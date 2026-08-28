// Command stats prints catalogue coverage. Handy for deciding what needs
// importing next.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/HypotheticalWhale/shelf/apps/api/internal/config"
	"github.com/HypotheticalWhale/shelf/apps/api/internal/db"
)

func main() {
	config.LoadDotEnv()
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	var total, cats, players, both int
	if err := pool.QueryRow(ctx, `
		SELECT count(*),
		       count(*) FILTER (WHERE cardinality(categories) > 0),
		       count(*) FILTER (WHERE min_players IS NOT NULL),
		       count(*) FILTER (WHERE cardinality(categories) > 0 AND min_players IS NOT NULL)
		  FROM games`).Scan(&total, &cats, &players, &both); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  total=%d  withCategories=%d  withPlayers=%d  withBoth=%d\n\n", total, cats, players, both)

	// Artwork coverage, split by where the cover came from. The backfill from
	// BoardGameGeek runs for hours, so this is the way to see how far it got.
	var art, fromBGG, aggregated, weights int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE image_url IS NOT NULL),
		       count(*) FILTER (WHERE image_url LIKE '%geekdo-images.com%'),
		       count(*) FILTER (WHERE image_url IS NOT NULL
		                          AND image_url NOT LIKE '%geekdo-images.com%'),
		       count(*) FILTER (WHERE weight IS NOT NULL AND weight > 0)
		  FROM games`).Scan(&art, &fromBGG, &aggregated, &weights); err != nil {
		log.Fatal(err)
	}
	pct := func(n int) float64 {
		if total == 0 {
			return 0
		}
		return float64(n) * 100 / float64(total)
	}
	fmt.Printf("  artwork:  %d of %d (%.1f%%)  —  bgg=%d  aggregated=%d\n",
		art, total, pct(art), fromBGG, aggregated)
	fmt.Printf("  weights:  %d of %d (%.1f%%)\n\n", weights, total, pct(weights))

	rows, err := pool.Query(ctx,
		`SELECT c, count(*) n FROM (SELECT unnest(categories) c FROM games) t GROUP BY c ORDER BY n DESC LIMIT 20`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var c string
		var n int
		if err := rows.Scan(&c, &n); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  %-30s %d\n", c, n)
	}
}
