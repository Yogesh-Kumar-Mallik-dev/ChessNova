#!/usr/bin/env bash
# ==============================================================================
# scripts/mobile.sh - Launch, initialize, or build Tauri mobile application (Android & iOS)
# ==============================================================================
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ACTION="${1:-check}"
shift || true

cd "$ROOT_DIR/apps/mobile"

# Ensure dependencies installed
if [ ! -d "node_modules" ]; then
  echo "==> [mobile] Installing mobile dependencies..."
  npm install
fi

# Ensure Android SDK environment variables are set if present
if [ -d "$HOME/Android/Sdk" ] && [ -z "${ANDROID_HOME:-}" ]; then
  export ANDROID_HOME="$HOME/Android/Sdk"
  export ANDROID_SDK_ROOT="$HOME/Android/Sdk"
  export PATH="$PATH:$ANDROID_HOME/platform-tools:$ANDROID_HOME/cmdline-tools/latest/bin"
fi

case "$ACTION" in
  check)
    echo "==> [mobile] Running cargo check on mobile Rust crate..."
    cargo check --manifest-path src-tauri/Cargo.toml
    ;;
  android-init|init-android)
    echo "==> [mobile] Initializing Android target in ChessPasant mobile..."
    npx tauri android init "$@"
    ;;
  android-dev|dev-android)
    echo "==> [mobile] Launching ChessPasant on connected Android device/emulator..."
    npx tauri android dev "$@"
    ;;
  android-build|build-android|apk)
    echo "==> [mobile] Building release Android APK / AAB bundle..."
    npx tauri android build "$@"
    ;;
  ios-init|init-ios)
    echo "==> [mobile] Initializing iOS target (requires macOS & Xcode)..."
    npx tauri ios init "$@"
    ;;
  ios-dev|dev-ios)
    echo "==> [mobile] Launching ChessPasant on iOS Simulator (requires macOS)..."
    npx tauri ios dev "$@"
    ;;
  ios-build|build-ios)
    echo "==> [mobile] Building release iOS IPA (requires macOS)..."
    npx tauri ios build "$@"
    ;;
  *)
    echo "Usage: ./scripts/mobile.sh [check|android-init|android-dev|android-build|ios-init|ios-dev|ios-build]"
    exit 1
    ;;
esac
