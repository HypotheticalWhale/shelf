#!/usr/bin/env bash
#
# Backfill covers, weights and tags from the BoardGameGeek API.
#
# The refresh walks the catalogue least-recently-imported first, so re-running
# it resumes where the last run stopped rather than starting over. A full pass
# over ~31k games takes around three hours: BGG asks for no more than one
# request every couple of seconds for bulk work, and that pacing is deliberate
# — going faster risks the token, which is the one thing here that cannot be
# replaced. A single failed batch ends a run, so this loops until a pass gets
# all the way through.
#
# Usage: scripts/backfill-artwork.sh [count]   (default: the whole catalogue)
set -uo pipefail

cd "$(dirname "$0")/../apps/api" || exit 1

if [ ! -f .env.local ]; then
  echo "apps/api/.env.local not found — it must supply DATABASE_URL and BGG_API_TOKEN" >&2
  exit 1
fi
set -a; . ./.env.local; set +a

if [ -z "${BGG_API_TOKEN:-}" ]; then
  echo "BGG_API_TOKEN is not set; the API refuses unauthenticated requests" >&2
  exit 1
fi

count="${1:-31500}"
max_passes=40
pass=0

while [ "$pass" -lt "$max_passes" ]; do
  pass=$((pass + 1))
  echo "=== pass $pass started $(date '+%Y-%m-%d %H:%M:%S') ==="

  if go run ./cmd/import -refresh "$count"; then
    echo "=== pass $pass finished cleanly ==="
    go run ./cmd/stats | head -4
    exit 0
  fi

  echo "=== pass $pass ended early; resuming in 30s ==="
  sleep 30
done

echo "gave up after $max_passes passes — something is failing repeatedly" >&2
go run ./cmd/stats | head -4
exit 1
