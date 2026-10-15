#!/usr/bin/env bash
set -euo pipefail

# ChessNova Zero-Downtime Rolling Release Script
# Target: 3 Load-Balanced Instances (8081, 8082, 8083) behind Nginx

VPS_HOST="${VPS_HOST:-162.35.173.209}"
VPS_USER="${VPS_USER:-root}"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/id_ed25519}"
REMOTE_DIR="/opt/chessnova"
INSTANCES=(8081 8082 8083)

SSH_CMD="ssh -i $SSH_KEY -o BatchMode=yes -o StrictHostKeyChecking=accept-new $VPS_USER@$VPS_HOST"
SCP_CMD="scp -i $SSH_KEY -o BatchMode=yes -o StrictHostKeyChecking=accept-new"

echo "=========================================================="
echo "  🚀 Starting ChessNova Zero-Downtime Rolling Deployment"
echo "  Host: $VPS_USER@$VPS_HOST"
echo "  Instances: ${INSTANCES[*]}"
echo "=========================================================="

# 1. Compile backend for Linux AMD64
echo "==> [1/5] Compiling Go backend for linux/amd64..."
cd "$(dirname "$0")/../apps/server"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /tmp/chessnova-server cmd/server/main.go
echo "    Backend binary compiled successfully ($(du -h /tmp/chessnova-server | cut -f1))"

# 2. Upload new binary to VPS staging path
echo "==> [2/5] Uploading new binary to VPS staging area..."
rsync -avzP -e "ssh -i $SSH_KEY -o BatchMode=yes -o StrictHostKeyChecking=accept-new" /tmp/chessnova-server "$VPS_USER@$VPS_HOST:$REMOTE_DIR/bin/server.new"
$SSH_CMD "chmod +x $REMOTE_DIR/bin/server.new && mv -f $REMOTE_DIR/bin/server.new $REMOTE_DIR/bin/server"
rm -f /tmp/chessnova-server
echo "    Binary atomically updated on VPS."

# 3. Ensure systemd template service exists
echo "==> [3/5] Verifying systemd multi-instance service template..."
$SSH_CMD "cat <<'EOF' > /etc/systemd/system/chessnova@.service
[Unit]
Description=ChessNova Real-Time Chess Server (Port %i)
After=network.target mongodb.service redis.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/chessnova
EnvironmentFile=/opt/chessnova/.env
Environment=\"SERVER_PORT=%i\"
Environment=\"INSTANCE_ID=instance-%i\"
ExecStart=/opt/chessnova/bin/server
Restart=always
RestartSec=3s
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF
sed -i '/^SERVER_PORT=/d' /opt/chessnova/.env
systemctl daemon-reload
"

# 4. Perform Rolling Update across all 3 instances
echo "==> [4/5] Executing rolling restart across instances..."
for PORT in "${INSTANCES[@]}"; do
    echo "    --------------------------------------------------"
    echo "    🔄 Restarting instance on port $PORT..."
    $SSH_CMD "systemctl enable --now chessnova@$PORT && systemctl restart chessnova@$PORT"
    
    echo "    ⏳ Awaiting health check on http://127.0.0.1:$PORT/health..."
    HEALTHY=false
    for i in {1..30}; do
        HEALTH_RES=$($SSH_CMD "curl -s -f http://127.0.0.1:$PORT/health || true")
        if [[ "$HEALTH_RES" =~ "\"status\":\"ok\"" ]]; then
            echo "    ✅ Instance on port $PORT is HEALTHY (Response: $HEALTH_RES)"
            HEALTHY=true
            break
        fi
        sleep 1
    done

    if [ "$HEALTHY" = false ]; then
        echo "    ❌ ERROR: Instance on port $PORT failed health check after 30 seconds!"
        echo "    Aborting rolling release to prevent total downtime."
        exit 1
    fi
done

# 5. Reload Nginx to ensure upstream pools are fresh and disable legacy single-instance service if active
echo "==> [5/6] Reloading Nginx load balancer..."
$SSH_CMD "nginx -t && nginx -s reload"
$SSH_CMD "if systemctl is-active --quiet chessnova.service; then systemctl disable --now chessnova.service; fi"

echo "==> [6/6] Checking metrics and cluster health..."
CLUSTER_STATUS=$($SSH_CMD "curl -s http://127.0.0.1:8081/health && echo '' && curl -s http://127.0.0.1:8082/health && echo '' && curl -s http://127.0.0.1:8083/health")
echo "$CLUSTER_STATUS"

echo "=========================================================="
echo "  🎉 Zero-downtime rolling update completed successfully!"
echo "  All 3 instances running and verified."
echo "=========================================================="
