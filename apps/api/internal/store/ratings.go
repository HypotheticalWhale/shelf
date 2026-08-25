package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/samueltansats/shelf/apps/api/internal/rating"
)

// SetRating records a user's rating and returns the game with its updated
// aggregates.
//
// The write and the read-back share one transaction. The stats trigger is AFTER
// ROW, so its effect is already visible to the re-read, which means the client
// gets the true new score in the same round trip — no second request and no
// window where the UI shows a stale number.
func (s *Store) SetRating(ctx context.Context, userID, slug string, value float64) (Game, error) {
	if err := rating.Validate(value); err != nil {
		return Game{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Game{}, fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	var gameID int64
	err = tx.QueryRow(ctx, `SELECT id FROM games WHERE slug = $1`, slug).Scan(&gameID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Game{}, ErrNotFound
	}
	if err != nil {
		return Game{}, fmt.Errorf("lookup game: %w", err)
	}

	const upsert = `
		INSERT INTO ratings (user_id, game_id, value)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, game_id) DO UPDATE
		   SET value = EXCLUDED.value, updated_at = now()`

	if _, err := tx.Exec(ctx, upsert, userID, gameID, value); err != nil {
		if isForeignKeyViolation(err) {
			return Game{}, ErrNotFound
		}
		return Game{}, fmt.Errorf("upsert rating: %w", err)
	}

	g, err := getGameBySlug(ctx, tx, slug, userID)
	if err != nil {
		return Game{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Game{}, fmt.Errorf("commit: %w", err)
	}
	return g, nil
}

// DeleteRating clears a user's rating and returns the updated game.
func (s *Store) DeleteRating(ctx context.Context, userID, slug string) (Game, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Game{}, fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	const del = `
		DELETE FROM ratings
		 WHERE user_id = $1
		   AND game_id = (SELECT id FROM games WHERE slug = $2)`

	if _, err := tx.Exec(ctx, del, userID, slug); err != nil {
		return Game{}, fmt.Errorf("delete rating: %w", err)
	}

	g, err := getGameBySlug(ctx, tx, slug, userID)
	if err != nil {
		return Game{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Game{}, fmt.Errorf("commit: %w", err)
	}
	return g, nil
}

// ListUserRatings returns a user's rated games, most recently rated first.
func (s *Store) ListUserRatings(ctx context.Context, userID string, limit, offset int) ([]Rating, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	const sql = `
		SELECT r.game_id, r.value, r.updated_at,
		       g.slug, g.name, g.thumbnail_url, g.year_published,
		       COALESCE(gs.num_ratings, 0), COALESCE(gs.rating_sum, 0),
		       gl.mean_rating, gl.prior_weight
		  FROM ratings r
		  JOIN games g       ON g.id = r.game_id
		  LEFT JOIN game_stats gs ON gs.game_id = g.id
		  CROSS JOIN global_stats gl
		 WHERE r.user_id = $1
		 ORDER BY r.updated_at DESC
		 LIMIT $2 OFFSET $3`

	rows, err := s.pool.Query(ctx, sql, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list ratings: %w", err)
	}
	defer rows.Close()

	out := []Rating{}
	for rows.Next() {
		var r Rating
		var g Game
		var prior Prior
		if err := rows.Scan(
			&r.GameID, &r.Value, &r.UpdatedAt,
			&g.Slug, &g.Name, &g.ThumbnailURL, &g.YearPublished,
			&g.NumRatings, &g.RatingSum,
			&prior.MeanRating, &prior.PriorWeight,
		); err != nil {
			return nil, fmt.Errorf("scan rating: %w", err)
		}
		g.ID = r.GameID
		g.Score = rating.Score(g.RatingSum, g.NumRatings, prior.MeanRating, prior.PriorWeight)
		g.Mean = rating.Mean(g.RatingSum, g.NumRatings)
		r.Game = &g
		out = append(out, r)
	}
	return out, rows.Err()
}
