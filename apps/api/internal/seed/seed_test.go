package seed

import "testing"

func TestSeedCatalogueIsWellFormed(t *testing.T) {
	games, err := Games()
	if err != nil {
		t.Fatalf("Games: %v", err)
	}
	if len(games) < 50 {
		t.Fatalf("seed catalogue holds only %d games", len(games))
	}

	seenID := map[int]string{}
	seenName := map[string]bool{}

	for _, g := range games {
		if g.BGGID <= 0 {
			t.Errorf("%s has a non-positive bgg id", g.Name)
		}
		if prev, dup := seenID[g.BGGID]; dup {
			t.Errorf("bgg id %d used by both %q and %q", g.BGGID, prev, g.Name)
		}
		seenID[g.BGGID] = g.Name

		if seenName[g.Name] {
			t.Errorf("duplicate title %q", g.Name)
		}
		seenName[g.Name] = true

		if g.Source != "seed" {
			t.Errorf("%s has source %q, want seed", g.Name, g.Source)
		}
		if g.MinPlayers != nil && g.MaxPlayers != nil && *g.MinPlayers > *g.MaxPlayers {
			t.Errorf("%s has min players above max", g.Name)
		}
		if g.MinPlaytime != nil && g.MaxPlaytime != nil && *g.MinPlaytime > *g.MaxPlaytime {
			t.Errorf("%s has min playtime above max", g.Name)
		}
		if len(g.Designers) == 0 {
			t.Errorf("%s lists no designer", g.Name)
		}
		if g.Weight == nil {
			t.Errorf("%s has no complexity weight", g.Name)
		} else if *g.Weight < 1 || *g.Weight > 5 {
			t.Errorf("%s has weight %v, outside BGG's 1-5 scale", g.Name, *g.Weight)
		}
		if len(g.Categories) == 0 {
			t.Errorf("%s has no categories", g.Name)
		}
		if len(g.Mechanics) == 0 {
			t.Errorf("%s has no mechanics", g.Name)
		}
	}
}
