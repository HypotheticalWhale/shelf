package store

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// GameInput is a game ready to be written. It mirrors what the BGG importer
// produces but is declared here so the store never depends on the bgg package.
type GameInput struct {
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

	// Source is "bgg" for imported games or "seed" for the bundled catalogue.
	// Empty means "bgg".
	Source string
}

// UpsertGames writes games keyed on bgg_id, so re-running an import refreshes
// metadata instead of duplicating rows. Ratings and stats are untouched.
//
// Returns the number of rows written.
func (s *Store) UpsertGames(ctx context.Context, games []GameInput) (int, error) {
	written := 0
	for _, g := range games {
		if g.BGGID <= 0 || g.Name == "" {
			continue
		}
		if err := s.upsertGame(ctx, g); err != nil {
			return written, err
		}
		written++
	}
	return written, nil
}

func (s *Store) upsertGame(ctx context.Context, g GameInput) error {
	const sql = `
		INSERT INTO games (
			bgg_id, slug, name, year_published, description,
			image_url, thumbnail_url,
			min_players, max_players, min_playtime, max_playtime, weight,
			designers, categories, mechanics, source, imported_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16, now())
		ON CONFLICT (bgg_id) DO UPDATE SET
			name           = EXCLUDED.name,
			year_published = EXCLUDED.year_published,
			description    = EXCLUDED.description,
			image_url      = EXCLUDED.image_url,
			thumbnail_url  = EXCLUDED.thumbnail_url,
			min_players    = EXCLUDED.min_players,
			max_players    = EXCLUDED.max_players,
			min_playtime   = EXCLUDED.min_playtime,
			max_playtime   = EXCLUDED.max_playtime,
			weight         = EXCLUDED.weight,
			designers      = EXCLUDED.designers,
			categories     = EXCLUDED.categories,
			mechanics      = EXCLUDED.mechanics,
			source         = EXCLUDED.source,
			imported_at    = now()`

	base := Slugify(g.Name)

	// Distinct games share titles ("Crokinole", reimplementations, regional
	// editions). Fall back to the year, then to the BGG ID, which is unique by
	// construction — so a collision can never fail an import.
	candidates := []string{base}
	if g.YearPublished != nil && *g.YearPublished > 0 {
		candidates = append(candidates, fmt.Sprintf("%s-%d", base, *g.YearPublished))
	}
	candidates = append(candidates, base+"-"+strconv.Itoa(g.BGGID))

	nullable := func(s string) *string {
		if s == "" {
			return nil
		}
		return &s
	}

	source := g.Source
	if source == "" {
		source = "bgg"
	}

	var lastErr error
	for _, slug := range candidates {
		_, err := s.pool.Exec(ctx, sql,
			g.BGGID, slug, g.Name, g.YearPublished, nullable(g.Description),
			nullable(g.ImageURL), nullable(g.ThumbnailURL),
			g.MinPlayers, g.MaxPlayers, g.MinPlaytime, g.MaxPlaytime, g.Weight,
			g.Designers, g.Categories, g.Mechanics, source,
		)
		if err == nil {
			return nil
		}
		if isUniqueViolation(err, "games_slug_key") {
			lastErr = err
			continue // slug held by a different game; try the next candidate
		}
		return fmt.Errorf("upsert game %d (%s): %w", g.BGGID, g.Name, err)
	}
	return fmt.Errorf("upsert game %d (%s): could not allocate a slug: %w", g.BGGID, g.Name, lastErr)
}

// ExistingBGGIDs returns the subset of ids already present, so a sweep can skip
// games it has already imported.
func (s *Store) ExistingBGGIDs(ctx context.Context, ids []int) (map[int]bool, error) {
	out := make(map[int]bool, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	rows, err := s.pool.Query(ctx, `SELECT bgg_id FROM games WHERE bgg_id = ANY($1)`, ids)
	if err != nil {
		return nil, fmt.Errorf("existing ids: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

// CountGames reports how many games are in the catalogue.
func (s *Store) CountGames(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM games`).Scan(&n)
	return n, err
}

// StaleBGGIDs returns the games whose metadata was refreshed longest ago, so a
// periodic job can keep the catalogue current without rescanning everything.
func (s *Store) StaleBGGIDs(ctx context.Context, limit int) ([]int, error) {
	if limit <= 0 || limit > 5000 {
		limit = 200
	}

	rows, err := s.pool.Query(ctx,
		`SELECT bgg_id FROM games ORDER BY imported_at ASC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("stale ids: %w", err)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// DeleteSeedGames removes catalogue rows that came from the bundled seed and
// were never reconciled against BGG.
//
// A real import promotes a seeded row to source='bgg' by bgg_id, so anything
// still marked 'seed' afterwards is a game BGG did not return — stale
// placeholder data worth clearing. Ratings and posts referencing it cascade,
// so this is deliberately a manual step rather than part of every import.
func (s *Store) DeleteSeedGames(ctx context.Context) (int, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM games WHERE source = 'seed'`)
	if err != nil {
		return 0, fmt.Errorf("delete seed games: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// InsertNewGames adds games that are not in the catalogue yet, leaving existing
// rows completely untouched.
//
// This is the bulk path: tens of thousands of rows, where the per-row upsert
// used by the BGG importer would mean tens of thousands of round trips. Slugs
// are made unique in Go against the slugs already in the database, so a batch
// cannot fail on a collision, and the insert still carries ON CONFLICT DO
// NOTHING so a concurrent run is harmless.
//
// Existing rows are skipped rather than updated on purpose: the hand-curated
// games carry better tagging and complexity weights than a bulk snapshot, and
// a broad import must not overwrite them.
func (s *Store) InsertNewGames(ctx context.Context, games []GameInput, progress func(done, total int)) (int, error) {
	existingIDs, existingSlugs, err := s.catalogueKeys(ctx)
	if err != nil {
		return 0, err
	}

	type row struct {
		g    GameInput
		slug string
	}

	pending := make([]row, 0, len(games))
	for _, g := range games {
		if g.BGGID <= 0 || g.Name == "" || existingIDs[g.BGGID] {
			continue
		}

		base := Slugify(g.Name)
		slug := base
		if existingSlugs[slug] {
			if g.YearPublished != nil && *g.YearPublished > 0 {
				slug = fmt.Sprintf("%s-%d", base, *g.YearPublished)
			}
			if existingSlugs[slug] {
				slug = base + "-" + strconv.Itoa(g.BGGID)
			}
		}
		if existingSlugs[slug] {
			continue // three collisions on one title is not worth a fourth guess
		}

		existingSlugs[slug] = true
		existingIDs[g.BGGID] = true
		pending = append(pending, row{g: g, slug: slug})
	}

	const chunk = 400
	written := 0

	for start := 0; start < len(pending); start += chunk {
		end := min(start+chunk, len(pending))
		batch := pending[start:end]

		var sb strings.Builder
		sb.WriteString(`INSERT INTO games (bgg_id, slug, name, year_published,
			min_players, max_players, min_playtime, max_playtime,
			designers, categories, mechanics, source, imported_at) VALUES `)

		args := make([]any, 0, len(batch)*12)
		for i, r := range batch {
			if i > 0 {
				sb.WriteString(",")
			}
			n := i * 12
			fmt.Fprintf(&sb, "($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d, now())",
				n+1, n+2, n+3, n+4, n+5, n+6, n+7, n+8, n+9, n+10, n+11, n+12)

			source := r.g.Source
			if source == "" {
				source = "seed"
			}
			args = append(args,
				r.g.BGGID, r.slug, r.g.Name, r.g.YearPublished,
				r.g.MinPlayers, r.g.MaxPlayers, r.g.MinPlaytime, r.g.MaxPlaytime,
				// These columns are NOT NULL DEFAULT '{}'; a nil Go slice sends
				// NULL, not an empty array, so most games would be rejected.
				emptyIfNil(r.g.Designers), emptyIfNil(r.g.Categories), emptyIfNil(r.g.Mechanics),
				source,
			)
		}
		sb.WriteString(" ON CONFLICT DO NOTHING")

		tag, err := s.pool.Exec(ctx, sb.String(), args...)
		if err != nil {
			return written, fmt.Errorf("insert games %d-%d: %w", start, end, err)
		}
		written += int(tag.RowsAffected())

		if progress != nil {
			progress(end, len(pending))
		}
	}
	return written, nil
}

// catalogueKeys loads the bgg ids and slugs already in use.
func (s *Store) catalogueKeys(ctx context.Context) (map[int]bool, map[string]bool, error) {
	rows, err := s.pool.Query(ctx, `SELECT bgg_id, slug FROM games`)
	if err != nil {
		return nil, nil, fmt.Errorf("read catalogue keys: %w", err)
	}
	defer rows.Close()

	ids := make(map[int]bool, 4096)
	slugs := make(map[string]bool, 4096)
	for rows.Next() {
		var id int
		var slug string
		if err := rows.Scan(&id, &slug); err != nil {
			return nil, nil, err
		}
		ids[id] = true
		slugs[slug] = true
	}
	return ids, slugs, rows.Err()
}

// emptyIfNil keeps a nil slice from being written as SQL NULL.
func emptyIfNil(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}
