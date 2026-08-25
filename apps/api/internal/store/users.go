package store

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
)

var nonSlug = regexp.MustCompile(`[^a-z0-9_-]+`)

// NormalizeUsername reduces arbitrary input to a safe URL segment.
func NormalizeUsername(raw string) string {
	u := nonSlug.ReplaceAllString(strings.ToLower(strings.TrimSpace(raw)), "-")
	u = strings.Trim(u, "-_")
	if len(u) > 32 {
		u = u[:32]
	}
	return u
}

// EnsureUser creates or refreshes the local row mirroring a Clerk identity.
//
// Clerk owns authentication, so this runs on the first authenticated request
// rather than through a signup webhook — no webhook endpoint to secure, and no
// window where a signed-in user has no row. Profile fields are refreshed on
// every call so a change made in Clerk propagates on the user's next request.
func (s *Store) EnsureUser(ctx context.Context, id, username, displayName, avatarURL string) (User, error) {
	if id == "" {
		return User{}, errors.New("user id is required")
	}

	candidate := NormalizeUsername(username)
	if candidate == "" {
		// Clerk does not guarantee a username. Derive a stable, unique
		// fallback from the Clerk ID rather than failing the request.
		suffix := id
		if len(suffix) > 8 {
			suffix = suffix[len(suffix)-8:]
		}
		candidate = "player-" + strings.ToLower(suffix)
	}

	var (
		nullable = func(v string) *string {
			if v == "" {
				return nil
			}
			return &v
		}
		u User
	)

	// Try the preferred username, then fall back to disambiguated variants if
	// somebody already holds it.
	for attempt := 0; attempt < 5; attempt++ {
		name := candidate
		if attempt > 0 {
			suffix := id
			if len(suffix) > 4+attempt {
				suffix = suffix[len(suffix)-(4+attempt):]
			}
			name = fmt.Sprintf("%s-%s", candidate, strings.ToLower(suffix))
		}

		const sql = `
			INSERT INTO users (id, username, display_name, avatar_url)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (id) DO UPDATE
			   SET display_name = COALESCE(EXCLUDED.display_name, users.display_name),
			       avatar_url   = COALESCE(EXCLUDED.avatar_url,   users.avatar_url),
			       updated_at   = now()
			RETURNING id, username, display_name, avatar_url, bio, created_at`

		err := s.pool.QueryRow(ctx, sql, id, name, nullable(displayName), nullable(avatarURL)).
			Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL, &u.Bio, &u.CreatedAt)
		if err == nil {
			return u, nil
		}
		if isUniqueViolation(err, "users_username_key") {
			continue // username taken by a different account; try a variant
		}
		return User{}, fmt.Errorf("ensure user: %w", err)
	}
	return User{}, fmt.Errorf("ensure user: could not allocate a username")
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (User, error) {
	const sql = `
		SELECT id, username, display_name, avatar_url, bio, created_at
		  FROM users WHERE username = $1`

	var u User
	err := s.pool.QueryRow(ctx, sql, username).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL, &u.Bio, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

// UpdateProfile changes the fields Shelf owns. Nil leaves a field untouched.
func (s *Store) UpdateProfile(ctx context.Context, id string, bio, displayName *string) (User, error) {
	const sql = `
		UPDATE users
		   SET bio          = COALESCE($2, bio),
		       display_name = COALESCE($3, display_name),
		       updated_at   = now()
		 WHERE id = $1
		RETURNING id, username, display_name, avatar_url, bio, created_at`

	var u User
	err := s.pool.QueryRow(ctx, sql, id, bio, displayName).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL, &u.Bio, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("update profile: %w", err)
	}
	return u, nil
}

// GetUserByID looks up the local row for a Clerk user ID.
func (s *Store) GetUserByID(ctx context.Context, id string) (User, error) {
	const sql = `
		SELECT id, username, display_name, avatar_url, bio, created_at
		  FROM users WHERE id = $1`

	var u User
	err := s.pool.QueryRow(ctx, sql, id).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL, &u.Bio, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

// Collector is a person plus enough of their shelf to show a preview.
type Collector struct {
	User       User    `json:"user"`
	OwnedCount int     `json:"ownedCount"`
	RatedCount int     `json:"ratedCount"`
	PostCount  int     `json:"postCount"`
	AvgRating  float64 `json:"avgRating"`
	ShelfPeek  []Game  `json:"shelfPeek"`
}

// ListCollectors returns people who have started a collection, busiest first.
//
// The preview games come from a lateral join rather than a follow-up query per
// person, so the directory stays one round trip however many people are listed.
func (s *Store) ListCollectors(ctx context.Context, limit, offset int) ([]Collector, error) {
	if limit <= 0 || limit > 60 {
		limit = 24
	}

	const sql = `
		SELECT u.id, u.username, u.display_name, u.avatar_url, u.bio, u.created_at,
		       COALESCE(o.n, 0)  AS owned,
		       COALESCE(r.n, 0)  AS rated,
		       COALESCE(p.n, 0)  AS posts,
		       COALESCE(r.avg, 0) AS avg_rating,
		       COALESCE(peek.games, '{}') AS peek_slugs,
		       COALESCE(peek.names, '{}') AS peek_names,
		       COALESCE(peek.thumbs, '{}') AS peek_thumbs
		  FROM users u
		  LEFT JOIN (SELECT user_id, count(*) n FROM shelf_items WHERE status = 'owned' GROUP BY user_id) o
		         ON o.user_id = u.id
		  LEFT JOIN (SELECT user_id, count(*) n, avg(value) avg FROM ratings GROUP BY user_id) r
		         ON r.user_id = u.id
		  LEFT JOIN (SELECT user_id, count(*) n FROM posts WHERE published_at IS NOT NULL GROUP BY user_id) p
		         ON p.user_id = u.id
		  LEFT JOIN LATERAL (
		        SELECT array_agg(g.slug ORDER BY si.created_at DESC) AS games,
		               array_agg(g.name ORDER BY si.created_at DESC) AS names,
		               array_agg(COALESCE(g.thumbnail_url, '') ORDER BY si.created_at DESC) AS thumbs
		          FROM (SELECT game_id, created_at FROM shelf_items
		                 WHERE user_id = u.id AND status = 'owned'
		                 ORDER BY created_at DESC LIMIT 12) si
		          JOIN games g ON g.id = si.game_id
		  ) peek ON true
		 ORDER BY (COALESCE(o.n,0) + COALESCE(r.n,0) + COALESCE(p.n,0)) DESC, u.created_at DESC
		 LIMIT $1 OFFSET $2`

	rows, err := s.pool.Query(ctx, sql, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list collectors: %w", err)
	}
	defer rows.Close()

	out := []Collector{}
	for rows.Next() {
		var c Collector
		var slugs, names, thumbs []string
		if err := rows.Scan(
			&c.User.ID, &c.User.Username, &c.User.DisplayName, &c.User.AvatarURL,
			&c.User.Bio, &c.User.CreatedAt,
			&c.OwnedCount, &c.RatedCount, &c.PostCount, &c.AvgRating,
			&slugs, &names, &thumbs,
		); err != nil {
			return nil, fmt.Errorf("scan collector: %w", err)
		}

		for i := range slugs {
			g := Game{Slug: slugs[i], Name: names[i]}
			if i < len(thumbs) && thumbs[i] != "" {
				t := thumbs[i]
				g.ThumbnailURL = &t
			}
			c.ShelfPeek = append(c.ShelfPeek, g)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
