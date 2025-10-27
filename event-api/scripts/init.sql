-- Инициализация БД для Event API
-- Этот скрипт выполняется при создании контейнера PostgreSQL

-- Создание расширения для UUID
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Создание расширения для генерации UUID v4
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Установка временной зоны
SET timezone = 'UTC';

-- Создание пользователя приложения (если не существует)
DO
$do$
BEGIN
   IF NOT EXISTS (SELECT FROM pg_user WHERE usename = 'event_api_user') THEN
      CREATE USER event_api_user WITH PASSWORD 'event_api_password';
   END IF;
END
$do$;

-- Даем права пользователю
ALTER USER event_api_user WITH PASSWORD 'event_api_password';
GRANT ALL PRIVILEGES ON DATABASE event_api TO event_api_user;

-- Подключаемся к нужной БД для создания схемы
\c event_api;

-- Даем права на schema
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO event_api_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO event_api_user;

-- Версионирование БД
CREATE TABLE IF NOT EXISTS db_version (
    id SERIAL PRIMARY KEY,
    version VARCHAR(50) NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO db_version (version, description) VALUES ('1.0', 'Initial schema')
ON CONFLICT DO NOTHING;
