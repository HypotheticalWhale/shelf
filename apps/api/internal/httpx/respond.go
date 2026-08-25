package httpx

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/HypotheticalWhale/shelf/apps/api/internal/auth"
	"github.com/HypotheticalWhale/shelf/apps/api/internal/rating"
	"github.com/HypotheticalWhale/shelf/apps/api/internal/store"
)

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		// The status line is already sent, so this can only be logged.
		log.Printf("write response: %v", err)
	}
}

type errorBody struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorBody{Error: message})
}

// fail maps a domain error to the right status code, so handlers can return
// store errors directly instead of each one re-deciding what a missing row means.
func fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	case errors.Is(err, store.ErrConflict):
		writeError(w, http.StatusConflict, "already exists")
	case errors.Is(err, auth.ErrUnauthenticated):
		writeError(w, http.StatusUnauthorized, "sign in to continue")
	case errors.Is(err, rating.ErrOutOfRange):
		writeError(w, http.StatusBadRequest, rating.ErrOutOfRange.Error())
	default:
		log.Printf("unhandled error: %v", err)
		writeError(w, http.StatusInternalServerError, "something went wrong")
	}
}

// decodeJSON reads a JSON body, rejecting unknown fields so a typo in a client
// payload surfaces as a clear 400 rather than being silently ignored.
func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
