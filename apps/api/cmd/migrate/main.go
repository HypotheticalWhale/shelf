// Command migrate applies Shelf's SQL migrations and exits.
//
//	go run ./cmd/migrate
package main

import (
	"context"
	"log"
	"time"

	"github.com/samueltansats/shelf/apps/api/internal/config"
	"github.com/samueltansats/shelf/apps/api/internal/db"
	"github.com/samueltansats/shelf/apps/api/internal/migrations"
)

func main() {
	config.LoadDotEnv()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, err := db.NewForMigrations(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	applied, err := migrations.Apply(ctx, pool)
	if err != nil {
		log.Fatalf("migrate: %v", err)
	}

	if len(applied) == 0 {
		log.Println("schema is already up to date")
		return
	}
	for _, name := range applied {
		log.Printf("applied %s", name)
	}
	log.Printf("%d migration(s) applied", len(applied))
}
