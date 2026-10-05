#!/usr/bin/env bash
# ==============================================================================
# scripts/build.sh - Compile production assets and binaries
# ==============================================================================
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "==> [build] Compiling Go server binary..."
cd "$ROOT_DIR/apps/server"
mkdir -p bin
go build -o bin/server ./cmd/server/main.go
echo "✅ [build] Go server binary built: apps/server/bin/server"

echo "==> [build] Building SvelteKit production bundle..."
cd "$ROOT_DIR/apps/web"
npm run build
echo "✅ [build] SvelteKit frontend bundle built."

echo "🎉 [build] All production artifacts built successfully."
