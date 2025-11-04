#!/bin/bash

# Binary Deployment Script for Event API
# This script sets up the production environment and deploys the Go binary

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Event API - Binary Deployment Setup  ${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Configuration
APP_NAME="event-api"
APP_USER="eventapi"
APP_DIR="/opt/event-api"
NGINX_SITE_CONFIG="/etc/nginx/sites-available/event-api-backend"
SYSTEMD_SERVICE="/etc/systemd/system/event-api.service"

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}❌ Please run as root (use: sudo)${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Running as root${NC}"

# Step 1: Update system
echo ""
echo -e "${YELLOW}📦 Updating system packages...${NC}"
apt-get update -qq

# Step 2: Install PostgreSQL
echo ""
echo -e "${YELLOW}🗄️  Installing PostgreSQL...${NC}"
if ! command -v psql &> /dev/null; then
    apt-get install -y postgresql postgresql-contrib
    systemctl start postgresql
    systemctl enable postgresql
    echo -e "${GREEN}✅ PostgreSQL installed${NC}"
else
    echo -e "${GREEN}✅ PostgreSQL already installed${NC}"
fi

# Step 3: Install Redis
echo ""
echo -e "${YELLOW}🔴 Installing Redis...${NC}"
if ! command -v redis-cli &> /dev/null; then
    apt-get install -y redis-server
    systemctl start redis-server
    systemctl enable redis-server
    echo -e "${GREEN}✅ Redis installed${NC}"
else
    echo -e "${GREEN}✅ Redis already installed${NC}"
fi

# Step 4: Create application user
echo ""
echo -e "${YELLOW}👤 Creating application user...${NC}"
if ! id "$APP_USER" &>/dev/null; then
    useradd -r -s /bin/bash -d "$APP_DIR" -m "$APP_USER"
    echo -e "${GREEN}✅ User $APP_USER created${NC}"
else
    echo -e "${GREEN}✅ User $APP_USER already exists${NC}"
fi

# Step 5: Create directory structure
echo ""
echo -e "${YELLOW}📁 Creating directory structure...${NC}"
mkdir -p "$APP_DIR"/{bin,logs,backups}
chown -R "$APP_USER":"$APP_USER" "$APP_DIR"
echo -e "${GREEN}✅ Directories created${NC}"

# Step 6: Setup PostgreSQL database
echo ""
echo -e "${YELLOW}🗄️  Setting up PostgreSQL database...${NC}"
DB_NAME="event_api"
DB_USER="eventapi"
DB_PASSWORD=$(openssl rand -base64 32)

sudo -u postgres psql -tc "SELECT 1 FROM pg_database WHERE datname = '$DB_NAME'" | grep -q 1 || \
sudo -u postgres psql -c "CREATE DATABASE $DB_NAME;"

sudo -u postgres psql -tc "SELECT 1 FROM pg_user WHERE usename = '$DB_USER'" | grep -q 1 || \
sudo -u postgres psql -c "CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';"

sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;"
sudo -u postgres psql -d "$DB_NAME" -c "GRANT ALL ON SCHEMA public TO $DB_USER;"

echo -e "${GREEN}✅ PostgreSQL database configured${NC}"
echo -e "${BLUE}   Database: $DB_NAME${NC}"
echo -e "${BLUE}   User: $DB_USER${NC}"
echo -e "${BLUE}   Password: $DB_PASSWORD${NC}"

# Step 7: Configure Redis
echo ""
echo -e "${YELLOW}🔴 Configuring Redis...${NC}"
REDIS_PASSWORD=$(openssl rand -base64 32)

# Update Redis configuration to require password
if ! grep -q "^requirepass" /etc/redis/redis.conf; then
    echo "requirepass $REDIS_PASSWORD" >> /etc/redis/redis.conf
    systemctl restart redis-server
    echo -e "${GREEN}✅ Redis password configured${NC}"
    echo -e "${BLUE}   Password: $REDIS_PASSWORD${NC}"
else
    echo -e "${YELLOW}⚠️  Redis password already set (keeping existing)${NC}"
    REDIS_PASSWORD="<existing_password>"
fi

# Step 8: Generate JWT secret
echo ""
echo -e "${YELLOW}🔐 Generating JWT secret...${NC}"
JWT_SECRET=$(openssl rand -base64 64)
echo -e "${GREEN}✅ JWT secret generated${NC}"

# Step 9: Create .env file
echo ""
echo -e "${YELLOW}⚙️  Creating environment configuration...${NC}"
cat > "$APP_DIR/.env" << EOF
# Server Configuration
PORT=8080
HOST=0.0.0.0
ENV=production

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=$DB_USER
DB_PASSWORD=$DB_PASSWORD
DB_NAME=$DB_NAME
DB_SSLMODE=disable

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=$REDIS_PASSWORD
REDIS_DB=0

# JWT Configuration
JWT_SECRET=$JWT_SECRET
JWT_EXPIRATION=24h

# Email Configuration (Update with your SMTP details)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your_email@gmail.com
SMTP_PASSWORD=your_app_password
SMTP_FROM=noreply@tuserduser.online

# SMS Configuration
SMS_PROVIDER=mock
SMS_API_KEY=
SMS_API_TOKEN=
SMS_FROM=EventAPI

# CORS Configuration
CORS_ALLOWED_ORIGINS=https://tuserduser.online,https://www.tuserduser.online,http://localhost:3000
EOF

chown "$APP_USER":"$APP_USER" "$APP_DIR/.env"
chmod 600 "$APP_DIR/.env"
echo -e "${GREEN}✅ Environment file created${NC}"

# Step 10: Create systemd service
echo ""
echo -e "${YELLOW}🔧 Creating systemd service...${NC}"
cat > "$SYSTEMD_SERVICE" << EOF
[Unit]
Description=Event API Service
After=network.target postgresql.service redis-server.service
Wants=postgresql.service redis-server.service

[Service]
Type=simple
User=$APP_USER
Group=$APP_USER
WorkingDirectory=$APP_DIR
EnvironmentFile=$APP_DIR/.env
ExecStart=$APP_DIR/bin/event-api
Restart=on-failure
RestartSec=10
StandardOutput=append:$APP_DIR/logs/event-api.log
StandardError=append:$APP_DIR/logs/event-api-error.log

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$APP_DIR

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
echo -e "${GREEN}✅ Systemd service created${NC}"

# Step 11: Create Nginx configuration
echo ""
echo -e "${YELLOW}🌐 Creating Nginx configuration...${NC}"

# First, add rate limiting to nginx.conf if not already there
if ! grep -q "limit_req_zone.*api_limit" /etc/nginx/nginx.conf; then
    # Add rate limiting zone to http block
    sed -i '/http {/a \    # Rate limiting for API\n    limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;' /etc/nginx/nginx.conf
    echo -e "${GREEN}✅ Rate limiting configured in nginx.conf${NC}"
fi

cat > "$NGINX_SITE_CONFIG" << 'EOF'
upstream event_api_backend {
    server 127.0.0.1:8080;
    keepalive 64;
}

server {
    listen 80;
    server_name api.tuserduser.online;
    
    # Logging
    access_log /var/log/nginx/event-api-access.log;
    error_log /var/log/nginx/event-api-error.log;
    
    # API endpoints
    location /v1/api/ {
        limit_req zone=api_limit burst=20 nodelay;
        
        proxy_pass http://event_api_backend;
        proxy_http_version 1.1;
        
        # Headers
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Connection "";
        
        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
    
    # Health check endpoint (no rate limit)
    location /v1/api/health {
        proxy_pass http://event_api_backend;
        access_log off;
    }
    
    # Swagger documentation
    location /swagger/ {
        proxy_pass http://event_api_backend;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
    
    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    
    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_types text/plain text/css text/xml text/javascript application/json application/javascript application/xml+rss;
}
EOF

# Enable the site
ln -sf "$NGINX_SITE_CONFIG" /etc/nginx/sites-enabled/event-api-backend

# Test Nginx configuration
if nginx -t; then
    systemctl reload nginx
    echo -e "${GREEN}✅ Nginx configuration created and loaded${NC}"
else
    echo -e "${RED}❌ Nginx configuration has errors${NC}"
    exit 1
fi

# Step 12: Setup log rotation
echo ""
echo -e "${YELLOW}📋 Setting up log rotation...${NC}"
cat > /etc/logrotate.d/event-api << EOF
$APP_DIR/logs/*.log {
    daily
    rotate 14
    compress
    delaycompress
    notifempty
    create 0640 $APP_USER $APP_USER
    sharedscripts
    postrotate
        systemctl reload event-api || true
    endscript
}
EOF
echo -e "${GREEN}✅ Log rotation configured${NC}"

# Step 13: Create backup script
echo ""
echo -e "${YELLOW}💾 Creating backup script...${NC}"
cat > "$APP_DIR/backup.sh" << 'BACKUP_EOF'
#!/bin/bash
BACKUP_DIR="/opt/event-api/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/event_api_backup_$TIMESTAMP.sql"

mkdir -p "$BACKUP_DIR"
pg_dump -U eventapi -d event_api > "$BACKUP_FILE"
gzip "$BACKUP_FILE"

# Keep only last 7 days of backups
find "$BACKUP_DIR" -name "event_api_backup_*.sql.gz" -mtime +7 -delete

echo "Backup completed: ${BACKUP_FILE}.gz"
BACKUP_EOF

chmod +x "$APP_DIR/backup.sh"
chown "$APP_USER":"$APP_USER" "$APP_DIR/backup.sh"

# Setup daily backup cron
(crontab -u "$APP_USER" -l 2>/dev/null; echo "0 2 * * * $APP_DIR/backup.sh >> $APP_DIR/logs/backup.log 2>&1") | crontab -u "$APP_USER" -
echo -e "${GREEN}✅ Backup script created and scheduled${NC}"

# Summary
echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}✅ Setup Complete!${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${YELLOW}📋 Configuration Summary:${NC}"
echo -e "   App Directory: ${BLUE}$APP_DIR${NC}"
echo -e "   App User: ${BLUE}$APP_USER${NC}"
echo -e "   PostgreSQL Database: ${BLUE}$DB_NAME${NC}"
echo -e "   PostgreSQL User: ${BLUE}$DB_USER${NC}"
echo -e "   API URL: ${BLUE}http://api.tuserduser.online${NC}"
echo ""
echo -e "${YELLOW}🔐 Credentials saved in: ${BLUE}$APP_DIR/.env${NC}"
echo ""
echo -e "${YELLOW}📝 Next Steps:${NC}"
echo -e "   1. Update SMTP settings in ${BLUE}$APP_DIR/.env${NC}"
echo -e "   2. Build and upload binary: ${BLUE}make build && scp bin/server tuser:$APP_DIR/bin/event-api${NC}"
echo -e "   3. Start service: ${BLUE}systemctl start event-api${NC}"
echo -e "   4. Check status: ${BLUE}systemctl status event-api${NC}"
echo -e "   5. View logs: ${BLUE}tail -f $APP_DIR/logs/event-api.log${NC}"
echo -e "   6. Setup SSL: ${BLUE}certbot --nginx -d api.tuserduser.online${NC}"
echo ""
echo -e "${GREEN}🎉 Ready for deployment!${NC}"
