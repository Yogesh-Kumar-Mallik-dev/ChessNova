#!/usr/bin/env bash
# ==============================================================================
# scripts/desktop.sh - Launch or build Tauri desktop application
# ==============================================================================
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ACTION="${1:-dev}"

cd "$ROOT_DIR/apps/web"

case "$ACTION" in
  dev)
    echo "==> [desktop] Launching ChessNova Tauri desktop in development mode..."
    npm run desktop:dev
    ;;
  build)
    echo "==> [desktop] Building ChessNova Tauri desktop production binary..."
    npm run desktop:build
    ;;
  *)
    echo "Usage: ./scripts/desktop.sh [dev|build]"
    exit 1
    ;;
esac
