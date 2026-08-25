// Package catalogue builds a broad game list from published BoardGameGeek
// snapshots, for use while a BGG API token is pending.
//
// Two public datasets are combined:
//
//   - beefsack/bgg-ranking-historicals — a daily CSV of every ranked game.
//     Current and broad (30k+ entries) but thin: id, name, year, rank.
//   - a 2016 metadata snapshot — richer per game (player counts, playtime,
//     categories, mechanics, designers) but stops in 2016.
//
// The rank file supplies breadth and recency; the older file fills in detail
// where the two overlap. Everything is keyed by BGG id, so a later import with
// a real token reconciles every row in place rather than duplicating it.
//
// Deliberately not imported:
//
//   - BGG's own ratings and averages. Shelf's scores are its own; borrowing
//     another site's numbers would make the whole ranking meaningless.
//   - Descriptions. Those are copyrighted text written by BGG's members.
//   - Full-size images. The rank file carries only a 64x64 thumbnail and the
//     CDN's transforms are HMAC-signed, so no larger variant can be derived
//     from one. Those thumbnails are imported for colour and recognition, and
//     real box art arrives with a BGG token.
package catalogue

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	rankedBase   = "https://raw.githubusercontent.com/beefsack/bgg-ranking-historicals/master/"
	metadataURL  = "https://raw.githubusercontent.com/lunadu321/bgg_dataset/main/data/board_games_2023.csv"
	lookBackDays = 90
)

// Entry is one ranked game.
type Entry struct {
	BGGID int
	Name  string
	Year  *int
	Rank  int
	// Thumbnail is BGG's 64x64 cover crop. The CDN signs its transforms, so no
	// larger variant can be derived from it — it is a colour and shape cue, not
	// a substitute for real box art.
	Thumbnail string
}

// Extra is the richer metadata available for older games.
type Extra struct {
	MinPlayers  *int
	MaxPlayers  *int
	MinPlaytime *int
	MaxPlaytime *int
	Categories  []string
	Mechanics   []string
	Designers   []string
}

type Client struct {
	http *http.Client
	Logf func(string, ...any)
}

func NewClient() *Client {
	return &Client{http: &http.Client{Timeout: 5 * time.Minute}}
}

func (c *Client) logf(format string, args ...any) {
	if c.Logf != nil {
		c.Logf(format, args...)
	}
}

// FetchRanked downloads the most recent rankings snapshot.
//
// The files are named by date and published irregularly, so this walks back
// from today until one exists rather than pinning a date that goes stale.
func (c *Client) FetchRanked(ctx context.Context) ([]Entry, error) {
	var body io.ReadCloser
	var used string

	for i := 0; i < lookBackDays; i++ {
		day := time.Now().UTC().AddDate(0, 0, -i).Format("2006-01-02")
		url := rankedBase + day + ".csv"

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetch rankings: %w", err)
		}
		if resp.StatusCode == http.StatusOK {
			body, used = resp.Body, day
			break
		}
		resp.Body.Close()
	}
	if body == nil {
		return nil, fmt.Errorf("no rankings snapshot found in the last %d days", lookBackDays)
	}
	defer body.Close()
	c.logf("using rankings snapshot %s", used)

	r := csv.NewReader(body)
	r.FieldsPerRecord = -1
	head, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	col := index(head)

	var out []Entry
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return out, fmt.Errorf("read rankings: %w", err)
		}

		id := atoi(get(rec, col, "ID"))
		name := strings.TrimSpace(get(rec, col, "Name"))
		if id <= 0 || name == "" {
			continue
		}
		out = append(out, Entry{
			BGGID:     id,
			Name:      name,
			Year:      parseYear(get(rec, col, "Year")),
			Rank:      atoi(get(rec, col, "Rank")),
			Thumbnail: strings.TrimSpace(get(rec, col, "Thumbnail")),
		})
	}
	return out, nil
}

// FetchExtras downloads the older, richer metadata snapshot.
func (c *Client) FetchExtras(ctx context.Context) (map[int]Extra, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metadataURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch metadata: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata snapshot returned %d", resp.StatusCode)
	}

	r := csv.NewReader(resp.Body)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	head, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	col := index(head)

	out := make(map[int]Extra, 12000)
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// One malformed row should not discard thousands of good ones.
			continue
		}

		id := atoi(get(rec, col, "game_id"))
		if id <= 0 {
			continue
		}

		e := Extra{
			MinPlayers:  atoiPtr(get(rec, col, "min_players")),
			MaxPlayers:  atoiPtr(get(rec, col, "max_players")),
			MinPlaytime: atoiPtr(get(rec, col, "min_playtime")),
			MaxPlaytime: atoiPtr(get(rec, col, "max_playtime")),
			Categories:  splitList(get(rec, col, "category")),
			Mechanics:   splitList(get(rec, col, "mechanic")),
			Designers:   splitList(get(rec, col, "designer")),
		}
		// The dataset often leaves min/max playtime at zero but fills
		// playing_time.
		if e.MaxPlaytime == nil {
			e.MaxPlaytime = atoiPtr(get(rec, col, "playing_time"))
		}
		if e.MinPlaytime == nil {
			e.MinPlaytime = e.MaxPlaytime
		}
		out[id] = e
	}
	return out, nil
}

func index(head []string) map[string]int {
	m := make(map[string]int, len(head))
	for i, h := range head {
		m[strings.TrimSpace(strings.TrimPrefix(h, "\ufeff"))] = i
	}
	return m
}

func get(rec []string, col map[string]int, name string) string {
	i, ok := col[name]
	if !ok || i >= len(rec) {
		return ""
	}
	return rec[i]
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

// parseYear reads a publication year, repairing the snapshot's BC entries.
//
// The rankings file stores ancient games as positive years — Senet as 3500,
// Backgammon as 3000 — having lost BGG's minus sign somewhere upstream. Left
// alone they sort as the newest games on the site. Nothing is published in the
// fourth millennium, so anything comfortably beyond the present is BC.
func parseYear(s string) *int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n == 0 {
		return nil
	}
	if n > time.Now().UTC().Year()+5 {
		n = -n
	}
	return &n
}

func atoiPtr(s string) *int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n <= 0 {
		return nil
	}
	return &n
}

func splitList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}
