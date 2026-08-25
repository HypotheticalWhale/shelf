package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Slugify turns a post title into a URL segment.
func Slugify(title string) string {
	s := nonSlug.ReplaceAllString(strings.ToLower(strings.TrimSpace(title)), "-")
	s = strings.Trim(s, "-_")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	if len(s) > 60 {
		s = strings.Trim(s[:60], "-")
	}
	if s == "" {
		s = "post"
	}
	return s
}

// NewPost describes a post being written.
type NewPost struct {
	Title   string
	BodyMD  string
	GameID  *int64
	Publish bool
}

// CreatePost writes a post, allocating a slug unique within the author's blog.
func (s *Store) CreatePost(ctx context.Context, userID string, in NewPost) (Post, error) {
	if strings.TrimSpace(in.Title) == "" {
		return Post{}, errors.New("title is required")
	}

	base := Slugify(in.Title)
	var published *time.Time
	if in.Publish {
		now := time.Now().UTC()
		published = &now
	}

	const sql = `
		INSERT INTO posts (user_id, game_id, slug, title, body_md, published_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, slug, title, body_md, published_at, created_at, updated_at`

	// Slugs collide only within one author's blog, so a short numeric suffix is
	// enough to disambiguate a repeated title.
	for attempt := 0; attempt < 25; attempt++ {
		slug := base
		if attempt > 0 {
			slug = fmt.Sprintf("%s-%d", base, attempt+1)
		}

		var p Post
		err := s.pool.QueryRow(ctx, sql, userID, in.GameID, slug, in.Title, in.BodyMD, published).
			Scan(&p.ID, &p.Slug, &p.Title, &p.BodyMD, &p.PublishedAt, &p.CreatedAt, &p.UpdatedAt)
		if err == nil {
			return p, nil
		}
		if isUniqueViolation(err, "posts_user_id_slug_key") {
			continue
		}
		if isForeignKeyViolation(err) {
			return Post{}, ErrNotFound
		}
		return Post{}, fmt.Errorf("create post: %w", err)
	}
	return Post{}, ErrConflict
}

// UpdatePost edits a post. Nil fields are left unchanged. The author check is
// part of the WHERE clause, so another user's post reads as not found rather
// than leaking its existence.
func (s *Store) UpdatePost(ctx context.Context, userID string, id int64, title, body *string, publish *bool) (Post, error) {
	const sql = `
		UPDATE posts
		   SET title        = COALESCE($3, title),
		       body_md      = COALESCE($4, body_md),
		       published_at = CASE
		           WHEN $5::boolean IS NULL THEN published_at
		           WHEN $5::boolean THEN COALESCE(published_at, now())
		           ELSE NULL
		       END,
		       updated_at   = now()
		 WHERE id = $1 AND user_id = $2
		RETURNING id, slug, title, body_md, published_at, created_at, updated_at`

	var p Post
	err := s.pool.QueryRow(ctx, sql, id, userID, title, body, publish).
		Scan(&p.ID, &p.Slug, &p.Title, &p.BodyMD, &p.PublishedAt, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Post{}, ErrNotFound
	}
	if err != nil {
		return Post{}, fmt.Errorf("update post: %w", err)
	}
	return p, nil
}

func (s *Store) DeletePost(ctx context.Context, userID string, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM posts WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete post: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

const postSelect = `
	SELECT p.id, p.slug, p.title, p.body_md, p.published_at, p.created_at, p.updated_at,
	       u.id, u.username, u.display_name, u.avatar_url,
	       g.id, g.slug, g.name, g.thumbnail_url
	  FROM posts p
	  JOIN users u      ON u.id = p.user_id
	  LEFT JOIN games g ON g.id = p.game_id`

func scanPost(row pgx.Row) (Post, error) {
	var p Post
	var author User
	var gameID *int64
	var gameSlug, gameName, gameThumb *string

	err := row.Scan(
		&p.ID, &p.Slug, &p.Title, &p.BodyMD, &p.PublishedAt, &p.CreatedAt, &p.UpdatedAt,
		&author.ID, &author.Username, &author.DisplayName, &author.AvatarURL,
		&gameID, &gameSlug, &gameName, &gameThumb,
	)
	if err != nil {
		return Post{}, err
	}
	p.Author = &author
	if gameID != nil && gameSlug != nil && gameName != nil {
		p.Game = &Game{ID: *gameID, Slug: *gameSlug, Name: *gameName, ThumbnailURL: gameThumb}
	}
	return p, nil
}

// GetPost fetches one post by author and slug. Drafts are visible only to their
// author, so viewerID gates them.
func (s *Store) GetPost(ctx context.Context, username, slug, viewerID string) (Post, error) {
	sql := postSelect + `
		 WHERE u.username = $1 AND p.slug = $2
		   AND (p.published_at IS NOT NULL OR p.user_id = $3)`

	p, err := scanPost(s.pool.QueryRow(ctx, sql, username, slug, viewerID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Post{}, ErrNotFound
	}
	if err != nil {
		return Post{}, fmt.Errorf("get post: %w", err)
	}
	return p, nil
}

// ListPostsByUser returns one person's blog, newest first.
func (s *Store) ListPostsByUser(ctx context.Context, username, viewerID string, limit, offset int) ([]Post, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	sql := postSelect + `
		 WHERE u.username = $1
		   AND (p.published_at IS NOT NULL OR p.user_id = $2)
		 ORDER BY COALESCE(p.published_at, p.created_at) DESC
		 LIMIT $3 OFFSET $4`

	return s.queryPosts(ctx, sql, username, viewerID, limit, offset)
}

// ListRecentPosts is the community feed on the home page.
func (s *Store) ListRecentPosts(ctx context.Context, limit int) ([]Post, error) {
	if limit <= 0 || limit > 50 {
		limit = 6
	}

	sql := postSelect + `
		 WHERE p.published_at IS NOT NULL
		 ORDER BY p.published_at DESC
		 LIMIT $1`

	return s.queryPosts(ctx, sql, limit)
}

// ListPostsForGame returns published reviews attached to a game.
func (s *Store) ListPostsForGame(ctx context.Context, gameID int64, limit int) ([]Post, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	sql := postSelect + `
		 WHERE p.game_id = $1 AND p.published_at IS NOT NULL
		 ORDER BY p.published_at DESC
		 LIMIT $2`

	return s.queryPosts(ctx, sql, gameID, limit)
}

func (s *Store) queryPosts(ctx context.Context, sql string, args ...any) ([]Post, error) {
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("list posts: %w", err)
	}
	defer rows.Close()

	out := []Post{}
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, fmt.Errorf("scan post: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
