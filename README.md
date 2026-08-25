# Shelf

A board game collection and rating site — BoardGameGeek's data model with an
interface built around one idea: **rating a game should take one tap.**

- Rate from the browse grid, not just a game page. No form, no submit button.
- Aggregate scores use a **Bayesian average**, so one enthusiastic vote can't
  push an unknown game to number one.
- Every account is a **personal blog**: a shelf, a rating history, and writing.

Go API + TypeScript frontend, both deployed on Vercel.

---

## Stack

| Piece | Choice | Why |
|---|---|---|
| API | **Go 1.26**, chi, pgx | Vercel's Go preset detects `cmd/api/main.go` with zero config |
| Frontend | **Next.js 16**, TypeScript, Tailwind 4, motion | Server components fetch the API directly for first paint |
| Database | **Postgres** (Neon in production) | A trigger keeps rating aggregates exact under concurrency |
| Auth | **Clerk** | `@clerk/nextjs` on the frontend; `clerk-sdk-go/v2` verifies the JWT in Go |

The web app rewrites `/api/*` to the Go service, so the browser only ever talks
to one origin — no CORS preflight on the interaction the product is built around.

```
shelf/
├── apps/api/    Go service  → Vercel project "shelf-api"
└── apps/web/    Next.js app → Vercel project "shelf"
```

---

## Run it locally

No Postgres install required — `devdb` downloads and supervises its own.

```bash
# 1. Database (first run downloads Postgres into .devdb/)
cd apps/api && go run ./cmd/devdb

# 2. API — in a second terminal
cd apps/api && go run ./cmd/api          # :8080

# 3. Frontend — in a third terminal
cd apps/web && npm install && npm run dev # :3000
```

`devdb` applies migrations and loads the bundled 65-game catalogue on start.

Copy `.env.example` to `.env.local` and fill it in. Auth needs Clerk keys:
`cd apps/web && npx clerk@latest init` writes them without needing an account.

> **Node 20.12+ is required** for the Clerk CLI, and Next 16's lint tooling wants
> 20.19+. Vercel builds on Node 24.

## Tests

```bash
cd apps/api && go test ./...
```

Unit tests cover the scoring maths and the BGG XML parser. The integration and
API tests start a real Postgres in-process, so they exercise the migrations, the
stats trigger under concurrent writes, and the live HTTP routes. Set
`SHELF_SKIP_DB=1` to skip the database-backed ones.

---

## How scoring works

```
shelf_score = (rating_sum + m × C) / (num_ratings + m)
```

`C` is the site-wide mean rating, `m` is the prior weight (5 — BGG uses 100,
right for their volume and far too aggressive for a new site).

A plain average is the wrong tool: a single 10/10 would rank an unknown game
above Gloomhaven. The prior pulls sparsely-rated games toward the mean and
releases them as real votes arrive. An unrated game scores exactly `C`, and the
UI says "not rated yet" rather than printing that number.

`game_stats` is kept exact by a Postgres trigger that applies O(1) deltas on
every rating insert, update and delete. The upsert takes a row lock, so
simultaneous raters serialise and no update is lost — there is no reconciliation
job, because the numbers cannot drift. `C` is refreshed hourly by cron.

## The catalogue

**BoardGameGeek closed its XML API in late 2025.** It now requires a bearer
token issued when you register an application; unauthenticated requests return
401 regardless of user agent.

Shelf therefore ships a small hand-curated catalogue (65 well-known games:
titles, years, player counts, playtimes, designers, tags) so the site is usable
immediately and local development never depends on an external API. Those rows
are marked `source = 'seed'`.

To import the real thing, register at
<https://boardgamegeek.com/using_the_xml_api>, set `BGG_API_TOKEN`, then:

```bash
cd apps/api
go run ./cmd/import -hot                      # BGG's current hot list
go run ./cmd/import -sweep 1:200000 -min 750  # discover popular games (slow, resumable)
go run ./cmd/import -refresh 200              # refresh stale metadata
go run ./cmd/import -clear-seed               # drop seed rows BGG never matched
```

Importing by `bgg_id` promotes a seeded row to `source = 'bgg'` and corrects
every field, so the two catalogues reconcile rather than duplicate.

BGG permits non-commercial use with attribution — the footer credits them.
**Monetising Shelf would require a commercial licence from BGG.**

---

## Deploying

Two Vercel projects from this one repo, distinguished by Root Directory.

```bash
# API
cd apps/api && vercel link          # Root Directory: apps/api
vercel env add DATABASE_URL         # Neon pooled connection string
vercel env add CLERK_SECRET_KEY
vercel env add CRON_SECRET
vercel env add BGG_API_TOKEN        # optional
vercel --prod

# Frontend
cd apps/web && vercel link          # Root Directory: apps/web
vercel env add SHELF_API_URL        # the API deployment's URL
vercel env add CLERK_SECRET_KEY
vercel env add NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY
vercel --prod
```

Provision the database with `vercel integration add neon` and use the **pooled**
connection string — pgx is configured for PgBouncer transaction mode.

Run migrations once against production with `DATABASE_URL=… go run ./cmd/migrate`,
or set `MIGRATE_ON_BOOT=1` and let the API migrate itself (an advisory lock makes
concurrent boots safe).

`apps/api/vercel.json` registers two crons: refresh the global mean hourly, and
refresh catalogue metadata weekly.
