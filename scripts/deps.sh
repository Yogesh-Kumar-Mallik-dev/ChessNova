#!/usr/bin/env bash
# ==============================================================================
# scripts/deps.sh - Install repository dependencies
# ==============================================================================
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "==> [deps] Checking toolchains..."
command -v go >/dev/null 2>&1 || { echo "❌ Go is required but not installed."; exit 1; }
command -v npm >/dev/null 2>&1 || { echo "❌ npm is required but not installed."; exit 1; }

echo "==> [deps] Installing Go backend dependencies..."
cd "$ROOT_DIR/apps/server"
go mod download
go mod tidy

echo "==> [deps] Installing web frontend dependencies..."
cd "$ROOT_DIR/apps/web"
npm install

echo "✅ [deps] All dependencies installed successfully."
