// Command artwork fills in box art.
//
//	go run ./cmd/artwork -limit 500       fill the next 500 games without a cover
//	go run ./cmd/artwork -coverage        report what has been found so far
//	go run ./cmd/artwork -recheck         look again at games that came up empty
//
// It aggregates from several sources, re-hosts what it finds on our own
// storage, and records the provenance of every image. Runs are resumable: a
// game that has been looked at is not looked at again unless asked.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/HypotheticalWhale/shelf/apps/api/internal/artwork"
	"github.com/HypotheticalWhale/shelf/apps/api/internal/blobstore"
	"github.com/HypotheticalWhale/shelf/apps/api/internal/config"
	"github.com/HypotheticalWhale/shelf/apps/api/internal/db"
	"github.com/HypotheticalWhale/shelf/apps/api/internal/store"
)

func main() {
	var (
		limit    = flag.Int("limit", 500, "how many games to attempt")
		recheck  = flag.Bool("recheck", false, "look again at games that previously came up empty")
		coverage = flag.Bool("coverage", false, "report coverage and exit")
		dryRun   = flag.Bool("dry-run", false, "resolve sources but store nothing")
	)
	flag.Parse()

	config.LoadDotEnv()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	connectCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	pool, err := db.New(connectCtx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()
	st := store.New(pool)

	if *coverage {
		report(ctx, st)
		return
	}

	blob := blobstore.New()
	if !blob.Configured() && !*dryRun {
		log.Fatal("BLOB_READ_WRITE_TOKEN is not set; run with -dry-run to resolve without storing")
	}

	targets, err := st.GamesNeedingArtwork(ctx, *limit, *recheck)
	if err != nil {
		log.Fatalf("select games: %v", err)
	}
	if len(targets) == 0 {
		log.Println("nothing to do — every game already has a cover")
		return
	}
	log.Printf("attempting %d games", len(targets))

	log.Println("loading wikidata facts...")
	facts, err := artwork.LoadFacts(ctx)
	if err != nil {
		log.Fatalf("wikidata: %v", err)
	}
	log.Printf("wikidata knows %d games by BoardGameGeek id", len(facts))

	// Publisher art first: it is the box the publisher chose. Commons is the
	// freely-licensed fallback.
	resolver := &artwork.Resolver{
		Sources: []artwork.Source{
			&artwork.PublisherSource{Facts: facts},
			&artwork.CommonsSource{Facts: facts},
		},
		Logf: func(f string, a ...any) {},
	}

	client := &http.Client{Timeout: 45 * time.Second}
	var found, stored, empty int
	bySource := map[string]int{}
	started := time.Now()

	for i, t := range targets {
		if ctx.Err() != nil {
			break
		}

		c := resolver.Resolve(ctx, artwork.Game{
			BGGID: t.BGGID, Slug: t.Slug, Name: t.Name, Year: t.Year,
		})
		if c == nil {
			empty++
			if !*dryRun {
				st.MarkArtworkChecked(ctx, t.ID)
			}
			continue
		}
		found++
		bySource[c.Source]++

		if *dryRun {
			log.Printf("  %-38s %-9s %s", trim(t.Name, 38), c.Source, trim(c.URL, 70))
			continue
		}

		body, contentType, err := artwork.Fetch(ctx, client, c.URL)
		if err != nil {
			log.Printf("  %-38s %s: %v", trim(t.Name, 38), c.Source, err)
			st.MarkArtworkChecked(ctx, t.ID)
			continue
		}

		name := t.Slug + extensionFor(contentType, c.URL)
		hosted, err := blob.Put(ctx, path.Join("covers", name), contentType, body)
		if err != nil {
			log.Printf("  %-38s upload failed: %v", trim(t.Name, 38), err)
			continue
		}

		if err := st.SetArtwork(ctx, t.ID, hosted, c.Source, c.License, c.Credit, c.Origin); err != nil {
			log.Printf("  %-38s record failed: %v", trim(t.Name, 38), err)
			continue
		}
		stored++

		if stored%25 == 0 {
			log.Printf("  stored %d (%d attempted)", stored, i+1)
		}
	}

	log.Printf("attempted=%d found=%d stored=%d nothing-found=%d in %s",
		len(targets), found, stored, empty, time.Since(started).Round(time.Second))
	for src, n := range bySource {
		log.Printf("  from %-10s %d", src, n)
	}
	report(ctx, st)
}

func report(ctx context.Context, st *store.Store) {
	cov, err := st.ArtworkCoverage(ctx)
	if err != nil {
		log.Printf("coverage: %v", err)
		return
	}
	fmt.Println("\n  cover art by source:")
	for k, v := range cov {
		fmt.Printf("    %-16s %d\n", k, v)
	}
}

func extensionFor(contentType, url string) string {
	switch {
	case strings.Contains(contentType, "png"):
		return ".png"
	case strings.Contains(contentType, "webp"):
		return ".webp"
	case strings.Contains(contentType, "gif"):
		return ".gif"
	default:
		return ".jpg"
	}
}

func trim(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
