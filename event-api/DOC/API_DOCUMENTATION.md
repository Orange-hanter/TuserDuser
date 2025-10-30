# Event API - Authentication Endpoints

## Документация API для аутентификации

### Базовый URL

```sh
http://localhost:8080
```

---

## Endpoints

### 1. POST `/api/auth/register` - Регистрация нового пользователя

**Описание:** Создает новый аккаунт пользователя

**Request:**

```json
{
  "email": "user@example.com",
  "phone": "+79991234567",
  "password": "password123"
}
```

**Параметры:**

- `email` (string, required): Email адрес пользователя
- `phone` (string, required): Номер телефона
- `password` (string, required): Пароль (минимум 8 символов)

**Response (201 Created):**

```json
{
  "user": {
    "id": "ad58c1e9e3c617185f36e336d956339f",
    "email": "user@example.com",
    "phone": "+79991234567",
    "verified": false,
    "created_at": "2025-10-25T23:54:06.374939+03:00",
    "updated_at": "2025-10-25T23:54:06.374939+03:00"
  },
  "verify_code": "436194"
}
```

**Примеры ошибок:**

- `409 Conflict`: Пользователь с таким email уже существует
- `400 Bad Request`: Неверный формат данных

---

### 2. POST `/api/auth/verify` - Верификация email по коду

**Описание:** Проверяет код верификации и подтверждает email

**Request:**

```json
{
  "email": "user@example.com",
  "code": "436194"
}
```

**Параметры:**

- `email` (string, required): Email адрес пользователя
- `code` (string, required): 6-значный код верификации

**Response (200 OK):**

```json
{
  "message": "Email успешно верифицирован",
  "verified": true
}
```

**Примеры ошибок:**

- `400 Bad Request`: Неверный код верификации

---

### 3. POST `/api/auth/login` - Вход в систему

**Описание:** Аутентифицирует пользователя и выдает JWT токен

**Request:**

```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Параметры:**

- `email` (string, required): Email адрес
- `password` (string, required): Пароль

**Response (200 OK):**

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "ad58c1e9e3c617185f36e336d956339f",
    "email": "user@example.com",
    "phone": "+79991234567",
    "verified": true,
    "created_at": "2025-10-25T23:54:06.374939+03:00",
    "updated_at": "2025-10-25T23:54:16.477171+03:00"
  },
  "expires_in": 3600,
  "expires_at": "2025-10-26T00:54:28.132343+03:00"
}
```

**Примеры ошибок:**

- `401 Unauthorized`: Пользователь не найден или неверный пароль
- `400 Bad Request`: Неверный формат данных

---

### 4. POST `/api/auth/logout` - Выход из системы

**Описание:** Отзывает JWT токен (добавляет его в черный список)

**Request:**

```
Headers:
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json

Body: {}
```

Или в теле запроса:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response (200 OK):**
```json
{
  "message": "Успешно вышли из системы"
}
```

**Примеры ошибок:**
- `400 Bad Request`: Token не найден

---

### 5. GET `/api/auth/me` - Получить текущего пользователя

**Описание:** Возвращает информацию о текущем аутентифицированном пользователе

**Request:**
```
Headers:
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Response (200 OK):**
```json
{
  "id": "ad58c1e9e3c617185f36e336d956339f",
  "email": "user@example.com",
  "phone": "+79991234567",
  "verified": true,
  "created_at": "2025-10-25T23:54:06.374939+03:00",
  "updated_at": "2025-10-25T23:54:16.477171+03:00"
}
```

**Примеры ошибок:**
- `401 Unauthorized`: Токен отсутствует или недействителен
- `404 Not Found`: Пользователь не найден

---

## Авторизация

Для защищенных endpoints используется **Bearer Token (JWT)**:

```
Authorization: Bearer <token>
```

Где `<token>` - это JWT токен, полученный при логине.

### JWT Token Claims:
```json
{
  "user_id": "ad58c1e9e3c617185f36e336d956339f",
  "email": "user@example.com",
  "phone": "+79991234567",
  "verified": true,
  "exp": 1761429268,
  "iat": 1761425668
}
```

---

## Примеры использования

### Полный процесс аутентификации:

```bash
# 1. Регистрация
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "phone": "+79991234567",
    "password": "password123"
  }'

# 2. Верификация (используем код из ответа регистрации)
curl -X POST http://localhost:8080/api/auth/verify \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "code": "436194"
  }'

# 3. Вход
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'

# 4. Получение текущего пользователя
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
curl -X GET http://localhost:8080/api/auth/me \
  -H "Authorization: Bearer $TOKEN"

# 5. Выход
curl -X POST http://localhost:8080/api/auth/logout \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'
```

---

## Статус коды

| Код | Описание |
|-----|---------|
| 200 | OK - Успешный запрос |
| 201 | Created - Ресурс создан |
| 400 | Bad Request - Неверный формат запроса |
| 401 | Unauthorized - Требуется аутентификация |
| 404 | Not Found - Ресурс не найден |
| 409 | Conflict - Конфликт (например, пользователь уже существует) |
| 500 | Internal Server Error - Ошибка сервера |

---

## Переменные окружения

```env
PORT=8080                                              # Порт сервера
ENV=development                                        # Окружение
CORS_ALLOWED_ORIGINS=http://localhost:3000,...        # Allowed origins
JWT_SECRET=your-secret-key-change-in-production       # Secret для подписи JWT
```

---

## Безопасность

- ✅ Пароли хешируются с помощью bcrypt
- ✅ JWT токены подписаны с HMAC-SHA256
- ✅ Поддержка черного списка токенов (logout)
- ✅ Security headers включены (X-Content-Type-Options, X-Frame-Options, HSTS)
- ✅ CORS настроен

---

## Примечания

- Коды верификации в текущей реализации хранятся в памяти (для production используйте Redis или БД)
- Данные пользователей в текущей реализации хранятся в памяти (для production используйте базу данных)
- JWT token действителен 1 час (настраивается в config)
- Logout добавляет токен в черный список, хранящийся в памяти
