package httpx

import (
	"net/http"
	"strconv"

	"github.com/HypotheticalWhale/shelf/apps/api/internal/auth"
	"github.com/HypotheticalWhale/shelf/apps/api/internal/store"
	"github.com/go-chi/chi/v5"
)

func queryInt(r *http.Request, key string) int {
	n, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return 0
	}
	return n
}

func queryFloat(r *http.Request, key string) float64 {
	f, err := strconv.ParseFloat(r.URL.Query().Get(key), 64)
	if err != nil {
		return 0
	}
	return f
}

func (s *Server) handleListGames(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page, err := s.store.ListGames(r.Context(), store.GameFilter{
		Query:        q.Get("q"),
		Mechanic:     q.Get("mechanic"),
		DetailedOnly: q.Get("detailed") == "1",
		Players:      queryInt(r, "players"),
		MaxTime:      queryInt(r, "maxTime"),
		MinWeight:    queryFloat(r, "minWeight"),
		MaxWeight:    queryFloat(r, "maxWeight"),
		Sort:         q.Get("sort"),
		Limit:        queryInt(r, "limit"),
		Offset:       queryInt(r, "offset"),
		ViewerID:     auth.UserID(r.Context()),
	})
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

type gameDetail struct {
	store.Game
	Histogram [10]int `json:"histogram"`
}

func (s *Server) handleGetGame(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	g, err := s.store.GetGameBySlug(ctx, chi.URLParam(r, "slug"), auth.UserID(ctx))
	if err != nil {
		fail(w, err)
		return
	}

	// A failed histogram should not take down the page it decorates.
	hist, err := s.store.RatingHistogram(ctx, g.ID)
	if err != nil {
		hist = [10]int{}
	}
	writeJSON(w, http.StatusOK, gameDetail{Game: g, Histogram: hist})
}

func (s *Server) handleGamePosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	g, err := s.store.GetGameBySlug(ctx, chi.URLParam(r, "slug"), "")
	if err != nil {
		fail(w, err)
		return
	}

	posts, err := s.store.ListPostsForGame(ctx, g.ID, queryInt(r, "limit"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"posts": posts})
}
