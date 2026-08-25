package store

import (
	"context"
	"fmt"

	"github.com/HypotheticalWhale/shelf/apps/api/internal/rating"
)

// ValidShelfStatus reports whether status is one Shelf recognises.
func ValidShelfStatus(status string) bool {
	switch status {
	case "owned", "wishlist", "played":
		return true
	}
	return false
}

// SetShelfStatus adds a game to one of a user's shelves.
func (s *Store) SetShelfStatus(ctx context.Context, userID, slug, status string) error {
	if !ValidShelfStatus(status) {
		return fmt.Errorf("unknown shelf status %q", status)
	}

	const sql = `
		INSERT INTO shelf_items (user_id, game_id, status)
		SELECT $1, g.id, $3 FROM games g WHERE g.slug = $2
		ON CONFLICT DO NOTHING`

	tag, err := s.pool.Exec(ctx, sql, userID, slug, status)
	if err != nil {
		if isForeignKeyViolation(err) {
			return ErrNotFound
		}
		return fmt.Errorf("set shelf status: %w", err)
	}
	// No row inserted means either the game does not exist or the entry was
	// already there; distinguish so the caller can 404 correctly.
	if tag.RowsAffected() == 0 {
		var exists bool
		if err := s.pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM games WHERE slug = $1)`, slug).Scan(&exists); err != nil {
			return fmt.Errorf("set shelf status: %w", err)
		}
		if !exists {
			return ErrNotFound
		}
	}
	return nil
}

// RemoveShelfStatus takes a game off one of a user's shelves.
func (s *Store) RemoveShelfStatus(ctx context.Context, userID, slug, status string) error {
	const sql = `
		DELETE FROM shelf_items
		 WHERE user_id = $1
		   AND status  = $3
		   AND game_id = (SELECT id FROM games WHERE slug = $2)`

	if _, err := s.pool.Exec(ctx, sql, userID, slug, status); err != nil {
		return fmt.Errorf("remove shelf status: %w", err)
	}
	return nil
}

// ListShelf returns a user's shelf, optionally narrowed to one status.
func (s *Store) ListShelf(ctx context.Context, userID, status string, limit, offset int) ([]ShelfItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	const sql = `
		SELECT si.status, si.created_at,
		       g.id, g.slug, g.name, g.thumbnail_url, g.year_published,
		       g.min_players, g.max_players, g.mechanics,
		       COALESCE(gs.num_ratings, 0), COALESCE(gs.rating_sum, 0),
		       gl.mean_rating, gl.prior_weight
		  FROM shelf_items si
		  JOIN games g            ON g.id = si.game_id
		  LEFT JOIN game_stats gs ON gs.game_id = g.id
		  CROSS JOIN global_stats gl
		 WHERE si.user_id = $1
		   AND ($2 = '' OR si.status = $2)
		 ORDER BY si.created_at DESC
		 LIMIT $3 OFFSET $4`

	rows, err := s.pool.Query(ctx, sql, userID, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list shelf: %w", err)
	}
	defer rows.Close()

	out := []ShelfItem{}
	for rows.Next() {
		var it ShelfItem
		var g Game
		var prior Prior
		if err := rows.Scan(
			&it.Status, &it.CreatedAt,
			&g.ID, &g.Slug, &g.Name, &g.ThumbnailURL, &g.YearPublished,
			&g.MinPlayers, &g.MaxPlayers, &g.Mechanics,
			&g.NumRatings, &g.RatingSum,
			&prior.MeanRating, &prior.PriorWeight,
		); err != nil {
			return nil, fmt.Errorf("scan shelf item: %w", err)
		}
		g.Score = rating.Score(g.RatingSum, g.NumRatings, prior.MeanRating, prior.PriorWeight)
		g.Mean = rating.Mean(g.RatingSum, g.NumRatings)
		it.Game = &g
		out = append(out, it)
	}
	return out, rows.Err()
}
