// Command import fills the catalogue from BoardGameGeek.
//
//	go run ./cmd/import -seed                      load the bundled catalogue
//	go run ./cmd/import -hot                       seed from BGG's hot list
//	go run ./cmd/import -ids 174430,224517         import specific games
//	go run ./cmd/import -sweep 1:200000 -min 750   discover popular games
//	go run ./cmd/import -refresh 200               refresh stale metadata
//
// A sweep is resumable: already-imported IDs are skipped, so it can be stopped
// and restarted freely.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/HypotheticalWhale/shelf/apps/api/internal/config"
	"github.com/HypotheticalWhale/shelf/apps/api/internal/db"
	"github.com/HypotheticalWhale/shelf/apps/api/internal/importer"
	"github.com/HypotheticalWhale/shelf/apps/api/internal/seed"
	"github.com/HypotheticalWhale/shelf/apps/api/internal/store"
)

func main() {
	var (
		useSeed    = flag.Bool("seed", false, "load the bundled catalogue (no BGG token needed)")
		clearSeed  = flag.Bool("clear-seed", false, "delete catalogue rows still marked as seed data")
		hot        = flag.Bool("hot", false, "import BGG's current hot list")
		ids        = flag.String("ids", "", "comma-separated BGG IDs to import")
		sweep      = flag.String("sweep", "", "BGG ID range to scan, as from:to")
		minRatings = flag.Int("min", 500, "with -sweep, the minimum BGG rating count to keep a game")
		refresh    = flag.Int("refresh", 0, "re-fetch this many least-recently-imported games")
	)
	flag.Parse()

	config.LoadDotEnv()

	if (*hot || *ids != "" || *sweep != "" || *refresh > 0) && os.Getenv("BGG_API_TOKEN") == "" {
		log.Fatal("this mode needs BGG_API_TOKEN.\n\n" +
			"BoardGameGeek closed its XML API in late 2025; requests without a bearer\n" +
			"token return 401. Register an application at\n" +
			"  https://boardgamegeek.com/using_the_xml_api\n" +
			"then set BGG_API_TOKEN in .env.local.\n\n" +
			"Until then, load the bundled catalogue instead:\n" +
			"  go run ./cmd/import -seed")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// A sweep can run for hours; Ctrl-C should stop it cleanly and keep every
	// game written so far.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	pool, err := db.New(connectCtx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	st := store.New(pool)
	im := importer.New(st, cfg.BGGAPIToken)

	if (*hot || *ids != "" || *sweep != "" || *refresh > 0) && !im.HasToken() {
		log.Fatalf("this mode needs BGG_API_TOKEN.\n\n" +
			"BoardGameGeek closed its XML API in late 2025; requests without a bearer\n" +
			"token return 401. Register an application at\n" +
			"  https://boardgamegeek.com/using_the_xml_api\n" +
			"then set BGG_API_TOKEN in .env.local.\n\n" +
			"Until then, load the bundled catalogue instead:\n" +
			"  go run ./cmd/import -seed")
	}
	started := time.Now()

	var res importer.Result
	switch {
	case *useSeed:
		games, serr := seed.Games()
		if serr != nil {
			log.Fatalf("seed: %v", serr)
		}
		var written int
		written, err = st.UpsertGames(ctx, games)
		res = importer.Result{Requested: len(games), Fetched: len(games), Written: written}

	case *clearSeed:
		var removed int
		removed, err = st.DeleteSeedGames(ctx)
		log.Printf("removed %d unreconciled seed games", removed)

	case *hot:
		res, err = im.ImportHot(ctx)
	case *ids != "":
		parsed, perr := parseIDs(*ids)
		if perr != nil {
			log.Fatalf("parse -ids: %v", perr)
		}
		res, err = im.ImportIDs(ctx, parsed, 0)
	case *sweep != "":
		from, to, perr := parseRange(*sweep)
		if perr != nil {
			log.Fatalf("parse -sweep: %v", perr)
		}
		log.Printf("sweeping BGG IDs %d..%d keeping games with >= %d ratings", from, to, *minRatings)
		res, err = im.Sweep(ctx, from, to, *minRatings)
	case *refresh > 0:
		res, err = im.RefreshStale(ctx, *refresh)
	default:
		flag.Usage()
		os.Exit(2)
	}

	total, cerr := st.CountGames(context.WithoutCancel(ctx))
	if cerr == nil {
		log.Printf("catalogue now holds %d games", total)
	}
	log.Printf("requested=%d fetched=%d written=%d skipped=%d in %s",
		res.Requested, res.Fetched, res.Written, res.Skipped, started.Round(time.Second))

	if err != nil {
		// Partial progress is still progress — report the failure but do not
		// pretend nothing was imported.
		log.Fatalf("import ended early: %v", err)
	}
}

func parseIDs(raw string) ([]int, error) {
	parts := strings.Split(raw, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

func parseRange(raw string) (int, int, error) {
	fromStr, toStr, ok := strings.Cut(raw, ":")
	if !ok {
		return 0, 0, errFormat
	}
	from, err := strconv.Atoi(strings.TrimSpace(fromStr))
	if err != nil {
		return 0, 0, err
	}
	to, err := strconv.Atoi(strings.TrimSpace(toStr))
	if err != nil {
		return 0, 0, err
	}
	return from, to, nil
}

var errFormat = errStr("expected a range like 1:200000")

type errStr string

func (e errStr) Error() string { return string(e) }
