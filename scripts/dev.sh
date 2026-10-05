#!/usr/bin/env bash
# ==============================================================================
# scripts/dev.sh - Start local development servers
# ==============================================================================
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cleanup() {
  echo ""
  echo "==> [dev] Shutting down services..."
  kill $(jobs -p) 2>/dev/null || true
  exit 0
}

trap cleanup SIGINT SIGTERM EXIT

# 1. Ensure environment & docker containers are up
if [ -f "$ROOT_DIR/scripts/envi.sh" ]; then
  "$ROOT_DIR/scripts/envi.sh"
fi

# 2. Deterministic Port Cleanup (Ports 8080 and 5173)
for PORT in 8080 5173; do
  PID=$(lsof -ti :$PORT 2>/dev/null || true)
  if [ -n "$PID" ]; then
    echo "==> [dev] Port $PORT in use by PID $PID. Terminating..."
    kill -9 $PID 2>/dev/null || true
  fi
done

echo "==> [dev] Starting Go backend on http://localhost:8080..."
cd "$ROOT_DIR/apps/server"
go run cmd/server/main.go &
BACKEND_PID=$!

echo "==> [dev] Starting SvelteKit frontend on http://localhost:5173..."
cd "$ROOT_DIR/apps/web"
npm run dev &
FRONTEND_PID=$!

echo "=================================================================="
echo "  ♟ ChessNova Platform is running in dev mode!"
echo "  Frontend: http://localhost:5173"
echo "  Backend:  http://localhost:8080"
echo "  Press Ctrl+C to stop all services."
echo "=================================================================="

wait
