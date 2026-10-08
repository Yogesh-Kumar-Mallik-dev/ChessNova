#!/usr/bin/env bash
# ==============================================================================
# scripts/uenvi.sh - 1-Click Environment Teardown
# ==============================================================================
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if command -v docker >/dev/null 2>&1; then
  echo "==> [uenvi] Stopping and removing containers..."
  cd "$ROOT_DIR"
  docker compose down
  for CNAME in chess-mongo chess-redis; do
    if docker ps -a --format '{{.Names}}' | grep -Eq "^${CNAME}$"; then
      docker rm -f "$CNAME" >/dev/null 2>&1 || true
    fi
  done
  echo "✅ [uenvi] Containers stopped."
fi
