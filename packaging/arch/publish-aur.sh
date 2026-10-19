#!/usr/bin/env bash
# ==============================================================================
# packaging/arch/publish-aur.sh - Automates publishing chessnova-bin to the AUR
# Usage: ./packaging/arch/publish-aur.sh
# ==============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
PKG_NAME="chessnova-bin"
AUR_REPO="ssh://aur@aur.archlinux.org/${PKG_NAME}.git"

echo "=================================================================="
echo "  🚀 Arch User Repository (AUR) Publisher: ${PKG_NAME}"
echo "=================================================================="

# 1. Regenerate .SRCINFO to ensure it matches PKGBUILD perfectly
echo "==> Regenerating .SRCINFO..."
cd "$SCRIPT_DIR"
makepkg --printsrcinfo > .SRCINFO

TMP_DIR=$(mktemp -d -t aur-chessnova-XXXXXX)
trap 'rm -rf "$TMP_DIR"' EXIT

echo "==> Cloning AUR repository: ${AUR_REPO}..."
if ! git clone "$AUR_REPO" "$TMP_DIR/${PKG_NAME}"; then
  echo ""
  echo "⚠️  Cloning failed. Please ensure:"
  echo "  1. You have registered an account at https://aur.archlinux.org"
  echo "  2. You uploaded your public SSH key to your AUR profile"
  echo "  3. SSH config is configured for aur.archlinux.org"
  exit 1
fi

echo "==> Copying PKGBUILD and .SRCINFO to AUR repository..."
cp "$SCRIPT_DIR/PKGBUILD" "$TMP_DIR/${PKG_NAME}/"
cp "$SCRIPT_DIR/.SRCINFO" "$TMP_DIR/${PKG_NAME}/"

cd "$TMP_DIR/${PKG_NAME}"

if git diff --quiet PKGBUILD .SRCINFO 2>/dev/null; then
  echo "ℹ️  No changes detected in PKGBUILD or .SRCINFO. AUR is already up-to-date."
  exit 0
fi

git add PKGBUILD .SRCINFO
VERSION=$(grep -m1 '^pkgver' PKGBUILD | cut -d'=' -f2 | tr -d ' ')
git commit -m "Update ${PKG_NAME} to v${VERSION}"

echo "==> Pushing to AUR master branch..."
git push origin master

echo "✅ Successfully published ${PKG_NAME} v${VERSION} to AUR!"
echo "   Inspect package: https://aur.archlinux.org/packages/${PKG_NAME}"
echo "   Install command: yay -S ${PKG_NAME}"
