// Package config loads runtime configuration from the environment.
package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL    string
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
		ClerkSecretKey: os.Getenv("CLERK_SECRET_KEY"),
		CronSecret:     os.Getenv("CRON_SECRET"),
		BGGAPIToken:    os.Getenv("BGG_API_TOKEN"),
		Port:           envOr("PORT", "8080"),
		AllowedOrigins: splitAndTrim(envOr("ALLOWED_ORIGINS", "http://localhost:3000")),
	}
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL is required")
	}
	return c, nil
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
