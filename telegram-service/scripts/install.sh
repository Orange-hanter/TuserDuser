#!/bin/bash

# Telegram Service Installation and Deployment Script
# This script is meant to run on the server after the binary is copied to /tmp
# Usage: ./install.sh [SERVICE_NAME] [BINARY_PATH] [HEALTH_CHECK_URL]

set -euo pipefail

# Configuration
SERVICE_NAME="${1:-telegram-service}"
BINARY_PATH="${2:-/opt/telegram-service/bin/telegram-service}"
HEALTH_CHECK_URL="${3:-http://localhost:9090/health}"
BINARY_SOURCE="/tmp/dist/telegram-service"
LOG_FILE="/opt/telegram-service/logs/telegram-service.log"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log() {
	echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1"
}

success() {
	echo -e "${GREEN}✅ $1${NC}"
}

warning() {
	echo -e "${YELLOW}⚠️  $1${NC}"
}

error() {
	echo -e "${RED}❌ $1${NC}" >&2
	exit 1
}

# Main deployment steps
main() {
	log "Starting Telegram Service deployment..."
	echo ""

	# Step 1: Verify source binary exists
	if [ ! -f "$BINARY_SOURCE" ]; then
		error "Binary not found at $BINARY_SOURCE"
	fi
	success "Source binary found: $BINARY_SOURCE"
	echo ""

	# Step 2: Stop the service
	log "Stopping $SERVICE_NAME service..."
	if sudo systemctl stop "$SERVICE_NAME" 2>&1; then
		success "Service stopped"
	else
		warning "Service was not running or failed to stop, continuing..."
	fi
	sleep 2
	echo ""

	# Step 3: Backup old binary
	if [ -f "$BINARY_PATH" ]; then
		BACKUP_BINARY="${BINARY_PATH}.backup.$(date +%s)"
		log "Backing up existing binary to $BACKUP_BINARY..."
		if sudo cp "$BINARY_PATH" "$BACKUP_BINARY"; then
			success "Old binary backed up"
		else
			warning "Failed to backup old binary, but continuing..."
		fi
	fi
	echo ""

	# Step 4: Install new binary with correct permissions
	log "Installing new binary..."
	# Ensure directory exists
	sudo mkdir -p "$(dirname "$BINARY_PATH")"
	if sudo install -o telegramservice -g telegramservice -m 755 "$BINARY_SOURCE" "$BINARY_PATH" 2>/dev/null ||
		sudo install -o root -g root -m 755 "$BINARY_SOURCE" "$BINARY_PATH"; then
		success "Binary installed and permissions set"
	else
		error "Failed to install binary"
	fi
	echo ""

	# Step 5: Start the service
	log "Starting $SERVICE_NAME service..."
	if sudo systemctl start "$SERVICE_NAME"; then
		success "Service started"
	else
		error "Failed to start service"
	fi
	sleep 2
	echo ""

	# Step 6: Check service status
	log "Checking service status..."
	if sudo systemctl is-active --quiet "$SERVICE_NAME"; then
		success "Service is running"
		echo ""
		sudo systemctl status --no-pager "$SERVICE_NAME" | head -10
	else
		error "Service is not running"
	fi
	echo ""

	# Step 7: Health check (gRPC service - check via HTTP metrics endpoint or grpc_health_probe)
	log "Performing health check at $HEALTH_CHECK_URL..."
	if command -v curl &>/dev/null; then
		# Try HTTP health endpoint first
		HEALTH_RESPONSE=$(curl -s -w "\n%{http_code}" "$HEALTH_CHECK_URL" 2>&1 || echo "error")
		HTTP_CODE=$(echo "$HEALTH_RESPONSE" | tail -1)

		if [ "$HTTP_CODE" = "200" ]; then
			success "Health check passed (HTTP $HTTP_CODE)"
		else
			warning "Health check returned HTTP $HTTP_CODE, but service is running"
		fi
	else
		warning "curl not found, skipping health check"
	fi
	echo ""

	# Step 8: Show recent logs
	log "Last 20 lines of service logs:"
	echo ""
	if [ -f "$LOG_FILE" ]; then
		tail -20 "$LOG_FILE"
	else
		warning "Log file not found at $LOG_FILE"
		log "Trying journalctl..."
		sudo journalctl -u "$SERVICE_NAME" --no-pager -n 20 2>/dev/null || warning "journalctl not available"
	fi
	echo ""

	# Cleanup
	log "Cleaning up temporary files..."
	if rm -f "$BINARY_SOURCE"; then
		success "Temporary files cleaned up"
	else
		warning "Failed to remove temporary binary"
	fi
	echo ""

	success "Deployment completed successfully!"
	log "Service $SERVICE_NAME is ready"
}

# Run main function
main "$@"
