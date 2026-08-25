package bgg

import (
	"encoding/xml"
	"os"
	"strings"
	"testing"
)

func parseFixture(t *testing.T) []Game {
	t.Helper()

	raw, err := os.ReadFile("testdata/thing.xml")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var doc thingResponse
	if err := xml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var games []Game
	for _, item := range doc.Items {
		if item.Type != "boardgame" {
			continue
		}
		games = append(games, item.toGame())
	}
	return games
}

func TestParseFiltersNonBaseGames(t *testing.T) {
	games := parseFixture(t)
	if len(games) != 2 {
		t.Fatalf("got %d games, want 2 (the expansion must be dropped)", len(games))
	}
	for _, g := range games {
		if strings.Contains(g.Name, "Jaws of the Lion") {
			t.Error("expansion leaked into results")
		}
	}
}

func TestParsePicksPrimaryNameNotFirstName(t *testing.T) {
	// The fixture lists a Russian alternate name before the primary one, which
	// is exactly how BGG orders it for many titles.
	if got := parseFixture(t)[0].Name; got != "Gloomhaven" {
		t.Fatalf("name = %q, want %q", got, "Gloomhaven")
	}
}

func TestParseDecodesDoubleEncodedDescription(t *testing.T) {
	desc := parseFixture(t)[0].Description
	if strings.Contains(desc, "&#10;") || strings.Contains(desc, "&amp;") {
		t.Fatalf("description still holds encoded entities: %q", desc)
	}
	if !strings.Contains(desc, "mercenary") {
		t.Fatalf("description lost content: %q", desc)
	}
	if !strings.Contains(desc, "&") {
		t.Fatalf("ampersand should decode to a literal &: %q", desc)
	}
	if !strings.Contains(desc, "\n") {
		t.Fatalf("newline entities should decode to real newlines: %q", desc)
	}
}

func TestParseFallsBackToPlayingTime(t *testing.T) {
	// BGG reports 0/0 for min and max playtime here while playingtime is 120.
	g := parseFixture(t)[0]
	if g.MaxPlaytime == nil || *g.MaxPlaytime != 120 {
		t.Fatalf("MaxPlaytime = %v, want 120 from the playingtime fallback", g.MaxPlaytime)
	}
	if g.MinPlaytime == nil || *g.MinPlaytime != 120 {
		t.Fatalf("MinPlaytime = %v, want 120", g.MinPlaytime)
	}
}

func TestParseScalarsAndLinks(t *testing.T) {
	g := parseFixture(t)[0]

	if g.BGGID != 174430 {
		t.Errorf("BGGID = %d", g.BGGID)
	}
	if g.YearPublished == nil || *g.YearPublished != 2017 {
		t.Errorf("YearPublished = %v", g.YearPublished)
	}
	if g.Weight == nil || *g.Weight != 3.9 {
		t.Errorf("Weight = %v", g.Weight)
	}
	if g.UsersRated != 61234 {
		t.Errorf("UsersRated = %d", g.UsersRated)
	}
	if g.Rank != 2 {
		t.Errorf("Rank = %d, want 2 from the boardgame subtype", g.Rank)
	}
	if len(g.Categories) != 2 || g.Categories[0] != "Adventure" {
		t.Errorf("Categories = %v", g.Categories)
	}
	if len(g.Designers) != 1 || g.Designers[0] != "Isaac Childres" {
		t.Errorf("Designers = %v", g.Designers)
	}
	if len(g.Mechanics) != 1 {
		t.Errorf("Mechanics = %v", g.Mechanics)
	}
	// Publishers are not imported, so they must not leak into another slice.
	for _, c := range g.Categories {
		if c == "Cephalofair Games" {
			t.Error("publisher leaked into categories")
		}
	}
}

func TestParseHandlesNotRankedAndZeroWeight(t *testing.T) {
	g := parseFixture(t)[1]

	if g.Rank != 0 {
		t.Errorf(`Rank = %d, want 0 for "Not Ranked"`, g.Rank)
	}
	if g.Weight != nil {
		t.Errorf("Weight = %v, want nil when BGG reports 0", g.Weight)
	}
	if g.MaxPlaytime != nil {
		t.Errorf("MaxPlaytime = %v, want nil when absent", g.MaxPlaytime)
	}
}
