// Package seed provides a small hand-curated catalogue.
//
// BoardGameGeek closed its XML API in late 2025 — it now needs a bearer token
// issued on application registration. Shelf ships this seed so the site is
// usable immediately, and so local development never depends on an external
// API being reachable.
//
// Titles, years, player counts, playtimes, designers, categories and mechanics
// are accurate. Complexity weights follow BGG's 1-5 scale and are close
// community-consensus figures rather than exact ones — good enough to sort and
// filter by, and overwritten the moment a real BGG import runs.
//
// Cover art is deliberately absent. Board game box art is copyrighted and there
// is no freely-licensed bulk source; Wikipedia's covers are non-free fair-use
// files whose licence does not extend to this site. The UI draws a typographic
// cover instead, and real art arrives with a BGG token.
//
// Games are written with source='seed'; importing the same bgg_id from BGG
// promotes the row to source='bgg' and corrects every field.
package seed

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/HypotheticalWhale/shelf/apps/api/internal/store"
)

//go:embed games.json
var raw []byte

type seedGame struct {
	BGGID       int      `json:"bggId"`
	Name        string   `json:"name"`
	Year        *int     `json:"year"`
	MinPlayers  *int     `json:"minPlayers"`
	MaxPlayers  *int     `json:"maxPlayers"`
	MinPlaytime *int     `json:"minPlaytime"`
	MaxPlaytime *int     `json:"maxPlaytime"`
	Weight      *float64 `json:"weight"`
	Designers   []string `json:"designers"`
	Categories  []string `json:"categories"`
	Mechanics   []string `json:"mechanics"`
}

// Games returns the bundled catalogue ready for insertion.
func Games() ([]store.GameInput, error) {
	var parsed []seedGame
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("parse seed catalogue: %w", err)
	}

	out := make([]store.GameInput, 0, len(parsed))
	for _, g := range parsed {
		out = append(out, store.GameInput{
			BGGID:         g.BGGID,
			Name:          g.Name,
			YearPublished: g.Year,
			MinPlayers:    g.MinPlayers,
			MaxPlayers:    g.MaxPlayers,
			MinPlaytime:   g.MinPlaytime,
			MaxPlaytime:   g.MaxPlaytime,
			Weight:        g.Weight,
			Designers:     g.Designers,
			Categories:    g.Categories,
			Mechanics:     g.Mechanics,
			Source:        "seed",
		})
	}
	return out, nil
}
