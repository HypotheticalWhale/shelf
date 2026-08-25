// Package migrations embeds and applies Shelf's SQL schema.
package migrations

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed sql/*.sql
var files embed.FS

// Apply runs every migration that has not been applied yet, in filename order.
//
// Each file runs inside its own transaction alongside the bookkeeping insert,
// so a failure part-way leaves no partially-applied migration behind. An
// advisory lock serialises concurrent runners — without it, two deploys racing
// to migrate can both see a migration as pending and apply it twice.
func Apply(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	const lockID = 8_474_531 // arbitrary, but stable across releases

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, lockID); err != nil {
		return nil, fmt.Errorf("acquire advisory lock: %w", err)
	}
	defer func() {
		conn.Exec(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock($1)`, lockID)
	}()

	if _, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    text PRIMARY KEY,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`); err != nil {
		return nil, fmt.Errorf("create schema_migrations: %w", err)
	}

	applied := map[string]bool{}
	rows, err := conn.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("read schema_migrations: %w", err)
	}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return nil, err
		}
		applied[v] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	entries, err := fs.Glob(files, "sql/*.sql")
	if err != nil {
		return nil, err
	}
	sort.Strings(entries)

	var ran []string
	for _, path := range entries {
		version := path[len("sql/"):]
		if applied[version] {
			continue
		}

		body, err := files.ReadFile(path)
		if err != nil {
			return ran, fmt.Errorf("read %s: %w", version, err)
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			return ran, fmt.Errorf("begin %s: %w", version, err)
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			tx.Rollback(ctx)
			return ran, fmt.Errorf("apply %s: %w", version, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			tx.Rollback(ctx)
			return ran, fmt.Errorf("record %s: %w", version, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return ran, fmt.Errorf("commit %s: %w", version, err)
		}
		ran = append(ran, version)
	}
	return ran, nil
}
