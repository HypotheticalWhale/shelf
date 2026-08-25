package config

import (
	"strings"
	"testing"
)

func TestMigrationURLDerivesDirectEndpoint(t *testing.T) {
	t.Setenv("MIGRATION_DATABASE_URL", "")

	pooled := "postgresql://u:p@ep-gentle-sunset-aw8ga60d-pooler.c-12.us-east-1.aws.neon.tech/shelf?sslmode=require"
	got := directURL(pooled)

	if strings.Contains(got, "-pooler") {
		t.Fatalf("pooler host survived: %s", got)
	}
	if !strings.Contains(got, "ep-gentle-sunset-aw8ga60d.c-12.us-east-1.aws.neon.tech") {
		t.Fatalf("derived host is wrong: %s", got)
	}
	if !strings.Contains(got, "sslmode=require") {
		t.Fatalf("query string was dropped: %s", got)
	}
}

func TestMigrationURLFollowsDatabaseURL(t *testing.T) {
	t.Setenv("MIGRATION_DATABASE_URL", "")

	// The bug this guards: pointing DATABASE_URL at a local database must not
	// leave migrations aimed at a remote one.
	local := "postgres://shelf:shelf@localhost:5432/shelf?sslmode=disable"
	if got := directURL(local); got != local {
		t.Fatalf("directURL(%q) = %q, want it unchanged", local, got)
	}
}

func TestMigrationURLExplicitOverrideWins(t *testing.T) {
	t.Setenv("MIGRATION_DATABASE_URL", "postgres://direct/db")
	if got := directURL("postgres://pooled-pooler/db"); got != "postgres://direct/db" {
		t.Fatalf("explicit override ignored, got %q", got)
	}
}
