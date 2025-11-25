# 🚀 Quick Start Guide

## За 5 минут до первого запроса

### 1️⃣ Запуск сервера

````bash
cd /Users/dakh/Git/TuserDuser/event-api
make run
```bash
Сервер запустится на `http://localhost:8080`

### 2️⃣ Регистрация пользователя

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "phone": "+79991234567",
    "password": "SecurePassword123"
  }'
````

**Ответ:**

````json
{
  "user": {
    "id": "abc123...",
    "email": "john@example.com",
    "phone": "+79991234567",
    "verified": false,
    "created_at": "2025-10-25T23:54:06Z"
  },
  "verify_code": "123456"
}
```bash
### 3️⃣ Верификация email

Используйте `verify_code` из предыдущего ответа:

```bash
curl -X POST http://localhost:8080/api/auth/verify \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "code": "123456"
  }'
````

### 4️⃣ Логин (получение JWT)

````bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "SecurePassword123"
  }'
```json
**Ответ:**

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "abc123...",
    "email": "john@example.com",
    "verified": true,
    "created_at": "2025-10-25T23:54:06Z"
  },
  "expires_in": 3600,
  "expires_at": "2025-10-26T00:54:28Z"
}
````

### 5️⃣ Использование protected endpoint

Скопируйте `access_token` и используйте его:

````bash
curl -X GET http://localhost:8080/api/auth/me \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```json
**Ответ:**

```json
{
  "id": "abc123...",
  "email": "john@example.com",
  "phone": "+79991234567",
  "verified": true,
  "created_at": "2025-10-25T23:54:06Z",
  "updated_at": "2025-10-25T23:54:16Z"
}
````

### 6️⃣ Выход из системы

````bash
curl -X POST http://localhost:8080/api/auth/logout \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{}'
```bash
---

## 🔧 Команды

```bash
## Компиляция
## Компиляция
make build

## Запуск
## Запуск
make run

## Тесты
## Тесты
make test

## Docker сборка
## Docker сборка
make docker-build

## Docker запуск
## Docker запуск
make docker-run

## Clean
## Clean
make clean
````

---

## ⚙️ Переменные окружения (.env)

````env
PORT=8080
ENV=development
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000
JWT_SECRET=super-secret-key-change-in-production-please
```bash
---

## 📝 Требования

- Go 1.25.0+
- Make (для Makefile)
- curl или Postman (для тестирования)
- Docker (опционально)

---

## 🐛 Troubleshooting

### Порт 8080 занят

Измените PORT в .env файле:

```env
PORT=8081
````

### Ошибка "JWT token is invalid"

Убедитесь, что:

1. Токен скопирован полностью
2. Используется Header: `Authorization: Bearer <token>`
3. Токен не истёк (default 1 час)

### Ошибка при регистрации "user already exists"

Пользователь с таким email уже зарегистрирован. Используйте другой email.

### Сервер не запускается

1. Проверьте, что Go установлен: `go version`
2. Скачайте зависимости: `go mod download`
3. Проверьте ошибки компиляции: `go build ./...`

---

## 📚 Дополнительно

- [API Документация](./API_DOCUMENTATION.md)
- [Архитектура](./ARCHITECTURE.md)
- [Отчёт реализации](./IMPLEMENTATION_REPORT.md)
- [Return to documentation menu](../README.md)

---

## 🎯 Что дальше?

1. **Интеграция с БД**: Замените in-memory storage на PostgreSQL
2. **Добавьте Redis**: Для кеша и черного списка токенов
3. **Email верификация**: Отправляйте коды по почте
4. **2FA**: Добавьте двухфакторную аутентификацию
5. **Тесты**: Напишите unit тесты

---

Готово к использованию! 🎉
