package httpx

import (
	"net/http"

	"github.com/HypotheticalWhale/shelf/apps/api/internal/store"
	"github.com/go-chi/chi/v5"
)

type setRatingRequest struct {
	Value float64 `json:"value"`
}

// handleSetRating records a rating and answers with the game's new aggregates.
//
// Returning the updated game rather than a bare 204 is what lets the UI stay
// honest: the optimistic score the client painted on click is immediately
// reconciled against the real one, in the same round trip.
func (s *Server) handleSetRating(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user, err := s.auth.RequireUser(ctx)
	if err != nil {
		fail(w, err)
		return
	}

	var body setRatingRequest
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "expected a JSON body like {\"value\": 8.5}")
		return
	}

	game, err := s.store.SetRating(ctx, user.ID, chi.URLParam(r, "slug"), body.Value)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, game)
}

func (s *Server) handleDeleteRating(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user, err := s.auth.RequireUser(ctx)
	if err != nil {
		fail(w, err)
		return
	}

	game, err := s.store.DeleteRating(ctx, user.ID, chi.URLParam(r, "slug"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, game)
}

type shelfRequest struct {
	Status string `json:"status"`
}

func (s *Server) handleSetShelf(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user, err := s.auth.RequireUser(ctx)
	if err != nil {
		fail(w, err)
		return
	}

	var body shelfRequest
	if err := decodeJSON(r, &body); err != nil || !store.ValidShelfStatus(body.Status) {
		writeError(w, http.StatusBadRequest, "status must be one of owned, wishlist, played")
		return
	}

	if err := s.store.SetShelfStatus(ctx, user.ID, chi.URLParam(r, "slug"), body.Status); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": body.Status})
}

func (s *Server) handleRemoveShelf(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user, err := s.auth.RequireUser(ctx)
	if err != nil {
		fail(w, err)
		return
	}

	status := r.URL.Query().Get("status")
	if !store.ValidShelfStatus(status) {
		writeError(w, http.StatusBadRequest, "status must be one of owned, wishlist, played")
		return
	}

	if err := s.store.RemoveShelfStatus(ctx, user.ID, chi.URLParam(r, "slug"), status); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
