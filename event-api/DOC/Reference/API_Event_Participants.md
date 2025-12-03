# Event Participants API

[← Вернуться к документации](./INDEX.md)

## Endpoint: GET /api/events/{event_id}/participants

### Overview

Возвращает список подтвержденных участников события. API является публичным (не требует аутентификации) и может использоваться для отображения информации об участниках события.

### Request

```bash
curl 'http://localhost:8080/v1/api/events/75736520-e2b1-446a-a7b2-6c1dea6f6ce7/participants' \
  -H 'Accept: application/json' \
  -H 'Content-Type: application/json'
```

### URL Parameters

| Parameter | Type | Required | Description                                |
| --------- | ---- | -------- | ------------------------------------------ |
| event_id  | UUID | Yes      | ID события для получения списка участников |

### Headers

```
Accept: application/json
Content-Type: application/json
Authorization: Bearer <token> (optional, не требуется для этого эндпоинта)
```

### Response

#### Success (200 OK)

```json
[
  {
    "user_id": "941b955e-ea57-dee3-565f-5684f81c4f14",
    "public_name": "Иван Петров",
    "avatar_url": "https://example.com/avatar.jpg",
    "status": "confirmed"
  },
  {
    "user_id": "850c123e-eb67-aaa3-656f-5684f81c4f15",
    "public_name": "Мария Сидорова",
    "avatar_url": null,
    "status": "confirmed"
  },
  {
    "user_id": "760d234e-ec78-bbb3-757f-5684f81c4f16",
    "public_name": "Петр Иванов",
    "avatar_url": "https://example.com/avatar2.jpg",
    "status": "confirmed"
  }
]
```

#### Error (400 Bad Request)

```json
{
  "error": "bad_request",
  "message": "Event ID is required"
}
```

#### Error (500 Internal Server Error)

```json
{
  "error": "internal_error",
  "message": "Failed to fetch participants"
}
```

### Response Fields

| Field       | Type         | Description                                                 |
| ----------- | ------------ | ----------------------------------------------------------- |
| user_id     | string       | UUID пользователя-участника события                         |
| public_name | string       | Публичное имя участника                                     |
| avatar_url  | string\|null | URL аватара участника (может быть null)                     |
| status      | string       | Статус участника: `confirmed`, `waitlisted` или `cancelled` |

### Notes

- Возвращает только участников со статусом `confirmed` (подтвержденные)
- Результаты отсортированы по времени подписки (ascending)
- API является публичным и не требует аутентификации
- Если событие не существует, возвращается пустой список

### Implementation Details

**Model**: `models.Participant`

```go
type Participant struct {
	UserID    string  `json:"user_id"`
	PublicName string `json:"public_name"`
	AvatarURL *string `json:"avatar_url,omitempty"`
	Status    string  `json:"status"` // "confirmed", "waitlisted", "cancelled"
}
```

**Database Table**: `event_subscriptions` (consolidated)

> **Note**: С миграции 014 таблица `event_registrations` удалена. Все данные об участниках теперь читаются из `event_subscriptions` с JOIN на `telegram_bindings` и `users` для получения публичного имени.

```sql
-- Primary table for subscriptions
CREATE TABLE IF NOT EXISTS event_subscriptions (
    user_id UUID NOT NULL,
    event_id UUID NOT NULL,
    status VARCHAR(50) NOT NULL,
    subscribed_at TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, event_id)
);

-- Public name is derived from telegram_bindings or users.email
-- via JOIN in the service query
```

**Service Method**: `UserService.GetEventParticipants()`

- Выполняет запрос к `event_subscriptions` с JOIN на `telegram_bindings` и `users`
- Фильтрует по `event_id` и статусу `confirmed`
- Публичное имя берётся из Telegram (first_name + last_name), username или email
- Сортирует результаты по `subscribed_at` в порядке возрастания
- Возвращает `[]models.Participant`

**SQL Query**:

```sql
SELECT
    es.user_id,
    COALESCE(
        NULLIF(TRIM(CONCAT(tb.telegram_first_name, ' ', tb.telegram_last_name)), ''),
        tb.telegram_username,
        u.email,
        'Anonymous'
    ) AS public_name,
    NULL::text AS avatar_url,
    es.status
FROM event_subscriptions es
LEFT JOIN telegram_bindings tb ON tb.user_id = es.user_id
LEFT JOIN users u ON u.id = es.user_id
WHERE es.event_id = $1 AND es.status = 'confirmed'
ORDER BY es.subscribed_at ASC
```

**Handler**: `UserHandler.GetEventParticipants()`

- Парсит `event_id` из URL параметров
- Вызывает `userService.GetEventParticipants()`
- Возвращает JSON ответ с кодом 200

**Router**: Зарегистрирован в публичных (неаутентифицированных) роутах:

```go
r.Get("/api/events/{event_id}/participants", userHandler.GetEventParticipants)
```

### Error Handling

| Error          | HTTP Status | Description                       |
| -------------- | ----------- | --------------------------------- |
| Empty event_id | 400         | Отсутствует обязательный параметр |
| Database error | 500         | Ошибка при запросе к БД           |

### Performance Considerations

- Используется составной индекс на `event_subscriptions(event_id)` для быстрого поиска
- JOIN на `telegram_bindings` и `users` оптимизирован (LEFT JOIN, nullable)
- Результаты отсортированы в БД (не в приложении)
- Рекомендуется добавить пагинацию для больших событий (100+ участников)

### Future Enhancements

1. **Пагинация**: Добавить query параметры `limit` и `offset`
2. **Фильтрация**: Возможность фильтровать по статусу (только confirmed/all)
3. **Сортировка**: Опции сортировки (по имени, дате регистрации)
4. **Кэширование**: Redis кэш списка участников для популярных событий
5. **WebSocket**: Real-time обновления при добавлении новых участников
6. **Avatar URL**: Добавить поле avatar_url в telegram_bindings для отображения аватаров

### Related Endpoints

- `POST /v1/api/users/me/events/{event_id}/subscribe` - Подписаться на событие
- `GET /v1/api/users/me/events/upcoming` - Получить предстоящие события пользователя
- `GET /v1/api/events/{id}` - Получить информацию о событии
