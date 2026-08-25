package httpx

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/samueltansats/shelf/apps/api/internal/auth"
	"github.com/samueltansats/shelf/apps/api/internal/store"
)

func (s *Server) handleRecentPosts(w http.ResponseWriter, r *http.Request) {
	posts, err := s.store.ListRecentPosts(r.Context(), queryInt(r, "limit"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"posts": posts})
}

func (s *Server) handleUserPosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	posts, err := s.store.ListPostsByUser(ctx, chi.URLParam(r, "username"), auth.UserID(ctx),
		queryInt(r, "limit"), queryInt(r, "offset"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"posts": posts})
}

func (s *Server) handleGetPost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	post, err := s.store.GetPost(ctx, chi.URLParam(r, "username"), chi.URLParam(r, "slug"), auth.UserID(ctx))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, post)
}

type createPostRequest struct {
	Title    string `json:"title"`
	BodyMD   string `json:"bodyMd"`
	GameSlug string `json:"gameSlug"`
	Publish  bool   `json:"publish"`
}

func (s *Server) handleCreatePost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user, err := s.auth.RequireUser(ctx)
	if err != nil {
		fail(w, err)
		return
	}

	var body createPostRequest
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "expected a JSON body")
		return
	}
	if strings.TrimSpace(body.Title) == "" {
		writeError(w, http.StatusBadRequest, "a title is required")
		return
	}

	in := store.NewPost{
		Title:   strings.TrimSpace(body.Title),
		BodyMD:  body.BodyMD,
		Publish: body.Publish,
	}

	// Posts may stand alone or hang off a game; resolve the slug here so the
	// client never has to know internal IDs.
	if body.GameSlug != "" {
		game, err := s.store.GetGameBySlug(ctx, body.GameSlug, "")
		if err != nil {
			fail(w, err)
			return
		}
		in.GameID = &game.ID
	}

	post, err := s.store.CreatePost(ctx, user.ID, in)
	if err != nil {
		fail(w, err)
		return
	}
	post.Author = &user
	writeJSON(w, http.StatusCreated, post)
}

type updatePostRequest struct {
	Title   *string `json:"title"`
	BodyMD  *string `json:"bodyMd"`
	Publish *bool   `json:"publish"`
}

func (s *Server) handleUpdatePost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user, err := s.auth.RequireUser(ctx)
	if err != nil {
		fail(w, err)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid post id")
		return
	}

	var body updatePostRequest
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "expected a JSON body")
		return
	}

	post, err := s.store.UpdatePost(ctx, user.ID, id, body.Title, body.BodyMD, body.Publish)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, post)
}

func (s *Server) handleDeletePost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user, err := s.auth.RequireUser(ctx)
	if err != nil {
		fail(w, err)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid post id")
		return
	}

	if err := s.store.DeletePost(ctx, user.ID, id); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
