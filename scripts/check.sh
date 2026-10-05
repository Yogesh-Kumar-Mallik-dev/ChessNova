#!/usr/bin/env bash
# ==============================================================================
# scripts/check.sh - Run static analyzers and typecheckers
# ==============================================================================
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "==> [check] Running Go static analysis (go vet)..."
cd "$ROOT_DIR/apps/server"
go vet ./...

echo "==> [check] Running SvelteKit typecheck (svelte-check)..."
cd "$ROOT_DIR/apps/web"
npm run check

echo "✅ [check] All static and type checks passed."
