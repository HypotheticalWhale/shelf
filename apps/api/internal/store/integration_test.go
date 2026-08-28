package store_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/HypotheticalWhale/shelf/apps/api/internal/db"
	"github.com/HypotheticalWhale/shelf/apps/api/internal/migrations"
	"github.com/HypotheticalWhale/shelf/apps/api/internal/rating"
	"github.com/HypotheticalWhale/shelf/apps/api/internal/seed"
	"github.com/HypotheticalWhale/shelf/apps/api/internal/store"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

// These tests run against a real Postgres started for the package. The stats
// trigger and the Bayesian ordering are the two things unit tests genuinely
// cannot check — both are SQL, and both are only interesting under concurrency.
//
// Set SHELF_SKIP_DB=1 to skip (the first run downloads a Postgres binary).

const testDSN = "postgres://postgres:postgres@localhost:5433/shelf_test?sslmode=disable"

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	if os.Getenv("SHELF_SKIP_DB") != "" {
		os.Exit(0)
	}

	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Username("postgres").
			Password("postgres").
			Database("shelf_test").
			Port(5433).
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
			fmt.Fprintf(os.Stderr, "connect for migrations: %v\n", err)
			return 1
		}
		if _, err := migrations.Apply(ctx, mpool); err != nil {
			fmt.Fprintf(os.Stderr, "apply migrations: %v\n", err)
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

		return m.Run()
	}()

	if err := pg.Stop(); err != nil {
		fmt.Fprintf(os.Stderr, "stop embedded postgres: %v\n", err)
	}
	os.Exit(code)
}

func newStore(t *testing.T) *store.Store {
	t.Helper()
	return store.New(testPool)
}

// reset clears user-generated data between tests while keeping the catalogue.
func reset(t *testing.T, ctx context.Context) {
	t.Helper()
	for _, q := range []string{
		"DELETE FROM posts",
		"DELETE FROM shelf_items",
		"DELETE FROM ratings",
		"DELETE FROM users",
		"UPDATE game_stats SET num_ratings = 0, rating_sum = 0",
	} {
		if _, err := testPool.Exec(ctx, q); err != nil {
			t.Fatalf("reset (%s): %v", q, err)
		}
	}
}

func seedCatalogue(t *testing.T, ctx context.Context, st *store.Store) {
	t.Helper()
	games, err := seed.Games()
	if err != nil {
		t.Fatalf("seed games: %v", err)
	}
	if _, err := st.UpsertGames(ctx, games); err != nil {
		t.Fatalf("upsert seed: %v", err)
	}
}

func TestMigrationsAreIdempotent(t *testing.T) {
	ctx := context.Background()

	mpool, err := db.NewForMigrations(ctx, testDSN)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer mpool.Close()

	applied, err := migrations.Apply(ctx, mpool)
	if err != nil {
		t.Fatalf("re-apply: %v", err)
	}
	if len(applied) != 0 {
		t.Fatalf("re-running migrations applied %v, want nothing", applied)
	}
}

func TestSeedImportIsIdempotent(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)

	seedCatalogue(t, ctx, st)
	first, err := st.CountGames(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Re-importing must refresh in place, never duplicate.
	seedCatalogue(t, ctx, st)
	second, err := st.CountGames(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("catalogue grew from %d to %d on re-import", first, second)
	}
	if first < 50 {
		t.Fatalf("only %d games imported", first)
	}
}

func TestRatingTriggerKeepsStatsExactUnderConcurrency(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	reset(t, ctx)
	seedCatalogue(t, ctx, st)

	const raters = 40
	for i := range raters {
		if _, err := st.EnsureUser(ctx, fmt.Sprintf("user_%d", i), fmt.Sprintf("rater%d", i), "", ""); err != nil {
			t.Fatalf("ensure user: %v", err)
		}
	}

	// Everyone rates the same game at once — the case where a read-modify-write
	// aggregate would lose updates.
	var wg sync.WaitGroup
	errs := make(chan error, raters)
	for i := range raters {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			value := float64(1 + i%10)
			if _, err := st.SetRating(ctx, fmt.Sprintf("user_%d", i), "wingspan", value); err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent rating failed: %v", err)
	}

	assertStatsMatchRatings(t, ctx, "wingspan")

	game, err := st.GetGameBySlug(ctx, "wingspan", "")
	if err != nil {
		t.Fatal(err)
	}
	if game.NumRatings != raters {
		t.Fatalf("num_ratings = %d, want %d", game.NumRatings, raters)
	}
}

func TestRatingUpdateAndDeleteAdjustStats(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	reset(t, ctx)
	seedCatalogue(t, ctx, st)

	if _, err := st.EnsureUser(ctx, "user_a", "ann", "Ann", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := st.EnsureUser(ctx, "user_b", "bob", "Bob", ""); err != nil {
		t.Fatal(err)
	}

	if _, err := st.SetRating(ctx, "user_a", "azul", 6); err != nil {
		t.Fatal(err)
	}
	if _, err := st.SetRating(ctx, "user_b", "azul", 8); err != nil {
		t.Fatal(err)
	}

	g, err := st.GetGameBySlug(ctx, "azul", "")
	if err != nil {
		t.Fatal(err)
	}
	if g.NumRatings != 2 || g.Mean != 7 {
		t.Fatalf("after two ratings: n=%d mean=%v, want 2 and 7", g.NumRatings, g.Mean)
	}

	// Re-rating must move the sum without changing the count.
	if _, err := st.SetRating(ctx, "user_a", "azul", 10); err != nil {
		t.Fatal(err)
	}
	g, _ = st.GetGameBySlug(ctx, "azul", "")
	if g.NumRatings != 2 || g.Mean != 9 {
		t.Fatalf("after re-rating: n=%d mean=%v, want 2 and 9", g.NumRatings, g.Mean)
	}
	assertStatsMatchRatings(t, ctx, "azul")

	// Deleting must remove exactly one vote and its value.
	if _, err := st.DeleteRating(ctx, "user_b", "azul"); err != nil {
		t.Fatal(err)
	}
	g, _ = st.GetGameBySlug(ctx, "azul", "")
	if g.NumRatings != 1 || g.Mean != 10 {
		t.Fatalf("after delete: n=%d mean=%v, want 1 and 10", g.NumRatings, g.Mean)
	}
	assertStatsMatchRatings(t, ctx, "azul")

	// The viewer's own rating comes back on their request.
	g, _ = st.GetGameBySlug(ctx, "azul", "user_a")
	if g.ViewerRating == nil || *g.ViewerRating != 10 {
		t.Fatalf("viewer rating = %v, want 10", g.ViewerRating)
	}
	g, _ = st.GetGameBySlug(ctx, "azul", "user_b")
	if g.ViewerRating != nil {
		t.Fatalf("viewer rating = %v, want nil after delete", g.ViewerRating)
	}
}

// assertStatsMatchRatings is the invariant the whole scoring model rests on.
func assertStatsMatchRatings(t *testing.T, ctx context.Context, slug string) {
	t.Helper()

	var storedN int
	var storedSum float64
	var actualN int
	var actualSum float64

	err := testPool.QueryRow(ctx, `
		SELECT gs.num_ratings, gs.rating_sum,
		       (SELECT count(*) FROM ratings r WHERE r.game_id = g.id),
		       COALESCE((SELECT sum(value) FROM ratings r WHERE r.game_id = g.id), 0)
		  FROM games g JOIN game_stats gs ON gs.game_id = g.id
		 WHERE g.slug = $1`, slug).Scan(&storedN, &storedSum, &actualN, &actualSum)
	if err != nil {
		t.Fatalf("verify stats: %v", err)
	}

	if storedN != actualN {
		t.Errorf("game_stats.num_ratings = %d but ratings holds %d rows", storedN, actualN)
	}
	if storedSum != actualSum {
		t.Errorf("game_stats.rating_sum = %v but ratings sum to %v", storedSum, actualSum)
	}
}

func TestBayesianRankingBeatsNaiveAverage(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	reset(t, ctx)
	seedCatalogue(t, ctx, st)

	for i := range 30 {
		if _, err := st.EnsureUser(ctx, fmt.Sprintf("u%d", i), fmt.Sprintf("voter%d", i), "", ""); err != nil {
			t.Fatal(err)
		}
	}

	// Establish a realistic global mean first.
	//
	// The prior pulls every game toward C, so C has to reflect the whole site
	// before the comparison below means anything. Rate a spread of other games
	// in the 5-8 band, which is roughly how a real rating distribution sits.
	filler := []string{"catan", "carcassonne", "splendor", "jaipur", "hive",
		"patchwork", "kingdomino", "santorini", "codenames", "love-letter"}
	for gi, slug := range filler {
		for i := range 12 {
			value := float64(5 + (i+gi)%4) // 5..8
			if _, err := st.SetRating(ctx, fmt.Sprintf("u%d", i), slug, value); err != nil {
				t.Fatalf("filler rating %s: %v", slug, err)
			}
		}
	}

	// One person gives an obscure game a perfect 10.
	if _, err := st.SetRating(ctx, "u0", "onitama", 10); err != nil {
		t.Fatal(err)
	}
	// Twenty-five people rate a heavyweight at 9.
	for i := range 25 {
		if _, err := st.SetRating(ctx, fmt.Sprintf("u%d", i), "gloomhaven", 9); err != nil {
			t.Fatal(err)
		}
	}

	if err := st.RefreshGlobalStats(ctx); err != nil {
		t.Fatal(err)
	}

	prior, err := st.Prior(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if prior.MeanRating < 5 || prior.MeanRating > 8 {
		t.Fatalf("global mean %v is outside the realistic band the test assumes", prior.MeanRating)
	}

	lone, err := st.GetGameBySlug(ctx, "onitama", "")
	if err != nil {
		t.Fatal(err)
	}
	classic, err := st.GetGameBySlug(ctx, "gloomhaven", "")
	if err != nil {
		t.Fatal(err)
	}

	// The naive average would rank these the other way round.
	if lone.Mean <= classic.Mean {
		t.Fatalf("test setup is wrong: the lone rating (%v) should have the higher plain mean than %v",
			lone.Mean, classic.Mean)
	}
	if lone.Score >= classic.Score {
		t.Fatalf("a single 10/10 (score %v) outranked 25 nines (score %v) with C=%v",
			lone.Score, classic.Score, prior.MeanRating)
	}

	page, err := st.ListGames(ctx, store.GameFilter{Sort: "score", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Games) == 0 {
		t.Fatal("no games returned")
	}
	if page.Games[0].Slug != "gloomhaven" {
		t.Fatalf("top game is %q, want gloomhaven", page.Games[0].Slug)
	}

	// The score the API returns must match the tested Go implementation.
	top := page.Games[0]
	want := rating.Score(top.RatingSum, top.NumRatings, page.Prior.MeanRating, page.Prior.PriorWeight)
	if top.Score != want {
		t.Fatalf("returned score %v disagrees with rating.Score %v", top.Score, want)
	}

	// SQL ordering and the Go-computed score must agree, or the list would be
	// sorted by one number and displayed with another.
	for i := 1; i < len(page.Games); i++ {
		if page.Games[i-1].Score < page.Games[i].Score-1e-9 {
			t.Fatalf("results not ordered by score: %v then %v",
				page.Games[i-1].Score, page.Games[i].Score)
		}
	}
}

// TestFiltersNarrowResults exercises the placeholder builder, where a clause
// that references one argument twice previously renumbered incorrectly.
func TestFiltersNarrowResults(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	reset(t, ctx)
	seedCatalogue(t, ctx, st)

	two, err := st.ListGames(ctx, store.GameFilter{Players: []int{2}, Limit: 100})
	if err != nil {
		t.Fatalf("filter by players: %v", err)
	}
	if two.Total == 0 {
		t.Fatal("no games support 2 players")
	}
	for _, g := range two.Games {
		if g.MinPlayers == nil || g.MaxPlayers == nil || *g.MinPlayers > 2 || *g.MaxPlayers < 2 {
			t.Fatalf("%s does not support 2 players (%v-%v)", g.Name, g.MinPlayers, g.MaxPlayers)
		}
	}

	seven, err := st.ListGames(ctx, store.GameFilter{Players: []int{7}, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if seven.Total >= two.Total {
		t.Fatalf("7-player filter (%d) should be narrower than 2-player (%d)", seven.Total, two.Total)
	}

	search, err := st.ListGames(ctx, store.GameFilter{Query: "wing", Limit: 10})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if search.Total == 0 {
		t.Fatal(`search for "wing" found nothing`)
	}

	quick, err := st.ListGames(ctx, store.GameFilter{MaxTime: 30, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range quick.Games {
		if g.MaxPlaytime != nil && *g.MaxPlaytime > 30 {
			t.Fatalf("%s runs %d minutes, over the 30 minute filter", g.Name, *g.MaxPlaytime)
		}
	}
}

// TestPostsAndDraftVisibility covers the personal-blog rules.
func TestPostsAndDraftVisibility(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	reset(t, ctx)
	seedCatalogue(t, ctx, st)

	author, err := st.EnsureUser(ctx, "author_1", "sam", "Sam", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.EnsureUser(ctx, "reader_1", "reader", "Reader", ""); err != nil {
		t.Fatal(err)
	}

	game, err := st.GetGameBySlug(ctx, "root", "")
	if err != nil {
		t.Fatal(err)
	}

	draft, err := st.CreatePost(ctx, author.ID, store.NewPost{
		Title: "Root, three plays in", BodyMD: "Still working it out.", GameID: &game.ID,
	})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	if draft.PublishedAt != nil {
		t.Fatal("a post created without publish should be a draft")
	}

	// A draft is invisible to everyone but its author.
	if _, err := st.GetPost(ctx, "sam", draft.Slug, "reader_1"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("reader could see a draft: %v", err)
	}
	if _, err := st.GetPost(ctx, "sam", draft.Slug, author.ID); err != nil {
		t.Fatalf("author cannot see their own draft: %v", err)
	}

	if _, err := st.UpdatePost(ctx, author.ID, draft.ID, nil, nil, ptr(true)); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if _, err := st.GetPost(ctx, "sam", draft.Slug, "reader_1"); err != nil {
		t.Fatalf("reader cannot see a published post: %v", err)
	}

	// Another user must not be able to edit or delete it.
	if _, err := st.UpdatePost(ctx, "reader_1", draft.ID, ptr("Hijacked"), nil, nil); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("a non-author edited the post: %v", err)
	}
	if err := st.DeletePost(ctx, "reader_1", draft.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("a non-author deleted the post: %v", err)
	}

	// A repeated title must still produce a unique slug.
	second, err := st.CreatePost(ctx, author.ID, store.NewPost{
		Title: "Root, three plays in", BodyMD: "Take two.", Publish: true,
	})
	if err != nil {
		t.Fatalf("create second post: %v", err)
	}
	if second.Slug == draft.Slug {
		t.Fatalf("duplicate slug %q", second.Slug)
	}

	feed, err := st.ListRecentPosts(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed) != 2 {
		t.Fatalf("feed holds %d posts, want 2 published", len(feed))
	}
	for _, p := range feed {
		if p.Author == nil || p.Author.Username != "sam" {
			t.Fatalf("feed post is missing its author: %+v", p)
		}
	}
}

func ptr[T any](v T) *T { return &v }

// TestDeletingARatedGameCascades covers the trigger's delete path.
//
// Cascading from games to ratings fires the stats trigger, which used to
// re-insert the game_stats row it was about to lose and trip the foreign key,
// making any rated game impossible to delete.
func TestDeletingARatedGameCascades(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	reset(t, ctx)
	seedCatalogue(t, ctx, st)

	if _, err := st.EnsureUser(ctx, "deleter_1", "deleter", "Deleter", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := st.SetRating(ctx, "deleter_1", "hive", 8); err != nil {
		t.Fatal(err)
	}

	var gameID int64
	if err := testPool.QueryRow(ctx, `SELECT id FROM games WHERE slug = 'hive'`).Scan(&gameID); err != nil {
		t.Fatal(err)
	}

	if _, err := testPool.Exec(ctx, `DELETE FROM games WHERE id = $1`, gameID); err != nil {
		t.Fatalf("deleting a rated game failed: %v", err)
	}

	for _, q := range []string{
		`SELECT count(*) FROM ratings    WHERE game_id = $1`,
		`SELECT count(*) FROM game_stats WHERE game_id = $1`,
	} {
		var n int
		if err := testPool.QueryRow(ctx, q, gameID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("%s left %d rows behind", q, n)
		}
	}
}

// TestClearSeedRemovesEverySeededGame exercises the same path in bulk.
func TestClearSeedRemovesEverySeededGame(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	reset(t, ctx)
	seedCatalogue(t, ctx, st)

	if _, err := st.EnsureUser(ctx, "clear_1", "clearer", "Clearer", ""); err != nil {
		t.Fatal(err)
	}
	for _, slug := range []string{"catan", "azul", "root"} {
		if _, err := st.SetRating(ctx, "clear_1", slug, 7); err != nil {
			t.Fatal(err)
		}
	}

	removed, err := st.DeleteSeedGames(ctx)
	if err != nil {
		t.Fatalf("clear seed: %v", err)
	}
	if removed < 50 {
		t.Fatalf("removed only %d seed games", removed)
	}

	left, err := st.CountGames(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if left != 0 {
		t.Errorf("%d games survived a full seed clear", left)
	}
}

// TestMultiSelectFilters covers the OR-within / AND-across semantics a set of
// filter chips implies.
func TestMultiSelectFilters(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	reset(t, ctx)
	seedCatalogue(t, ctx, st)

	two, err := st.ListGames(ctx, store.GameFilter{Players: []int{2}, Limit: 200})
	if err != nil {
		t.Fatal(err)
	}
	five, err := st.ListGames(ctx, store.GameFilter{Players: []int{5}, Limit: 200})
	if err != nil {
		t.Fatal(err)
	}
	both, err := st.ListGames(ctx, store.GameFilter{Players: []int{2, 5}, Limit: 200})
	if err != nil {
		t.Fatal(err)
	}

	// Player counts describe the group sizes to cover, so asking for two of
	// them narrows: every result must seat both.
	if both.Total > two.Total || both.Total > five.Total {
		t.Fatalf("2-and-5 (%d) is wider than 2 alone (%d) or 5 alone (%d)",
			both.Total, two.Total, five.Total)
	}
	if both.Total == 0 {
		t.Fatal("no games seat both 2 and 5 players, which cannot be right")
	}
	for _, g := range both.Games {
		if g.MinPlayers == nil || g.MaxPlayers == nil || *g.MinPlayers > 2 || *g.MaxPlayers < 5 {
			t.Fatalf("%s cannot seat both 2 and 5 (%v-%v)", g.Name, g.MinPlayers, g.MaxPlayers)
		}
	}

	// Same for mechanics.
	deck, err := st.ListGames(ctx, store.GameFilter{Mechanics: []string{"Deck Building"}, Limit: 200})
	if err != nil {
		t.Fatal(err)
	}
	pair, err := st.ListGames(ctx, store.GameFilter{
		Mechanics: []string{"Deck Building", "Worker Placement"}, Limit: 200,
	})
	if err != nil {
		t.Fatal(err)
	}
	if pair.Total < deck.Total {
		t.Fatalf("two mechanics (%d) returned fewer than one (%d)", pair.Total, deck.Total)
	}
	for _, g := range pair.Games {
		has := false
		for _, m := range g.Mechanics {
			if m == "Deck Building" || m == "Worker Placement" {
				has = true
			}
		}
		if !has {
			t.Fatalf("%s has neither mechanic: %v", g.Name, g.Mechanics)
		}
	}

	// Across facets the filters combine.
	combo, err := st.ListGames(ctx, store.GameFilter{
		Mechanics: []string{"Deck Building"}, Players: []int{2}, Limit: 200,
	})
	if err != nil {
		t.Fatal(err)
	}
	if combo.Total > deck.Total {
		t.Fatalf("adding a player filter widened the result: %d > %d", combo.Total, deck.Total)
	}
}

// TestShelfCountsOnGameDetail covers the figures the game page shows under
// "on other shelves". They are counted per shelf, not per person, so one
// person who owns a game and has also played it must show up in both columns
// — and a game nobody has kept must report three honest zeroes rather than
// inheriting a neighbour's totals.
func TestShelfCountsOnGameDetail(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	reset(t, ctx)
	seedCatalogue(t, ctx, st)

	for _, u := range []string{"shelf_a", "shelf_b", "shelf_c"} {
		if _, err := st.EnsureUser(ctx, u, u, u, ""); err != nil {
			t.Fatal(err)
		}
	}

	// a owns and has played it; b has only played it; c wants it.
	for _, e := range []struct{ user, status string }{
		{"shelf_a", "owned"},
		{"shelf_a", "played"},
		{"shelf_b", "played"},
		{"shelf_c", "wishlist"},
	} {
		if err := st.SetShelfStatus(ctx, e.user, "root", e.status); err != nil {
			t.Fatalf("%s/%s: %v", e.user, e.status, err)
		}
	}

	got, err := st.GetGameBySlug(ctx, "root", "shelf_a")
	if err != nil {
		t.Fatal(err)
	}
	if got.Owners != 1 || got.Players != 2 || got.Wanters != 1 {
		t.Errorf("root: owners=%d players=%d wanters=%d, want 1/2/1",
			got.Owners, got.Players, got.Wanters)
	}

	// Setting the same status twice is idempotent — the button is a toggle,
	// not a counter.
	if err := st.SetShelfStatus(ctx, "shelf_a", "root", "owned"); err != nil {
		t.Fatal(err)
	}
	if got, err = st.GetGameBySlug(ctx, "root", ""); err != nil {
		t.Fatal(err)
	}
	if got.Owners != 1 {
		t.Errorf("re-owning counted twice: owners=%d, want 1", got.Owners)
	}

	// Taking it off one shelf must not disturb the others.
	if err := st.RemoveShelfStatus(ctx, "shelf_a", "root", "owned"); err != nil {
		t.Fatal(err)
	}
	if got, err = st.GetGameBySlug(ctx, "root", ""); err != nil {
		t.Fatal(err)
	}
	if got.Owners != 0 || got.Players != 2 || got.Wanters != 1 {
		t.Errorf("after removal: owners=%d players=%d wanters=%d, want 0/2/1",
			got.Owners, got.Players, got.Wanters)
	}

	// An untouched game reports zeroes, which is what drives the empty state.
	other, err := st.GetGameBySlug(ctx, "azul", "")
	if err != nil {
		t.Fatal(err)
	}
	if other.Owners != 0 || other.Players != 0 || other.Wanters != 0 {
		t.Errorf("azul: owners=%d players=%d wanters=%d, want 0/0/0",
			other.Owners, other.Players, other.Wanters)
	}
}

// TestUpsertGameWithNoDesigners guards a refresh against the games that carry
// no credited designer, no category or no mechanic. The columns are NOT NULL
// DEFAULT '{}', and a nil Go slice sends NULL rather than an empty array — one
// such game (Space Hulk, Fourth Edition) aborted a full catalogue refresh 660
// games in.
func TestUpsertGameWithNoDesigners(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	reset(t, ctx)

	year := 2014
	n, err := st.UpsertGames(ctx, []store.GameInput{{
		BGGID:         165838,
		Name:          "Space Hulk (Fourth Edition)",
		YearPublished: &year,
		Source:        "bgg",
		// Designers, Categories and Mechanics deliberately left nil.
	}})
	if err != nil {
		t.Fatalf("upserting a game with no designers failed: %v", err)
	}
	if n != 1 {
		t.Fatalf("wrote %d games, want 1", n)
	}

	got, err := st.GetGameBySlug(ctx, "space-hulk-fourth-edition", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Designers) != 0 || len(got.Categories) != 0 || len(got.Mechanics) != 0 {
		t.Errorf("expected empty tag arrays, got designers=%v categories=%v mechanics=%v",
			got.Designers, got.Categories, got.Mechanics)
	}

	// And it must still be updatable, since a refresh upserts over it.
	if _, err := st.UpsertGames(ctx, []store.GameInput{{
		BGGID: 165838, Name: "Space Hulk (Fourth Edition)",
		YearPublished: &year, Source: "bgg",
		Designers: []string{"Richard Halliwell"},
	}}); err != nil {
		t.Fatalf("re-upserting failed: %v", err)
	}
}

// TestRefreshKeepsArtworkProvenanceHonest covers the two ways a refresh can
// lie about a cover: by crediting the wrong source, and by discarding art it
// cannot replace.
func TestRefreshKeepsArtworkProvenanceHonest(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	reset(t, ctx)

	const id = 999001
	mk := func(img string) []store.GameInput {
		return []store.GameInput{{
			BGGID: id, Name: "Provenance Test", Source: "bgg", ImageURL: img,
		}}
	}

	// The aggregation pipeline found a cover elsewhere and credited it.
	if _, err := st.UpsertGames(ctx, mk("https://blob.example/cover.png")); err != nil {
		t.Fatal(err)
	}
	if _, err := testPool.Exec(ctx,
		`UPDATE games SET image_credit = 'Wikimedia Commons', image_source = 'commons'
		  WHERE bgg_id = $1`, id); err != nil {
		t.Fatal(err)
	}

	// A refresh with no artwork must not throw away the cover we already have.
	if _, err := st.UpsertGames(ctx, mk("")); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetGameBySlug(ctx, "provenance-test", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.ImageURL == nil || *got.ImageURL != "https://blob.example/cover.png" {
		t.Fatalf("an empty refresh discarded the existing cover: %v", got.ImageURL)
	}
	if got.ImageCredit == nil || *got.ImageCredit != "Wikimedia Commons" {
		t.Errorf("credit changed without the image: %v", got.ImageCredit)
	}

	// A refresh that does supply artwork supersedes it — and the credit has to
	// move with the image rather than staying behind on the old source.
	if _, err := st.UpsertGames(ctx, mk("https://cf.geekdo-images.com/real.png")); err != nil {
		t.Fatal(err)
	}
	if got, err = st.GetGameBySlug(ctx, "provenance-test", ""); err != nil {
		t.Fatal(err)
	}
	if got.ImageURL == nil || *got.ImageURL != "https://cf.geekdo-images.com/real.png" {
		t.Fatalf("refresh did not take the new cover: %v", got.ImageURL)
	}
	if got.ImageCredit == nil || *got.ImageCredit != "BoardGameGeek" {
		t.Errorf("cover is BoardGameGeek's but credit says %v", got.ImageCredit)
	}
}
