package bgg

import (
	"context"
	"os"
	"testing"
	"time"
)

// Opt-in check against the real API, so CI never depends on BGG being up:
//
//	BGG_LIVE=1 go test ./internal/bgg -run Live -v
func TestLiveFetch(t *testing.T) {
	if os.Getenv("BGG_LIVE") == "" {
		t.Skip("set BGG_LIVE=1 to exercise the real BoardGameGeek API")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	c := NewClient(os.Getenv("BGG_API_TOKEN"))
	if !c.HasToken() {
		t.Skip("set BGG_API_TOKEN to exercise the real BoardGameGeek API")
	}

	hot, err := c.Hot(ctx)
	if err != nil {
		t.Fatalf("Hot: %v", err)
	}
	if len(hot) == 0 {
		t.Fatal("hot list came back empty")
	}
	t.Logf("hot list: %d ids, first=%d", len(hot), hot[0])

	games, err := c.Things(ctx, hot[:min(5, len(hot))])
	if err != nil {
		t.Fatalf("Things: %v", err)
	}
	if len(games) == 0 {
		t.Fatal("no games parsed from a real response")
	}

	for _, g := range games {
		if g.BGGID == 0 || g.Name == "" {
			t.Errorf("game parsed with empty identity: %+v", g)
		}
		t.Logf("%-45s id=%-7d year=%v players=%v-%v weight=%v rated=%d",
			g.Name, g.BGGID, deref(g.YearPublished),
			deref(g.MinPlayers), deref(g.MaxPlayers), derefF(g.Weight), g.UsersRated)
	}
}

func deref(p *int) any {
	if p == nil {
		return "-"
	}
	return *p
}

func derefF(p *float64) any {
	if p == nil {
		return "-"
	}
	return *p
}
