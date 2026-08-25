// Package config loads runtime configuration from the environment.
package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL string
	// MigrationURL is a direct (non-pooled) connection used for schema work.
	MigrationURL   string
	ClerkSecretKey string
	CronSecret     string
	BGGAPIToken    string
	Port           string
	// AllowedOrigins is only consulted in local development. In production the
	// Next.js app rewrites /api/* to this service, so requests are same-origin
	// and never trigger a CORS preflight.
	AllowedOrigins []string
}

func Load() (Config, error) {
	c := Config{
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		MigrationURL:   migrationURL(),
		ClerkSecretKey: os.Getenv("CLERK_SECRET_KEY"),
		CronSecret:     os.Getenv("CRON_SECRET"),
		BGGAPIToken:    os.Getenv("BGG_API_TOKEN"),
		Port:           envOr("PORT", "8080"),
		AllowedOrigins: splitAndTrim(envOr("ALLOWED_ORIGINS", "http://localhost:3000")),
	}
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL is required")
	}
	if c.MigrationURL == "" {
		c.MigrationURL = c.DatabaseURL
	}
	return c, nil
}

// migrationURL picks a direct connection for schema changes.
//
// Neon's DATABASE_URL points at its PgBouncer pooler in transaction mode, where
// a connection is handed back to the pool between statements. The migrator
// holds a session-level pg_advisory_lock across the whole run, and a session
// that does not survive between statements cannot hold one — the lock could be
// taken on one backend and released on another, letting two deploys migrate at
// once. Neon publishes an unpooled endpoint for exactly this, so prefer it.
func migrationURL() string {
	for _, key := range []string{"DATABASE_URL_UNPOOLED", "POSTGRES_URL_NON_POOLING", "MIGRATION_DATABASE_URL"} {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return ""
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
