#!/bin/bash

# Скрипт для деплоя приложения на production сервер
# Использование: ./scripts/deploy.sh [environment]

set -e

ENVIRONMENT=${1:-production}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "🚀 Deploying to $ENVIRONMENT..."

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Функция для вывода ошибок
error() {
    echo -e "${RED}❌ Error: $1${NC}" >&2
    exit 1
}

# Функция для вывода успеха
success() {
    echo -e "${GREEN}✅ $1${NC}"
}

# Функция для вывода предупреждений
warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

# Проверка окружения
case $ENVIRONMENT in
    production|prod)
        ENV_FILE=".env.production"
        COMPOSE_FILE="docker-compose.prod.yml"
        ;;
    staging)
        ENV_FILE=".env.staging"
        COMPOSE_FILE="docker-compose.prod.yml"
        ;;
    *)
        error "Unknown environment: $ENVIRONMENT. Use 'production' or 'staging'"
        ;;
esac

# Проверка наличия .env файла
if [ ! -f "$PROJECT_DIR/$ENV_FILE" ]; then
    warning "$ENV_FILE not found. Using .env"
    ENV_FILE=".env"
fi

cd "$PROJECT_DIR"

# Проверка Git статуса
if [ -n "$(git status --porcelain)" ]; then
    warning "Working directory is not clean. Uncommitted changes detected."
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Получение текущей ветки и коммита
BRANCH=$(git rev-parse --abbrev-ref HEAD)
COMMIT=$(git rev-parse --short HEAD)

echo "📋 Deployment Info:"
echo "   Environment: $ENVIRONMENT"
echo "   Branch: $BRANCH"
echo "   Commit: $COMMIT"
echo ""

# Подтверждение деплоя
if [ "$ENVIRONMENT" = "production" ] || [ "$ENVIRONMENT" = "prod" ]; then
    warning "You are about to deploy to PRODUCTION!"
    read -p "Are you sure? (yes/N) " -r
    echo
    if [[ ! $REPLY = "yes" ]]; then
        echo "Deployment cancelled."
        exit 0
    fi
fi

# Создание бэкапа базы данных (если на том же сервере)
if command -v docker-compose &> /dev/null; then
    echo "📦 Creating database backup..."
    BACKUP_DIR="$PROJECT_DIR/backups"
    mkdir -p "$BACKUP_DIR"
    BACKUP_FILE="$BACKUP_DIR/db_backup_$(date +%Y%m%d_%H%M%S).sql"
    
    if docker-compose ps | grep -q postgres; then
        docker-compose exec -T postgres pg_dump -U postgres event_api > "$BACKUP_FILE" 2>/dev/null || true
        if [ -f "$BACKUP_FILE" ]; then
            success "Database backup created: $BACKUP_FILE"
        fi
    fi
fi

# Pull последних изменений
echo "🔄 Pulling latest changes..."
git pull origin "$BRANCH" || error "Failed to pull latest changes"

# Сборка Docker образа
echo "🔨 Building Docker image..."
docker build -t event-api:latest -t event-api:$COMMIT . || error "Failed to build Docker image"
success "Docker image built successfully"

# Загрузка переменных окружения
if [ -f "$ENV_FILE" ]; then
    export $(cat "$ENV_FILE" | grep -v '^#' | xargs)
fi

# Остановка старых контейнеров
echo "🛑 Stopping old containers..."
docker-compose -f "$COMPOSE_FILE" down

# Запуск новых контейнеров
echo "🚀 Starting new containers..."
docker-compose -f "$COMPOSE_FILE" up -d || error "Failed to start containers"

# Ожидание запуска приложения
echo "⏳ Waiting for application to start..."
RETRY_COUNT=0
MAX_RETRIES=30

until curl -f http://localhost:8080/v1/api/health &> /dev/null; do
    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $RETRY_COUNT -ge $MAX_RETRIES ]; then
        error "Application failed to start within expected time"
    fi
    echo "   Attempt $RETRY_COUNT/$MAX_RETRIES..."
    sleep 2
done

success "Application is running!"

# Проверка состояния контейнеров
echo ""
echo "📊 Container Status:"
docker-compose -f "$COMPOSE_FILE" ps

# Вывод логов
echo ""
echo "📝 Recent logs:"
docker-compose -f "$COMPOSE_FILE" logs --tail=20 app

# Очистка старых образов
echo ""
echo "🧹 Cleaning up old images..."
docker image prune -f

success "Deployment completed successfully!"
echo ""
echo "🌍 Application is available at: http://localhost:8080"
echo "📊 Health check: http://localhost:8080/v1/api/health"
echo ""
echo "To view logs: docker-compose -f $COMPOSE_FILE logs -f app"
echo "To stop: docker-compose -f $COMPOSE_FILE down"
