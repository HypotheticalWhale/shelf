// Package db owns the Postgres connection pool.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// New builds a pool tuned for serverless request handling.
//
// Two deliberate choices here:
//
//   - MaxConns is small. Fluid Compute reuses instances but can still fan out,
//     and Neon's pooler has a finite backend budget; a large per-instance pool
//     exhausts it under load for no gain.
//
//   - QueryExecModeExec disables pgx's prepared-statement cache. Neon's pooled
//     endpoint is PgBouncer in transaction mode, where a prepared statement can
//     land on a different backend than the one that prepared it. Leaving the
//     cache on produces intermittent "prepared statement does not exist" errors
//     that only surface under concurrency.
func New(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	return newPool(ctx, databaseURL, pgx.QueryExecModeExec, 4)
}

// NewForMigrations builds a single-connection pool that speaks the simple
// protocol.
//
// Migration files contain many statements separated by semicolons. The extended
// protocol used by QueryExecModeExec permits exactly one statement per Exec and
// would reject them, so schema work needs its own connection configured this
// way rather than reusing the request pool.
func NewForMigrations(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	return newPool(ctx, databaseURL, pgx.QueryExecModeSimpleProtocol, 1)
}

func newPool(ctx context.Context, databaseURL string, mode pgx.QueryExecMode, maxConns int32) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	cfg.MaxConns = maxConns
	cfg.MinConns = 0
	cfg.MaxConnLifetime = 5 * time.Minute
	cfg.MaxConnIdleTime = 1 * time.Minute
	cfg.ConnConfig.DefaultQueryExecMode = mode

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	// Neon suspends idle compute, and waking it can take longer than a typical
	// connect timeout — a cold start was failing CLI commands outright with
	// "context deadline exceeded" while the endpoint was perfectly reachable.
	// Retry briefly rather than giving up on the first slow handshake.
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		err := pool.Ping(pingCtx)
		cancel()
		if err == nil {
			return pool, nil
		}
		lastErr = err
		if ctx.Err() != nil {
			break
		}
	}
	pool.Close()
	return nil, fmt.Errorf("ping database (the endpoint may be waking): %w", lastErr)
}
