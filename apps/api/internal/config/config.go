// Package config loads runtime configuration from the environment.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL string
	// DirectURL is a non-pooled connection. Schema changes need it for their
	// session-level advisory lock, and LISTEN/NOTIFY needs it because a
	// transaction pooler drops the subscription between statements.
	DirectURL      string
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
		DatabaseURL: os.Getenv("DATABASE_URL"),

		ClerkSecretKey: os.Getenv("CLERK_SECRET_KEY"),
		CronSecret:     os.Getenv("CRON_SECRET"),
		BGGAPIToken:    os.Getenv("BGG_API_TOKEN"),
		Port:           envOr("PORT", "8080"),
		AllowedOrigins: splitAndTrim(envOr("ALLOWED_ORIGINS", "http://localhost:3000")),
	}
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL is required")
	}
	c.DirectURL = directURL(c.DatabaseURL)
	if c.DirectURL == "" {
		c.DirectURL = c.DatabaseURL
	}
	return c, nil
}

// directURL derives a non-pooled connection from databaseURL.
//
// Neon's DATABASE_URL points at its PgBouncer pooler in transaction mode, where
// a connection returns to the pool between statements. The migrator holds a
// session-level pg_advisory_lock across the whole run, and a session that does
// not survive between statements cannot hold one — the lock could be taken on
// one backend and released on another, letting two deploys migrate at once.
//
// The direct endpoint is derived from the pooled one by dropping "-pooler" from
// the host rather than read from a separate environment variable. Reading
// DATABASE_URL_UNPOOLED independently meant that overriding DATABASE_URL — to
// point at a local database, say — left migrations still aimed at whatever the
// env file described, silently migrating the wrong database. Deriving it keeps
// the two in lockstep by construction.
//
// MIGRATION_DATABASE_URL overrides this for setups that publish an unrelated
// direct endpoint.
func directURL(databaseURL string) string {
	if v := os.Getenv("MIGRATION_DATABASE_URL"); v != "" {
		return v
	}
	if databaseURL == "" {
		return ""
	}

	u, err := url.Parse(databaseURL)
	if err != nil || u.Host == "" {
		return databaseURL
	}
	if !strings.Contains(u.Host, "-pooler") {
		return databaseURL // already direct, or not a pooled provider
	}

	u.Host = strings.Replace(u.Host, "-pooler", "", 1)
	return u.String()
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
