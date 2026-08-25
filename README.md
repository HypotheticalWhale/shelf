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

Shelf therefore builds its catalogue from two sources, neither of which needs a
token. Both mark their rows `source = 'seed'`, and both key on `bgg_id`, so a
real BGG import later reconciles every row in place rather than duplicating it.

**A hand-curated core of 65 well-known games** (`import -seed`) — accurate
titles, years, player counts, playtimes, designers, categories and mechanics,
plus BGG-scale complexity weights. This is the highest-quality data in the
catalogue and a bulk import never overwrites it.

**A bulk catalogue of ~31,000 games** (`import -catalogue`) from two published
snapshots: a [daily rankings
file](https://github.com/beefsack/bgg-ranking-historicals) for breadth and
recency, and a 2016 metadata snapshot for player counts, playtime, categories
and mechanics where the two overlap. Roughly a third of these carry full
metadata; the rest have title and year until a BGG import fills them in.

```bash
go run ./cmd/import -catalogue            # everything (~31k)
go run ./cmd/import -catalogue -top 5000  # highest-ranked only
```

Deliberately **not** imported from those snapshots: BGG's own ratings, which
would make Shelf's own rankings meaningless; descriptions, which are members'
copyrighted text; and images, since the rank file's thumbnails are 64×64 and the
CDN's transforms are HMAC-signed, so no larger variant can be derived.

For the curated 65, **complexity weights are close community-consensus figures
on BGG's 1–5 scale, not exact values** — good enough to sort and filter by, and
overwritten the moment a real import runs. The bulk catalogue has no weights.

**Cover art is deliberately absent.** Box art is copyrighted and there is no
freely-licensed bulk source: Wikipedia's covers are non-free fair-use files
whose licence does not extend to this site, and Wikidata's board game images are
often photos of components rather than boxes. The UI draws a typographic cover
from each game's initials instead, and real art arrives with a BGG token.

Every seeded `bgg_id` is cross-checked against Wikidata's BoardGameGeek ID
property, because a wrong id would not fail loudly — it would quietly overwrite
a game with a different game's data on the next import:

```bash
SEED_VERIFY=1 go test ./internal/seed -run Wikidata -v
```

To import the real thing — box art, exact weights, full tagging — you need a
BGG token. **Approval is manual and takes a few days**, so start it early:

1. Sign in at <https://boardgamegeek.com/using_the_xml_api> and create a new
   application.
2. Wait for the approval mail from `api@boardgamegeek.com`.
3. Return to the applications page and create a token for that application.
4. `vercel env add BGG_API_TOKEN production --value <token>` on `shelf-api`,
   and add it to `.env.local` for local runs.

Then:

Once the token is set, this upgrades every cover in the catalogue from the
64x64 crop the public snapshots carry to BGG's real artwork, and replaces the
approximate complexity weights with exact ones. It respects BGG's rate limit,
so a full run over ~31,000 games takes roughly an hour:

```bash
go run ./cmd/import -refresh 31500
```

Other modes:

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

`apps/api/vercel.json` registers two crons: refresh the global mean daily, and
refresh catalogue metadata weekly.

Vercel's Hobby plan allows at most one cron run per day, which is why the mean
is refreshed daily rather than hourly. It moves slowly, and scores are computed
at query time from the stored value, so nothing is stale in between. On Pro,
tightening `0 3 * * *` to `0 * * * *` is the only change needed.
