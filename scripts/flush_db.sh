#!/usr/bin/env bash
# ==============================================================================
# scripts/flush_db.sh - Flush databases and caches for clean-slate testing
# ==============================================================================
set -euo pipefail

echo "==> [flush] Flushing Redis cache..."
if command -v redis-cli >/dev/null 2>&1; then
  redis-cli flushall || true
elif command -v docker >/dev/null 2>&1; then
  docker exec chess-redis redis-cli flushall 2>/dev/null || true
fi

echo "==> [flush] Resetting MongoDB databases..."
DB_NAME="${MONGODB_DATABASE:-chess}"
if command -v mongosh >/dev/null 2>&1; then
  mongosh --eval "use ${DB_NAME}; db.dropDatabase(); use chess_test; db.dropDatabase();" || true
elif command -v docker >/dev/null 2>&1; then
  docker exec chess-mongo mongosh --eval "use ${DB_NAME}; db.dropDatabase(); use chess_test; db.dropDatabase();" 2>/dev/null || true
fi

echo "✅ [flush] Database and caches flushed successfully."
