# Creator Events API - Справка для фронтенда

## Обзор

API для управления своими событиями авторами (создателями). Позволяет просматривать статус модерации, отклонённые события и комментарии админа.

---

## Аутентификация

Все эндпоинты требуют JWT токен в заголовке:

```
Authorization: Bearer <jwt_token>
```

---

## Эндпоинты

### 1. Получить мои события

**GET** `/v1/api/creator/events`

Возвращает события автора, сгруппированные по статусам.

**Требуемые роли:** `creator`, `admin`

**Ответ (200):**

```json
{
  "pending": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "type": "workshop",
      "startTime": "2025-12-01T14:00:00Z",
      "endTime": "2025-12-01T16:00:00Z",
      "duration": 120,
      "place": "Room 101",
      "priceType": "paid",
      "needRegistration": true,
      "status": "pending",
      "reviewComment": "",
      "createdAt": "2025-11-27T10:00:00Z",
      "updatedAt": "2025-11-27T10:00:00Z"
    },
    {
      "id": "550e8400-e29b-41d4-a716-446655440002",
      "type": "seminar",
      "status": "needs_revision",
      "reviewComment": "Пожалуйста, исправьте описание события",
      "startTime": "2025-12-05T15:00:00Z",
      "endTime": "2025-12-05T17:00:00Z",
      "duration": 120,
      "place": "Online",
      "priceType": "free",
      "needRegistration": false,
      "createdAt": "2025-11-25T12:00:00Z",
      "updatedAt": "2025-11-27T09:30:00Z"
    }
  ],
  "active": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440003",
      "type": "conference",
      "status": "approved",
      "startTime": "2025-12-10T09:00:00Z",
      "endTime": "2025-12-10T17:00:00Z",
      "duration": 480,
      "place": "Convention Center",
      "priceType": "paid",
      "needRegistration": true,
      "reviewComment": "",
      "createdAt": "2025-11-20T14:00:00Z",
      "updatedAt": "2025-11-21T08:00:00Z"
    }
  ],
  "rejected": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440004",
      "type": "meetup",
      "status": "rejected",
      "reviewComment": "Событие не соответствует нашим стандартам",
      "startTime": "2025-11-30T18:00:00Z",
      "endTime": "2025-11-30T20:00:00Z",
      "duration": 120,
      "place": "Café",
      "priceType": "free",
      "needRegistration": false,
      "createdAt": "2025-11-15T10:00:00Z",
      "updatedAt": "2025-11-26T16:00:00Z"
    }
  ]
}
```

**Поля ответа:**

- `pending` — ожидающие проверки и требующие доработки события
- `active` — одобренные события, которые ещё не закончились
- `rejected` — отклонённые события

**Статусы событий:**

- `pending` — ждёт проверки модератором
- `needs_revision` — нужна доработка (см. `reviewComment`)
- `approved` — одобрено и активно
- `rejected` — отклонено

**Ошибки:**

- `401` — не авторизован
- `500` — внутренняя ошибка сервера

---

### 2. Получить заблокированные события

**GET** `/v1/api/creator/events/blocked`

Возвращает события, которые были заблокированы администратором.

**Требуемые роли:** `creator`, `admin`

**Ответ (200):**

```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440005",
    "type": "workshop",
    "startTime": "2025-12-15T10:00:00Z",
    "endTime": "2025-12-15T12:00:00Z",
    "duration": 120,
    "place": "Room 202",
    "priceType": "paid",
    "needRegistration": true,
    "status": "blocked",
    "reviewComment": "",
    "createdAt": "2025-11-20T14:00:00Z",
    "updatedAt": "2025-11-27T08:00:00Z",
    "blockReason": "Нарушение правил содержания контента",
    "blockedAt": "2025-11-27T08:00:00Z"
  }
]
```

**Ошибки:**

- `401` — не авторизован
- `500` — внутренняя ошибка сервера

---

### 3. Получить комментарии к событию

**GET** `/v1/api/creator/events/{eventId}/comments`

Возвращает историю всех комментариев модерации для конкретного события.

**Параметры:**

- `eventId` (path, обязателен) — UUID события

**Требуемые роли:** `creator`, `admin`

**Ответ (200):**

```json
[
  {
    "id": 1,
    "eventId": "550e8400-e29b-41d4-a716-446655440002",
    "authorId": "550e8400-e29b-41d4-a716-446655440099",
    "authorRole": "admin",
    "comment": "Пожалуйста, исправьте описание события",
    "createdAt": "2025-11-27T09:30:00Z"
  },
  {
    "id": 2,
    "eventId": "550e8400-e29b-41d4-a716-446655440002",
    "authorId": "550e8400-e29b-41d4-a716-446655440001",
    "authorRole": "creator",
    "comment": "Исправил описание, спасибо за замечание",
    "createdAt": "2025-11-27T10:00:00Z"
  }
]
```

**Ошибки:**

- `401` — не авторизован
- `404` — событие не найдено
- `500` — внутренняя ошибка сервера

---

### 4. Добавить комментарий к событию

**POST** `/v1/api/creator/events/{eventId}/comments`

Позволяет автору (или админу) добавить комментарий к событию.

**Параметры:**

- `eventId` (path, обязателен) — UUID события

**Тело запроса:**

```json
{
  "comment": "Спасибо за замечание, исправил"
}
```

**Требуемые роли:** `creator`, `admin`

**Ответ (200):**

```json
{
  "status": "ok"
}
```

**Ошибки:**

- `400` — неверный формат или пустой комментарий
- `401` — не авторизован
- `500` — внутренняя ошибка сервера

---

### 5. Запросить доработку события (Admin only)

**POST** `/v1/api/admin/events/{eventId}/request-revision`

Администратор может перевести событие в статус `needs_revision` с комментарием.

**Параметры:**

- `eventId` (path, обязателен) — UUID события

**Тело запроса:**

```json
{
  "comment": "Пожалуйста, обновите изображение события и добавьте описание"
}
```

**Требуемые роли:** `admin`

**Ответ (200):**

```json
{
  "status": "ok"
}
```

**Ошибки:**

- `400` — неверный формат или пустой комментарий
- `401` — не авторизован
- `404` — событие не найдено
- `500` — внутренняя ошибка сервера

---

### 6. Заблокировать событие (Admin only)

**POST** `/v1/api/admin/events/{eventId}/block`

Администратор может заблокировать событие с указанием причины.

**Параметры:**

- `eventId` (path, обязателен) — UUID события

**Тело запроса:**

```json
{
  "reason": "Нарушение правил содержания контента - ненормативная лексика в описании"
}
```

**Требуемые роли:** `admin`

**Ответ (200):**

```json
{
  "status": "ok"
}
```

**Ошибки:**

- `400` — неверный формат или пустая причина
- `401` — не авторизован
- `404` — событие не найдено
- `500` — внутренняя ошибка сервера

---

## Типы данных

### CreatorEvent

```typescript
{
  id: string; // UUID события
  type: string; // Тип события (workshop, seminar и т.д.)
  startTime: DateTime; // Начало события
  endTime: DateTime; // Конец события
  duration: number; // Длительность в минутах
  place: string; // Место проведения
  priceType: string; // Тип цены (free, paid и т.д.)
  needRegistration: boolean; // Требуется ли регистрация
  details: object; // Дополнительные данные (JSON)
  status: string; // Статус (pending, needs_revision, approved, rejected, blocked)
  reviewComment: string; // Комментарий модератора
  createdAt: DateTime; // Время создания
  updatedAt: DateTime; // Время последнего обновления
}
```

### BlockedEvent (extends CreatorEvent)

```typescript
{
  // Все поля из CreatorEvent
  blockReason: string; // Причина блокировки
  blockedAt: DateTime; // Время блокировки
}
```

### ReviewComment

```typescript
{
  id: number; // ID комментария
  eventId: string; // UUID события
  authorId: string; // UUID автора комментария
  authorRole: string; // Роль автора (admin, creator)
  comment: string; // Текст комментария
  createdAt: DateTime; // Время создания
}
```

---

## Примеры использования

### JavaScript/TypeScript

```typescript
// Получить мои события
const getMyEvents = async () => {
  const response = await fetch("/v1/api/creator/events", {
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
  });
  return response.json();
};

// Получить комментарии к событию
const getComments = async (eventId: string) => {
  const response = await fetch(`/v1/api/creator/events/${eventId}/comments`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });
  return response.json();
};

// Добавить комментарий
const addComment = async (eventId: string, comment: string) => {
  const response = await fetch(`/v1/api/creator/events/${eventId}/comments`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ comment }),
  });
  return response.json();
};
```

### cURL

```bash
# Получить мои события
curl -X GET http://localhost:8080/v1/api/creator/events \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json"

# Получить заблокированные события
curl -X GET http://localhost:8080/v1/api/creator/events/blocked \
  -H "Authorization: Bearer YOUR_TOKEN"

# Добавить комментарий
curl -X POST http://localhost:8080/v1/api/creator/events/{eventId}/comments \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"comment": "Спасибо за замечание"}'
```

---

## Коды ошибок

| Код | Статус                | Описание                                          |
| --- | --------------------- | ------------------------------------------------- |
| 200 | OK                    | Успешный запрос                                   |
| 201 | Created               | Ресурс создан                                     |
| 400 | Bad Request           | Неверный формат запроса или валидация не пройдена |
| 401 | Unauthorized          | Требуется авторизация                             |
| 404 | Not Found             | Ресурс не найден                                  |
| 500 | Internal Server Error | Внутренняя ошибка сервера                         |

**Формат ошибки:**

```json
{
  "error": "unauthorized",
  "message": "Требуется авторизация",
  "code": 401
}
```

---

## Примечания для фронтенда

### UI компоненты для реализации:

1. **Таб "Мои события"** → Показывает три категории:
   - Ожидание проверки (pending)
   - Активные (active)
   - Отклонённые (rejected)
   - Каждое событие с кнопкой "Комментарии модератора"

2. **Таб "Заблокированные"** → Отдельная страница с заблокированными событиями и причинами

3. **Модальное окно комментариев** → История общения с модератором, возможность добавить свой комментарий

4. **Статусные индикаторы:**
   - 🟡 `pending` — жёлтый (ожидание)
   - 🟠 `needs_revision` — оранжевый (требует доработки)
   - 🟢 `approved` — зелёный (одобрено)
   - 🔴 `rejected` — красный (отклонено)
   - ⚫ `blocked` — чёрный (заблокировано)

### Обработка ошибок:

- При `401` → перенаправить на страницу логина
- При `404` → показать сообщение "События не найдено"
- При `500` → показать сообщение "Ошибка сервера, попробуйте позже"

---

## История изменений

**v1.0** (27 Nov 2025) -初始 release с основным функционалом Creator Events API

- Поддержка всех статусов модерации
- Система комментариев модерации
