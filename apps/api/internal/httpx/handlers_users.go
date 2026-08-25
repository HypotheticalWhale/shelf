package httpx

import (
	"net/http"

	"github.com/HypotheticalWhale/shelf/apps/api/internal/auth"
	"github.com/go-chi/chi/v5"
)

func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user, err := s.store.GetUserByUsername(ctx, chi.URLParam(r, "username"))
	if err != nil {
		fail(w, err)
		return
	}

	// A profile is the person's home page, so it carries enough to render
	// without the client fanning out into three more requests.
	ratings, err := s.store.ListUserRatings(ctx, user.ID, 12, 0)
	if err != nil {
		fail(w, err)
		return
	}
	shelf, err := s.store.ListShelf(ctx, user.ID, "", 12, 0)
	if err != nil {
		fail(w, err)
		return
	}
	posts, err := s.store.ListPostsByUser(ctx, user.Username, auth.UserID(ctx), 10, 0)
	if err != nil {
		fail(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user":          user,
		"recentRatings": ratings,
		"shelf":         shelf,
		"posts":         posts,
	})
}

func (s *Server) handleUserRatings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user, err := s.store.GetUserByUsername(ctx, chi.URLParam(r, "username"))
	if err != nil {
		fail(w, err)
		return
	}

	ratings, err := s.store.ListUserRatings(ctx, user.ID, queryInt(r, "limit"), queryInt(r, "offset"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ratings": ratings})
}

func (s *Server) handleUserShelf(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user, err := s.store.GetUserByUsername(ctx, chi.URLParam(r, "username"))
	if err != nil {
		fail(w, err)
		return
	}

	items, err := s.store.ListShelf(ctx, user.ID, r.URL.Query().Get("status"),
		queryInt(r, "limit"), queryInt(r, "offset"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"shelf": items})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user, err := s.auth.RequireUser(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

type updateMeRequest struct {
	Bio         *string `json:"bio"`
	DisplayName *string `json:"displayName"`
}

func (s *Server) handleUpdateMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user, err := s.auth.RequireUser(ctx)
	if err != nil {
		fail(w, err)
		return
	}

	var body updateMeRequest
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "expected a JSON body")
		return
	}

	updated, err := s.store.UpdateProfile(ctx, user.ID, body.Bio, body.DisplayName)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}
