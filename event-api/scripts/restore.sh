#!/bin/bash

# Скрипт для восстановления базы данных из бэкапа
# Использование: ./scripts/restore.sh <backup_file> [database_name]

set -e

BACKUP_FILE=$1
DB_NAME=${2:-event_api}

# Цвета для вывода
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

if [ -z "$BACKUP_FILE" ]; then
    echo -e "${RED}❌ Usage: ./scripts/restore.sh <backup_file> [database_name]${NC}"
    echo ""
    echo "Available backups:"
    ls -lh backups/*.sql.gz 2>/dev/null || echo "No backups found"
    exit 1
fi

if [ ! -f "$BACKUP_FILE" ]; then
    echo -e "${RED}❌ Backup file not found: $BACKUP_FILE${NC}"
    exit 1
fi

echo -e "${YELLOW}⚠️  WARNING: This will OVERWRITE the current database!${NC}"
echo "   Database: $DB_NAME"
echo "   Backup: $BACKUP_FILE"
read -p "Are you sure? (yes/N) " -r
echo

if [[ ! $REPLY = "yes" ]]; then
    echo "Restore cancelled."
    exit 0
fi

# Проверка наличия Docker контейнера с PostgreSQL
if ! docker-compose ps postgres | grep -q "Up"; then
    echo -e "${RED}❌ PostgreSQL container is not running${NC}"
    exit 1
fi

echo "🔄 Restoring database..."

# Если файл сжат, распаковываем его
if [[ $BACKUP_FILE == *.gz ]]; then
    echo "📦 Decompressing backup..."
    gunzip -c "$BACKUP_FILE" | docker-compose exec -T postgres psql -U postgres -d "$DB_NAME"
else
    docker-compose exec -T postgres psql -U postgres -d "$DB_NAME" < "$BACKUP_FILE"
fi

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Database restored successfully${NC}"
else
    echo -e "${RED}❌ Restore failed${NC}"
    exit 1
fi
