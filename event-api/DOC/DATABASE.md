# Database Infrastructure

## 📋 Описание

Event API использует PostgreSQL как основную базу данных. Инфраструктура включает:

- **PostgreSQL** - основная БД
- **Docker Compose** - для простого запуска контейнеров
- **Migrations System** - встроенная система миграций
- **pgAdmin** - веб-интерфейс для управления БД (опционально)

---

## 🚀 Быстрый старт

### 1. Запуск PostgreSQL с Docker Compose

```bash
cd /Users/dakh/Git/TuserDuser/event-api

# Запуск контейнеров
docker-compose up -d

# Проверка статуса
docker-compose ps
```

### 2. Переменные окружения

Добавьте в `.env` файл:

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=event_api
DB_SSLMODE=disable
```

### 3. Запуск приложения

```bash
make run
```

Приложение автоматически:
1. Подключится к БД
2. Создаст таблицу `schema_migrations`
3. Выполнит все миграции
4. Запустит сервер

---

## 📦 Архитектура

### Database Package (`internal/database/db.go`)

```go
// Подключение к БД
db, err := database.NewDatabase(dbConfig, logger)

// Методы доступа
rows, err := db.Query(query, args...)
row := db.QueryRow(query, args...)
result, err := db.Exec(query, args...)

// Транзакции
tx, err := db.BeginTx()
```

**Особенности:**
- Пул подключений (по умолчанию: 25 макс, 5 минимум)
- Параметризованные запросы (защита от SQL инъекций)
- Health checks
- Логирование запросов

### Migrations Package (`internal/migrations/migrations.go`)

```go
// Инициализация
migrator := migrations.NewMigrator(db.DB, logger)

// Запуск миграций
migrator.RunMigrations()

// Откат (только для разработки)
migrator.Rollback()
```

**Особенности:**
- Таблица отслеживания миграций: `schema_migrations`
- Транзакционные миграции
- Логирование каждой миграции
- Защита от двойного запуска

---

## 🗄️ Структура БД

### Таблица: `users`

```sql
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email VARCHAR(255) UNIQUE NOT NULL,
  phone VARCHAR(20) NOT NULL,
  password VARCHAR(255) NOT NULL,
  verified BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Индексы
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_verified ON users(verified);
```

### Таблица: `schema_migrations`

Служебная таблица для отслеживания примененных миграций:

```sql
CREATE TABLE schema_migrations (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) UNIQUE NOT NULL,
  applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

---

## 📝 Работа с миграциями

### Добавление новой миграции

1. Откройте `internal/migrations/migrations.go`
2. Добавьте новую миграцию в список:

```go
migrations := []migration{
  {
    name: "001_create_users_table",
    up:   createUsersTable,
    down: dropUsersTable,
  },
  {
    name: "002_create_events_table",  // ← Новая
    up:   createEventsTable,          // ← SQL для создания
    down: dropEventsTable,            // ← SQL для удаления
  },
}
```

3. Добавьте SQL константы:

```go
const createEventsTable = `
CREATE TABLE IF NOT EXISTS events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  title VARCHAR(255) NOT NULL,
  description TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`

const dropEventsTable = `DROP TABLE IF EXISTS events CASCADE;`
```

4. Пересоберите и запустите приложение:

```bash
make run
```

---

## 🔧 Docker Compose

### Services

**PostgreSQL**
- Image: `postgres:15-alpine`
- Port: 5432
- Volume: `postgres_data`
- Health check: каждые 10 секунд

**pgAdmin** (опционально)
- Image: `dpage/pgadmin4:latest`
- Port: 5050
- Доступ: `http://localhost:5050`

### Команды

```bash
# Запуск контейнеров
docker-compose up -d

# Остановка контейнеров
docker-compose down

# Просмотр логов
docker-compose logs -f postgres

# Удаление данных
docker-compose down -v

# Пересоздание контейнеров
docker-compose up -d --force-recreate
```

---

## 📊 Подключение через SQL Client

### psql (коммандная строка)

```bash
psql -h localhost -p 5432 -U postgres -d event_api
```

### Connection String

```
postgres://postgres:postgres@localhost:5432/event_api?sslmode=disable
```

### DBeaver / DataGrip

```
Host:     localhost
Port:     5432
Database: event_api
User:     postgres
Password: postgres
```

### pgAdmin (веб)

1. Откройте http://localhost:5050
2. Email: `admin@example.com`
3. Password: `admin`
4. Добавьте сервер:
   - Host: `postgres` (имя сервиса в docker-compose)
   - Port: `5432`
   - Username: `postgres`
   - Password: `postgres`

---

## 🛠️ SQL Скрипты

### `scripts/init.sql`

Выполняется при создании контейнера:
- Создает расширения (uuid-ossp, pgcrypto)
- Создает пользователя приложения
- Инициализирует таблицу версий БД

### `scripts/test_data.sql`

Тестовые данные:
- 3 тестовых пользователя
- Структура примеров для разработки

Запуск:
```bash
docker-compose exec postgres psql -U postgres -d event_api -f /docker-entrypoint-initdb.d/test_data.sql
```

---

## 🔍 Мониторинг и отладка

### Проверка здоровья БД

Приложение имеет встроенный health check:

```go
if err := db.Health(); err != nil {
    logger.Error("Database health check failed", zap.Error(err))
}
```

### Логирование SQL запросов

В `internal/database/db.go` используется:

```go
d.Logger.Debug("Выполняем запрос", zap.String("query", query))
```

Для полного логирования установите `LOG_LEVEL=debug`

### Просмотр логов контейнера

```bash
# PostgreSQL логи
docker-compose logs -f postgres

# Последние 100 строк
docker-compose logs --tail=100 postgres
```

---

## 🔒 Безопасность

### Защита от SQL инъекций

Используются параметризованные запросы:

```go
// ✅ Правильно
db.Query("SELECT * FROM users WHERE email = $1", email)

// ❌ Неправильно
db.Query("SELECT * FROM users WHERE email = '" + email + "'")
```

### Пароли БД

- **Development**: используются default значения из .env
- **Production**: используйте vault или secrets manager

```bash
# Генерация сильного пароля
openssl rand -base64 32
```

### SSL/TLS

В production используйте:

```env
DB_SSLMODE=require
```

---

## 📈 Оптимизация

### Настройка пула подключений

```go
db.SetMaxOpenConns(25)   // Максимум открытых подключений
db.SetMaxIdleConns(5)    // Максимум неиспользуемых
db.SetConnMaxLifetime(5 * time.Minute)
```

### Индексирование

При добавлении новых полей для поиска:

```sql
CREATE INDEX idx_table_field ON table(field);
CREATE INDEX idx_table_field_status ON table(field, status);  -- Составной индекс
```

### Аналитика медленных запросов

```sql
-- В postgresql.conf
log_min_duration_statement = 1000  -- Запросы > 1 сек
```

---

## 🐛 Troubleshooting

### "Connection refused"

```bash
# Проверьте, запущен ли контейнер
docker-compose ps

# Проверьте логи
docker-compose logs postgres

# Перезапустите
docker-compose restart postgres
```

### "Database does not exist"

```bash
# Создайте БД через psql
docker-compose exec postgres psql -U postgres -c "CREATE DATABASE event_api;"
```

### "Permission denied"

Проверьте права пользователя:

```sql
SELECT * FROM information_schema.role_table_grants 
WHERE grantee = 'event_api_user';
```

### "Connection pool exhausted"

Увеличьте `MaxOpenConns` в config или проверьте утечки подключений:

```sql
SELECT datname, count(*) FROM pg_stat_activity GROUP BY datname;
```

---

## 📚 Дополнительные ресурсы

- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Docker Compose Reference](https://docs.docker.com/compose/compose-file/)
- [pgAdmin Documentation](https://www.pgadmin.org/docs/)
- [Golang database/sql](https://pkg.go.dev/database/sql)

---

## 🔄 Миграция с другой БД

Если нужна миграция с MySQL, SQLite и т.д.:

1. Экспортируйте данные
2. Создайте PostgreSQL версию таблиц
3. Импортируйте данные
4. Обновите connection string

---

## ✅ Checklist для production

- [ ] Используется сильный пароль БД
- [ ] SSL/TLS включен (`DB_SSLMODE=require`)
- [ ] Настроены backups
- [ ] Включены логи запросов
- [ ] Настроены monitoring alerts
- [ ] Создан replica для failover
- [ ] Проведены load tests
- [ ] Документированы все миграции
- [ ] Подготовлен rollback plan

---

Инфраструктура готова к использованию! 🎉
