package artwork

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Facts is what Wikidata knows about one game, keyed by BoardGameGeek id.
type Facts struct {
	CommonsImage string // P18, a freely-licensed file on Wikimedia Commons
	OfficialSite string // P856, usually the publisher's own product page
	Label        string
}

// LoadFacts fetches the whole mapping in one query.
//
// Wikidata holds a BoardGameGeek id for a few thousand games, so asking once
// and joining locally costs a single request instead of one per game — and
// keeps the pipeline polite enough to run over a full catalogue.
func LoadFacts(ctx context.Context) (map[int]Facts, error) {
	const query = `
SELECT ?bggId ?image ?site ?itemLabel WHERE {
  ?item wdt:P2339 ?bggId .
  OPTIONAL { ?item wdt:P18  ?image }
  OPTIONAL { ?item wdt:P856 ?site }
  SERVICE wikibase:label { bd:serviceParam wikibase:language "en". }
}`

	endpoint := "https://query.wikidata.org/sparql?" + url.Values{
		"query": {query}, "format": {"json"},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/sparql-results+json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := newHTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("query wikidata: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wikidata returned %d", resp.StatusCode)
	}

	var doc struct {
		Results struct {
			Bindings []map[string]struct {
				Value string `json:"value"`
			} `json:"bindings"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode wikidata: %w", err)
	}

	out := make(map[int]Facts, len(doc.Results.Bindings))
	for _, b := range doc.Results.Bindings {
		raw, ok := b["bggId"]
		if !ok {
			continue
		}
		id, err := strconv.Atoi(strings.TrimSpace(raw.Value))
		if err != nil {
			continue
		}
		f := out[id]
		if v, ok := b["image"]; ok {
			f.CommonsImage = v.Value
		}
		if v, ok := b["site"]; ok {
			f.OfficialSite = v.Value
		}
		if v, ok := b["itemLabel"]; ok {
			f.Label = v.Value
		}
		out[id] = f
	}
	return out, nil
}
