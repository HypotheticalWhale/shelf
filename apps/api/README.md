# Shelf API

Go service backing Shelf. See the [root README](../../README.md) for the full
picture.

```
cmd/api        HTTP server — the entrypoint Vercel's Go preset detects
cmd/migrate    applies migrations and exits
cmd/import     fills the catalogue from BoardGameGeek, or the bundled seed
cmd/devdb      throwaway local Postgres, no install required

internal/
  auth         Clerk session verification and just-in-time user provisioning
  bgg          BoardGameGeek XML API2 client (needs BGG_API_TOKEN since 2025)
  httpx        router, middleware, handlers
  importer     ties the BGG client to the store
  migrations   embedded SQL, applied under an advisory lock
  rating       the Bayesian scoring model
  seed         bundled 65-game catalogue
  store        pgx queries
```

## Routes

| Method | Path | Auth | |
|---|---|---|---|
| GET | `/health` | – | status, catalogue size, whether a BGG token is set |
| GET | `/games` | – | `q`, `players`, `maxTime`, `minWeight`, `maxWeight`, `sort`, `limit`, `offset` |
| GET | `/games/{slug}` | – | detail, viewer's rating, rating histogram |
| PUT DELETE | `/games/{slug}/rating` | ✅ | returns the game with recomputed aggregates |
| PUT DELETE | `/shelf/{slug}` | ✅ | `owned` / `wishlist` / `played` |
| GET | `/users/{username}` | – | profile, shelf, recent ratings, posts |
| GET | `/users/{username}/posts/{slug}` | – | drafts visible only to their author |
| POST PATCH DELETE | `/posts[/{id}]` | ✅ | |
| GET PATCH | `/me` | ✅ | provisions the local user row on first call |
| GET POST | `/cron/{refresh-stats,import}` | `CRON_SECRET` | |

A token that fails to verify does not fail the request — it means "anonymous",
so an expired session never takes down public reads.
