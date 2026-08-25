package seed

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Cross-checks every seeded bggId against Wikidata's BoardGameGeek ID property.
//
// This matters because the BGG importer keys on bgg_id: a wrong id would not
// fail loudly, it would quietly overwrite a game with a different game's title,
// art and weight. Wikidata is an independent source for that mapping.
//
// Opt-in so the suite never depends on the network:
//
//	SEED_VERIFY=1 go test ./internal/seed -run Wikidata -v
func TestSeedIDsAgainstWikidata(t *testing.T) {
	if os.Getenv("SEED_VERIFY") == "" {
		t.Skip("set SEED_VERIFY=1 to cross-check seed ids against Wikidata")
	}

	games, err := Games()
	if err != nil {
		t.Fatalf("load seed: %v", err)
	}

	const query = `SELECT ?bggId ?itemLabel WHERE {
		?item wdt:P2339 ?bggId .
		SERVICE wikibase:label { bd:serviceParam wikibase:language "en". }
	}`

	endpoint := "https://query.wikidata.org/sparql?" + url.Values{
		"query": {query}, "format": {"json"},
	}.Encode()

	req, _ := http.NewRequest(http.MethodGet, endpoint, nil)
	req.Header.Set("Accept", "application/sparql-results+json")
	req.Header.Set("User-Agent", "Shelf/1.0 (https://github.com/HypotheticalWhale/shelf)")

	resp, err := (&http.Client{Timeout: 2 * time.Minute}).Do(req)
	if err != nil {
		t.Fatalf("query wikidata: %v", err)
	}
	defer resp.Body.Close()

	var doc struct {
		Results struct {
			Bindings []struct {
				BGGID     struct{ Value string } `json:"bggId"`
				ItemLabel struct{ Value string } `json:"itemLabel"`
			} `json:"bindings"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		t.Fatalf("decode: %v", err)
	}

	known := make(map[int]string, len(doc.Results.Bindings))
	for _, b := range doc.Results.Bindings {
		if id, err := strconv.Atoi(b.BGGID.Value); err == nil {
			known[id] = b.ItemLabel.Value
		}
	}
	if len(known) < 1000 {
		t.Fatalf("wikidata returned only %d ids — query is probably wrong", len(known))
	}
	t.Logf("wikidata knows %d BoardGameGeek ids", len(known))

	var checked, absent int
	for _, g := range games {
		label, ok := known[g.BGGID]
		if !ok {
			// Not every edition is catalogued; absence is not a failure.
			absent++
			t.Logf("not in wikidata: %d %s", g.BGGID, g.Name)
			continue
		}
		checked++

		// Only English labels are requested, so an item Wikidata has no English
		// label for comes back as its Q-id. Comparing against that proves
		// nothing — several of these games are only labelled in Czech, Japanese
		// or Ukrainian — so skip rather than report a false conflict.
		if strings.HasPrefix(label, "Q") && len(label) > 1 && label[1] >= '0' && label[1] <= '9' {
			continue
		}
		if !similar(g.Name, label) {
			t.Errorf("bggId %d is %q on wikidata but %q in the seed", g.BGGID, label, g.Name)
		}
	}
	t.Logf("cross-checked %d ids (%d absent from wikidata)", checked, absent)
}

func similar(a, b string) bool {
	na, nb := alnum(a), alnum(b)
	return na == nb || strings.Contains(na, nb) || strings.Contains(nb, na)
}

func alnum(s string) string {
	var out strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
		}
	}
	return out.String()
}
