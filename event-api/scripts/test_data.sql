-- Тестовые данные для Event API
-- Этот скрипт можно запустить после создания миграций

-- Вставляем тестовых пользователей (пароли должны быть захеширована в приложении)
-- Это только примеры структуры

INSERT INTO users (id, email, phone, password, verified, created_at, updated_at) 
VALUES 
  (
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11'::uuid,
    'test.user1@example.com',
    '+79991234567',
    '$2a$12$abcdefghijklmnopqrstuvwxyz', -- Хеш пароля (пример)
    true,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
  ),
  (
    'b1ffcd00-0d1c-5fg9-cc7e-7cc0ce491b22'::uuid,
    'test.user2@example.com',
    '+79991234568',
    '$2a$12$bcdefghijklmnopqrstuvwxyza', -- Хеш пароля (пример)
    true,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
  ),
  (
    'c2ggde11-1e2d-6gh0-dd8f-8dd1df502c33'::uuid,
    'test.user3@example.com',
    '+79991234569',
    '$2a$12$cdefghijklmnopqrstuvwxyzab', -- Хеш пароля (пример)
    false,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
  )
ON CONFLICT (email) DO NOTHING;

-- Проверяем вставленные данные
SELECT 'Users inserted' as status, COUNT(*) as count FROM users;
