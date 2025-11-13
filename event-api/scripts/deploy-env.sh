#!/bin/bash

# deploy-env.sh
# Скрипт для обновления .env конфигурации на удаленном сервере из локального источника
#
# Использование:
#   ./scripts/deploy-env.sh [опции]
#
# Опции:
#   -h, --host HOST       IP адрес или домен удаленного сервера (по умолчанию: 5.35.127.138)
#   -u, --user USER       Пользователь для SSH (по умолчанию: root)
#   -k, --key KEY         Путь к SSH ключу (по умолчанию: ~/.ssh/deployer_eventapi_root)
#   -s, --source SOURCE   Путь к локальному .env файлу (по умолчанию: .env)
#   -d, --dest DEST       Путь к .env на удаленном сервере (по умолчанию: /root/event-api/.env)
#   -b, --backup          Создать резервную копию перед обновлением (по умолчанию: да)
#   --no-backup           Не создавать резервную копию
#   -r, --restart         Перезапустить сервис после обновления (по умолчанию: да)
#   --no-restart          Не перезапускать сервис
#   --service NAME        Имя systemd сервиса (по умолчанию: eventapi)
#   --dry-run             Показать что будет сделано без выполнения
#   --help                Показать справку

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Значения по умолчанию
REMOTE_HOST="5.35.127.138"
REMOTE_USER="root"
SSH_KEY="$HOME/.ssh/deployer_eventapi_root"
SOURCE_ENV=".env"
DEST_ENV="/root/event-api/.env"
CREATE_BACKUP=true
RESTART_SERVICE=true
SERVICE_NAME="eventapi"
DRY_RUN=false

# Функция для вывода сообщений
log_info() {
	echo -e "${BLUE}ℹ${NC} $1"
}

log_success() {
	echo -e "${GREEN}✓${NC} $1"
}

log_warning() {
	echo -e "${YELLOW}⚠${NC} $1"
}

log_error() {
	echo -e "${RED}✗${NC} $1"
}

# Функция для показа справки
show_help() {
	cat <<EOF
Скрипт для обновления .env конфигурации на удаленном сервере

Использование:
    $0 [опции]

Опции:
    -h, --host HOST       IP адрес или домен удаленного сервера
                          (по умолчанию: $REMOTE_HOST)
    -u, --user USER       Пользователь для SSH (по умолчанию: $REMOTE_USER)
    -k, --key KEY         Путь к SSH ключу
                          (по умолчанию: $SSH_KEY)
    -s, --source SOURCE   Путь к локальному .env файлу
                          (по умолчанию: $SOURCE_ENV)
    -d, --dest DEST       Путь к .env на удаленном сервере
                          (по умолчанию: $DEST_ENV)
    -b, --backup          Создать резервную копию перед обновлением (по умолчанию)
    --no-backup           Не создавать резервную копию
    -r, --restart         Перезапустить сервис после обновления (по умолчанию)
    --no-restart          Не перезапускать сервис
    --service NAME        Имя systemd сервиса (по умолчанию: $SERVICE_NAME)
    --dry-run             Показать что будет сделано без выполнения
    --help                Показать эту справку

Примеры:
    # Обновить .env на сервере с резервной копией и перезапуском
    $0

    # Обновить из другого файла
    $0 -s .env.production

    # Обновить без перезапуска сервиса
    $0 --no-restart

    # Проверить что будет сделано
    $0 --dry-run

    # Обновить на другом сервере
    $0 -h 192.168.1.100 -u deploy -k ~/.ssh/id_rsa

EOF
}

# Парсинг аргументов командной строки
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
	-k | --key)
		SSH_KEY="$2"
		shift 2
		;;
	-s | --source)
		SOURCE_ENV="$2"
		shift 2
		;;
	-d | --dest)
		DEST_ENV="$2"
		shift 2
		;;
	-b | --backup)
		CREATE_BACKUP=true
		shift
		;;
	--no-backup)
		CREATE_BACKUP=false
		shift
		;;
	-r | --restart)
		RESTART_SERVICE=true
		shift
		;;
	--no-restart)
		RESTART_SERVICE=false
		shift
		;;
	--service)
		SERVICE_NAME="$2"
		shift 2
		;;
	--dry-run)
		DRY_RUN=true
		shift
		;;
	--help)
		show_help
		exit 0
		;;
	*)
		log_error "Неизвестный параметр: $1"
		echo "Используйте --help для справки"
		exit 1
		;;
	esac
done

# Функция для выполнения команды или показа в dry-run режиме
execute_or_show() {
	local description="$1"
	local command="$2"

	if [ "$DRY_RUN" = true ]; then
		log_info "[DRY-RUN] $description"
		echo "  Команда: $command"
	else
		log_info "$description"
		eval "$command"
	fi
}

# Функция для выполнения SSH команды
ssh_execute() {
	local command="$1"
	ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no -o ConnectTimeout=10 \
		"${REMOTE_USER}@${REMOTE_HOST}" "$command"
}

# Баннер
echo ""
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║         Обновление .env конфигурации на сервере                ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

if [ "$DRY_RUN" = true ]; then
	log_warning "РЕЖИМ ПРОВЕРКИ (DRY-RUN) - команды не будут выполнены"
	echo ""
fi

# Шаг 1: Проверка локального файла
log_info "Проверка локального .env файла..."

if [ ! -f "$SOURCE_ENV" ]; then
	log_error "Локальный файл не найден: $SOURCE_ENV"
	exit 1
fi

log_success "Локальный файл найден: $SOURCE_ENV"
local_lines=$(wc -l <"$SOURCE_ENV" | tr -d ' ')
log_info "  Размер: $(du -h "$SOURCE_ENV" | cut -f1)"
log_info "  Строк: $local_lines"

# Показываем некоторые переменные (без значений для безопасности)
log_info "  Переменные (обнаружено):"
grep -v '^#' "$SOURCE_ENV" | grep '=' | cut -d'=' -f1 | head -10 | while read var; do
	echo "    - $var"
done
if [ "$local_lines" -gt 10 ]; then
	echo "    ... и еще $((local_lines - 10))"
fi
echo ""

# Шаг 2: Проверка SSH ключа
log_info "Проверка SSH ключа..."

if [ ! -f "$SSH_KEY" ]; then
	log_error "SSH ключ не найден: $SSH_KEY"
	exit 1
fi

log_success "SSH ключ найден: $SSH_KEY"
echo ""

# Шаг 3: Проверка подключения к серверу
log_info "Проверка подключения к серверу..."
log_info "  Сервер: ${REMOTE_USER}@${REMOTE_HOST}"

if [ "$DRY_RUN" = false ]; then
	if ! ssh_execute "echo 'Connection successful'" >/dev/null 2>&1; then
		log_error "Не удалось подключиться к серверу"
		log_error "Проверьте:"
		log_error "  - IP адрес/домен сервера: $REMOTE_HOST"
		log_error "  - Пользователя: $REMOTE_USER"
		log_error "  - SSH ключ: $SSH_KEY"
		exit 1
	fi
	log_success "Подключение установлено"
else
	log_info "[DRY-RUN] Проверка подключения пропущена"
fi
echo ""

# Шаг 4: Проверка существующего файла на сервере
log_info "Проверка существующего .env на сервере..."

if [ "$DRY_RUN" = false ]; then
	if ssh_execute "test -f $DEST_ENV" 2>/dev/null; then
		log_success "Файл существует: $DEST_ENV"
		remote_lines=$(ssh_execute "wc -l < $DEST_ENV 2>/dev/null | tr -d ' '")
		log_info "  Строк: $remote_lines"
	else
		log_warning "Файл не существует на сервере (будет создан)"
		CREATE_BACKUP=false
	fi
else
	log_info "[DRY-RUN] Проверка существующего файла пропущена"
fi
echo ""

# Шаг 5: Создание резервной копии
if [ "$CREATE_BACKUP" = true ]; then
	BACKUP_FILE="${DEST_ENV}.backup.$(date +%Y%m%d_%H%M%S)"
	execute_or_show "Создание резервной копии..." \
		"ssh_execute 'cp $DEST_ENV $BACKUP_FILE'"

	if [ "$DRY_RUN" = false ]; then
		log_success "Резервная копия создана: $BACKUP_FILE"
	fi
	echo ""
fi

# Шаг 6: Загрузка нового .env файла
execute_or_show "Загрузка нового .env файла на сервер..." \
	"scp -i '$SSH_KEY' -o StrictHostKeyChecking=no '$SOURCE_ENV' '${REMOTE_USER}@${REMOTE_HOST}:${DEST_ENV}'"

if [ "$DRY_RUN" = false ]; then
	log_success "Файл успешно загружен"
fi
echo ""

# Шаг 7: Проверка файла на сервере
if [ "$DRY_RUN" = false ]; then
	log_info "Проверка загруженного файла..."
	new_remote_lines=$(ssh_execute "wc -l < $DEST_ENV 2>/dev/null | tr -d ' '")
	log_info "  Строк на сервере: $new_remote_lines"
	log_info "  Строк в локальном: $local_lines"

	if [ "$new_remote_lines" = "$local_lines" ]; then
		log_success "Количество строк совпадает"
	else
		log_warning "Количество строк отличается (может быть нормально)"
	fi
	echo ""
fi

# Шаг 8: Перезапуск сервиса
if [ "$RESTART_SERVICE" = true ]; then
	log_info "Перезапуск сервиса $SERVICE_NAME..."

	if [ "$DRY_RUN" = false ]; then
		# Проверяем существует ли сервис
		if ssh_execute "systemctl list-units --full --all | grep -q $SERVICE_NAME.service" 2>/dev/null; then
			execute_or_show "Остановка сервиса..." \
				"ssh_execute 'sudo systemctl stop $SERVICE_NAME'"

			sleep 2

			execute_or_show "Запуск сервиса..." \
				"ssh_execute 'sudo systemctl start $SERVICE_NAME'"

			sleep 2

			# Проверка статуса
			if ssh_execute "sudo systemctl is-active --quiet $SERVICE_NAME"; then
				log_success "Сервис успешно перезапущен"

				# Показываем последние логи
				log_info "Последние логи сервиса:"
				ssh_execute "sudo journalctl -u $SERVICE_NAME -n 5 --no-pager" || true
			else
				log_error "Сервис не запущен!"
				log_error "Проверьте логи: sudo journalctl -u $SERVICE_NAME -n 50"

				if [ "$CREATE_BACKUP" = true ]; then
					log_warning "Восстановление из резервной копии..."
					ssh_execute "cp $BACKUP_FILE $DEST_ENV"
					ssh_execute "sudo systemctl start $SERVICE_NAME"
					log_info "Файл восстановлен из резервной копии"
				fi
				exit 1
			fi
		else
			log_warning "Сервис $SERVICE_NAME не найден в systemd"
			log_info "Пропускаем перезапуск"
		fi
	else
		log_info "[DRY-RUN] Перезапуск сервиса $SERVICE_NAME"
	fi
	echo ""
else
	log_info "Перезапуск сервиса отключен (используйте -r для включения)"
	echo ""
fi

# Шаг 9: Итоговая информация
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║                   Развертывание завершено                      ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

if [ "$DRY_RUN" = true ]; then
	log_warning "Это был РЕЖИМ ПРОВЕРКИ - изменения не применены"
	log_info "Запустите без --dry-run для реального развертывания"
else
	log_success "Конфигурация успешно обновлена на сервере"

	if [ "$CREATE_BACKUP" = true ]; then
		log_info "Резервная копия: $BACKUP_FILE"
	fi

	log_info "Для проверки конфигурации выполните:"
	echo "  ssh -i $SSH_KEY ${REMOTE_USER}@${REMOTE_HOST} 'cat $DEST_ENV'"

	if [ "$RESTART_SERVICE" = true ]; then
		log_info "Для просмотра логов выполните:"
		echo "  ssh -i $SSH_KEY ${REMOTE_USER}@${REMOTE_HOST} 'sudo journalctl -u $SERVICE_NAME -f'"
	fi
fi

echo ""
log_success "Готово!"
