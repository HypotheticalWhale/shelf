package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/samueltansats/shelf/apps/api/internal/rating"
)

// GameFilter describes a browse query. Zero values mean "no constraint".
type GameFilter struct {
	Query     string
	Players   int
	MaxTime   int
	MinWeight float64
	MaxWeight float64
	Sort      string
	Limit     int
	Offset    int
	ViewerID  string
}

// GamePage is one page of browse results.
type GamePage struct {
	Games []Game `json:"games"`
	Total int    `json:"total"`
	Prior Prior  `json:"prior"`
}

// bayesOrder is the Bayesian score expressed in SQL, used only for ORDER BY.
// The value returned to clients is computed in Go by rating.Score from the same
// inputs, so the tested implementation is the one users see.
const bayesOrder = `((COALESCE(gs.rating_sum,0) + gl.prior_weight * gl.mean_rating)
                     / NULLIF(COALESCE(gs.num_ratings,0) + gl.prior_weight, 0))`

// ListGames runs a filtered, sorted, paginated browse query.
func (s *Store) ListGames(ctx context.Context, f GameFilter) (GamePage, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 24
	}
	if f.Offset < 0 {
		f.Offset = 0
	}

	// ph appends an argument and returns its placeholder. Returning the token
	// rather than renumbering a template keeps clauses that reference the same
	// argument twice (player counts, the trigram search) correct.
	var where []string
	var args []any
	ph := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if q := strings.TrimSpace(f.Query); q != "" {
		// Trigram similarity tolerates the typos and partial titles people
		// actually type; the ILIKE arm keeps exact substrings matching too.
		p := ph(q)
		where = append(where, fmt.Sprintf(
			"(g.name ILIKE '%%' || %s || '%%' OR g.name %% %s)", p, p))
	}
	if f.Players > 0 {
		p := ph(f.Players)
		where = append(where, fmt.Sprintf(
			"(g.min_players <= %s AND g.max_players >= %s)", p, p))
	}
	if f.MaxTime > 0 {
		where = append(where, fmt.Sprintf(
			"(COALESCE(g.max_playtime, g.min_playtime) <= %s)", ph(f.MaxTime)))
	}
	if f.MinWeight > 0 {
		where = append(where, fmt.Sprintf("(g.weight >= %s)", ph(f.MinWeight)))
	}
	if f.MaxWeight > 0 {
		where = append(where, fmt.Sprintf("(g.weight <= %s)", ph(f.MaxWeight)))
	}

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	orderSQL := bayesOrder + " DESC NULLS LAST, gs.num_ratings DESC"
	switch f.Sort {
	case "popular":
		orderSQL = "gs.num_ratings DESC NULLS LAST, " + bayesOrder + " DESC"
	case "new":
		orderSQL = "g.year_published DESC NULLS LAST, " + bayesOrder + " DESC"
	case "name":
		orderSQL = "g.name ASC"
	}

	viewerIdx := len(args) + 1
	args = append(args, f.ViewerID, f.Limit, f.Offset)

	sql := fmt.Sprintf(`
		SELECT g.id, g.bgg_id, g.slug, g.name, g.year_published,
		       g.image_url, g.thumbnail_url,
		       g.min_players, g.max_players, g.min_playtime, g.max_playtime, g.weight,
		       COALESCE(gs.num_ratings, 0), COALESCE(gs.rating_sum, 0),
		       r.value,
		       gl.mean_rating, gl.prior_weight,
		       COUNT(*) OVER () AS total
		  FROM games g
		  LEFT JOIN game_stats gs ON gs.game_id = g.id
		  CROSS JOIN global_stats gl
		  LEFT JOIN ratings r ON r.game_id = g.id AND r.user_id = $%d
		  %s
		 ORDER BY %s
		 LIMIT $%d OFFSET $%d`,
		viewerIdx, whereSQL, orderSQL, viewerIdx+1, viewerIdx+2)

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return GamePage{}, fmt.Errorf("list games: %w", err)
	}
	defer rows.Close()

	page := GamePage{Games: []Game{}}
	for rows.Next() {
		var g Game
		var prior Prior
		var total int
		if err := rows.Scan(
			&g.ID, &g.BGGID, &g.Slug, &g.Name, &g.YearPublished,
			&g.ImageURL, &g.ThumbnailURL,
			&g.MinPlayers, &g.MaxPlayers, &g.MinPlaytime, &g.MaxPlaytime, &g.Weight,
			&g.NumRatings, &g.RatingSum,
			&g.ViewerRating,
			&prior.MeanRating, &prior.PriorWeight,
			&total,
		); err != nil {
			return GamePage{}, fmt.Errorf("scan game: %w", err)
		}
		g.Score = rating.Score(g.RatingSum, g.NumRatings, prior.MeanRating, prior.PriorWeight)
		g.Mean = rating.Mean(g.RatingSum, g.NumRatings)
		page.Games = append(page.Games, g)
		page.Total = total
		page.Prior = prior
	}
	if err := rows.Err(); err != nil {
		return GamePage{}, fmt.Errorf("list games: %w", err)
	}

	if page.Total == 0 {
		if prior, err := s.Prior(ctx); err == nil {
			page.Prior = prior
		}
	}
	return page, nil
}

// GetGameBySlug returns one game with full detail.
func (s *Store) GetGameBySlug(ctx context.Context, slug, viewerID string) (Game, error) {
	return getGameBySlug(ctx, s.pool, slug, viewerID)
}

// getGameBySlug runs against either the pool or an open transaction, so a write
// can read back the freshly-updated aggregates without leaving its transaction.
func getGameBySlug(ctx context.Context, q querier, slug, viewerID string) (Game, error) {
	const sql = `
		SELECT g.id, g.bgg_id, g.slug, g.name, g.year_published, g.description,
		       g.image_url, g.thumbnail_url,
		       g.min_players, g.max_players, g.min_playtime, g.max_playtime, g.weight,
		       g.designers, g.categories, g.mechanics,
		       COALESCE(gs.num_ratings, 0), COALESCE(gs.rating_sum, 0),
		       r.value,
		       gl.mean_rating, gl.prior_weight,
		       COALESCE(
		           (SELECT array_agg(si.status ORDER BY si.status)
		              FROM shelf_items si
		             WHERE si.game_id = g.id AND si.user_id = $2),
		           '{}'
		       )
		  FROM games g
		  LEFT JOIN game_stats gs ON gs.game_id = g.id
		  CROSS JOIN global_stats gl
		  LEFT JOIN ratings r ON r.game_id = g.id AND r.user_id = $2
		 WHERE g.slug = $1`

	var g Game
	var prior Prior
	err := q.QueryRow(ctx, sql, slug, viewerID).Scan(
		&g.ID, &g.BGGID, &g.Slug, &g.Name, &g.YearPublished, &g.Description,
		&g.ImageURL, &g.ThumbnailURL,
		&g.MinPlayers, &g.MaxPlayers, &g.MinPlaytime, &g.MaxPlaytime, &g.Weight,
		&g.Designers, &g.Categories, &g.Mechanics,
		&g.NumRatings, &g.RatingSum,
		&g.ViewerRating,
		&prior.MeanRating, &prior.PriorWeight,
		&g.ViewerShelf,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Game{}, ErrNotFound
	}
	if err != nil {
		return Game{}, fmt.Errorf("get game: %w", err)
	}

	g.Score = rating.Score(g.RatingSum, g.NumRatings, prior.MeanRating, prior.PriorWeight)
	g.Mean = rating.Mean(g.RatingSum, g.NumRatings)
	return g, nil
}

// RatingHistogram returns how many people gave each whole-number rating,
// indexed 1..10. It is what the score breakdown on a game page is drawn from.
func (s *Store) RatingHistogram(ctx context.Context, gameID int64) ([10]int, error) {
	const sql = `
		SELECT width_bucket(value, 1, 11, 10) AS bucket, count(*)
		  FROM ratings
		 WHERE game_id = $1
		 GROUP BY bucket`

	var hist [10]int
	rows, err := s.pool.Query(ctx, sql, gameID)
	if err != nil {
		return hist, fmt.Errorf("histogram: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var bucket, n int
		if err := rows.Scan(&bucket, &n); err != nil {
			return hist, err
		}
		if bucket >= 1 && bucket <= 10 {
			hist[bucket-1] = n
		}
	}
	return hist, rows.Err()
}

// Prior reads the current Bayesian parameters.
func (s *Store) Prior(ctx context.Context) (Prior, error) {
	var p Prior
	err := s.pool.QueryRow(ctx,
		`SELECT mean_rating, prior_weight FROM global_stats WHERE id = true`).
		Scan(&p.MeanRating, &p.PriorWeight)
	if err != nil {
		return Prior{MeanRating: 7.0, PriorWeight: 5}, fmt.Errorf("read prior: %w", err)
	}
	return p, nil
}

// RefreshGlobalStats recomputes the global mean. Invoked by the hourly cron.
func (s *Store) RefreshGlobalStats(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `SELECT refresh_global_stats()`)
	return err
}
