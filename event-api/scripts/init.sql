-- Инициализация БД для Event API
-- Этот скрипт выполняется автоматически при первом запуске PostgreSQL контейнера

-- Создание расширений для работы с UUID
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Установка временной зоны
SET timezone = 'UTC';

-- Информационное сообщение
DO $$
BEGIN
    RAISE NOTICE '✅ Database "%" initialized successfully', current_database();
    RAISE NOTICE '📊 Extensions: uuid-ossp, pgcrypto';
    RAISE NOTICE '🕐 Timezone: UTC';
END $$;

