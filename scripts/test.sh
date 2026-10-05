#!/usr/bin/env bash
# ==============================================================================
# scripts/test.sh - Run test suites
# ==============================================================================
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "==> [test] Running Pure Go Chess Engine unit tests..."
cd "$ROOT_DIR/apps/server"
go test -v ./internal/chess/...

echo "==> [test] Running end-to-end integration tests..."
go test -v ./internal/...

echo "✅ [test] All test suites passed."
