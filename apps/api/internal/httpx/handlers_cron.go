package httpx

import (
	"context"
	"net/http"
	"time"
)

func (s *Server) handleCronRefreshStats(w http.ResponseWriter, r *http.Request) {
	if err := s.store.RefreshGlobalStats(r.Context()); err != nil {
		fail(w, err)
		return
	}

	prior, err := s.store.Prior(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"refreshed": true, "prior": prior})
}

// handleCronImport refreshes catalogue metadata from BGG.
//
// The work is detached from the request context and capped well inside the
// platform's function timeout, so a slow BGG does not leave the schedule
// hanging or the response blocked.
func (s *Server) handleCronImport(w http.ResponseWriter, r *http.Request) {
	if !s.imp.HasToken() {
		writeJSON(w, http.StatusOK, map[string]any{
			"skipped": true,
			"reason":  "BGG_API_TOKEN is not configured",
		})
		return
	}

	limit := queryInt(r, "limit")
	if limit <= 0 {
		limit = 100
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 4*time.Minute)
	defer cancel()

	res, err := s.imp.RefreshStale(ctx, limit)
	if err != nil {
		// Partial progress still counts; report it alongside the failure.
		writeJSON(w, http.StatusPartialContent, map[string]any{
			"result": res,
			"error":  err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": res})
}
