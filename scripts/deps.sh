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

echo "==> [deps] Ensuring Stockfish binary exists in apps/server/bin/stockfish..."
STOCKFISH_BIN="$ROOT_DIR/apps/server/bin/stockfish"
if [ ! -f "$STOCKFISH_BIN" ] && ! command -v stockfish >/dev/null 2>&1; then
  mkdir -p "$ROOT_DIR/apps/server/bin"
  ARCH="$(uname -m)"
  OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
  if [ "$OS" = "linux" ] && [ "$ARCH" = "x86_64" ]; then
    echo "==> [deps] Downloading Stockfish 19 for Linux x86_64..."
    TMP_SF="$(mktemp -d)"
    curl -sL -o "$TMP_SF/sf.tar.gz" "https://github.com/official-stockfish/Stockfish/releases/download/sf_19/stockfish-linux-x86-64-universal.tar.gz"
    tar -xzf "$TMP_SF/sf.tar.gz" -C "$TMP_SF"
    cp "$TMP_SF/stockfish/stockfish-linux-x86-64-universal" "$STOCKFISH_BIN"
    chmod +x "$STOCKFISH_BIN"
    rm -rf "$TMP_SF"
    echo "✅ [deps] Stockfish binary downloaded successfully."
  else
    echo "⚠️ [deps] Prebuilt Stockfish download not available for $OS/$ARCH. Fallback engine will be used or install 'stockfish' via your package manager."
  fi
fi

echo "==> [deps] Installing web frontend dependencies..."
cd "$ROOT_DIR/apps/web"
npm install


echo "✅ [deps] All dependencies installed successfully."

