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
    echo "==> [desktop] Launching ChessPasant Tauri desktop in development mode..."
    npm run dev
    ;;
  build)
    echo "==> [desktop] Building ChessPasant Tauri desktop production binary..."
    npm run build
    ;;
  check)
    echo "==> [desktop] Running cargo check on Tauri desktop crate..."
    cargo check --manifest-path src-tauri/Cargo.toml
    ;;
  arch|package-arch)
    echo "==> [desktop] Packaging ChessPasant for Arch Linux (.pkg.tar.zst & .tar.gz)..."
    "$ROOT_DIR/packaging/arch/package-arch.sh" "0.1.1" "$ROOT_DIR/dist/arch"
    ;;
  deploy-arch|deploy-repo)
    echo "==> [desktop] Deploying Arch Linux repository to production VPS..."
    shift || true
    "$ROOT_DIR/packaging/arch/deploy-repo.sh" "$@"
    ;;
  *)
    echo "Usage: ./scripts/desktop.sh [dev|build|check|arch|deploy-arch]"
    exit 1
    ;;
esac
