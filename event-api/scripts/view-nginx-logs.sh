#!/bin/bash

###############################################################################
# Script to view remote nginx logs with filtering options
# Скрипт для просмотра логов nginx на удаленном сервере с фильтрацией
#
# Usage / Использование:
#   ./scripts/view-nginx-logs.sh [OPTIONS]
#
# Environment variables / Переменные окружения:
#   REMOTE_HOST, REMOTE_USER, SSH_PORT, SSH_KEY_PATH
###############################################################################

set -euo pipefail

###############################################################################
# Colors / Цвета
###############################################################################
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color
BOLD='\033[1m'

###############################################################################
# Global variables / Глобальные переменные
###############################################################################
REMOTE_HOST="${REMOTE_HOST-}"
REMOTE_USER="${REMOTE_USER-}"
SSH_PORT="${SSH_PORT:-22}"
SSH_KEY_PATH="${SSH_KEY_PATH-}"
LOG_TYPE=""
LINES="${LINES:-50}"
FOLLOW="${FOLLOW:-false}"
FILTER="${FILTER-}"

###############################################################################
# Output functions / Функции вывода
###############################################################################
error() {
	echo -e "${RED}❌ Error: $1${NC}" >&2
	exit 1
}

success() {
	echo -e "${GREEN}✅ $1${NC}"
}

info() {
	echo -e "${BLUE}ℹ️  $1${NC}"
}

warning() {
	echo -e "${YELLOW}⚠️  $1${NC}"
}

###############################################################################
# Show help / Показать помощь
###############################################################################
show_help() {
	cat <<EOF
${BOLD}${CYAN}Nginx Logs Viewer - Remote Server${NC}

${BOLD}Usage / Использование:${NC}
    $0 [OPTIONS]

${BOLD}Options / Опции:${NC}
    -h, --host HOST           Remote server hostname or IP (required)
    -u, --user USER           SSH username (required)
    -p, --port PORT           SSH port (default: 22)
    -k, --key PATH            Path to SSH private key
    -t, --type TYPE           Log type to view (required):
                                access-staging  - Staging access logs
                                error-staging   - Staging error logs
                                access-api      - API access logs
                                error-api       - API error logs
                                nginx-access    - Main nginx access log
                                nginx-error     - Main nginx error log
                                all-errors      - All error logs
    -n, --lines NUM           Number of lines to show (default: 50)
    -f, --follow              Follow log output (like tail -f)
    --filter TEXT             Filter logs by text (grep)
    --help                    Show this help message

${BOLD}Environment Variables / Переменные окружения:${NC}
    REMOTE_HOST              Remote server hostname or IP
    REMOTE_USER              SSH username
    SSH_PORT                 SSH port (default: 22)
    SSH_KEY_PATH             Path to SSH private key

${BOLD}Examples / Примеры:${NC}
    # View last 50 lines of staging access logs
    $0 --host 5.35.127.138 --user root --key ~/.ssh/id_rsa --type access-staging

    # Follow API error logs in real-time
    $0 -h 5.35.127.138 -u root -k ~/.ssh/id_rsa -t error-api --follow

    # View last 100 lines of nginx error log
    $0 -h 5.35.127.138 -u root -t nginx-error -n 100

    # Filter staging logs for specific endpoint
    $0 -h 5.35.127.138 -u root -t access-staging --filter "/v1/api/auth"

    # Using environment variables
    REMOTE_HOST=5.35.127.138 REMOTE_USER=root SSH_KEY_PATH=~/.ssh/id_rsa \\
    $0 --type access-api --follow

${BOLD}Available Log Types / Доступные типы логов:${NC}
    ${GREEN}access-staging${NC}   - /var/log/nginx/staging-access.log
    ${GREEN}error-staging${NC}    - /var/log/nginx/staging-error.log
    ${GREEN}access-api${NC}       - /var/log/nginx/api-access.log
    ${GREEN}error-api${NC}        - /var/log/nginx/api-error.log
    ${GREEN}nginx-access${NC}     - /var/log/nginx/access.log
    ${GREEN}nginx-error${NC}      - /var/log/nginx/error.log
    ${GREEN}all-errors${NC}       - All error logs combined
EOF
}

###############################################################################
# Parse arguments / Парсинг аргументов
###############################################################################
parse_arguments() {
	while [[ $# -gt 0 ]]; do
		case $1 in
		-h | --host)
			REMOTE_HOST="$2"
			shift 2
			;;
		-u | --user)
			REMOTE_USER="$2"
			shift 2
			;;
		-p | --port)
			SSH_PORT="$2"
			shift 2
			;;
		-k | --key)
			SSH_KEY_PATH="$2"
			shift 2
			;;
		-t | --type)
			LOG_TYPE="$2"
			shift 2
			;;
		-n | --lines)
			LINES="$2"
			shift 2
			;;
		-f | --follow)
			FOLLOW="true"
			shift
			;;
		--filter)
			FILTER="$2"
			shift 2
			;;
		--help)
			show_help
			exit 0
			;;
		*)
			error "Unknown option: $1. Use --help for usage information."
			;;
		esac
	done
}

###############################################################################
# Validate parameters / Валидация параметров
###############################################################################
validate_parameters() {
	if [ -z "$REMOTE_HOST" ]; then
		error "REMOTE_HOST is required (use -h/--host or set REMOTE_HOST environment variable)"
	fi

	if [ -z "$REMOTE_USER" ]; then
		error "REMOTE_USER is required (use -u/--user or set REMOTE_USER environment variable)"
	fi

	if [ -z "$LOG_TYPE" ]; then
		error "LOG_TYPE is required (use -t/--type). Available: access-staging, error-staging, access-api, error-api, nginx-access, nginx-error, all-errors"
	fi

	# Validate log type
	case $LOG_TYPE in
	access-staging | error-staging | access-api | error-api | nginx-access | nginx-error | all-errors) ;;
	*)
		error "Invalid log type: $LOG_TYPE. Use --help to see available types."
		;;
	esac

	if [ -n "$SSH_KEY_PATH" ] && [ ! -f "$SSH_KEY_PATH" ]; then
		error "SSH key file not found: $SSH_KEY_PATH"
	fi
}

###############################################################################
# Get log file path / Получить путь к файлу лога
###############################################################################
get_log_path() {
	case $LOG_TYPE in
	access-staging)
		echo "/var/log/nginx/staging-access.log"
		;;
	error-staging)
		echo "/var/log/nginx/staging-error.log"
		;;
	access-api)
		echo "/var/log/nginx/api-access.log"
		;;
	error-api)
		echo "/var/log/nginx/api-error.log"
		;;
	nginx-access)
		echo "/var/log/nginx/access.log"
		;;
	nginx-error)
		echo "/var/log/nginx/error.log"
		;;
	all-errors)
		echo "/var/log/nginx/*error.log"
		;;
	esac
}

###############################################################################
# View logs / Просмотр логов
###############################################################################
view_logs() {
	info "Connecting to $REMOTE_USER@$REMOTE_HOST:$SSH_PORT"

	SSH_CMD="ssh -p $SSH_PORT"
	if [ -n "$SSH_KEY_PATH" ]; then
		SSH_CMD="$SSH_CMD -i $SSH_KEY_PATH"
	fi
	SSH_CMD="$SSH_CMD -o StrictHostKeyChecking=no -o ConnectTimeout=10"
	SSH_CMD="$SSH_CMD $REMOTE_USER@$REMOTE_HOST"

	LOG_PATH=$(get_log_path)

	echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════════════${NC}"
	echo -e "${BOLD}${CYAN}  Viewing: ${MAGENTA}$LOG_TYPE${NC}"
	echo -e "${BOLD}${CYAN}  Path: ${NC}$LOG_PATH"
	if [ "$FOLLOW" = "true" ]; then
		echo -e "${BOLD}${CYAN}  Mode: ${YELLOW}Follow (Ctrl+C to stop)${NC}"
	else
		echo -e "${BOLD}${CYAN}  Lines: ${NC}$LINES"
	fi
	if [ -n "$FILTER" ]; then
		echo -e "${BOLD}${CYAN}  Filter: ${GREEN}$FILTER${NC}"
	fi
	echo -e "${BOLD}${CYAN}═══════════════════════════════════════════════════════════════${NC}"
	echo ""

	# Build the command
	if [ "$FOLLOW" = "true" ]; then
		CMD="sudo tail -f $LOG_PATH"
	else
		CMD="sudo tail -n $LINES $LOG_PATH"
	fi

	# Add filter if specified
	if [ -n "$FILTER" ]; then
		CMD="$CMD | grep --color=always -E '$FILTER|$'"
	fi

	# Execute remote command
	$SSH_CMD "$CMD"
}

###############################################################################
# Main function / Главная функция
###############################################################################
main() {
	echo -e "${BOLD}${BLUE}"
	echo "╔══════════════════════════════════════════════════════════════╗"
	echo "║           Remote Nginx Logs Viewer                          ║"
	echo "╚══════════════════════════════════════════════════════════════╝"
	echo -e "${NC}"
	echo ""

	parse_arguments "$@"
	validate_parameters
	view_logs
}

###############################################################################
# Run script / Запуск скрипта
###############################################################################
main "$@"
