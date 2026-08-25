package httpx

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/HypotheticalWhale/shelf/apps/api/internal/rating"
)

// statsEvent is one score change, as sent to the browser.
type statsEvent struct {
	Slug       string  `json:"slug"`
	NumRatings int     `json:"numRatings"`
	Score      float64 `json:"score"`
	Mean       float64 `json:"mean"`
}

// notifyPayload is what the database publishes.
type notifyPayload struct {
	Slug        string  `json:"slug"`
	NumRatings  int     `json:"numRatings"`
	RatingSum   float64 `json:"ratingSum"`
	MeanRating  float64 `json:"meanRating"`
	PriorWeight int     `json:"priorWeight"`
}

// handleEvents streams score changes to the browser over Server-Sent Events.
//
// A dedicated connection LISTENs on the Postgres channel the stats trigger
// publishes to, so an update reaches every open page the moment the rating
// commits — no polling, and no work at all while the catalogue is quiet.
//
// The stream closes itself before the platform's function timeout would; the
// browser's EventSource reconnects on its own, which also covers an instance
// being recycled underneath it.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	// ResponseController reaches through middleware wrappers via Unwrap, so
	// this keeps working however the chain is composed.
	rc := http.NewResponseController(w)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	// Tell the client how soon to come back if this drops.
	fmt.Fprint(w, "retry: 3000\n\n")
	_ = rc.Flush()

	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Minute)
	defer cancel()

	pool, err := s.listenPool(ctx)
	if err != nil {
		log.Printf("events: direct pool: %v", err)
		return
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		// Every direct connection is busy; the client will retry shortly.
		log.Printf("events: acquire: %v", err)
		return
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "LISTEN shelf_game_stats"); err != nil {
		log.Printf("events: listen: %v", err)
		return
	}

	// A comment line every 25s keeps proxies from closing an idle stream.
	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ping.C:
				fmt.Fprint(w, ": ping\n\n")
				_ = rc.Flush()
			}
		}
	}()

	for {
		n, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			return // context finished, or the connection went away
		}

		var p notifyPayload
		if err := json.Unmarshal([]byte(n.Payload), &p); err != nil {
			continue
		}

		// The score is computed here by the same tested implementation the
		// rest of the API uses, rather than duplicated in SQL or in the client.
		out := statsEvent{
			Slug:       p.Slug,
			NumRatings: p.NumRatings,
			Score:      rating.Score(p.RatingSum, p.NumRatings, p.MeanRating, p.PriorWeight),
			Mean:       rating.Mean(p.RatingSum, p.NumRatings),
		}

		body, err := json.Marshal(out)
		if err != nil {
			continue
		}
		fmt.Fprintf(w, "event: stats\ndata: %s\n\n", body)
		_ = rc.Flush()
	}
}
