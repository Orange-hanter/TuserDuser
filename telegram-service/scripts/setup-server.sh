#!/bin/bash

# Telegram Service Initial Server Setup Script
# Run this script ONCE on a fresh server to prepare for telegram-service deployment
# Usage: sudo ./setup-server.sh

set -euo pipefail

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1"; }
success() { echo -e "${GREEN}✅ $1${NC}"; }
warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }
error() {
	echo -e "${RED}❌ $1${NC}" >&2
	exit 1
}

# Check if running as root
if [ "$EUID" -ne 0 ]; then
	error "Please run this script as root (sudo)"
fi

log "Starting Telegram Service server setup..."
echo ""

# Step 1: Create service user
log "Creating service user 'telegramservice'..."
if id "telegramservice" &>/dev/null; then
	warning "User 'telegramservice' already exists"
else
	useradd -r -s /bin/false -d /opt/telegram-service telegramservice
	success "User 'telegramservice' created"
fi
echo ""

# Step 2: Create directory structure
log "Creating directory structure..."
mkdir -p /opt/telegram-service/{bin,logs,scripts}
success "Directories created"
echo ""

# Step 3: Set ownership
log "Setting ownership..."
chown -R telegramservice:telegramservice /opt/telegram-service
chmod 755 /opt/telegram-service
chmod 750 /opt/telegram-service/logs
success "Ownership set"
echo ""

# Step 4: Copy and enable systemd service
log "Setting up systemd service..."
if [ -f "./telegram-service.service" ]; then
	cp ./telegram-service.service /etc/systemd/system/
elif [ -f "/tmp/telegram-service.service" ]; then
	cp /tmp/telegram-service.service /etc/systemd/system/
else
	cat >/etc/systemd/system/telegram-service.service <<'EOF'
[Unit]
Description=Telegram Service - gRPC microservice for Telegram Bot API
Documentation=https://github.com/Orange-hanter/TuserDuser
After=network-online.target postgresql.service
Wants=network-online.target

[Service]
Type=simple
User=telegramservice
Group=telegramservice
WorkingDirectory=/opt/telegram-service

ExecStart=/opt/telegram-service/bin/telegram-service
ExecReload=/bin/kill -HUP $MAINPID

Restart=on-failure
RestartSec=5
TimeoutStartSec=30
TimeoutStopSec=30

# Environment
EnvironmentFile=-/opt/telegram-service/.env

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=/opt/telegram-service/logs

# Logging
StandardOutput=append:/opt/telegram-service/logs/telegram-service.log
StandardError=append:/opt/telegram-service/logs/telegram-service.log

# Limits
LimitNOFILE=65535
LimitNPROC=4096

[Install]
WantedBy=multi-user.target
EOF
fi

chmod 644 /etc/systemd/system/telegram-service.service
systemctl daemon-reload
systemctl enable telegram-service
success "Systemd service installed and enabled"
echo ""

# Step 5: Create .env template
log "Creating .env template..."
if [ ! -f "/opt/telegram-service/.env" ]; then
	cat >/opt/telegram-service/.env <<'EOF'
# Telegram Service Environment Configuration

# Environment
ENV=production

# gRPC Server
GRPC_PORT=50051

# HTTP Metrics Server
METRICS_PORT=9090

# Database
DATABASE_URL=postgres://telegramservice:password@localhost:5432/telegram_service?sslmode=disable

# Telegram Bot
TELEGRAM_BOT_TOKEN=your_bot_token_here

# Logging
LOG_LEVEL=info
EOF
	chown telegramservice:telegramservice /opt/telegram-service/.env
	chmod 600 /opt/telegram-service/.env
	warning "Created .env template - PLEASE EDIT with actual values!"
else
	warning ".env file already exists, skipping"
fi
echo ""

# Step 6: Setup log rotation
log "Setting up log rotation..."
cat >/etc/logrotate.d/telegram-service <<'EOF'
/opt/telegram-service/logs/*.log {
    daily
    missingok
    rotate 14
    compress
    delaycompress
    notifempty
    create 0640 telegramservice telegramservice
    sharedscripts
    postrotate
        systemctl reload telegram-service > /dev/null 2>&1 || true
    endscript
}
EOF
success "Log rotation configured"
echo ""

success "Server setup completed!"
echo ""
log "Next steps:"
echo "  1. Edit /opt/telegram-service/.env with actual configuration values"
echo "  2. Ensure PostgreSQL database is accessible"
echo "  3. Run CI/CD pipeline or manually deploy the binary"
echo ""
log "Commands:"
echo "  - Start service:   sudo systemctl start telegram-service"
echo "  - Check status:    sudo systemctl status telegram-service"
echo "  - View logs:       sudo journalctl -u telegram-service -f"
echo ""
