#!/bin/bash

# Quick deployment script for Event API binary updates
# Usage: ./scripts/deploy-binary.sh

set -e

# Configuration
REMOTE_USER="root"
REMOTE_HOST="tuser"
APP_DIR="/opt/event-api"
SERVICE_NAME="event-api"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Event API - Binary Deployment${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Step 1: Build the binary
echo -e "${YELLOW}🔨 Building binary...${NC}"
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/event-api-linux ./cmd/server
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Binary built successfully${NC}"
else
    echo -e "${RED}❌ Build failed${NC}"
    exit 1
fi

# Step 2: Create backup on server
echo ""
echo -e "${YELLOW}💾 Creating backup on server...${NC}"
ssh "$REMOTE_HOST" "sudo -u eventapi $APP_DIR/backup.sh" || echo "Backup script not found, skipping..."

# Step 3: Stop the service
echo ""
echo -e "${YELLOW}🛑 Stopping service...${NC}"
ssh "$REMOTE_HOST" "systemctl stop $SERVICE_NAME" || echo "Service not running"

# Step 4: Upload new binary
echo ""
echo -e "${YELLOW}📤 Uploading binary...${NC}"
scp bin/event-api-linux "$REMOTE_HOST:/tmp/event-api"
ssh "$REMOTE_HOST" "mv /tmp/event-api $APP_DIR/bin/event-api && chown eventapi:eventapi $APP_DIR/bin/event-api && chmod +x $APP_DIR/bin/event-api"
echo -e "${GREEN}✅ Binary uploaded${NC}"

# Step 5: Start the service
echo ""
echo -e "${YELLOW}🚀 Starting service...${NC}"
ssh "$REMOTE_HOST" "systemctl start $SERVICE_NAME"

# Step 6: Check status
echo ""
echo -e "${YELLOW}🔍 Checking service status...${NC}"
sleep 3
ssh "$REMOTE_HOST" "systemctl is-active $SERVICE_NAME && echo '✅ Service is running' || echo '❌ Service failed to start'"

# Step 7: Test health endpoint
echo ""
echo -e "${YELLOW}🏥 Testing health endpoint...${NC}"
sleep 2
HEALTH_CHECK=$(ssh "$REMOTE_HOST" "curl -s http://localhost:8080/v1/api/health" || echo "failed")
if [[ "$HEALTH_CHECK" == *"ok"* ]] || [[ "$HEALTH_CHECK" == *"healthy"* ]]; then
    echo -e "${GREEN}✅ Health check passed${NC}"
else
    echo -e "${YELLOW}⚠️  Health check response: $HEALTH_CHECK${NC}"
fi

# Step 8: Show logs
echo ""
echo -e "${YELLOW}📋 Recent logs:${NC}"
ssh "$REMOTE_HOST" "tail -20 $APP_DIR/logs/event-api.log"

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}✅ Deployment Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${BLUE}🔗 API URL: http://api.tuserduser.online${NC}"
echo -e "${BLUE}📊 Health: http://api.tuserduser.online/v1/api/health${NC}"
echo -e "${BLUE}📖 Swagger: http://api.tuserduser.online/swagger/index.html${NC}"
echo ""
echo -e "${YELLOW}View logs: ssh $REMOTE_HOST tail -f $APP_DIR/logs/event-api.log${NC}"
echo -e "${YELLOW}Service status: ssh $REMOTE_HOST systemctl status $SERVICE_NAME${NC}"
