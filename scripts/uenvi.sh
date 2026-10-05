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
  echo "✅ [uenvi] Containers stopped."
fi
