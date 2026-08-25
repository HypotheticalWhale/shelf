// Package bgg is a client for the BoardGameGeek XML API2.
//
// Since late 2025 the API is closed: every request must carry a bearer token
// issued when you register an application with BGG. Unauthenticated calls
// return 401 regardless of user agent. Register at
// https://boardgamegeek.com/using_the_xml_api and set BGG_API_TOKEN.
//
// BGG permits non-commercial use with attribution. Shelf credits BoardGameGeek
// in its UI. Monetising the site would require a commercial licence from BGG.
//
// The API is easy to abuse, so this client is deliberately polite: requests are
// spaced by a minimum interval, IDs are batched, and the retry path honours the
// 202 "your request is queued" and 429 responses BGG uses to shed load.
package bgg

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	baseURL = "https://boardgamegeek.com/xmlapi2"
	// BatchSize is how many IDs go into one /thing call. BGG accepts up to 20.
	BatchSize = 20
	// minInterval spaces consecutive requests. BGG's published guidance is
	// roughly one request every couple of seconds for bulk work.
	minInterval = 2 * time.Second
	maxAttempts = 6
)

// ErrNoToken is returned when no bearer token is configured. It carries the
// registration URL so the failure is actionable rather than a bare 401.
var ErrNoToken = errors.New(
	"BGG_API_TOKEN is not set: the BoardGameGeek XML API has required a bearer " +
		"token since late 2025 — register an application at " +
		"https://boardgamegeek.com/using_the_xml_api and set BGG_API_TOKEN")

type Client struct {
	http  *http.Client
	token string

	mu       sync.Mutex
	lastCall time.Time
}

// NewClient builds a client authenticated with the given bearer token.
func NewClient(token string) *Client {
	return &Client{
		http:  &http.Client{Timeout: 45 * time.Second},
		token: strings.TrimSpace(token),
	}
}

// HasToken reports whether the client is able to make authenticated calls.
func (c *Client) HasToken() bool { return c.token != "" }

// throttle blocks until minInterval has elapsed since the previous request.
func (c *Client) throttle(ctx context.Context) error {
	c.mu.Lock()
	wait := time.Until(c.lastCall.Add(minInterval))
	c.lastCall = time.Now().Add(max(wait, 0))
	c.mu.Unlock()

	if wait <= 0 {
		return nil
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// get fetches a path with retry. BGG answers 202 while it builds a response and
// 429 when it wants you to slow down; both mean "try again shortly" rather than
// "this failed", so they are retried with a growing backoff.
func (c *Client) get(ctx context.Context, path string, query url.Values) ([]byte, error) {
	if c.token == "" {
		return nil, ErrNoToken
	}
	endpoint := baseURL + path + "?" + query.Encode()

	backoff := 3 * time.Second
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := c.throttle(ctx); err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("User-Agent", "Shelf/1.0 (+https://github.com/HypotheticalWhale/shelf)")
		req.Header.Set("Accept", "text/xml,application/xml")

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
		} else {
			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()

			switch {
			case resp.StatusCode == http.StatusOK:
				if readErr != nil {
					return nil, fmt.Errorf("read body: %w", readErr)
				}
				return body, nil
			case resp.StatusCode == http.StatusAccepted,
				resp.StatusCode == http.StatusTooManyRequests,
				resp.StatusCode >= 500:
				lastErr = fmt.Errorf("bgg returned %d", resp.StatusCode)
			case resp.StatusCode == http.StatusUnauthorized,
				resp.StatusCode == http.StatusForbidden:
				// Retrying will not fix a bad token; fail immediately.
				return nil, fmt.Errorf("bgg rejected the token (%d) — check BGG_API_TOKEN: %w",
					resp.StatusCode, ErrNoToken)
			case resp.StatusCode == http.StatusNotFound:
				return nil, fmt.Errorf("bgg returned 404 for %s", path)
			default:
				return nil, fmt.Errorf("bgg returned %d: %s", resp.StatusCode,
					strings.TrimSpace(string(body[:min(len(body), 200)])))
			}
		}

		if attempt == maxAttempts {
			break
		}
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
		backoff *= 2
	}
	return nil, fmt.Errorf("bgg request failed after %d attempts: %w", maxAttempts, lastErr)
}

// Things fetches full detail for up to BatchSize game IDs.
func (c *Client) Things(ctx context.Context, ids []int) ([]Game, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	if len(ids) > BatchSize {
		return nil, fmt.Errorf("at most %d ids per request, got %d", BatchSize, len(ids))
	}

	strIDs := make([]string, len(ids))
	for i, id := range ids {
		strIDs[i] = strconv.Itoa(id)
	}

	body, err := c.get(ctx, "/thing", url.Values{
		"id":    {strings.Join(strIDs, ",")},
		"stats": {"1"},
		"type":  {"boardgame"},
	})
	if err != nil {
		return nil, err
	}

	var doc thingResponse
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parse thing response: %w", err)
	}

	games := make([]Game, 0, len(doc.Items))
	for _, item := range doc.Items {
		// Expansions and accessories share the endpoint; keep base games only.
		if item.Type != "boardgame" {
			continue
		}
		games = append(games, item.toGame())
	}
	return games, nil
}

// Hot returns BGG's current hot list — roughly 50 games trending right now.
// It needs no ID knowledge, so it is what a fresh database is seeded from.
func (c *Client) Hot(ctx context.Context) ([]int, error) {
	body, err := c.get(ctx, "/hot", url.Values{"type": {"boardgame"}})
	if err != nil {
		return nil, err
	}

	var doc hotResponse
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parse hot response: %w", err)
	}

	ids := make([]int, 0, len(doc.Items))
	for _, item := range doc.Items {
		if item.ID > 0 {
			ids = append(ids, item.ID)
		}
	}
	return ids, nil
}
