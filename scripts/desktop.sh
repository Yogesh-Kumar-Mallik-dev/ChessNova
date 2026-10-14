#!/usr/bin/env bash
# ==============================================================================
# scripts/desktop.sh - Launch or build Tauri desktop application
# ==============================================================================
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ACTION="${1:-dev}"

cd "$ROOT_DIR/apps/desktop"

# Ensure dependencies installed
if [ ! -d "node_modules" ]; then
  echo "==> [desktop] Installing desktop dependencies..."
  npm install
fi

case "$ACTION" in
  dev)
    echo "==> [desktop] Launching ChessNova Tauri desktop in development mode..."
    npm run dev
    ;;
  build)
    echo "==> [desktop] Building ChessNova Tauri desktop production binary..."
    npm run build
    ;;
  check)
    echo "==> [desktop] Running cargo check on Tauri desktop crate..."
    cargo check --manifest-path src-tauri/Cargo.toml
    ;;
  *)
    echo "Usage: ./scripts/desktop.sh [dev|build|check]"
    exit 1
    ;;
esac
