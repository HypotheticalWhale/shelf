// Package httpx wires Shelf's HTTP API.
package httpx

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/samueltansats/shelf/apps/api/internal/auth"
	"github.com/samueltansats/shelf/apps/api/internal/config"
	"github.com/samueltansats/shelf/apps/api/internal/importer"
	"github.com/samueltansats/shelf/apps/api/internal/store"
)

type Server struct {
	store *store.Store
	auth  *auth.Authenticator
	imp   *importer.Importer
	cfg   config.Config
}

func NewServer(cfg config.Config, st *store.Store) *Server {
	return &Server{
		store: st,
		auth:  auth.New(st, cfg.ClerkSecretKey),
		imp:   importer.New(st, cfg.BGGAPIToken),
		cfg:   cfg,
	}
}

// Handler builds the router.
//
// Every route runs through the Clerk middleware so handlers can read an
// optional viewer, and the handful that require a signed-in user assert it
// themselves via requireUser. That keeps "who is looking" available to public
// endpoints — a game page shows your own rating — without duplicating the
// auth wiring per route.
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()

	r.Use(recoverPanics)
	r.Use(requestLogger)
	r.Use(cors(s.cfg.AllowedOrigins))
	r.Use(s.auth.Middleware)

	r.Get("/health", s.handleHealth)

	r.Route("/games", func(r chi.Router) {
		r.Get("/", s.handleListGames)
		r.Get("/{slug}", s.handleGetGame)
		r.Get("/{slug}/posts", s.handleGamePosts)
		r.Put("/{slug}/rating", s.handleSetRating)
		r.Delete("/{slug}/rating", s.handleDeleteRating)
	})

	r.Route("/shelf", func(r chi.Router) {
		r.Put("/{slug}", s.handleSetShelf)
		r.Delete("/{slug}", s.handleRemoveShelf)
	})

	r.Route("/users/{username}", func(r chi.Router) {
		r.Get("/", s.handleGetUser)
		r.Get("/ratings", s.handleUserRatings)
		r.Get("/shelf", s.handleUserShelf)
		r.Get("/posts", s.handleUserPosts)
		r.Get("/posts/{slug}", s.handleGetPost)
	})

	r.Route("/posts", func(r chi.Router) {
		r.Get("/", s.handleRecentPosts)
		r.Post("/", s.handleCreatePost)
		r.Patch("/{id}", s.handleUpdatePost)
		r.Delete("/{id}", s.handleDeletePost)
	})

	r.Route("/me", func(r chi.Router) {
		r.Get("/", s.handleMe)
		r.Patch("/", s.handleUpdateMe)
	})

	// Vercel Cron invokes schedules with GET, carrying CRON_SECRET as a bearer
	// token; POST is kept so the same endpoints can be triggered by hand.
	r.Route("/cron", func(r chi.Router) {
		r.Use(requireCronSecret(s.cfg.CronSecret))
		r.Get("/refresh-stats", s.handleCronRefreshStats)
		r.Post("/refresh-stats", s.handleCronRefreshStats)
		r.Get("/import", s.handleCronImport)
		r.Post("/import", s.handleCronImport)
	})

	return r
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	games, err := s.store.CountGames(r.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status": "degraded",
			"error":  "database unreachable",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"games":    games,
		"bggToken": s.imp.HasToken(),
	})
}
