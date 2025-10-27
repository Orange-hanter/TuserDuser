# 🗄️ Инструкция: Работа с PostgreSQL инфраструктурой

## ⚡ За 2 минуты до первого подключения к БД

### Шаг 1: Запуск контейнеров

```bash
cd /Users/dakh/Git/TuserDuser/event-api

# Запуск PostgreSQL и pgAdmin
docker-compose up -d
```

**Что произойдет:**
```
✅ PostgreSQL запустится на localhost:5432
✅ pgAdmin запустится на http://localhost:5050
✅ БД "event_api" будет создана
✅ Таблица db_version инициализируется
```

### Шаг 2: Проверка статуса

```bash
# Проверка контейнеров
docker-compose ps

# Вывод:
# NAME                       STATE       PORTS
# event_api_postgres         running     0.0.0.0:5432->5432/tcp
# event_api_pgadmin          running     0.0.0.0:5050->80/tcp
```

### Шаг 3: Запуск приложения

```bash
make run

# или

go run ./cmd/server
```

**Приложение автоматически:**
```
1. Подключится к PostgreSQL
2. Создаст таблицу schema_migrations
3. Выполнит все миграции
4. Создаст таблицу users с индексами
5. Запустит сервер на :8080
```

**Ожидаемый вывод:**
```
{"level":"info","msg":"✅ Успешно подключились к БД","host":"localhost","port":"5432","dbname":"event_api"}
{"level":"info","msg":"🔄 Запускаем миграции БД"}
{"level":"info","msg":"▶️  Выполняем миграцию","name":"001_create_users_table"}
{"level":"info","msg":"✅ Миграция успешно применена","name":"001_create_users_table"}
{"level":"info","msg":"✅ Все миграции успешно выполнены"}
{"level":"info","msg":"Сервер запущен","port":":8080","env":"development"}
```

---

## 🔍 Проверка данных в БД

### Способ 1: psql (коммандная строка)

```bash
# Подключение
psql -h localhost -p 5432 -U postgres -d event_api

# Или через docker
docker-compose exec postgres psql -U postgres -d event_api
```

**Примеры команд:**
```sql
-- Список таблиц
\dt

-- Структура таблицы users
\d users

-- Таблица миграций
SELECT * FROM schema_migrations;

-- Все пользователи
SELECT * FROM users;

-- Выход
\q
```

### Способ 2: pgAdmin (веб-интерфейс)

1. Откройте http://localhost:5050
2. Войдите: `admin@example.com` / `admin`
3. Добавьте сервер:
   - **Hostname:** `postgres`
   - **Port:** `5432`
   - **Username:** `postgres`
   - **Password:** `postgres`
4. Просмотрите таблицы и данные

### Способ 3: DBeaver / DataGrip

Создайте новое подключение:
```
Host:     localhost
Port:     5432
Database: event_api
User:     postgres
Password: postgres
SSL Mode: disable
```

---

## 📝 Добавление первой миграции

Таблица `users` уже создана автоматически. Вот как добавить новую:

### Пример: Создание таблицы `events`

**1. Отредактируйте** `internal/migrations/migrations.go`:

```go
// Добавьте в список миграций:
migrations := []migration{
  {
    name: "001_create_users_table",
    up:   createUsersTable,
    down: dropUsersTable,
  },
  {
    name: "002_create_events_table",  // ← НОВАЯ
    up:   createEventsTable,
    down: dropEventsTable,
  },
}
```

**2. Добавьте SQL константы в конце файла:**

```go
const createEventsTable = `
CREATE TABLE IF NOT EXISTS events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  title VARCHAR(255) NOT NULL,
  description TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_events_user_id ON events(user_id);
CREATE INDEX idx_events_created_at ON events(created_at);
`

const dropEventsTable = `DROP TABLE IF EXISTS events CASCADE;`
```

**3. Пересоберите и запустите:**

```bash
go build ./cmd/server && ./cmd/server
```

**Результат:**
```
{"level":"info","msg":"▶️  Выполняем миграцию","name":"002_create_events_table"}
{"level":"info","msg":"✅ Миграция успешно применена","name":"002_create_events_table"}
```

---

## 🧪 Вставка тестовых данных

### Способ 1: SQL скрипт через psql

```bash
# Через docker-compose
docker-compose exec postgres psql -U postgres -d event_api -f /docker-entrypoint-initdb.d/test_data.sql

# Или локально
psql -h localhost -p 5432 -U postgres -d event_api -f scripts/test_data.sql
```

### Способ 2: Вручную через psql

```bash
psql -h localhost -p 5432 -U postgres -d event_api
```

```sql
INSERT INTO users (email, phone, password, verified) VALUES 
  ('user1@example.com', '+79991234567', '$2a$12$hash', true),
  ('user2@example.com', '+79991234568', '$2a$12$hash', true);

SELECT COUNT(*) FROM users;
```

---

## 🔧 Полезные команды

### Docker Compose

```bash
# Запуск в фоне
docker-compose up -d

# Остановка контейнеров
docker-compose down

# Перезагрузка PostgreSQL
docker-compose restart postgres

# Просмотр логов PostgreSQL
docker-compose logs -f postgres

# Просмотр логов pgAdmin
docker-compose logs -f pgadmin

# Удаление всех данных (с пересозданием)
docker-compose down -v && docker-compose up -d

# Вход в контейнер БД
docker-compose exec postgres bash
```

### PostgreSQL (psql)

```bash
# Список БД
psql -h localhost -U postgres -l

# Подключение к БД
psql -h localhost -p 5432 -U postgres -d event_api

# Выполнение SQL файла
psql -h localhost -U postgres -d event_api -f script.sql

# Экспорт данных
pg_dump -h localhost -U postgres -d event_api > backup.sql

# Импорт данных
psql -h localhost -U postgres -d event_api < backup.sql
```

### Go приложение

```bash
# Компиляция
go build ./cmd/server

# Запуск
./cmd/server

# С переменными окружения
DB_HOST=localhost DB_PORT=5432 go run ./cmd/server

# С логированием
LOG_LEVEL=debug go run ./cmd/server
```

---

## 🐛 Troubleshooting

### ❌ "Connection refused" при подключении к БД

**Проверка:**
```bash
# Проверьте, запущен ли контейнер
docker-compose ps

# Должен быть статус "running"
```

**Решение:**
```bash
# Перезагрузите контейнер
docker-compose restart postgres

# Или пересоздайте
docker-compose down && docker-compose up -d
```

### ❌ "Database does not exist"

**Проверка:**
```bash
psql -h localhost -U postgres -l
```

**Решение:**
```bash
# Создайте БД вручную
docker-compose exec postgres psql -U postgres -c "CREATE DATABASE event_api;"
```

### ❌ "Permission denied" при вставке данных

**Проверка:**
```sql
SELECT * FROM information_schema.role_table_grants 
WHERE grantee = 'postgres';
```

**Решение:**
```sql
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO postgres;
```

### ❌ Порт 5432 уже используется

**Измените в docker-compose.yml:**
```yaml
ports:
  - "5433:5432"  # Используйте 5433 вместо 5432
```

Или в `.env`:
```env
DB_PORT=5433
```

---

## 📊 Мониторинг

### Активные подключения

```sql
SELECT datname, count(*) as connections 
FROM pg_stat_activity 
GROUP BY datname;
```

### Размер БД

```sql
SELECT 
  datname,
  pg_size_pretty(pg_database_size(datname)) as size
FROM pg_database 
WHERE datname = 'event_api';
```

### Таблицы и индексы

```sql
SELECT 
  schemaname,
  tablename,
  pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables 
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

---

## 🔐 Безопасность

### Изменение пароля PostgreSQL

```bash
# В docker-compose.yml
docker-compose down && docker-compose up -d

# Или вручную
docker-compose exec postgres psql -U postgres -c "ALTER USER postgres WITH PASSWORD 'newpassword';"
```

### Экспорт данных для backup

```bash
# Полный dump
pg_dump -h localhost -U postgres -d event_api > backup_$(date +%Y%m%d).sql

# Только структура
pg_dump -h localhost -U postgres -d event_api -s > schema.sql

# Только данные
pg_dump -h localhost -U postgres -d event_api -a > data.sql
```

### Импорт из backup

```bash
psql -h localhost -U postgres -d event_api < backup_20251025.sql
```

---

## 📚 Ссылки

- 🗄️ [DATABASE.md](./DATABASE.md) - Полная документация БД
- 📖 [PostgreSQL Docs](https://www.postgresql.org/docs/)
- 🐳 [Docker Compose Docs](https://docs.docker.com/compose/)
- 💾 [pgAdmin Docs](https://www.pgadmin.org/docs/)

---

## ✅ Checklist

- [ ] Docker установлен (`docker --version`)
- [ ] docker-compose установлен (`docker-compose --version`)
- [ ] Go 1.25+ установлен (`go version`)
- [ ] PostgreSQL запущен (`docker-compose ps`)
- [ ] Приложение скомпилировано (`go build ./cmd/server`)
- [ ] Приложение запущено (`make run`)
- [ ] Можно подключиться через psql
- [ ] Таблицы созданы через миграции
- [ ] Тестовые данные вставлены

---

🎉 **Готово! Инфраструктура БД полностью настроена и готова к использованию.**
