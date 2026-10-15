#!/usr/bin/env bash
# ==============================================================================
# ChessNova App Deployment Script (Binaries, Web Bundle, SSL)
# ==============================================================================
set -euo pipefail

TARGET_HOST="${1:-162.35.173.209}"
TARGET_USER="${2:-root}"

echo "Deploying ChessNova to ${TARGET_USER}@${TARGET_HOST}..."

# 1. Run provisioning script on VPS
ssh -i ~/.ssh/id_ed25519 -o StrictHostKeyChecking=accept-new "${TARGET_USER}@${TARGET_HOST}" "bash -s" < scripts/vps-setup.sh

# 2. Deploy .env file
ssh -i ~/.ssh/id_ed25519 "${TARGET_USER}@${TARGET_HOST}" 'cat << "EOF" > /opt/chessnova/.env
MONGODB_URI=mongodb://127.0.0.1:27017
MONGODB_DATABASE=chess
REDIS_URL=127.0.0.1:6379
SERVER_PORT=8080
CORS_ORIGIN=https://chess.yogeshmallik.in
ALLOWED_ORIGINS=https://chess.yogeshmallik.in,http://chess.yogeshmallik.in,http://162.35.173.209,http://localhost,http://127.0.0.1,tauri://localhost,http://tauri.localhost
JWT_ACCESS_SECRET=chessnova-production-jwt-access-secret-32-chars-long-987123
JWT_REFRESH_SECRET=chessnova-production-jwt-refresh-secret-32-chars-long-456789
LOCAL_UPLOAD_DIR=/opt/chessnova/uploads/avatars
SMTP_HOST=127.0.0.1
SMTP_PORT=25
SMTP_USER=
SMTP_PASS=
SMTP_FROM=ChessNova <noreply@chess.yogeshmallik.in>
EOF
chown yogesh:yogesh /opt/chessnova/.env
chmod 600 /opt/chessnova/.env
'

# 3. Deploy Go Server Binary and Stockfish
echo "Uploading backend binaries..."
rsync -avzP apps/server/bin/server "${TARGET_USER}@${TARGET_HOST}:/opt/chessnova/bin/server"
if [ -f apps/server/bin/stockfish ]; then
    rsync -avzP apps/server/bin/stockfish "${TARGET_USER}@${TARGET_HOST}:/opt/chessnova/bin/stockfish"
fi

ssh -i ~/.ssh/id_ed25519 "${TARGET_USER}@${TARGET_HOST}" "chown -R yogesh:yogesh /opt/chessnova/bin && chmod +x /opt/chessnova/bin/*"

# 4. Deploy Frontend Static Web Build
echo "Uploading frontend static build..."
rsync -avzP --delete apps/web/build/ "${TARGET_USER}@${TARGET_HOST}:/var/www/chessnova/html/"
ssh -i ~/.ssh/id_ed25519 "${TARGET_USER}@${TARGET_HOST}" "chown -R www-data:www-data /var/www/chessnova/html"

# 5. Restart Backend Service
echo "Restarting ChessNova service..."
ssh -i ~/.ssh/id_ed25519 "${TARGET_USER}@${TARGET_HOST}" "systemctl restart chessnova && systemctl status chessnova --no-pager"

# 6. Obtain Let's Encrypt SSL certificate for chess.yogeshmallik.in
echo "Configuring SSL with Certbot..."
ssh -i ~/.ssh/id_ed25519 "${TARGET_USER}@${TARGET_HOST}" "certbot --nginx -d chess.yogeshmallik.in --non-interactive --agree-tos -m yogesh@yogeshmallik.in --redirect || true"

echo "=========================================="
echo " ChessNova Deployment Complete!"
echo " Visit: https://chess.yogeshmallik.in"
echo "=========================================="
