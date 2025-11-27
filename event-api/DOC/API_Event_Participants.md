# Event Participants API

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
- Результаты отсортированы по времени регистрации (ascending)
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

**Database Table**: `event_registrations`

```sql
CREATE TABLE IF NOT EXISTS event_registrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    public_name VARCHAR(255) NOT NULL,
    avatar_url TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'confirmed',
    registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_event_registrations_event_id ON event_registrations(event_id);
CREATE INDEX idx_event_registrations_user_id ON event_registrations(user_id);
CREATE INDEX idx_event_registrations_event_status ON event_registrations(event_id, status);
CREATE UNIQUE INDEX idx_event_registrations_unique ON event_registrations(event_id, user_id);
```

**Service Method**: `UserService.GetEventParticipants()`

- Выполняет запрос к БД с фильтром по `event_id` и статусу `confirmed`
- Сортирует результаты по `registered_at` в порядке возрастания
- Возвращает `[]models.Participant`

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

- Используется индекс `idx_event_registrations_event_status(event_id, status)` для быстрого поиска
- Результаты отсортированы в БД (не в приложении)
- Рекомендуется добавить пагинацию для больших событий (100+ участников)

### Future Enhancements

1. **Пагинация**: Добавить query параметры `limit` и `offset`
2. **Фильтрация**: Возможность фильтровать по статусу (только confirmed/all)
3. **Сортировка**: Опции сортировки (по имени, дате регистрации)
4. **Кэширование**: Redis кэш списка участников для популярных событий
5. **WebSocket**: Real-time обновления при добавлении новых участников

### Related Endpoints

- `POST /v1/api/users/me/events/{event_id}/subscribe` - Подписаться на событие
- `GET /v1/api/users/me/events/upcoming` - Получить предстоящие события пользователя
- `GET /v1/api/events/{id}` - Получить информацию о событии
