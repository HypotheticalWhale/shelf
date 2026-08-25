// Package seed provides a small hand-curated catalogue.
//
// BoardGameGeek closed its XML API in late 2025 — it now needs a bearer token
// issued on application registration. Shelf ships this seed so the site is
// usable immediately, and so local development never depends on an external
// API being reachable.
//
// The data here is deliberately conservative: title, year, player count,
// playtime, designers and tags only. Complexity weights, descriptions and cover
// art are left empty rather than guessed, because a real BGG import fills them
// in accurately. Games are written with source='seed'; importing the same
// bgg_id from BGG promotes the row to source='bgg' and corrects every field.
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
			Designers:     g.Designers,
			Categories:    g.Categories,
			Mechanics:     g.Mechanics,
			Source:        "seed",
		})
	}
	return out, nil
}
