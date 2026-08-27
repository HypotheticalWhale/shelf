// Package artwork aggregates box art from several sources and re-hosts it.
//
//	sources → resolver → download → blob storage → catalogue
//
// No single source covers the hobby. BoardGameGeek has the art but gates its
// API; publishers have the originals but no shared interface; Wikimedia has a
// freely-licensed subset. The resolver tries them in order of quality and takes
// the first that answers, recording which one won so every image can be
// credited and audited later.
package artwork

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Candidate is one source's answer for a game.
type Candidate struct {
	URL     string // where to fetch the image from
	Source  string // "bgg" | "publisher" | "commons"
	License string // best-known licence for the image
	Credit  string // attribution to display
	Origin  string // the page the image was found on, for auditing
}

// Source finds artwork for a game. Returning a nil candidate means "no opinion"
// rather than an error, so one quiet source never stops the pipeline.
type Source interface {
	Name() string
	Find(ctx context.Context, g Game) (*Candidate, error)
}

// Game is the little the sources need to know.
type Game struct {
	BGGID int
	Slug  string
	Name  string
	Year  *int
}

// Resolver asks each source in turn and keeps the first answer.
type Resolver struct {
	Sources []Source
	Logf    func(string, ...any)
}

func (r *Resolver) logf(format string, args ...any) {
	if r.Logf != nil {
		r.Logf(format, args...)
	}
}

// Resolve returns the best available candidate, or nil if nobody has one.
func (r *Resolver) Resolve(ctx context.Context, g Game) *Candidate {
	for _, s := range r.Sources {
		c, err := s.Find(ctx, g)
		if err != nil {
			r.logf("  %s failed for %s: %v", s.Name(), g.Slug, err)
			continue
		}
		if c != nil && c.URL != "" {
			return c
		}
	}
	return nil
}

// Fetch downloads an image, rejecting anything that is not one or is too small
// to be a cover. A 200 response is not proof of a picture: sites answer with
// HTML error pages, tracking pixels and placeholder graphics.
func Fetch(ctx context.Context, client *http.Client, url string) (body []byte, contentType string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "image/*")

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("returned %d", resp.StatusCode)
	}

	contentType = resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return nil, "", fmt.Errorf("not an image (%s)", contentType)
	}
	if strings.Contains(contentType, "svg") {
		return nil, "", fmt.Errorf("svg is not usable as a cover")
	}

	// 12 MB is generous for a box shot and stops a stray asset from being
	// pulled into memory.
	body, err = io.ReadAll(io.LimitReader(resp.Body, 12<<20))
	if err != nil {
		return nil, "", err
	}
	if len(body) < 6000 {
		return nil, "", fmt.Errorf("only %d bytes; too small to be cover art", len(body))
	}
	return body, contentType, nil
}

const userAgent = "Shelf/1.0 (+https://github.com/HypotheticalWhale/shelf) artwork aggregator"

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: 45 * time.Second}
}
