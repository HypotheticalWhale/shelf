// Package store holds Shelf's database access layer.
package store

import (
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a lookup matches no row. Handlers translate it
// into a 404 so callers never see a raw pgx error.
var ErrNotFound = errors.New("not found")

// ErrConflict is returned when a write violates a uniqueness constraint, such
// as two posts sharing a slug under one author.
var ErrConflict = errors.New("conflict")

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Pool exposes the underlying pool for the few callers that need it, such as
// the importer's batch writes.
func (s *Store) Pool() *pgxpool.Pool { return s.pool }

type User struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	DisplayName *string   `json:"displayName"`
	AvatarURL   *string   `json:"avatarUrl"`
	Bio         *string   `json:"bio"`
	CreatedAt   time.Time `json:"createdAt"`
}

// Game carries both the raw aggregate inputs and the derived figures.
//
// NumRatings and RatingSum are the stored aggregates; Score and Mean are
// computed in Go by the rating package so that a single tested implementation
// produces every number a user actually sees.
type Game struct {
	ID            int64    `json:"id"`
	BGGID         int      `json:"bggId"`
	Slug          string   `json:"slug"`
	Name          string   `json:"name"`
	YearPublished *int     `json:"yearPublished"`
	Description   *string  `json:"description,omitempty"`
	ImageURL      *string  `json:"imageUrl"`
	ThumbnailURL  *string  `json:"thumbnailUrl"`
	MinPlayers    *int     `json:"minPlayers"`
	MaxPlayers    *int     `json:"maxPlayers"`
	MinPlaytime   *int     `json:"minPlaytime"`
	MaxPlaytime   *int     `json:"maxPlaytime"`
	Weight        *float64 `json:"weight"`
	Designers     []string `json:"designers,omitempty"`
	Categories    []string `json:"categories,omitempty"`
	Mechanics     []string `json:"mechanics,omitempty"`

	NumRatings int     `json:"numRatings"`
	RatingSum  float64 `json:"-"`
	Score      float64 `json:"score"`
	Mean       float64 `json:"mean"`

	// ViewerRating is the requesting user's own rating, when signed in.
	ViewerRating *float64 `json:"viewerRating"`
	// ViewerShelf lists the requesting user's shelf statuses for this game.
	ViewerShelf []string `json:"viewerShelf,omitempty"`
}

type Rating struct {
	GameID    int64     `json:"gameId"`
	Value     float64   `json:"value"`
	UpdatedAt time.Time `json:"updatedAt"`
	Game      *Game     `json:"game,omitempty"`
}

type Post struct {
	ID          int64      `json:"id"`
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	BodyMD      string     `json:"bodyMd"`
	PublishedAt *time.Time `json:"publishedAt"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`

	Author *User `json:"author,omitempty"`
	Game   *Game `json:"game,omitempty"`
}

type ShelfItem struct {
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	Game      *Game     `json:"game,omitempty"`
}

// Prior holds the Bayesian parameters read from global_stats.
type Prior struct {
	MeanRating  float64 `json:"meanRating"`
	PriorWeight int     `json:"priorWeight"`
}
