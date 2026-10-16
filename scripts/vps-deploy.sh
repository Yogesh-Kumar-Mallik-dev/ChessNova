#!/usr/bin/env bash
# ==============================================================================
# ChessNova Production VPS Full Provisioning & Deployment Script
# ==============================================================================
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TARGET_HOST="${1:-162.35.173.209}"
TARGET_USER="${2:-root}"
SSH_KEY="${HOME}/.ssh/id_ed25519"
SSH_CMD="ssh -i ${SSH_KEY} -o StrictHostKeyChecking=accept-new ${TARGET_USER}@${TARGET_HOST}"

echo "=========================================================="
echo "  🚀 Starting ChessNova Full Production Deployment"
echo "  Target: ${TARGET_USER}@${TARGET_HOST}"
echo "=========================================================="

# 1. Run base provisioning and hardening script on VPS
echo "==> [1/6] Running VPS provisioning & hardening on remote host..."
$SSH_CMD "bash -s" < "${PROJECT_ROOT}/scripts/vps-setup.sh"

# 2. Deploy .env file
echo "==> [2/6] Writing production environment configuration..."
$SSH_CMD 'cat << "EOF" > /opt/chessnova/.env
MONGODB_URI=mongodb://127.0.0.1:27017
MONGODB_DATABASE=chess
REDIS_URL=127.0.0.1:6379
CORS_ORIGIN=https://chess.yogeshmallik.in
ALLOWED_ORIGINS=https://chess.yogeshmallik.in,http://chess.yogeshmallik.in,http://162.35.173.209,http://localhost,http://127.0.0.1,tauri://localhost,http://tauri.localhost
JWT_ACCESS_SECRET=chessnova-production-jwt-access-secret-32-chars-long-987123
JWT_REFRESH_SECRET=chessnova-production-jwt-refresh-secret-32-chars-long-456789
LOCAL_UPLOAD_DIR=/opt/chessnova/uploads/avatars
RESEND_API_KEY=
SMTP_HOST=
SMTP_PORT=587
SMTP_USER=
SMTP_PASS=
SMTP_FROM=ChessNova <onboarding@resend.dev>
EOF
chown -R yogesh:yogesh /opt/chessnova/.env
chmod 600 /opt/chessnova/.env
'

# 3. Compile & Deploy Backend Go Binary
echo "==> [3/6] Compiling backend binary for linux/amd64..."
cd "${PROJECT_ROOT}/apps/server"
mkdir -p bin
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/server cmd/server/main.go
echo "    Binary compiled successfully ($(du -h bin/server | cut -f1))"

echo "    Uploading backend binary to VPS..."
rsync -avzP -e "ssh -i ${SSH_KEY} -o StrictHostKeyChecking=accept-new" bin/server "${TARGET_USER}@${TARGET_HOST}:/opt/chessnova/bin/server"
$SSH_CMD "chmod +x /opt/chessnova/bin/server && chown -R yogesh:yogesh /opt/chessnova/bin"

# 4. Build & Deploy Frontend Static Bundle
echo "==> [4/6] Building frontend static bundle..."
cd "${PROJECT_ROOT}/apps/web"
npm run build

echo "    Uploading frontend static assets..."
rsync -avzP --delete -e "ssh -i ${SSH_KEY} -o StrictHostKeyChecking=accept-new" build/ "${TARGET_USER}@${TARGET_HOST}:/var/www/chessnova/html/"
$SSH_CMD "chown -R www-data:www-data /var/www/chessnova/html"

# 5. Start/Restart Multi-Instance Cluster (8081, 8082, 8083)
echo "==> [5/6] Starting multi-instance backend cluster..."
$SSH_CMD "systemctl daemon-reload && systemctl enable --now chessnova@8081 chessnova@8082 chessnova@8083 && systemctl restart chessnova@8081 chessnova@8082 chessnova@8083"

# 6. Configure Let's Encrypt SSL
echo "==> [6/6] Ensuring Let's Encrypt SSL certificate..."
$SSH_CMD "certbot --nginx -d chess.yogeshmallik.in --non-interactive --agree-tos -m yogesh@yogeshmallik.in --redirect || true"
$SSH_CMD "systemctl reload nginx"

# Verification
echo "=========================================================="
echo "  🔍 Verifying cluster health..."
sleep 2
for PORT in 8081 8082 8083; do
    echo -n "  Instance ${PORT}: "
    $SSH_CMD "curl -s http://127.0.0.1:${PORT}/health || echo 'FAILED'"
    echo ""
done

echo "  Public Health Endpoint:"
curl -s https://chess.yogeshmallik.in/health || curl -s http://162.35.173.209/health || true
echo ""

echo "=========================================================="
echo "  🎉 ChessNova Production Deployment Completed!"
echo "  URL: https://chess.yogeshmallik.in"
echo "=========================================================="
