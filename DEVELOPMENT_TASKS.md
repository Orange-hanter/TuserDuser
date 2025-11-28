# 🛠️ Детальное описание задач для разработки

## Приоритет 1: Критические улучшения

### 1.1 Оптимизация Discovery Engine

**Проблема:**

- Current: History lookup O(n) при фильтрации
- Current: Каждый запрос сканирует всю историю
- Current: Исключённые события проверяются линейно

**Решение:**

```go
// Вместо полного сканирования истории:
// OLD: func (s *Service) GetExcludedEventIDs(ctx, userID) -> O(n)

// NEW: Использовать Redis HyperLogLog для быстрой проверки
type ExcludedEventsCache interface {
    Add(eventID string)
    Contains(eventID string) bool
    Count() int64
}

// Или Redis Bitmap (ещё быстрее для небольших наборов)
// Или Bloom Filter для выделенной памяти
```

**Шаги:**

1. Добавить `redis.BitMap` операции (SETBIT, GETBIT)
2. При добавлении в историю — обновлять bitmap
3. При поиске исключённых — проверять O(1) вместо O(n)
4. Добавить TTL как у истории

**Файлы для изменения:**

- `internal/discovery/service.go` — изменить GetExcludedEventIDs
- `internal/discovery/redis_history_repository.go` — добавить bitmap update

**Приблизительное время:** 4-6 часов

---

### 1.2 Увеличение throughput базы данных

**Проблема:**

- N+1 queries при загрузке событий с деталями
- Нет connection pooling оптимизации
- Bulk inserts делаются по одному

**Решение:**

```go
// Batch insert вместо одного за раз
func (r *PostgresRepository) InsertBatch(ctx context.Context, events []Event) error {
    // Использовать multirow insert:
    // INSERT INTO events (id, title, ...) VALUES ($1, $2, ...), ($3, $4, ...) ...

    query := "INSERT INTO events (id, title, ...) VALUES "
    values := []interface{}{}

    for i, event := range events {
        offset := i * 3
        query += fmt.Sprintf("($%d, $%d, $%d),", offset+1, offset+2, offset+3)
        values = append(values, event.ID, event.Title, ...)
    }
    query = strings.TrimSuffix(query, ",")

    return r.db.ExecContext(ctx, query, values...).Err()
}
```

**Шаги:**

1. Профилировать текущие queries (pg_stat_statements)
2. Добавить batch insert операции
3. Добавить JOIN для избежания N+1
4. Настроить connection pool (max_connections, min_connections)

**Файлы для изменения:**

- `internal/database/repository.go` — добавить BatchInsert
- `internal/config/config.go` — настройки pool
- `internal/handlers/events.go` — использовать batch

**Приблизительное время:** 6-8 часов

---

### 1.3 Мониторинг Redis

**Что нужно:**

- Prometheus metrics для Redis
- Grafana dashboard
- Alerts на критические события

**Метрики:**

```
redis_memory_used_bytes
redis_memory_max_bytes
redis_keys_total{pattern="queue:user:*"}
redis_keys_total{pattern="history:user:*"}
redis_operations_total{operation="GET"}
redis_operations_total{operation="SET"}
redis_operation_duration_seconds
redis_evictions_total
```

**Шаги:**

1. Использовать `prometheus/client_golang`
2. Добавить `internal/metrics/redis.go`
3. Интегрировать в Redis repositories
4. Создать Grafana dashboard JSON
5. Настроить alerts (alertmanager)

**Файлы для создания:**

- `internal/metrics/redis.go` — Redis metrics collector
- `internal/metrics/prometheus.go` — Prometheus registry
- `monitoring/grafana/redis-dashboard.json` — Grafana dashboard
- `monitoring/alerts/redis-alerts.yml` — Alert rules

**Приблизительное время:** 8 часов

---

## Приоритет 2: Функциональность

### 2.1 Real-time уведомления через WebSocket

**Архитектура:**

```
┌─ Admin1 ─┐
│ WS Conn  │
└────┬─────┘
     │ Redis Pub/Sub
     │
┌────▼────────────────┐
│   Redis Channel:    │
│ creator:12345:msgs  │
└────┬────────────────┘
     │
┌────▼─────────────┐
│ Creator (Mobile) │
│   WS Listener    │
└──────────────────┘
```

**Шаги:**

1. Создать WebSocket handler в handlers
2. Использовать Redis Pub/Sub для broadcast
3. Отправлять уведомления при новом комментарии
4. На фронте добавить WebSocket слушатель
5. Обновлять UI при получении сообщения

**Код на бэке:**

```go
// internal/handlers/websocket.go
func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) {
    userID := r.Header.Get("X-User-ID")
    channel := fmt.Sprintf("creator:%s:msgs", userID)

    pubsub := h.redis.Subscribe(r.Context(), channel)
    defer pubsub.Close()

    // WebSocket upgrade...
    ws.OnMessage(func(msg []byte) {
        pubsub.Send(channel, msg)
    })
}
```

**Файлы для создания:**

- `internal/handlers/websocket.go` — WebSocket handler
- `internal/redis/pubsub.go` — Redis Pub/Sub wrapper
- `admin-panel/src/hooks/useWebSocket.js` — React hook

**Приблизительное время:** 10-12 часов

---

### 2.2 Full-text search для событий

**Опция 1: PostgreSQL FTS (быстрее внедрить)**

```sql
-- Добавить tsvector column
ALTER TABLE events ADD COLUMN search_vector tsvector;

-- Создать индекс
CREATE INDEX events_search_idx ON events USING GIN(search_vector);

-- Заполнить
UPDATE events SET search_vector =
    to_tsvector('russian', title || ' ' || description);

-- Использовать в запросе
SELECT * FROM events
WHERE search_vector @@ plainto_tsquery('russian', 'concert')
ORDER BY ts_rank(search_vector, query) DESC;
```

**Опция 2: Elasticsearch (более мощно, но сложнее)**

```go
type SearchService struct {
    es *elasticsearch.Client
}

func (s *SearchService) IndexEvent(ctx context.Context, event Event) error {
    doc, _ := json.Marshal(event)
    req := esapi.IndexRequest{
        Index:      "events",
        DocumentID: event.ID,
        Body:       bytes.NewReader(doc),
    }
    return req.Do(ctx, s.es)
}

func (s *SearchService) Search(ctx context.Context, query string) ([]Event, error) {
    // ...
}
```

**Рекомендация:** Начать с PostgreSQL FTS (встроено, нет доп. инфраструктуры)

**Файлы для изменения:**

- `internal/migrations/` — новая миграция
- `internal/search/search.go` — поиск логика
- `internal/handlers/search.go` — HTTP handler

**Приблизительное время:** 6-8 часов (PostgreSQL) или 12-16 часов (Elasticsearch)

---

### 2.3 Event Analytics

**Какие метрики собирать:**

```go
type EventAnalytics struct {
    EventID    string
    DateID     string    // YYYY-MM-DD
    ViewCount  int       // Сколько раз было shown
    LikeCount  int       // Сколько раз было liked
    BookCount  int       // Сколько раз было booked
    SkipCount  int       // Сколько раз было skipped
    UpdatedAt  time.Time
}
```

**Как собирать:**

1. При каждом action (like, skip, book) — инкрементировать счётчик
2. Использовать Redis для live счётчиков
3. Каждый час/день перемещать в PostgreSQL для истории
4. Вычислять конверсию: LikeCount / ViewCount

**Шаги:**

1. Добавить миграцию для таблицы event_analytics
2. Добавить Redis counters (INCR)
3. Создать batch job для flush в PostgreSQL
4. Добавить endpoint /analytics/event/{id}

**Файлы для создания:**

- `internal/analytics/analytics.go` — основная логика
- `internal/analytics/aggregator.go` — batch job
- `internal/handlers/analytics.go` — HTTP endpoints

**Приблизительное время:** 8-10 часов

---

## Quick Wins для быстрого улучшения

### 1. Автоскролл в чате (30 мин)

```javascript
// EventCommentChat.js
const flatListRef = useRef(null);

const handleSendComment = useCallback(async () => {
  // ... send logic
  await fetchComments();

  // Scroll to bottom
  setTimeout(() => {
    flatListRef.current?.scrollToEnd({ animated: true });
  }, 100);
}, []);

return (
  <FlatList
    ref={flatListRef}
    onContentSizeChange={() => flatListRef.current?.scrollToEnd()}
    // ...
  />
);
```

### 2. Фильтры в admin-panel (1-2 часа)

```javascript
// PendingEventsScreen.js добавить:
const [statusFilter, setStatusFilter] = useState("all");
const [typeFilter, setTypeFilter] = useState("all");

const filteredEvents = events.filter((e) => {
  if (statusFilter !== "all" && e.status !== statusFilter) return false;
  if (typeFilter !== "all" && e.type !== typeFilter) return false;
  return true;
});
```

### 3. Dark mode (2-3 часа)

```javascript
// Использовать useColorScheme() из react-native
const colorScheme = useColorScheme();
const isDark = colorScheme === 'dark';

const colors = {
    light: { bg: '#fff', text: '#000' },
    dark: { bg: '#1a1a1a', text: '#fff' }
};

return (
    <View style={{
        backgroundColor: colors[isDark ? 'dark' : 'light'].bg
    }}>
```

---

## Dependencies для различных задач

### Real-time

- gorilla/websocket
- go-redis (уже есть)

### Search

- github.com/elastic/go-elasticsearch (Elasticsearch)
- github.com/lib/pq (PostgreSQL FTS встроено)

### Monitoring

- github.com/prometheus/client_golang
- Grafana (отдельно)

### Load Testing

- k6 (js-based, легко)
- artillery (node-based)
- locust (python-based)

### Testing

- testify/assert
- testify/mock
- pgx/pgmock
- goredis/testutil

---

## Оценка сложности

| Задача           | Сложность | Время  | Блокеры           |
| ---------------- | --------- | ------ | ----------------- |
| Автоскролл       | ⭐        | 30м    | -                 |
| Фильтры          | ⭐        | 1-2ч   | -                 |
| Redis bitmap     | ⭐⭐      | 4-6ч   | Redis knowledge   |
| Batch ops        | ⭐⭐      | 6-8ч   | DB profiling      |
| WebSocket        | ⭐⭐⭐    | 10-12ч | Async complexity  |
| Full-text search | ⭐⭐⭐    | 6-16ч  | Depends on option |
| Analytics        | ⭐⭐      | 8-10ч  | -                 |
| Redis Sentinel   | ⭐⭐⭐    | 12-16ч | Ops knowledge     |
