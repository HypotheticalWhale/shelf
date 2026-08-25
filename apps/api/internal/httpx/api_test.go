package httpx_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/HypotheticalWhale/shelf/apps/api/internal/config"
	"github.com/HypotheticalWhale/shelf/apps/api/internal/db"
	"github.com/HypotheticalWhale/shelf/apps/api/internal/httpx"
	"github.com/HypotheticalWhale/shelf/apps/api/internal/migrations"
	"github.com/HypotheticalWhale/shelf/apps/api/internal/seed"
	"github.com/HypotheticalWhale/shelf/apps/api/internal/store"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Exercises the real router against a real database. Clerk is not configured
// here, so every request is anonymous — which is exactly what is needed to pin
// down the public surface and prove the write endpoints are closed.

const testDSN = "postgres://postgres:postgres@localhost:5434/shelf_api_test?sslmode=disable"

var (
	testPool *pgxpool.Pool
	testSrv  *httptest.Server
)

func TestMain(m *testing.M) {
	if os.Getenv("SHELF_SKIP_DB") != "" {
		os.Exit(0)
	}

	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Username("postgres").
			Password("postgres").
			Database("shelf_api_test").
			Port(5434).
			RuntimePath(os.TempDir() + "/shelf-api-pg").
			StartTimeout(90 * time.Second),
	)
	if err := pg.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "start embedded postgres: %v\n", err)
		os.Exit(1)
	}

	code := func() int {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		mpool, err := db.NewForMigrations(ctx, testDSN)
		if err != nil {
			fmt.Fprintf(os.Stderr, "connect: %v\n", err)
			return 1
		}
		if _, err := migrations.Apply(ctx, mpool); err != nil {
			fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
			mpool.Close()
			return 1
		}
		mpool.Close()

		testPool, err = db.New(ctx, testDSN)
		if err != nil {
			fmt.Fprintf(os.Stderr, "connect: %v\n", err)
			return 1
		}
		defer testPool.Close()

		st := store.New(testPool)
		games, err := seed.Games()
		if err != nil {
			fmt.Fprintf(os.Stderr, "seed: %v\n", err)
			return 1
		}
		if _, err := st.UpsertGames(ctx, games); err != nil {
			fmt.Fprintf(os.Stderr, "upsert seed: %v\n", err)
			return 1
		}

		cfg := config.Config{CronSecret: "test-cron-secret", Port: "0"}
		testSrv = httptest.NewServer(httpx.NewServer(cfg, st).Handler())
		defer testSrv.Close()

		return m.Run()
	}()

	if err := pg.Stop(); err != nil {
		fmt.Fprintf(os.Stderr, "stop embedded postgres: %v\n", err)
	}
	os.Exit(code)
}

func get(t *testing.T, path string) (*http.Response, []byte) {
	t.Helper()
	return do(t, http.MethodGet, path, nil)
}

func do(t *testing.T, method, path string, headers map[string]string) (*http.Response, []byte) {
	t.Helper()

	req, err := http.NewRequest(method, testSrv.URL+path, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := testSrv.Client().Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if err != nil {
			break
		}
	}
	return resp, buf
}

func TestHealth(t *testing.T) {
	resp, body := get(t, "/health")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", resp.StatusCode, body)
	}

	var payload struct {
		Status   string `json:"status"`
		Games    int    `json:"games"`
		BGGToken bool   `json:"bggToken"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode: %v (%s)", err, body)
	}
	if payload.Status != "ok" {
		t.Errorf("status = %q", payload.Status)
	}
	if payload.Games < 50 {
		t.Errorf("games = %d, want the seeded catalogue", payload.Games)
	}
	if payload.BGGToken {
		t.Error("bggToken should be false with no token configured")
	}
}

func TestBrowseAndDetail(t *testing.T) {
	resp, body := get(t, "/games?limit=5")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", resp.StatusCode, body)
	}

	var page struct {
		Games []store.Game `json:"games"`
		Total int          `json:"total"`
		Prior store.Prior  `json:"prior"`
	}
	if err := json.Unmarshal(body, &page); err != nil {
		t.Fatalf("decode: %v (%s)", err, body)
	}
	if len(page.Games) != 5 {
		t.Fatalf("got %d games, want 5", len(page.Games))
	}
	if page.Total < 50 {
		t.Errorf("total = %d", page.Total)
	}
	if page.Prior.PriorWeight == 0 {
		t.Error("prior weight should be exposed so the UI can explain the score")
	}
	for _, g := range page.Games {
		if g.Slug == "" || g.Name == "" {
			t.Errorf("game missing identity: %+v", g)
		}
		// With no ratings yet, every game should sit exactly at the mean.
		if g.NumRatings == 0 && g.Score != page.Prior.MeanRating {
			t.Errorf("%s has no ratings but scores %v, want the global mean %v",
				g.Slug, g.Score, page.Prior.MeanRating)
		}
	}

	resp, body = get(t, "/games/"+page.Games[0].Slug)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("detail status %d: %s", resp.StatusCode, body)
	}

	var detail struct {
		Slug      string  `json:"slug"`
		Histogram [10]int `json:"histogram"`
	}
	if err := json.Unmarshal(body, &detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if detail.Slug != page.Games[0].Slug {
		t.Errorf("detail slug = %q", detail.Slug)
	}
}

func TestUnknownGameIs404(t *testing.T) {
	resp, _ := get(t, "/games/definitely-not-a-real-game")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestWriteEndpointsRequireAuth(t *testing.T) {
	// Every mutating route must be closed to anonymous callers.
	cases := []struct{ method, path string }{
		{http.MethodPut, "/games/wingspan/rating"},
		{http.MethodDelete, "/games/wingspan/rating"},
		{http.MethodPut, "/shelf/wingspan"},
		{http.MethodDelete, "/shelf/wingspan?status=owned"},
		{http.MethodPost, "/posts"},
		{http.MethodPatch, "/posts/1"},
		{http.MethodDelete, "/posts/1"},
		{http.MethodGet, "/me"},
		{http.MethodPatch, "/me"},
	}
	for _, c := range cases {
		resp, body := do(t, c.method, c.path, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s %s = %d, want 401 (%s)", c.method, c.path, resp.StatusCode, body)
		}
	}
}

func TestCronRequiresSecret(t *testing.T) {
	resp, _ := do(t, http.MethodPost, "/cron/refresh-stats", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no secret = %d, want 401", resp.StatusCode)
	}

	resp, _ = do(t, http.MethodPost, "/cron/refresh-stats",
		map[string]string{"Authorization": "Bearer wrong-secret"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong secret = %d, want 401", resp.StatusCode)
	}

	resp, body := do(t, http.MethodPost, "/cron/refresh-stats",
		map[string]string{"Authorization": "Bearer test-cron-secret"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("valid secret = %d, want 200 (%s)", resp.StatusCode, body)
	}
}

func TestCronImportSkipsWithoutToken(t *testing.T) {
	resp, body := do(t, http.MethodPost, "/cron/import",
		map[string]string{"Authorization": "Bearer test-cron-secret"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", resp.StatusCode, body)
	}

	var payload struct {
		Skipped bool   `json:"skipped"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !payload.Skipped {
		t.Fatal("import should report itself skipped without a BGG token")
	}
}

func TestUnknownUserIs404(t *testing.T) {
	resp, _ := get(t, "/users/nobody-here")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestBadTokenStillAllowsPublicReads(t *testing.T) {
	// An expired or malformed session must not take down the public site.
	// Clerk's middleware answers 401 by default for anything it cannot verify,
	// which would break browsing for any visitor with a stale cookie.
	bad := map[string]string{"Authorization": "Bearer not-a-real-token"}

	for _, path := range []string{"/games", "/games/wingspan", "/posts", "/health"} {
		resp, body := do(t, http.MethodGet, path, bad)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s with a bad token = %d, want 200 (%s)", path, resp.StatusCode, body)
		}
	}

	// It must still count as signed out where authentication is required.
	resp, _ := do(t, http.MethodGet, "/me", bad)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("GET /me with a bad token = %d, want 401", resp.StatusCode)
	}
}

func TestCronAcceptsGetForVercelSchedules(t *testing.T) {
	// Vercel Cron issues GET, not POST. If these routes were POST-only every
	// scheduled run would 405 and the failure would be silent.
	resp, body := do(t, http.MethodGet, "/cron/refresh-stats",
		map[string]string{"Authorization": "Bearer test-cron-secret"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /cron/refresh-stats = %d, want 200 (%s)", resp.StatusCode, body)
	}

	resp, _ = do(t, http.MethodGet, "/cron/refresh-stats", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated GET = %d, want 401", resp.StatusCode)
	}
}
