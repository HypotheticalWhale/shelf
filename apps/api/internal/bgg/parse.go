package bgg

import (
	"html"
	"strconv"
	"strings"
)

// Game is one board game as Shelf consumes it, flattened from BGG's XML.
type Game struct {
	BGGID         int
	Name          string
	YearPublished *int
	Description   string
	ImageURL      string
	ThumbnailURL  string
	MinPlayers    *int
	MaxPlayers    *int
	MinPlaytime   *int
	MaxPlaytime   *int
	Weight        *float64
	Designers     []string
	Categories    []string
	Mechanics     []string

	// UsersRated is BGG's own rating count. Shelf never imports BGG's scores —
	// its rankings come from its own users — but this is a good popularity
	// filter when deciding which games are worth importing at all.
	UsersRated int
	Rank       int
}

type thingResponse struct {
	Items []thingItem `xml:"item"`
}

type thingItem struct {
	Type          string      `xml:"type,attr"`
	ID            int         `xml:"id,attr"`
	Thumbnail     string      `xml:"thumbnail"`
	Image         string      `xml:"image"`
	Names         []nameEl    `xml:"name"`
	Description   string      `xml:"description"`
	YearPublished *valueAttr  `xml:"yearpublished"`
	MinPlayers    *valueAttr  `xml:"minplayers"`
	MaxPlayers    *valueAttr  `xml:"maxplayers"`
	MinPlaytime   *valueAttr  `xml:"minplaytime"`
	MaxPlaytime   *valueAttr  `xml:"maxplaytime"`
	PlayingTime   *valueAttr  `xml:"playingtime"`
	Links         []linkEl    `xml:"link"`
	Statistics    *statistics `xml:"statistics"`
}

type nameEl struct {
	Type  string `xml:"type,attr"`
	Value string `xml:"value,attr"`
}

type linkEl struct {
	Type  string `xml:"type,attr"`
	Value string `xml:"value,attr"`
}

type valueAttr struct {
	Value string `xml:"value,attr"`
}

type statistics struct {
	Ratings struct {
		UsersRated    valueAttr `xml:"usersrated"`
		AverageWeight valueAttr `xml:"averageweight"`
		Ranks         struct {
			Rank []rankEl `xml:"rank"`
		} `xml:"ranks"`
	} `xml:"ratings"`
}

type rankEl struct {
	Type  string `xml:"type,attr"`
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type hotResponse struct {
	Items []struct {
		ID int `xml:"id,attr"`
	} `xml:"item"`
}

func (it thingItem) toGame() Game {
	g := Game{
		BGGID:         it.ID,
		Name:          it.primaryName(),
		Description:   cleanDescription(it.Description),
		ImageURL:      strings.TrimSpace(it.Image),
		ThumbnailURL:  strings.TrimSpace(it.Thumbnail),
		YearPublished: atoiPtr(it.YearPublished),
		MinPlayers:    atoiPtr(it.MinPlayers),
		MaxPlayers:    atoiPtr(it.MaxPlayers),
		MinPlaytime:   atoiPtr(it.MinPlaytime),
		MaxPlaytime:   atoiPtr(it.MaxPlaytime),
	}

	// BGG often leaves min/max playtime at zero while populating playingtime.
	if g.MaxPlaytime == nil || *g.MaxPlaytime == 0 {
		if pt := atoiPtr(it.PlayingTime); pt != nil && *pt > 0 {
			g.MaxPlaytime = pt
		}
	}
	if g.MinPlaytime == nil || *g.MinPlaytime == 0 {
		g.MinPlaytime = g.MaxPlaytime
	}

	for _, l := range it.Links {
		switch l.Type {
		case "boardgamedesigner":
			g.Designers = append(g.Designers, l.Value)
		case "boardgamecategory":
			g.Categories = append(g.Categories, l.Value)
		case "boardgamemechanic":
			g.Mechanics = append(g.Mechanics, l.Value)
		}
	}

	if it.Statistics != nil {
		r := it.Statistics.Ratings
		if n, err := strconv.Atoi(strings.TrimSpace(r.UsersRated.Value)); err == nil {
			g.UsersRated = n
		}
		if w, err := strconv.ParseFloat(strings.TrimSpace(r.AverageWeight.Value), 64); err == nil && w > 0 {
			g.Weight = &w
		}
		for _, rank := range r.Ranks.Rank {
			// "Not Ranked" is a legitimate value, so a parse failure here is
			// expected rather than exceptional.
			if rank.Name == "boardgame" {
				if v, err := strconv.Atoi(strings.TrimSpace(rank.Value)); err == nil {
					g.Rank = v
				}
			}
		}
	}
	return g
}

// primaryName picks the game's canonical title. BGG lists every localised name
// alongside it and the primary is not guaranteed to come first.
func (it thingItem) primaryName() string {
	for _, n := range it.Names {
		if n.Type == "primary" {
			return strings.TrimSpace(n.Value)
		}
	}
	if len(it.Names) > 0 {
		return strings.TrimSpace(it.Names[0].Value)
	}
	return ""
}

// cleanDescription undoes BGG's double-encoding. Descriptions arrive with
// entities that survive one XML decode pass — "&amp;#10;" becomes the literal
// text "&#10;" rather than a newline — so a second unescape is needed before
// the text is fit to render.
func cleanDescription(raw string) string {
	s := html.UnescapeString(raw)
	if strings.Contains(s, "&#") || strings.Contains(s, "&amp;") {
		s = html.UnescapeString(s)
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	// Collapse the long runs of blank lines BGG uses as section breaks.
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(s)
}

func atoiPtr(v *valueAttr) *int {
	if v == nil {
		return nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(v.Value))
	if err != nil {
		return nil
	}
	return &n
}
