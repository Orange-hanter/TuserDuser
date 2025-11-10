#!/bin/bash

###############################################################################
# Скрипт для безопасной публикации конфигурации nginx на удаленный сервер
# Secure nginx configuration deployment script to remote server via SSH
#
# Использование / Usage:
#   ./scripts/deploy-nginx.sh [OPTIONS]
#
# Переменные окружения / Environment variables:
#   REMOTE_HOST, REMOTE_USER, SSH_PORT, SSH_KEY_PATH,
#   NGINX_CONF_SOURCE, NGINX_CONF_DEST, NGINX_SITES_AVAILABLE,
#   NGINX_SITES_ENABLED, SITE_NAME, BACKUP_EXISTING, AUTO_MODE
#
# Примеры / Examples:
#   ./scripts/deploy-nginx.sh \
#     --host 192.168.1.100 \
#     --user deploy \
#     --source nginx.conf \
#     --dest /etc/nginx/sites-available/event-api \
#     --site event-api
#
#   REMOTE_HOST=192.168.1.100 REMOTE_USER=deploy \
#   ./scripts/deploy-nginx.sh --source nginx.conf --site event-api
###############################################################################

set -euo pipefail

###############################################################################
# Цвета для вывода / Color output
###############################################################################
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color
BOLD='\033[1m'

###############################################################################
# Глобальные переменные / Global variables
###############################################################################
REMOTE_HOST=""
REMOTE_USER=""
SSH_PORT="${SSH_PORT:-22}"
SSH_KEY_PATH="${SSH_KEY_PATH-}"
NGINX_CONF_SOURCE=""
NGINX_CONF_DEST=""
NGINX_SITES_AVAILABLE="${NGINX_SITES_AVAILABLE:-/etc/nginx/sites-available}"
NGINX_SITES_ENABLED="${NGINX_SITES_ENABLED:-/etc/nginx/sites-enabled}"
SITE_NAME=""
BACKUP_EXISTING="${BACKUP_EXISTING:-true}"
AUTO_MODE="${AUTO_MODE:-false}"
ROLLBACK_ENABLED=false
BACKUP_FILE=""
TEMP_FILE=""

###############################################################################
# Функции для вывода / Output functions
###############################################################################
error() {
	echo -e "${RED}❌ Error: $1${NC}" >&2
	exit 1
}

success() {
	echo -e "${GREEN}✅ $1${NC}"
}

warning() {
	echo -e "${YELLOW}⚠️  $1${NC}"
}

info() {
	echo -e "${BLUE}ℹ️  $1${NC}"
}

step() {
	echo -e "${CYAN}${BOLD}▶ $1${NC}"
}

###############################################################################
# Функция очистки при выходе / Cleanup function on exit
###############################################################################
cleanup() {
	if [ -n "$TEMP_FILE" ] && [ -f "$TEMP_FILE" ]; then
		rm -f "$TEMP_FILE"
	fi
}

trap cleanup EXIT INT TERM

###############################################################################
# Функция отката изменений / Rollback function
###############################################################################
rollback() {
	if [ "$ROLLBACK_ENABLED" = false ]; then
		return 0
	fi

	warning "Starting rollback procedure..."

	SSH_CMD="ssh -p $SSH_PORT"
	if [ -n "$SSH_KEY_PATH" ]; then
		SSH_CMD="$SSH_CMD -i $SSH_KEY_PATH"
	fi
	SSH_CMD="$SSH_CMD -o StrictHostKeyChecking=no -o ConnectTimeout=10"
	SSH_CMD="$SSH_CMD $REMOTE_USER@$REMOTE_HOST"

	# Восстановление бэкапа / Restore backup
	if [ -n "$BACKUP_FILE" ]; then
		info "Restoring backup from $BACKUP_FILE..."
		if $SSH_CMD "test -f $BACKUP_FILE" 2>/dev/null; then
			$SSH_CMD "sudo cp $BACKUP_FILE $NGINX_CONF_DEST" 2>/dev/null || true
			info "Backup restored"
		fi
	fi

	# Удаление нового файла если он был создан / Remove new file if created
	if [ -n "$NGINX_CONF_DEST" ]; then
		$SSH_CMD "sudo rm -f $NGINX_CONF_DEST.new" 2>/dev/null || true
	fi

	# Удаление symlink если был создан / Remove symlink if created
	if [ -n "$SITE_NAME" ]; then
		$SSH_CMD "sudo rm -f $NGINX_SITES_ENABLED/$SITE_NAME" 2>/dev/null || true
	fi

	warning "Rollback completed"
}

###############################################################################
# Функция проверки синтаксиса nginx / Nginx syntax check function
###############################################################################
check_nginx_syntax_local() {
	if command -v nginx &>/dev/null; then
		info "Checking nginx syntax locally..."
		if nginx -t -c "$NGINX_CONF_SOURCE" &>/dev/null; then
			success "Local nginx syntax is valid"
			return 0
		else
			warning "Local nginx syntax check failed (nginx binary not in PATH or config is incomplete)"
			return 1
		fi
	else
		info "nginx binary not found locally, skipping local syntax check"
		return 0
	fi
}

check_nginx_syntax_remote() {
	step "Checking nginx syntax on remote server..."

	SSH_CMD="ssh -p $SSH_PORT"
	if [ -n "$SSH_KEY_PATH" ]; then
		SSH_CMD="$SSH_CMD -i $SSH_KEY_PATH"
	fi
	SSH_CMD="$SSH_CMD -o StrictHostKeyChecking=no -o ConnectTimeout=10"
	SSH_CMD="$SSH_CMD $REMOTE_USER@$REMOTE_HOST"

	# Проверка синтаксиса с выводом ошибок / Syntax check with error output
	local syntax_output
	syntax_output=$($SSH_CMD "sudo nginx -t" 2>&1) || true

	if echo "$syntax_output" | grep -q "successful\|syntax is ok\|test is successful"; then
		success "Remote nginx syntax is valid"
		return 0
	else
		warning "Nginx syntax check output:"
		echo "$syntax_output" | sed 's/^/  /'
		echo -e "${RED}❌ Remote nginx syntax check failed${NC}" >&2
		return 1
	fi
}

###############################################################################
# Функция помощи / Help function
###############################################################################
show_help() {
	cat <<EOF
${BOLD}Nginx Configuration Deployment Script${NC}

${BOLD}Использование / Usage:${NC}
    $0 [OPTIONS]

${BOLD}Опции / Options:${NC}
    -h, --host HOST              Remote server hostname or IP (required)
    -u, --user USER              SSH username (required)
    -p, --port PORT              SSH port (default: 22)
    -k, --key PATH               Path to SSH private key
    -s, --source PATH            Local nginx configuration file path (required)
    -d, --dest PATH              Destination path on remote server (required)
    --sites-available PATH       Path to sites-available (default: /etc/nginx/sites-available)
    --sites-enabled PATH         Path to sites-enabled (default: /etc/nginx/sites-enabled)
    -n, --site NAME              Site name for symlink creation (required)
    -b, --backup [true|false]    Backup existing configuration (default: true)
    -a, --auto                   Auto mode (no confirmation prompts)
    --help                       Show this help message

${BOLD}Переменные окружения / Environment variables:${NC}
    REMOTE_HOST                  Remote server hostname or IP
    REMOTE_USER                  SSH username
    SSH_PORT                     SSH port
    SSH_KEY_PATH                 Path to SSH private key
    NGINX_CONF_SOURCE            Local nginx configuration file path
    NGINX_CONF_DEST              Destination path on remote server
    NGINX_SITES_AVAILABLE        Path to sites-available
    NGINX_SITES_ENABLED          Path to sites-enabled
    SITE_NAME                    Site name for symlink creation
    BACKUP_EXISTING              Backup existing configuration (true/false)
    AUTO_MODE                    Auto mode (true/false)

${BOLD}Примеры / Examples:${NC}
    # Using command line arguments
    $0 --host 192.168.1.100 --user deploy \\
       --source nginx.conf --dest /etc/nginx/sites-available/event-api \\
       --site event-api --key ~/.ssh/id_rsa

    # Using environment variables
    REMOTE_HOST=192.168.1.100 REMOTE_USER=deploy \\
    NGINX_CONF_SOURCE=nginx.conf \\
    NGINX_CONF_DEST=/etc/nginx/sites-available/event-api \\
    SITE_NAME=event-api \\
    $0

    # Auto mode (no confirmation)
    $0 --host 192.168.1.100 --user deploy --source nginx.conf \\
       --dest /etc/nginx/sites-available/event-api --site event-api --auto
EOF
}

###############################################################################
# Парсинг аргументов командной строки / Parse command line arguments
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
		-s | --source)
			NGINX_CONF_SOURCE="$2"
			shift 2
			;;
		-d | --dest)
			NGINX_CONF_DEST="$2"
			shift 2
			;;
		--sites-available)
			NGINX_SITES_AVAILABLE="$2"
			shift 2
			;;
		--sites-enabled)
			NGINX_SITES_ENABLED="$2"
			shift 2
			;;
		-n | --site)
			SITE_NAME="$2"
			shift 2
			;;
		-b | --backup)
			BACKUP_EXISTING="$2"
			shift 2
			;;
		-a | --auto)
			AUTO_MODE="true"
			shift
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
# Валидация параметров / Parameter validation
###############################################################################
validate_parameters() {
	step "Validating parameters..."

	# Установка значений по умолчанию / Set default values
	SSH_PORT="${SSH_PORT:-22}"
	SSH_KEY_PATH="${SSH_KEY_PATH-}"
	NGINX_SITES_AVAILABLE="${NGINX_SITES_AVAILABLE:-/etc/nginx/sites-available}"
	NGINX_SITES_ENABLED="${NGINX_SITES_ENABLED:-/etc/nginx/sites-enabled}"
	BACKUP_EXISTING="${BACKUP_EXISTING:-true}"
	AUTO_MODE="${AUTO_MODE:-false}"

	# Проверка обязательных параметров / Check required parameters
	if [ -z "$REMOTE_HOST" ]; then
		error "REMOTE_HOST is required (use -h/--host or set REMOTE_HOST environment variable)"
	fi

	if [ -z "$REMOTE_USER" ]; then
		error "REMOTE_USER is required (use -u/--user or set REMOTE_USER environment variable)"
	fi

	if [ -z "$NGINX_CONF_SOURCE" ]; then
		error "NGINX_CONF_SOURCE is required (use -s/--source or set NGINX_CONF_SOURCE environment variable)"
	fi

	if [ -z "$NGINX_CONF_DEST" ]; then
		error "NGINX_CONF_DEST is required (use -d/--dest or set NGINX_CONF_DEST environment variable)"
	fi

	if [ -z "$SITE_NAME" ]; then
		error "SITE_NAME is required (use -n/--site or set SITE_NAME environment variable)"
	fi

	# Проверка существования локального файла / Check local file existence
	if [ ! -f "$NGINX_CONF_SOURCE" ]; then
		error "Local nginx configuration file not found: $NGINX_CONF_SOURCE"
	fi

	# Проверка SSH ключа если указан / Check SSH key if specified
	if [ -n "$SSH_KEY_PATH" ] && [ ! -f "$SSH_KEY_PATH" ]; then
		error "SSH key file not found: $SSH_KEY_PATH"
	fi

	# Проверка формата порта / Check port format
	if ! [[ $SSH_PORT =~ ^[0-9]+$ ]] || [ "$SSH_PORT" -lt 1 ] || [ "$SSH_PORT" -gt 65535 ]; then
		error "Invalid SSH port: $SSH_PORT (must be between 1 and 65535)"
	fi

	# Проверка формата BACKUP_EXISTING / Check BACKUP_EXISTING format
	if [ "$BACKUP_EXISTING" != "true" ] && [ "$BACKUP_EXISTING" != "false" ]; then
		error "Invalid BACKUP_EXISTING value: $BACKUP_EXISTING (must be 'true' or 'false')"
	fi

	# Проверка формата AUTO_MODE / Check AUTO_MODE format
	if [ "$AUTO_MODE" != "true" ] && [ "$AUTO_MODE" != "false" ]; then
		error "Invalid AUTO_MODE value: $AUTO_MODE (must be 'true' or 'false')"
	fi

	success "All parameters validated"
}

###############################################################################
# Проверка подключения к серверу / Server connection check
###############################################################################
check_server_connection() {
	step "Checking server connection..."

	SSH_CMD="ssh -p $SSH_PORT"
	if [ -n "$SSH_KEY_PATH" ]; then
		SSH_CMD="$SSH_CMD -i $SSH_KEY_PATH"
	fi
	SSH_CMD="$SSH_CMD -o StrictHostKeyChecking=no -o ConnectTimeout=10"
	SSH_CMD="$SSH_CMD -o BatchMode=yes"
	SSH_CMD="$SSH_CMD $REMOTE_USER@$REMOTE_HOST"

	# Проверка подключения / Connection check
	if $SSH_CMD "echo 'Connection successful'" &>/dev/null; then
		success "Server connection established"
	else
		error "Failed to connect to server $REMOTE_USER@$REMOTE_HOST:$SSH_PORT"
	fi

	# Проверка наличия nginx / Check nginx availability
	if $SSH_CMD "command -v nginx &> /dev/null" &>/dev/null ||
		$SSH_CMD "command -v sudo &> /dev/null && sudo command -v nginx &> /dev/null" &>/dev/null; then
		success "nginx is available on remote server"
	else
		error "nginx is not available on remote server"
	fi

	# Проверка прав sudo / Check sudo privileges
	if $SSH_CMD "sudo -n true" &>/dev/null; then
		success "sudo privileges confirmed"
	else
		warning "sudo password may be required (ensure passwordless sudo or SSH key with sudo access)"
	fi
}

###############################################################################
# Создание бэкапа существующей конфигурации / Backup existing configuration
###############################################################################
backup_existing_config() {
	if [ "$BACKUP_EXISTING" != "true" ]; then
		info "Backup is disabled, skipping..."
		return 0
	fi

	step "Creating backup of existing configuration..."

	SSH_CMD="ssh -p $SSH_PORT"
	if [ -n "$SSH_KEY_PATH" ]; then
		SSH_CMD="$SSH_CMD -i $SSH_KEY_PATH"
	fi
	SSH_CMD="$SSH_CMD -o StrictHostKeyChecking=no -o ConnectTimeout=10"
	SSH_CMD="$SSH_CMD $REMOTE_USER@$REMOTE_HOST"

	# Проверка существования файла / Check if file exists
	if $SSH_CMD "test -f $NGINX_CONF_DEST" &>/dev/null; then
		TIMESTAMP=$(date +%Y%m%d_%H%M%S)
		BACKUP_FILE="${NGINX_CONF_DEST}.backup.${TIMESTAMP}"

		if $SSH_CMD "sudo cp $NGINX_CONF_DEST $BACKUP_FILE" &>/dev/null; then
			success "Backup created: $BACKUP_FILE"
			ROLLBACK_ENABLED=true
		else
			echo -e "${RED}❌ Failed to create backup${NC}" >&2
			return 1
		fi
	else
		info "Configuration file does not exist yet, no backup needed"
	fi
}

###############################################################################
# Копирование конфигурации на сервер / Copy configuration to server
###############################################################################
copy_configuration() {
	step "Copying configuration to server..."

	SSH_CMD="ssh -p $SSH_PORT"
	if [ -n "$SSH_KEY_PATH" ]; then
		SSH_CMD="$SSH_CMD -i $SSH_KEY_PATH"
	fi
	SSH_CMD="$SSH_CMD -o StrictHostKeyChecking=no -o ConnectTimeout=10"
	SSH_CMD="$SSH_CMD $REMOTE_USER@$REMOTE_HOST"

	# Создание директории назначения если не существует / Create destination directory if it doesn't exist
	DEST_DIR=$(dirname "$NGINX_CONF_DEST")
	if ! $SSH_CMD "sudo test -d $DEST_DIR" &>/dev/null; then
		info "Creating destination directory: $DEST_DIR"
		if ! $SSH_CMD "sudo mkdir -p $DEST_DIR" &>/dev/null; then
			echo -e "${RED}❌ Failed to create destination directory${NC}" >&2
			return 1
		fi
	fi

	# Создание временного файла на сервере / Create temporary file on server
	TEMP_REMOTE_FILE="/tmp/nginx_conf_$(date +%s).conf"

	# Метод 1: Копирование через SCP во временный файл / Method 1: Copy via SCP to temp file
	SCP_CMD="scp -P $SSH_PORT"
	if [ -n "$SSH_KEY_PATH" ]; then
		SCP_CMD="$SCP_CMD -i $SSH_KEY_PATH"
	fi
	SCP_CMD="$SCP_CMD -o StrictHostKeyChecking=no -o ConnectTimeout=10"
	SCP_CMD="$SCP_CMD $NGINX_CONF_SOURCE $REMOTE_USER@$REMOTE_HOST:$TEMP_REMOTE_FILE"

	if $SCP_CMD &>/dev/null; then
		# Перемещение файла в нужное место с правами / Move file to destination with proper permissions
		if $SSH_CMD "sudo mv $TEMP_REMOTE_FILE $NGINX_CONF_DEST && sudo chmod 644 $NGINX_CONF_DEST && sudo chown root:root $NGINX_CONF_DEST" &>/dev/null; then
			success "Configuration copied to $NGINX_CONF_DEST"
			ROLLBACK_ENABLED=true
			return 0
		else
			# Очистка временного файла при ошибке / Cleanup temp file on error
			$SSH_CMD "rm -f $TEMP_REMOTE_FILE" &>/dev/null || true
		fi
	fi

	# Метод 2: Копирование через stdin (более надежный) / Method 2: Copy via stdin (more reliable)
	if cat "$NGINX_CONF_SOURCE" | $SSH_CMD "sudo tee $NGINX_CONF_DEST > /dev/null && sudo chmod 644 $NGINX_CONF_DEST && sudo chown root:root $NGINX_CONF_DEST" &>/dev/null; then
		success "Configuration copied to $NGINX_CONF_DEST"
		ROLLBACK_ENABLED=true
	else
		echo -e "${RED}❌ Failed to copy configuration file${NC}" >&2
		return 1
	fi
}

###############################################################################
# Активация конфигурации (создание symlink) / Activate configuration (create symlink)
###############################################################################
activate_configuration() {
	step "Activating configuration (creating symlink)..."

	SSH_CMD="ssh -p $SSH_PORT"
	if [ -n "$SSH_KEY_PATH" ]; then
		SSH_CMD="$SSH_CMD -i $SSH_KEY_PATH"
	fi
	SSH_CMD="$SSH_CMD -o StrictHostKeyChecking=no -o ConnectTimeout=10"
	SSH_CMD="$SSH_CMD $REMOTE_USER@$REMOTE_HOST"

	SYMLINK_PATH="$NGINX_SITES_ENABLED/$SITE_NAME"
	TARGET_PATH="$NGINX_CONF_DEST"

	# Проверка существования директорий / Check directory existence
	if ! $SSH_CMD "sudo test -d $NGINX_SITES_AVAILABLE" &>/dev/null; then
		info "Creating sites-available directory..."
		if ! $SSH_CMD "sudo mkdir -p $NGINX_SITES_AVAILABLE" &>/dev/null; then
			echo -e "${RED}❌ Failed to create sites-available directory${NC}" >&2
			return 1
		fi
	fi

	if ! $SSH_CMD "sudo test -d $NGINX_SITES_ENABLED" &>/dev/null; then
		info "Creating sites-enabled directory..."
		if ! $SSH_CMD "sudo mkdir -p $NGINX_SITES_ENABLED" &>/dev/null; then
			echo -e "${RED}❌ Failed to create sites-enabled directory${NC}" >&2
			return 1
		fi
	fi

	# Удаление существующего symlink если есть / Remove existing symlink if exists
	if $SSH_CMD "sudo test -L $SYMLINK_PATH" &>/dev/null; then
		info "Removing existing symlink..."
		$SSH_CMD "sudo rm -f $SYMLINK_PATH" &>/dev/null ||
			warning "Failed to remove existing symlink (may not exist)"
	fi

	# Создание symlink / Create symlink
	if $SSH_CMD "sudo ln -s $TARGET_PATH $SYMLINK_PATH" &>/dev/null; then
		success "Symlink created: $SYMLINK_PATH -> $TARGET_PATH"
	else
		echo -e "${RED}❌ Failed to create symlink${NC}" >&2
		return 1
	fi
}

###############################################################################
# Перезагрузка nginx / Reload nginx
###############################################################################
reload_nginx() {
	step "Reloading nginx..."

	SSH_CMD="ssh -p $SSH_PORT"
	if [ -n "$SSH_KEY_PATH" ]; then
		SSH_CMD="$SSH_CMD -i $SSH_KEY_PATH"
	fi
	SSH_CMD="$SSH_CMD -o StrictHostKeyChecking=no -o ConnectTimeout=10"
	SSH_CMD="$SSH_CMD $REMOTE_USER@$REMOTE_HOST"

	# Проверка синтаксиса перед reload / Syntax check before reload
	if ! check_nginx_syntax_remote; then
		echo -e "${RED}❌ Nginx syntax check failed, aborting reload${NC}" >&2
		return 1
	fi

	# Перезагрузка nginx / Reload nginx
	if $SSH_CMD "sudo systemctl reload nginx" &>/dev/null ||
		$SSH_CMD "sudo service nginx reload" &>/dev/null ||
		$SSH_CMD "sudo nginx -s reload" &>/dev/null; then
		success "Nginx reloaded successfully"
	else
		echo -e "${RED}❌ Failed to reload nginx${NC}" >&2
		return 1
	fi

	# Проверка статуса nginx / Check nginx status
	sleep 2
	if $SSH_CMD "sudo systemctl is-active nginx &> /dev/null" &>/dev/null ||
		$SSH_CMD "sudo service nginx status &> /dev/null" &>/dev/null ||
		$SSH_CMD "pgrep nginx &> /dev/null" &>/dev/null; then
		success "Nginx is running"
	else
		warning "Nginx status check inconclusive (service may use different method)"
	fi
}

###############################################################################
# Подтверждение действий пользователем / User confirmation
###############################################################################
confirm_deployment() {
	if [ "$AUTO_MODE" = "true" ]; then
		info "Auto mode enabled, skipping confirmation"
		return 0
	fi

	echo ""
	echo -e "${BOLD}${CYAN}Deployment Summary:${NC}"
	echo "  Remote Host:        $REMOTE_USER@$REMOTE_HOST:$SSH_PORT"
	echo "  Source File:        $NGINX_CONF_SOURCE"
	echo "  Destination:        $NGINX_CONF_DEST"
	echo "  Site Name:          $SITE_NAME"
	echo "  Symlink:            $NGINX_SITES_ENABLED/$SITE_NAME"
	echo "  Backup Existing:    $BACKUP_EXISTING"
	echo ""

	read -p "Do you want to continue? (yes/N) " -r
	echo
	if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
		info "Deployment cancelled by user"
		exit 0
	fi
}

###############################################################################
# Главная функция / Main function
###############################################################################
main() {
	echo -e "${BOLD}${BLUE}"
	echo "╔══════════════════════════════════════════════════════════════╗"
	echo "║     Nginx Configuration Deployment Script                    ║"
	echo "╚══════════════════════════════════════════════════════════════╝"
	echo -e "${NC}"
	echo ""

	# Парсинг аргументов / Parse arguments
	parse_arguments "$@"

	# Валидация параметров / Validate parameters
	validate_parameters

	# Подтверждение / Confirmation
	confirm_deployment

	# Проверка синтаксиса локально (если возможно) / Local syntax check (if possible)
	check_nginx_syntax_local || true

	# Проверка подключения к серверу / Check server connection
	check_server_connection

	# Установка обработчика ошибок для отката / Set error handler for rollback
	trap 'rollback; echo -e "\033[0;31m❌ Deployment failed - rollback completed\033[0m" >&2; exit 1' ERR

	# Создание бэкапа / Create backup
	backup_existing_config

	# Копирование конфигурации / Copy configuration
	copy_configuration

	# Активация конфигурации / Activate configuration
	activate_configuration

	# Перезагрузка nginx / Reload nginx
	reload_nginx

	# Отключение отката после успешного завершения / Disable rollback after successful completion
	trap - ERR
	ROLLBACK_ENABLED=false

	echo ""
	success "Deployment completed successfully!"
	echo ""
	info "Configuration: $NGINX_CONF_DEST"
	info "Symlink: $NGINX_SITES_ENABLED/$SITE_NAME"
	if [ -n "$BACKUP_FILE" ]; then
		info "Backup: $BACKUP_FILE"
	fi
	echo ""
}

###############################################################################
# Запуск скрипта / Script execution
###############################################################################
main "$@"
