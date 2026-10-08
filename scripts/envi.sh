#!/usr/bin/env bash
# ==============================================================================
# scripts/envi.sh - 1-Click Environment Setup & Docker Infrastructure
# ==============================================================================
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "==> [envi] Initializing environment configuration..."
if [ ! -f "$ROOT_DIR/.env" ]; then
  if [ -f "$ROOT_DIR/.env.example" ]; then
    cp "$ROOT_DIR/.env.example" "$ROOT_DIR/.env"
    echo "Created .env from .env.example"
  fi
fi

if command -v docker >/dev/null 2>&1; then
  echo "==> [envi] Starting MongoDB and Redis via Docker Compose..."
  cd "$ROOT_DIR"

  # Clean up any conflicting orphaned containers with fixed names from other compose projects
  for CNAME in chess-mongo chess-redis; do
    if docker ps -a --format '{{.Names}}' | grep -Eq "^${CNAME}$"; then
      PROJ=$(docker inspect "$CNAME" --format '{{index .Config.Labels "com.docker.compose.project"}}' 2>/dev/null || true)
      if [ "$PROJ" != "chessnova" ]; then
        echo "==> [envi] Cleaning up conflicting container $CNAME (project: '${PROJ:-none}')..."
        docker rm -f "$CNAME" >/dev/null 2>&1 || true
      fi
    fi
  done

  docker compose up -d
  echo "✅ [envi] Infrastructure containers started."
else
  echo "⚠️ Docker not detected. Please ensure MongoDB (27017) and Redis (6379) are running."
fi
