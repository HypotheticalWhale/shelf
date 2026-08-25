// Command devdb runs a throwaway Postgres for local development.
//
//	go run ./cmd/devdb
//
// It downloads and supervises a real Postgres, applies the migrations and
// loads the bundled catalogue, then blocks until interrupted. Nothing has to be
// installed first — no Homebrew, no Docker, no system service — which keeps
// "clone it and run it" true for anyone picking this repo up.
//
// Data lives under .devdb/ and is disposable; delete it for a clean slate.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/samueltansats/shelf/apps/api/internal/db"
	"github.com/samueltansats/shelf/apps/api/internal/migrations"
	"github.com/samueltansats/shelf/apps/api/internal/seed"
	"github.com/samueltansats/shelf/apps/api/internal/store"
)

func main() {
	port := flag.Uint("port", 5432, "port to listen on")
	skipSeed := flag.Bool("no-seed", false, "skip loading the bundled catalogue")
	flag.Parse()

	root, err := filepath.Abs(".devdb")
	if err != nil {
		log.Fatalf("resolve data directory: %v", err)
	}

	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Username("shelf").
			Password("shelf").
			Database("shelf").
			Port(uint32(*port)).
			RuntimePath(filepath.Join(root, "run")).
			DataPath(filepath.Join(root, "data")).
			BinariesPath(filepath.Join(root, "bin")).
			StartTimeout(3 * time.Minute),
	)

	log.Println("starting postgres (the first run downloads it)...")
	if err := pg.Start(); err != nil {
		log.Fatalf("start postgres: %v", err)
	}
	defer func() {
		if err := pg.Stop(); err != nil {
			log.Printf("stop postgres: %v", err)
		}
	}()

	dsn := "postgres://shelf:shelf@localhost:" +
		flag.Lookup("port").Value.String() + "/shelf?sslmode=disable"

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	setupCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	mpool, err := db.NewForMigrations(setupCtx, dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	applied, err := migrations.Apply(setupCtx, mpool)
	mpool.Close()
	if err != nil {
		log.Fatalf("migrate: %v", err)
	}
	log.Printf("applied %d migration(s)", len(applied))

	if !*skipSeed {
		pool, err := db.New(setupCtx, dsn)
		if err != nil {
			log.Fatalf("connect: %v", err)
		}
		st := store.New(pool)

		games, err := seed.Games()
		if err != nil {
			log.Fatalf("seed: %v", err)
		}
		written, err := st.UpsertGames(setupCtx, games)
		if err != nil {
			log.Fatalf("load catalogue: %v", err)
		}
		total, _ := st.CountGames(setupCtx)
		pool.Close()
		log.Printf("loaded %d seed games (catalogue holds %d)", written, total)
	}

	log.Printf("\n  DATABASE_URL=%s\n\n  ready — press Ctrl-C to stop\n", dsn)
	<-ctx.Done()
	log.Println("stopping postgres")
}
