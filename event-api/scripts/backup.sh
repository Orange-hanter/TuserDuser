#!/bin/bash

# Скрипт для создания бэкапа базы данных PostgreSQL
# Использование: ./scripts/backup.sh [database_name]

set -e

DB_NAME=${1:-event_api}
BACKUP_DIR="./backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/${DB_NAME}_backup_${TIMESTAMP}.sql"

# Цвета для вывода
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo "📦 Creating database backup..."

# Создание директории для бэкапов
mkdir -p "$BACKUP_DIR"

# Проверка наличия Docker контейнера с PostgreSQL
if ! docker-compose ps postgres | grep -q "Up"; then
    echo -e "${RED}❌ PostgreSQL container is not running${NC}"
    exit 1
fi

# Создание бэкапа
docker-compose exec -T postgres pg_dump -U postgres -d "$DB_NAME" > "$BACKUP_FILE"

if [ $? -eq 0 ]; then
    # Сжатие бэкапа
    gzip "$BACKUP_FILE"
    BACKUP_FILE="${BACKUP_FILE}.gz"
    
    FILE_SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
    echo -e "${GREEN}✅ Backup created successfully${NC}"
    echo "   File: $BACKUP_FILE"
    echo "   Size: $FILE_SIZE"
    
    # Удаление старых бэкапов (хранить последние 7 дней)
    find "$BACKUP_DIR" -name "${DB_NAME}_backup_*.sql.gz" -mtime +7 -delete
    echo "   Old backups (>7 days) cleaned up"
else
    echo -e "${RED}❌ Backup failed${NC}"
    rm -f "$BACKUP_FILE"
    exit 1
fi
