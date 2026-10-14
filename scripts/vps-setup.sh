#!/usr/bin/env bash
# ==============================================================================
# ChessNova Production VPS Provisioning & Deployment Script
# ==============================================================================
set -euo pipefail

echo "=========================================="
echo " Starting ChessNova VPS Setup"
echo "=========================================="

# 1. Base updates & essential packages
export DEBIAN_FRONTEND=noninteractive
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
    unzip

# 2. Configure SSH & prevent pam_faillock lockouts
mkdir -p /etc/ssh/sshd_config.d/
cat << 'EOF' > /etc/ssh/sshd_config.d/99-chessnova.conf
PubkeyAuthentication yes
PasswordAuthentication yes
KbdInteractiveAuthentication yes
ClientAliveInterval 60
ClientAliveCountMax 10
EOF

# Ensure faillock does not lock out legitimate admin users
if [ -f /etc/security/faillock.conf ]; then
    sed -i 's/^#*deny =.*/deny = 10/' /etc/security/faillock.conf || true
    sed -i 's/^#*unlock_time =.*/unlock_time = 300/' /etc/security/faillock.conf || true
fi

systemctl restart ssh || systemctl restart sshd || true

# 3. Create user yogesh if not exists and configure sudo + ssh key
PUBKEY="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICJ6OvB0ABT9gC5n9QLZ6FnXVI1FpdDngwifScFLTo0s yogesh@vps"

id yogesh >/dev/null 2>&1 || useradd -m -s /bin/bash yogesh
usermod -aG sudo yogesh

mkdir -p /root/.ssh
chmod 700 /root/.ssh
echo "$PUBKEY" >> /root/.ssh/authorized_keys
sort -u /root/.ssh/authorized_keys -o /root/.ssh/authorized_keys
chmod 600 /root/.ssh/authorized_keys

mkdir -p /home/yogesh/.ssh
chmod 700 /home/yogesh/.ssh
echo "$PUBKEY" >> /home/yogesh/.ssh/authorized_keys
sort -u /home/yogesh/.ssh/authorized_keys -o /home/yogesh/.ssh/authorized_keys
chmod 600 /home/yogesh/.ssh/authorized_keys
chown -R yogesh:yogesh /home/yogesh/.ssh

echo "yogesh ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/yogesh
chmod 440 /etc/sudoers.d/yogesh

# 4. Install Docker & Docker Compose if not installed
if ! command -v docker >/dev/null 2>&1; then
    echo "Installing Docker..."
    curl -fsSL https://get.docker.com -o /tmp/get-docker.sh
    sh /tmp/get-docker.sh
    rm -f /tmp/get-docker.sh
fi
usermod -aG docker yogesh || true

# 5. Create application directories
mkdir -p /opt/chessnova/bin \
         /opt/chessnova/uploads/avatars \
         /var/www/chessnova/html

chown -R yogesh:yogesh /opt/chessnova /var/www/chessnova

# 6. Setup Docker Compose for MongoDB and Redis (local only)
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

# 7. Setup systemd service for ChessNova Backend
cat << 'EOF' > /etc/systemd/system/chessnova.service
[Unit]
Description=ChessNova Real-Time Chess Server
After=network.target docker.service
Wants=docker.service

[Service]
Type=simple
User=yogesh
Group=yogesh
WorkingDirectory=/opt/chessnova
EnvironmentFile=/opt/chessnova/.env
ExecStart=/opt/chessnova/bin/server
Restart=always
RestartSec=3
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload

# 8. Setup Nginx Configuration
cat << 'EOF' > /etc/nginx/sites-available/chessnova
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

    # API Proxy to Go Backend
    location /api/ {
        proxy_pass http://127.0.0.1:8080/api/;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Health check
    location /health {
        proxy_pass http://127.0.0.1:8080/health;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
    }

    # WebSocket Proxy
    location /ws/ {
        proxy_pass http://127.0.0.1:8080/ws/;
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

# 9. Configure UFW Firewall safely
ufw allow 22/tcp || true
ufw allow 80/tcp || true
ufw allow 443/tcp || true
ufw --force enable || true

echo "=========================================="
echo " VPS Base Provisioning Completed Successfully!"
echo "=========================================="
