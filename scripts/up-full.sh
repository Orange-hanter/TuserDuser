#!/usr/bin/env bash
set -euo pipefail

# Start the full development stack by composing the existing service files.
# This avoids duplicating service definitions across compose files.

BASE_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
echo "Starting full stack using event-api and telegram-service compose files..."

docker compose -f "$BASE_DIR/event-api/docker-compose.yml" -f "$BASE_DIR/telegram-service/docker-compose.yml" up -d --remove-orphans

echo "Done. Use 'docker compose ps' to inspect running containers."
