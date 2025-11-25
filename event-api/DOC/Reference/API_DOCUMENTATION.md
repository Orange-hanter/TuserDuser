# Event API - Authentication Endpoints

## Документация API для аутентификации

### Базовый URL

```json
http://localhost:8080
```

---

## Endpoints

### 1. POST `/api/auth/register` - Регистрация нового пользователя

**Описание:** Создает новый аккаунт пользователя

**Request:**

<!-- markdownlint-disable MD013 -->

```json
{
  "email": "user@example.com",
  "phone": "+79991234567",
  "password": "password123"
}
```

<!-- markdownlint-enable MD013 -->

**Параметры:**

- `email` (string, required): Email адрес пользователя
- `phone` (string, required): Номер телефона
- `password` (string, required): Пароль (минимум 8 символов)

**Response (201 Created):**

<!-- markdownlint-disable MD013 -->

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

<!-- markdownlint-enable MD013 -->

**Примеры ошибок:**

- `409 Conflict`: Пользователь с таким email уже существует
- `400 Bad Request`: Неверный формат данных

---

### 2. POST `/api/auth/verify` - Верификация email по коду

**Описание:** Проверяет код верификации и подтверждает email

**Request:**

<!-- markdownlint-disable MD013 -->

```tex
{
  "email": "user@example.com",
  "code": "436194"
}
```

<!-- markdownlint-enable MD013 -->

**Параметры:**

- `email` (string, required): Email адрес пользователя
- `code` (string, required): 6-значный код верификации

**Response (200 OK):**

<!-- markdownlint-disable MD013 -->

```tex
{
  "message": "Email успешно верифицирован",
  "verified": true
}
```

<!-- markdownlint-enable MD013 -->

**Примеры ошибок:**

- `400 Bad Request`: Неверный код верификации

---

### 3. POST `/api/auth/login` - Вход в систему

**Описание:** Аутентифицирует пользователя и выдает JWT токен

**Request:**

<!-- markdownlint-disable MD013 -->

```tex
{
  "email": "user@example.com",
  "password": "password123"
}
```

<!-- markdownlint-enable MD013 -->

**Параметры:**

- `email` (string, required): Email адрес
- `password` (string, required): Пароль

**Response (200 OK):**

<!-- markdownlint-disable MD013 -->

```tex
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

<!-- markdownlint-enable MD013 -->

**Примеры ошибок:**

- `401 Unauthorized`: Пользователь не найден или неверный пароль
- `400 Bad Request`: Неверный формат данных

---

### 4. POST `/api/auth/logout` - Выход из системы

**Описание:** Отзывает JWT токен (добавляет его в черный список)

**Request:**

```tex

Headers:
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json

Body: {}

```

Или в теле запроса:

<!-- markdownlint-disable MD013 -->

```tex
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

<!-- markdownlint-enable MD013 -->

**Response (200 OK):**

<!-- markdownlint-disable MD013 -->

```tex
{
  "message": "Успешно вышли из системы"
}
```

<!-- markdownlint-enable MD013 -->

**Примеры ошибок:**

- `400 Bad Request`: Token не найден

---

### 5. GET `/api/auth/me` - Получить текущего пользователя

**Описание:** Возвращает информацию о текущем аутентифицированном пользователе

**Request:**

```tex

Headers:
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...

```

**Response (200 OK):**

<!-- markdownlint-disable MD013 -->

```tex
{
  "id": "ad58c1e9e3c617185f36e336d956339f",
  "email": "user@example.com",
  "phone": "+79991234567",
  "verified": true,
  "created_at": "2025-10-25T23:54:06.374939+03:00",
  "updated_at": "2025-10-25T23:54:16.477171+03:00"
}
```

<!-- markdownlint-enable MD013 -->

**Примеры ошибок:**

- `401 Unauthorized`: Токен отсутствует или недействителен
- `404 Not Found`: Пользователь не найден

---

## Авторизация

Для защищенных endpoints используется **Bearer Token (JWT)**:

```tex
Authorization: Bearer <token>
```

---

## Narrow Time-Slot Discovery Engine

Эти эндпоинты помогают пользователю быстро просмотреть события в узком временном
окне, фиксируя реакции и конфликты бронирований. Все запросы выполняются под
заголовком `Authorization: Bearer <token>`.

> Подробнее о внутреннем устройстве и сценариях использования см. в документе [`DOC/DISCOVERY_ENGINE.md`](DISCOVERY_ENGINE.md).

### Быстрый старт

1. Вызвать `GET /v1/api/discovery/next` и сохранить `eventId`.
2. Решить действие (`like`, `dislike`, `neutral`) и вызвать `POST /v1/api/discovery/action`.
3. При готовности забронировать — `POST /v1/api/discovery/book` с тем же `eventId`.
4. Для отображения ленты событий пользователю можно повторно дёргать
   `/next`, пока очередь не опустеет.

### Карта действий

- `like` — Удаляет событие из очереди и помечает как интересное.
  - Запись в истории: `history.action = "like"`
- `dislike` — Удаляет событие навсегда.
  - Запись в истории: `history.action = "dislike"`
- `neutral` — Переносит событие в конец очереди (или в конфликтную часть).
  - Запись в истории: `history.action = "neutral"`
- `book` — Бронирует событие и помечает пересекающиеся слоты как конфликтующие.
  - Запись в истории: `history.action = "book"` (и `context.conflictedEventIds`)

## 1. GET `/v1/api/discovery/next`

**Описание:** Возвращает следующее событие в очереди. События, помеченные
как конфликтующие после бронирования, отображаются только после исчерпания
основной очереди.

**Response (200 OK):**

<!-- markdownlint-disable MD013 -->

```tex
{
  "event": {
    "id": "evt_brunch",
    "title": "Morning Brunch",
    "description": "Morning Brunch @ Loft",
    "slot": {
      "start": "2025-11-16T10:00:00Z",
      "end": "2025-11-16T11:30:00Z"
    },
    "metadata": {
      "place": "Loft",
      "type": "food",
      "priceType": "free"
    }
  },
  "conflict": false,
  "remainingPrimary": 3,
  "remainingConflicts": 1
}
```

<!-- markdownlint-enable MD013 -->

`404` возвращается, когда очередь исчерпана.

## 2. POST `/v1/api/discovery/action`

**Описание:** Реакция пользователя на текущее событие. Поддерживаются
`like`, `dislike`, `neutral`.

**Request:**

<!-- markdownlint-disable MD013 -->

```tex
{
  "eventId": "evt_brunch",
  "action": "neutral"
}
```

<!-- markdownlint-enable MD013 -->

**Response (200 OK):**

<!-- markdownlint-disable MD013 -->

```tex
{
  "userId": "f84c9b46",
  "eventId": "evt_brunch",
  "action": "neutral",
  "timestamp": "2025-11-16T09:05:00Z"
}
```

<!-- markdownlint-enable MD013 -->

## 3. POST `/v1/api/discovery/book`

**Описание:** Подтверждает участие в событии. Все пересекающиеся по времени
события перемещаются в конец очереди с пометкой конфликта.

**Request:**

<!-- markdownlint-disable MD013 -->

```tex
{
  "eventId": "evt_brunch"
}
```

<!-- markdownlint-enable MD013 -->

**Response (200 OK):**

<!-- markdownlint-disable MD013 -->

```tex
{
  "bookedEvent": {
    "id": "evt_brunch",
    "slot": {
      "start": "2025-11-16T10:00:00Z",
      "end": "2025-11-16T11:30:00Z"
    },
    "metadata": {
      "type": "food"
    }
  },
  "conflictedEventIds": ["evt_meetup", "evt_run"]
}
```

<!-- markdownlint-enable MD013 -->

## 4. GET `/v1/api/discovery/history`

**Описание:** Возвращает хронологию действий пользователя над событиями окна.

**Response (200 OK):**

<!-- markdownlint-disable MD013 -->

```tex
[
  {
    "userId": "f84c9b46",
    "eventId": "evt_brunch",
    "action": "book",
    "timestamp": "2025-11-16T09:10:00Z",
    "context": {
      "conflictedEventIds": ["evt_meetup"]
    }
  },
  {
    "userId": "f84c9b46",
    "eventId": "evt_run",
    "action": "dislike",
    "timestamp": "2025-11-16T09:06:00Z"
  }
]
```

<!-- markdownlint-enable MD013 -->

**Коды ошибок для всех discovery эндпоинтов:**

- `400 Bad Request` — нарушена валидация входных данных.
- `401 Unauthorized` — отсутствует или недействителен JWT токен.
- `404 Not Found` — событие не найдено или очередь опустела.
- `409 Conflict` — пользователь пытается выполнить действие не по порядку.
- `500 Internal Server Error` — непредвиденная ошибка движка.

Где `<token>` - это JWT токен, полученный при логине.

### JWT Token Claims

<!-- markdownlint-disable MD013 -->

```tex
{
  "user_id": "ad58c1e9e3c617185f36e336d956339f",
  "email": "user@example.com",
  "phone": "+79991234567",
  "verified": true,
  "exp": 1761429268,
  "iat": 1761425668
}
```

<!-- markdownlint-enable MD013 -->

---

## Примеры использования

### Полный процесс аутентификации

```tex

## 1. Регистрация

## 1. Регистрация
curl -X POST http://localhost:8080/api/auth/registe
  -H "Content-Type: application/json"
  -d '{
    "email": "test@example.com",
    "phone": "+79991234567",
    "password": "password123"
  }'

## 2. Верификация (используем код из ответа регистрации)

## 2. Верификация (используем код из ответа регистрации)
curl -X POST http://localhost:8080/api/auth/verify
  -H "Content-Type: application/json"
  -d '{
    "email": "test@example.com",
    "code": "436194"
  }'

## 3. Вход

## 3. Вход
curl -X POST http://localhost:8080/api/auth/login
  -H "Content-Type: application/json"
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'

## 4. Получение текущего пользователя

## 4. Получение текущего пользователя
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
curl -X GET http://localhost:8080/api/auth/me
  -H "Authorization: Bearer $TOKEN"

## 5. Выход

## 5. Выход
curl -X POST http://localhost:8080/api/auth/logou
  -H "Authorization: Bearer $TOKEN"
  -H "Content-Type: application/json"
  -d '{}'
```

---

## Статус коды

| Код | Описание                                                    |
| --- | ----------------------------------------------------------- |
| 200 | OK - Успешный запрос                                        |
| 201 | Created - Ресурс создан                                     |
| 400 | Bad Request - Неверный формат запроса                       |
| 401 | Unauthorized - Требуется аутентификация                     |
| 404 | Not Found - Ресурс не найден                                |
| 409 | Conflict - Конфликт (например, пользователь уже существует) |
| 500 | Internal Server Error - Ошибка сервера                      |

---

## Переменные окружения

```tex
PORT=8080                                              # Порт сервера
ENV=development                                        # Окружение
CORS_ALLOWED_ORIGINS=http://localhost:3000,...        # Allowed origins
JWT_SECRET=your-secret-key-change-in-production       # Secret для подписи JWT
```

---

## Безопасность

- ✅ Пароли хешируются с помощью bcryp
- ✅ JWT токены подписаны с HMAC-SHA256
- ✅ Поддержка черного списка токенов (logout)
- ✅ Security headers включены (X-Content-Type-Options, X-Frame-Options, HSTS)
- ✅ CORS настроен

---

## Примечания

- Коды верификации в текущей реализации хранятся в памяти (для production
  используйте Redis или БД)
- Данные пользователей в текущей реализации хранятся в памяти (для
  production используйте базу данных)
- JWT token действителен 1 час (настраивается в config)
- Logout добавляет токен в черный список, хранящийся в памяти
