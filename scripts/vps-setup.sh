#!/usr/bin/env bash
# ==============================================================================
# ChessNova Production VPS Provisioning & Hardening Script
# ==============================================================================
set -euo pipefail

echo "=========================================="
echo " Starting ChessNova Production VPS Setup"
echo "=========================================="

export DEBIAN_FRONTEND=noninteractive

# 1. Base updates & essential packages
echo "==> [1/9] Updating system packages..."
apt-get update -y
apt-get install -y --no-install-recommends \
    curl \
    wget \
    ca-certificates \
    gnupg \
    lsb-release \
    nginx \
    certbot \
    python3-certbot-nginx \
    ufw \
    rsync \
    tar \
    unzip \
    cron \
    stockfish

# 2. Permanent SSH Access & Lockout Immunity
echo "==> [2/9] Hardening SSH and configuring permanent access..."
mkdir -p /etc/ssh/sshd_config.d/
cat << 'EOF' > /etc/ssh/sshd_config.d/99-chessnova.conf
PubkeyAuthentication yes
PasswordAuthentication yes
PermitRootLogin yes
KbdInteractiveAuthentication yes
ClientAliveInterval 60
ClientAliveCountMax 10
EOF

# Ensure faillock does not lock out legitimate admin users
if [ -f /etc/security/faillock.conf ]; then
    sed -i 's/^#*deny =.*/deny = 10/' /etc/security/faillock.conf || true
    sed -i 's/^#*unlock_time =.*/unlock_time = 300/' /etc/security/faillock.conf || true
fi

# Ensure user yogesh exists with passwordless sudo
id yogesh >/dev/null 2>&1 || useradd -m -s /bin/bash yogesh
usermod -aG sudo yogesh 2>/dev/null || true
echo "yogesh ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/yogesh
chmod 440 /etc/sudoers.d/yogesh

# Permanent SSH keys
KEY_USER="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICJ6OvB0ABT9gC5n9QLZ6FnXVI1FpdDngwifScFLTo0s yogesh@vps"
KEY_DEPLOY="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFiZWXZXdsauuG4nMv2qYJRuK54BXrY55jP/EXcwJCNh chessnova-github-actions"

mkdir -p /root/.ssh && chmod 700 /root/.ssh
echo "$KEY_USER" >> /root/.ssh/authorized_keys
echo "$KEY_DEPLOY" >> /root/.ssh/authorized_keys
sort -u /root/.ssh/authorized_keys -o /root/.ssh/authorized_keys
chmod 600 /root/.ssh/authorized_keys

mkdir -p /home/yogesh/.ssh && chmod 700 /home/yogesh/.ssh
cp /root/.ssh/authorized_keys /home/yogesh/.ssh/authorized_keys
chown -R yogesh:yogesh /home/yogesh/.ssh
chmod 600 /home/yogesh/.ssh/authorized_keys

# Restart SSH service
systemctl restart ssh || systemctl restart sshd || true

# Hourly cron to guarantee SSH keys are never lost or corrupted
cat << 'EOF' > /etc/cron.hourly/ensure-ssh-keys
#!/bin/sh
KEY_USER="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICJ6OvB0ABT9gC5n9QLZ6FnXVI1FpdDngwifScFLTo0s yogesh@vps"
KEY_DEPLOY="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFiZWXZXdsauuG4nMv2qYJRuK54BXrY55jP/EXcwJCNh chessnova-github-actions"
mkdir -p /root/.ssh && chmod 700 /root/.ssh
grep -q "$KEY_USER" /root/.ssh/authorized_keys 2>/dev/null || echo "$KEY_USER" >> /root/.ssh/authorized_keys
grep -q "$KEY_DEPLOY" /root/.ssh/authorized_keys 2>/dev/null || echo "$KEY_DEPLOY" >> /root/.ssh/authorized_keys
chmod 600 /root/.ssh/authorized_keys
mkdir -p /home/yogesh/.ssh && chmod 700 /home/yogesh/.ssh
cp /root/.ssh/authorized_keys /home/yogesh/.ssh/authorized_keys
chown -R yogesh:yogesh /home/yogesh/.ssh
chmod 600 /home/yogesh/.ssh/authorized_keys
EOF
chmod +x /etc/cron.hourly/ensure-ssh-keys

# 3. Install Docker & Docker Compose
echo "==> [3/9] Installing Docker and Docker Compose..."
if ! command -v docker >/dev/null 2>&1; then
    curl -fsSL https://get.docker.com -o /tmp/get-docker.sh
    sh /tmp/get-docker.sh
    rm -f /tmp/get-docker.sh
fi
usermod -aG docker yogesh || true

# 4. Create application directories
echo "==> [4/9] Creating directories..."
mkdir -p /opt/chessnova/bin \
         /opt/chessnova/uploads/avatars \
         /opt/chessnova/scripts \
         /opt/chessnova/logs \
         /var/www/chessnova/html

chown -R yogesh:yogesh /opt/chessnova
chown -R www-data:www-data /var/www/chessnova/html

# Link system stockfish binary if not in /opt/chessnova/bin
if [ -f /usr/games/stockfish ] && [ ! -f /opt/chessnova/bin/stockfish ]; then
    ln -sf /usr/games/stockfish /opt/chessnova/bin/stockfish
elif [ -f /usr/bin/stockfish ] && [ ! -f /opt/chessnova/bin/stockfish ]; then
    ln -sf /usr/bin/stockfish /opt/chessnova/bin/stockfish
fi

# 5. Setup Docker Compose for MongoDB and Redis (Local Only)
echo "==> [5/9] Starting MongoDB and Redis in Docker..."
cat << 'EOF' > /opt/chessnova/docker-compose.yml
name: chessnova
services:
  mongodb:
    image: mongo:7
    container_name: chess-mongo
    restart: always
    ports:
      - "127.0.0.1:27017:27017"
    command: mongod --wiredTigerCacheSizeGB 0.5
    volumes:
      - mongo_data:/data/db
    environment:
      - MONGO_INITDB_DATABASE=chess

  redis:
    image: redis:7-alpine
    container_name: chess-redis
    restart: always
    ports:
      - "127.0.0.1:6379:6379"
    volumes:
      - redis_data:/data

volumes:
  mongo_data:
  redis_data:
EOF

cd /opt/chessnova && docker compose up -d

# 6. Setup Multi-Instance Systemd Service Template (8081, 8082, 8083)
echo "==> [6/9] Configuring multi-instance systemd cluster template..."
cat << 'EOF' > /etc/systemd/system/chessnova@.service
[Unit]
Description=ChessNova Real-Time Chess Server (Port %i)
After=network.target docker.service
Wants=docker.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/chessnova
EnvironmentFile=/opt/chessnova/.env
Environment="SERVER_PORT=%i"
Environment="INSTANCE_ID=instance-%i"
ExecStart=/opt/chessnova/bin/server
Restart=always
RestartSec=3s
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

# Disable legacy single-instance service if exists
systemctl disable --now chessnova.service 2>/dev/null || true
rm -f /etc/systemd/system/chessnova.service

systemctl daemon-reload

# 7. Self-Healing Health Watchdog
echo "==> [7/9] Setting up self-healing watchdog..."
cat << 'EOF' > /opt/chessnova/scripts/health-watchdog.sh
#!/usr/bin/env bash
INSTANCES=(8081 8082 8083)
LOG_FILE="/opt/chessnova/logs/watchdog.log"
mkdir -p /opt/chessnova/logs

for PORT in "${INSTANCES[@]}"; do
    STATUS=$(curl -s -m 4 "http://127.0.0.1:${PORT}/health" 2>/dev/null || echo "")
    if ! echo "$STATUS" | grep -q '"status":"ok"'; then
        echo "[$(date -u +"%Y-%m-%dT%H:%M:%SZ")] ALERT: instance-${PORT} unhealthy. Restarting..." >> "$LOG_FILE"
        systemctl restart "chessnova@${PORT}"
    fi
done
EOF
chmod +x /opt/chessnova/scripts/health-watchdog.sh

# Add to root crontab every minute
(crontab -l 2>/dev/null | grep -v "health-watchdog.sh" ; echo "* * * * * /opt/chessnova/scripts/health-watchdog.sh >/dev/null 2>&1") | crontab -

# 8. Setup Nginx Cluster Load Balancer Configuration
echo "==> [8/9] Configuring Nginx load balancer..."
cat << 'EOF' > /etc/nginx/sites-available/chessnova
upstream chessnova_cluster {
    server 127.0.0.1:8081 max_fails=2 fail_timeout=5s;
    server 127.0.0.1:8082 max_fails=2 fail_timeout=5s;
    server 127.0.0.1:8083 max_fails=2 fail_timeout=5s;
}

server {
    listen 80;
    listen [::]:80;
    server_name chess.yogeshmallik.in 162.35.173.209;

    root /var/www/chessnova/html;
    index index.html;

    client_max_body_size 10M;

    # Gzip Compression
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_proxied expired no-cache no-store private auth;
    gzip_types text/plain text/css text/xml text/javascript application/x-javascript application/xml application/javascript application/json;

    # Static avatar uploads
    location /uploads/avatars/ {
        alias /opt/chessnova/uploads/avatars/;
        expires 7d;
        add_header Cache-Control "public, no-transform";
    }

    # API Proxy to Go Backend Cluster with instant fallback on failure
    location /api/ {
        proxy_pass http://chessnova_cluster;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_next_upstream error timeout http_502 http_503 http_504;
        proxy_next_upstream_tries 3;
        proxy_connect_timeout 2s;
        proxy_read_timeout 60s;
    }

    # Health & Metrics Proxy
    location ~ ^/(health|metrics) {
        proxy_pass http://chessnova_cluster;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_next_upstream error timeout http_502 http_503 http_504;
    }

    # WebSocket Proxy
    location /ws/ {
        proxy_pass http://chessnova_cluster;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 86400s;
        proxy_send_timeout 86400s;
    }

    # Frontend SPA routing
    location / {
        try_files $uri $uri/ $uri.html /index.html;
    }
}
EOF

rm -f /etc/nginx/sites-enabled/default
ln -sf /etc/nginx/sites-available/chessnova /etc/nginx/sites-enabled/chessnova
nginx -t && systemctl restart nginx

# 9. Configure UFW Firewall
echo "==> [9/9] Configuring firewall..."
ufw allow 22/tcp || true
ufw allow 80/tcp || true
ufw allow 443/tcp || true
ufw --force enable || true

echo "=========================================="
echo " VPS Provisioning & Hardening Complete!"
echo "=========================================="
