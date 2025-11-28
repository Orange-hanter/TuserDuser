# Проверка Metrics и Prometheus - План Исправлений

## 1. Регистрация /metrics endpoint

**Файл:** `event-api/cmd/server/main.go`

**Проблема:** Endpoint `/metrics` не зарегистрирован в маршрутах.

**Решение:** Добавить в функцию `buildHTTPHandler`:

```go
// Поместить перед r.Route("/v1", ...)
r.Handle("/metrics", handlers.MetricsHandler)
```

**Почему:** Без этого Prometheus не сможет собирать метрики с вашего API.

---

## 2. Инициализация RedisMetrics

**Файл:** `event-api/cmd/server/main.go` (функция `run`)

**Проблема:** Нет создания глобального экземпляра `RedisMetrics`.

**Решение:** Добавить после инициализации Redis:

```go
// После initRedis(cfg)
var redisMetrics *metrics.RedisMetrics
if redis != nil {
    redisMetrics = metrics.NewRedisMetrics()
    logger.Log.Info("✅ Prometheus metrics initialized")
} else {
    logger.Log.Warn("⚠️  Redis not available, metrics disabled")
}
```

---

## 3. Передача метрик в сервисы

**Файл:** `event-api/internal/redis/client.go`

**Проблема:** Redis операции не регистрируются в метриках.

**Решение:**

- Добавить поле `metrics *metrics.RedisMetrics` в структуру `Client`
- Записывать метрики при выполнении команд

**Пример:**

```go
// В методе Execute или BeforeExecute
if c.metrics != nil {
    c.metrics.RecordCommand(cmd.String())
    defer func(start time.Time) {
        duration := time.Since(start).Seconds()
        c.metrics.RecordCommandDuration(cmd.String(), duration)
        if err != nil {
            c.metrics.RecordCommandError(cmd.String(), err.Error())
        }
    }(time.Now())
}
```

---

## 4. Метрики Discovery Engine

**Файл:** `event-api/internal/discovery/engine.go`

**Проблема:** Метрики очередей Discovery не заполняются.

**Решение:** Передать `metrics.RedisMetrics` в `Engine` и вызывать:

```go
// При операциях с очередью
metrics.SetQueueSize(queueName, float64(size))
metrics.RecordQueueOperation(queueName, "push")
metrics.SetQueueErrorRate(queueName, errorRate)
```

---

## 5. Проверка через curl

После исправлений протестировать:

```bash
curl http://localhost:8080/metrics | head -20
```

Должны увидеть метрики вроде:

```
# HELP redis_commands_total Total number of Redis commands executed
# TYPE redis_commands_total counter
redis_commands_total{command="GET"} 0
```

---

## Метрики которые собираются:

### Redis операции

- ✅ `redis_commands_total` - счетчик команд
- ✅ `redis_command_errors_total` - ошибки
- ✅ `redis_command_duration_seconds` - время выполнения

### Redis здоровье

- ✅ `redis_connections_active` - активные подключения
- ✅ `redis_connection_errors_total` - ошибки подключения

### Redis данные

- ✅ `redis_memory_usage_bytes` - использованная память
- ✅ `redis_keys_total` - количество ключей

### Discovery Engine

- ✅ `discovery_queue_size` - размер очередей
- ✅ `discovery_queue_ops_total` - операции с очередями
- ✅ `discovery_queue_error_rate` - частота ошибок

---

## Порядок исправлений (приоритет):

1. **ВЫСОКИЙ:** Зарегистрировать `/metrics` endpoint (без этого ничего не работает)
2. **ВЫСОКИЙ:** Инициализировать `RedisMetrics`
3. **СРЕДНИЙ:** Добавить запись метрик в Redis операции
4. **СРЕДНИЙ:** Добавить запись метрик в Discovery Engine

---

## Проверка корректности

После всех исправлений выполнить:

```bash
# 1. Запустить сервер
go run ./cmd/server/main.go

# 2. В другом терминале - получить метрики
curl -s http://localhost:8080/metrics | grep redis_

# 3. Выполнить операцию (например, подписка на событие)
curl -X POST http://localhost:8080/v1/api/users/me/events/1/subscribe \
  -H "Authorization: Bearer YOUR_TOKEN"

# 4. Проверить что метрики увеличились
curl -s http://localhost:8080/metrics | grep redis_commands_total
```
