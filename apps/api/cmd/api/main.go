// Command api serves the Shelf HTTP API.
//
// Vercel's Go framework preset detects this entrypoint automatically from the
// module root, so no vercel.json routing is needed. Locally:
//
//	go run ./cmd/api
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/HypotheticalWhale/shelf/apps/api/internal/config"
	"github.com/HypotheticalWhale/shelf/apps/api/internal/db"
	"github.com/HypotheticalWhale/shelf/apps/api/internal/httpx"
	"github.com/HypotheticalWhale/shelf/apps/api/internal/migrations"
	"github.com/HypotheticalWhale/shelf/apps/api/internal/store"
)

func main() {
	config.LoadDotEnv()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	pool, err := db.New(connectCtx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	// Opt-in so a deploy can migrate itself. The migrator takes an advisory
	// lock, so several instances booting at once is safe.
	if os.Getenv("MIGRATE_ON_BOOT") == "1" {
		mpool, err := db.NewForMigrations(connectCtx, cfg.MigrationURL)
		if err != nil {
			log.Fatalf("migrate: %v", err)
		}
		applied, err := migrations.Apply(connectCtx, mpool)
		mpool.Close()
		if err != nil {
			log.Fatalf("migrate: %v", err)
		}
		if len(applied) > 0 {
			log.Printf("applied %d migration(s) on boot", len(applied))
		}
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpx.NewServer(cfg, store.New(pool)).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("shelf api listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("serve: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}
