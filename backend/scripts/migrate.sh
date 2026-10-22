#!/bin/bash
# DesignDocument/8_DBDesign.md 8.9 / 12_Operation.md 12.4
# Applies backend/migrations/*.sql in order with psql. Every file is idempotent.
# Usage: DATABASE_URL=postgres://... ./scripts/migrate.sh

set -euo pipefail

if [ -z "${DATABASE_URL:-}" ]; then
  echo "DATABASE_URL is required" >&2
  exit 1
fi

DIR="$(cd "$(dirname "$0")/../migrations" && pwd)"

for f in "$DIR"/*.sql; do
  echo "applying $(basename "$f")"
  PGOPTIONS="-c client_min_messages=warning" psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -q -f "$f"
done

echo "migrations applied"
