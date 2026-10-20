#!/usr/bin/env bash
# ==============================================================================
# packaging/arch/deploy-repo.sh - Deploys Arch repository to production VPS
# Usage: ./packaging/arch/deploy-repo.sh [TARGET_HOST] [TARGET_USER]
# ==============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

TARGET_HOST="${1:-162.35.173.209}"
TARGET_USER="${2:-root}"
SSH_KEY="${HOME}/.ssh/id_ed25519"
REPO_DIR="$ROOT_DIR/dist/arch/repo"

echo "=================================================================="
echo "  🚀 Deploying ChessPasant Arch Linux Repository to Production"
echo "  Target: ${TARGET_USER}@${TARGET_HOST}:/var/www/chessnova/html/repo/"
echo "=================================================================="

# 1. Ensure repository has been built
if [ ! -d "$REPO_DIR/x86_64" ] || [ ! -f "$REPO_DIR/x86_64/chesspasant.db.tar.zst" ]; then
    echo "==> Repository not found locally. Running packaging script first..."
    "$SCRIPT_DIR/package-arch.sh" "0.1.0" "$ROOT_DIR/dist/arch"
fi

# 2. Test SSH connection
echo "==> Testing SSH connection to ${TARGET_HOST}..."
if ! ssh -i "$SSH_KEY" -o BatchMode=yes -o ConnectTimeout=5 "${TARGET_USER}@${TARGET_HOST}" "true" 2>/dev/null; then
    # Fallback to standard agent if id_ed25519 not specified
    if ! ssh -o BatchMode=yes -o ConnectTimeout=5 "${TARGET_USER}@${TARGET_HOST}" "true"; then
        echo "ERROR: Unable to connect to ${TARGET_USER}@${TARGET_HOST} via SSH."
        exit 1
    fi
fi

# 3. Create remote repo directory
echo "==> Preparing remote directory structure..."
ssh -o BatchMode=yes "${TARGET_USER}@${TARGET_HOST}" "mkdir -p /var/www/chessnova/html/repo/x86_64"

# 4. Rsync repository files (preserving symlinks and permissions)
echo "==> Uploading repository packages, database, and setup script..."
rsync -avP --partial --copy-links \
    -e "ssh -o BatchMode=yes -o ServerAliveInterval=15 -o ServerAliveCountMax=6" \
    "$REPO_DIR/" \
    "${TARGET_USER}@${TARGET_HOST}:/var/www/chessnova/html/repo/"

# 5. Fix permissions on remote host
echo "==> Setting remote web server permissions..."
ssh -o BatchMode=yes "${TARGET_USER}@${TARGET_HOST}" "
    chown -R www-data:www-data /var/www/chessnova/html/repo
    chmod -R 755 /var/www/chessnova/html/repo
"

# 6. Verify live availability over HTTPS
echo "==> Verifying repository availability over HTTPS..."
STATUS=$(curl -s -o /dev/null -w "%{http_code}" https://chess.yogeshmallik.in/repo/x86_64/chesspasant.db.tar.zst || true)
if [ "$STATUS" = "200" ]; then
    echo "✅ [https://chess.yogeshmallik.in/repo/x86_64/chesspasant.db.tar.zst] is LIVE (HTTP 200)!"
else
    echo "⚠️  HTTP response code: $STATUS (may take a moment to propagate or reload nginx)"
fi

echo ""
echo "=================================================================="
echo "🎉 Arch Linux Repository Deployment Completed!"
echo "   Repository URL: https://chess.yogeshmallik.in/repo/\$arch"
echo "   Automated Install: curl -sSL https://chess.yogeshmallik.in/repo/setup-repo.sh | sudo bash"
echo "   Manual Config: Add [chesspasant] section to /etc/pacman.conf"
echo "=================================================================="
