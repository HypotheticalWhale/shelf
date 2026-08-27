package artwork

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	ogImageRe = regexp.MustCompile(`(?is)<meta[^>]+property=["']og:image["'][^>]*>`)
	contentRe = regexp.MustCompile(`(?is)content=["']([^"']+)["']`)
)

// PublisherSource reads the publisher's own product page.
//
// This is the best art available: the box shot the publisher chose to
// represent the game. There is no shared interface across publishers, so the
// page is found through Wikidata's "official website" property and the image
// through its OpenGraph tag — the one thing nearly every site agrees on,
// because social previews depend on it.
type PublisherSource struct {
	Facts map[int]Facts

	once   sync.Once
	client *http.Client
	// Publishers are small sites; one request at a time per host is polite.
	gate chan struct{}
}

func (s *PublisherSource) Name() string { return "publisher" }

func (s *PublisherSource) Find(ctx context.Context, g Game) (*Candidate, error) {
	s.once.Do(func() {
		s.client = &http.Client{Timeout: 20 * time.Second}
		s.gate = make(chan struct{}, 1)
	})

	f, ok := s.Facts[g.BGGID]
	if !ok || f.OfficialSite == "" {
		return nil, nil
	}

	select {
	case s.gate <- struct{}{}:
		defer func() { <-s.gate }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.OfficialSite, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned %d", f.OfficialSite, resp.StatusCode)
	}

	// Only the head is needed, and some product pages are enormous.
	head, err := io.ReadAll(io.LimitReader(resp.Body, 400<<10))
	if err != nil {
		return nil, err
	}

	tag := ogImageRe.Find(head)
	if tag == nil {
		return nil, nil
	}
	m := contentRe.FindSubmatch(tag)
	if m == nil {
		return nil, nil
	}

	img := strings.TrimSpace(string(m[1]))
	// Product pages routinely give a relative or protocol-relative URL.
	base, err := url.Parse(resp.Request.URL.String())
	if err != nil {
		return nil, err
	}
	ref, err := url.Parse(img)
	if err != nil {
		return nil, nil
	}
	abs := base.ResolveReference(ref).String()

	host := base.Hostname()
	return &Candidate{
		URL:     abs,
		Source:  "publisher",
		License: "Publisher promotional image",
		Credit:  host,
		Origin:  f.OfficialSite,
	}, nil
}
