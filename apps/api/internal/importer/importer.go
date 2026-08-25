// Package importer fills Shelf's catalogue from BoardGameGeek.
package importer

import (
	"context"
	"fmt"
	"log"

	"github.com/HypotheticalWhale/shelf/apps/api/internal/bgg"
	"github.com/HypotheticalWhale/shelf/apps/api/internal/store"
)

type Importer struct {
	client *bgg.Client
	store  *store.Store
	Logf   func(format string, args ...any)
}

// New builds an importer. The token authenticates against BGG's XML API, which
// has required one since late 2025.
func New(s *store.Store, token string) *Importer {
	return &Importer{
		client: bgg.NewClient(token),
		store:  s,
		Logf:   log.Printf,
	}
}

func (im *Importer) logf(format string, args ...any) {
	if im.Logf != nil {
		im.Logf(format, args...)
	}
}

// Result summarises one import run.
type Result struct {
	Requested int `json:"requested"`
	Fetched   int `json:"fetched"`
	Written   int `json:"written"`
	Skipped   int `json:"skipped"`
}

// ImportIDs fetches and writes specific BGG IDs, batching to respect the API.
func (im *Importer) ImportIDs(ctx context.Context, ids []int, minRatings int) (Result, error) {
	res := Result{Requested: len(ids)}

	for start := 0; start < len(ids); start += bgg.BatchSize {
		end := min(start+bgg.BatchSize, len(ids))
		batch := ids[start:end]

		games, err := im.client.Things(ctx, batch)
		if err != nil {
			// A single failed batch should not discard the work already done;
			// report it and return what was written so a cron run still makes
			// forward progress.
			return res, fmt.Errorf("fetch batch %d-%d: %w", start, end, err)
		}
		res.Fetched += len(games)

		inputs := make([]store.GameInput, 0, len(games))
		for _, g := range games {
			if minRatings > 0 && g.UsersRated < minRatings {
				res.Skipped++
				continue
			}
			inputs = append(inputs, toInput(g))
		}

		written, err := im.store.UpsertGames(ctx, inputs)
		res.Written += written
		if err != nil {
			return res, fmt.Errorf("write batch %d-%d: %w", start, end, err)
		}

		im.logf("imported %d/%d (written %d, skipped %d)", end, len(ids), res.Written, res.Skipped)
	}
	return res, nil
}

// ImportHot seeds from BGG's current hot list. It needs no prior ID knowledge,
// which makes it the right way to fill an empty database.
func (im *Importer) ImportHot(ctx context.Context) (Result, error) {
	ids, err := im.client.Hot(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("fetch hot list: %w", err)
	}
	im.logf("hot list returned %d games", len(ids))
	return im.ImportIDs(ctx, ids, 0)
}

// Sweep walks a range of BGG IDs and keeps the games that clear minRatings.
//
// BGG's XML API has no "top N" endpoint, so popularity has to be discovered by
// looking. IDs already in the catalogue are skipped, making a sweep resumable:
// stop it at any point and the next run continues where it left off.
func (im *Importer) Sweep(ctx context.Context, from, to, minRatings int) (Result, error) {
	if from < 1 {
		from = 1
	}
	if to < from {
		return Result{}, fmt.Errorf("invalid range %d..%d", from, to)
	}

	res := Result{}
	for start := from; start <= to; start += bgg.BatchSize {
		end := min(start+bgg.BatchSize-1, to)

		batch := make([]int, 0, bgg.BatchSize)
		for id := start; id <= end; id++ {
			batch = append(batch, id)
		}

		existing, err := im.store.ExistingBGGIDs(ctx, batch)
		if err != nil {
			return res, err
		}
		pending := batch[:0:0]
		for _, id := range batch {
			if !existing[id] {
				pending = append(pending, id)
			}
		}
		if len(pending) == 0 {
			continue
		}
		res.Requested += len(pending)

		games, err := im.client.Things(ctx, pending)
		if err != nil {
			return res, fmt.Errorf("sweep %d-%d: %w", start, end, err)
		}
		res.Fetched += len(games)

		inputs := make([]store.GameInput, 0, len(games))
		for _, g := range games {
			if g.UsersRated < minRatings {
				res.Skipped++
				continue
			}
			inputs = append(inputs, toInput(g))
		}

		written, err := im.store.UpsertGames(ctx, inputs)
		res.Written += written
		if err != nil {
			return res, err
		}
		if written > 0 {
			im.logf("sweep %d..%d — kept %d (total written %d)", start, end, written, res.Written)
		}
	}
	return res, nil
}

// RefreshStale re-fetches the least recently imported games.
func (im *Importer) RefreshStale(ctx context.Context, limit int) (Result, error) {
	ids, err := im.store.StaleBGGIDs(ctx, limit)
	if err != nil {
		return Result{}, err
	}
	if len(ids) == 0 {
		return Result{}, nil
	}
	return im.ImportIDs(ctx, ids, 0)
}

func toInput(g bgg.Game) store.GameInput {
	return store.GameInput{
		BGGID:         g.BGGID,
		Name:          g.Name,
		YearPublished: g.YearPublished,
		Description:   g.Description,
		ImageURL:      g.ImageURL,
		ThumbnailURL:  g.ThumbnailURL,
		MinPlayers:    g.MinPlayers,
		MaxPlayers:    g.MaxPlayers,
		MinPlaytime:   g.MinPlaytime,
		MaxPlaytime:   g.MaxPlaytime,
		Weight:        g.Weight,
		Designers:     g.Designers,
		Categories:    g.Categories,
		Mechanics:     g.Mechanics,
	}
}

// HasToken reports whether BGG imports are possible with the current config.
func (im *Importer) HasToken() bool { return im.client.HasToken() }
