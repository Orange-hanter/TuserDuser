# Redis в Event API

Полное руководство по интеграции Redis в проект event-api.

## Содержание

1. [Что такое Redis](#что-такое-redis)
2. [Зачем Redis в проекте](#зачем-redis-в-проекте)
3. [Архитектура](#архитектура)
4. [Быстрый старт](#быстрый-старт)
5. [Конфигурация](#конфигурация)
6. [Схема данных](#схема-данных)
7. [Использование в коде](#использование-в-коде)
8. [CLI и мониторинг](#cli-и-мониторинг)
9. [Troubleshooting](#troubleshooting)

---

## Что такое Redis

**Redis** (Remote Dictionary Server) — это высокопроизводительное хранилище данных в памяти, работающее по принципу "ключ-значение".

### Ключевые характеристики

| Характеристика      | Описание                                            |
| ------------------- | --------------------------------------------------- |
| **Скорость**        | ~100,000 операций/сек на одном ядре                 |
| **Типы данных**     | String, List, Set, Hash, Sorted Set, Stream         |
| **TTL**             | Автоматическое удаление ключей по истечении времени |
| **Персистентность** | RDB снапшоты + AOF журнал                           |
| **Репликация**      | Master-Slave, Redis Sentinel, Redis Cluster         |

### Сравнение с альтернативами

```
┌──────────────────┬─────────────┬─────────────┬─────────────┐
│                  │   Redis     │  PostgreSQL │  In-Memory  │
├──────────────────┼─────────────┼─────────────┼─────────────┤
│ Скорость         │ ⚡ ~0.1ms   │ ~1-10ms     │ ⚡ ~0.01ms  │
│ Персистентность  │ ✅ Да       │ ✅ Да       │ ❌ Нет      │
│ Масштабирование  │ ✅ Кластер  │ ✅ Репликация│ ❌ Нет      │
│ TTL              │ ✅ Встроен  │ ❌ Ручной   │ ❌ Ручной   │
│ Перезапуск       │ ✅ Сохраняет│ ✅ Сохраняет│ ❌ Теряет   │
└──────────────────┴─────────────┴─────────────┴─────────────┘
```

---

## Зачем Redis в проекте

Redis используется для хранения **временных данных** с автоматическим удалением:

### 1. Коды верификации (Auth)

```
Key:   verify:{email}
Value: 6-значный код
TTL:   10 минут
```

При регистрации пользователь получает код подтверждения email. Код автоматически истекает через 10 минут.

### 2. Token Blacklist (Auth)

```
Key:   blacklist:{jwt_token}
Value: "1"
TTL:   время жизни токена (1 час)
```

При logout токен добавляется в blacklist. Middleware проверяет каждый запрос на наличие токена в blacklist.

### 3. Discovery Queue (Discovery Module)

```
Key:   queue:user:{userID}
Value: JSON с очередью событий
TTL:   30 дней
```

Состояние swipe-очереди пользователя: текущее событие, primary/secondary списки.

### 4. Discovery History (Discovery Module)

```
Key:   history:user:{userID}
Value: Redis List с историей действий
TTL:   7 дней

Key:   last-action:user:{userID}:event:{eventID}
Value: JSON с последним действием
TTL:   7 дней
```

"Горячий" кэш истории для быстрого доступа. Полная история хранится в PostgreSQL.

---

## Архитектура

### До Redis (проблемы)

```
┌─ Server 1 ─┐    ┌─ Server 2 ─┐
│ Memory     │    │ Memory     │
│ (isolated) │    │ (isolated) │
└────────────┘    └────────────┘
      │                 │
      └────────┬────────┘
               ▼
         ┌──────────┐
         │PostgreSQL│
         └──────────┘

❌ Данные теряются при перезапуске
❌ Каждый инстанс имеет свою копию данных
❌ Нет возможности горизонтального масштабирования
```

### После Redis

```
┌─ Server 1 ─┐    ┌─ Server 2 ─┐
│   App      │    │   App      │
└─────┬──────┘    └──────┬─────┘
      │                  │
      └────────┬─────────┘
               ▼
         ┌──────────┐
         │  Redis   │  ← Временные данные, кэш
         └────┬─────┘
              │
         ┌────▼─────┐
         │PostgreSQL│  ← Постоянные данные
         └──────────┘

✅ Данные сохраняются при перезапуске
✅ Все инстансы работают с одними данными
✅ Горизонтальное масштабирование
```

### Graceful Degradation

Если Redis недоступен, приложение продолжает работать:

```go
if redis != nil {
    // Используем Redis
    queueRepo = NewRedisQueueRepository(...)
    historyRepo = NewRedisHistoryRepository(...)
} else {
    // Fallback на in-memory + PostgreSQL
    queueRepo = NewInMemoryQueueRepository()
    historyRepo = NewPostgresHistoryRepository(...)
    logger.Warn("Redis not available, using fallback")
}
```

⚠️ В режиме fallback данные не сохраняются между перезапусками.

---

## Быстрый старт

### 1. Запуск Redis

**Docker Compose (рекомендуется):**

```bash
docker-compose up -d redis
```

**Homebrew (macOS):**

```bash
brew install redis
brew services start redis
```

**Docker напрямую:**

```bash
docker run -d -p 6379:6379 --name redis redis:7-alpine
```

### 2. Проверка подключения

```bash
redis-cli PING
# Ответ: PONG
```

Если установлен пароль:

```bash
redis-cli -a devpass PING
```

### 3. Настройка приложения

Добавьте в `.env`:

```bash
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=devpass
REDIS_DB=0

# TTL для Discovery (опционально)
DISCOVERY_HISTORY_TTL=604800    # 7 дней в секундах
DISCOVERY_QUEUE_TTL=2592000     # 30 дней в секундах
```

### 4. Запуск сервера

```bash
cd event-api
go build -o bin/server ./cmd/server
./bin/server
```

В логах должно появиться:

```
✅ Redis queue repository initialized (ttl_seconds: 2592000)
✅ Redis history repository initialized (ttl_seconds: 604800)
```

---

## Конфигурация

### Переменные окружения

| Переменная              | Описание                 | По умолчанию        |
| ----------------------- | ------------------------ | ------------------- |
| `REDIS_HOST`            | Хост Redis               | `localhost`         |
| `REDIS_PORT`            | Порт Redis               | `6379`              |
| `REDIS_PASSWORD`        | Пароль                   | пусто               |
| `REDIS_DB`              | Номер базы данных (0-15) | `0`                 |
| `DISCOVERY_HISTORY_TTL` | TTL истории в секундах   | `604800` (7 дней)   |
| `DISCOVERY_QUEUE_TTL`   | TTL очереди в секундах   | `2592000` (30 дней) |

### Docker Compose конфигурация

```yaml
redis:
  image: redis:7-alpine
  container_name: event_api_redis
  ports:
    - "6379:6379"
  command: redis-server --appendonly yes --requirepass devpass
  volumes:
    - redis_data:/data
  healthcheck:
    test: ["CMD", "redis-cli", "-a", "devpass", "ping"]
    interval: 10s
    timeout: 5s
    retries: 5
```

**Параметры:**

- `--appendonly yes` — включает AOF персистентность
- `--requirepass devpass` — устанавливает пароль
- `redis_data:/data` — сохраняет данные между перезапусками

---

## Схема данных

### Паттерны ключей

```
verify:{email}                         # Код верификации
blacklist:{jwt_token}                  # Отозванный токен
queue:user:{userID}                    # Очередь Discovery
history:user:{userID}                  # История действий (List)
last-action:user:{userID}:event:{eventID}  # Последнее действие
```

### Примеры данных

**Код верификации:**

```bash
> GET verify:user@example.com
"123456"

> TTL verify:user@example.com
587  # секунд осталось
```

**Очередь Discovery:**

```bash
> GET queue:user:abc123
{
  "user_id": "abc123",
  "filter": {"category": "music"},
  "primary": ["event1", "event2", "event3"],
  "secondary": ["event5", "event6"],
  "conflict_flags": {}
}
```

**История (Redis List):**

```bash
> LRANGE history:user:abc123 0 2
1) {"action":"like","event_id":"ev1","timestamp":"..."}
2) {"action":"skip","event_id":"ev2","timestamp":"..."}
3) {"action":"like","event_id":"ev3","timestamp":"..."}
```

---

## Использование в коде

### Инициализация клиента

```go
import "event-api/internal/redis"

config := &redis.Config{
    Host:     cfg.RedisHost,
    Port:     cfg.RedisPort,
    Password: cfg.RedisPassword,
    DB:       cfg.RedisDB,
}

client, err := redis.NewClient(config, logger)
if err != nil {
    log.Fatal("Failed to connect to Redis:", err)
}
defer client.Close()
```

### Базовые операции

**Set с TTL:**

```go
err := client.Set(ctx, "key", "value", 10*time.Minute)
```

**Get:**

```go
value, err := client.Get(ctx, "key")
if err == redis.Nil {
    // Ключ не существует или истёк
}
```

**Delete:**

```go
err := client.Del(ctx, "key")
```

**Exists:**

```go
exists, err := client.Exists(ctx, "key")
```

### Работа со списками (History)

```go
// Добавить элемент в начало списка
err := client.LPush(ctx, "history:user:abc123", jsonData)

// Обрезать до 100 элементов
err := client.LTrim(ctx, "history:user:abc123", 0, 99)

// Получить все элементы
items, err := client.LRange(ctx, "history:user:abc123", 0, -1)

// Получить количество элементов
count, err := client.LLen(ctx, "history:user:abc123")
```

### Транзакции

```go
pipe := client.Pipeline()
pipe.Set(ctx, "key1", "val1", 0)
pipe.Set(ctx, "key2", "val2", 0)
pipe.Incr(ctx, "counter")
_, err := pipe.Exec(ctx)
```

---

## CLI и мониторинг

### Подключение к Redis CLI

```bash
# Docker
docker exec -it event_api_redis redis-cli -a devpass

# Локально
redis-cli -h localhost -p 6379 -a devpass
```

### Основные команды

| Команда                    | Описание                             |
| -------------------------- | ------------------------------------ |
| `PING`                     | Проверить соединение                 |
| `KEYS pattern`             | Найти ключи по паттерну              |
| `GET key`                  | Получить значение                    |
| `SET key value EX seconds` | Установить с TTL                     |
| `DEL key`                  | Удалить ключ                         |
| `TTL key`                  | Время до истечения                   |
| `EXPIRE key seconds`       | Установить TTL                       |
| `EXISTS key`               | Проверить существование              |
| `TYPE key`                 | Тип значения                         |
| `DBSIZE`                   | Количество ключей в БД               |
| `INFO`                     | Информация о сервере                 |
| `MONITOR`                  | Мониторинг команд в реальном времени |

### Примеры использования

```bash
# Все ключи верификации
KEYS "verify:*"

# Все очереди пользователей
KEYS "queue:user:*"

# Количество очередей
KEYS "queue:user:*" | wc -l

# Просмотр очереди конкретного пользователя
GET queue:user:abc123

# История пользователя (первые 5)
LRANGE history:user:abc123 0 4

# Проверить TTL
TTL queue:user:abc123

# Удалить данные пользователя
DEL queue:user:abc123
DEL history:user:abc123
```

### Мониторинг

```bash
# Мониторинг всех команд в реальном времени
MONITOR

# Информация о памяти
INFO memory

# Статистика команд
INFO commandstats

# Статистика по базам данных
INFO keyspace
```

### Проверка состояния

```bash
# Использование памяти
redis-cli INFO memory | grep used_memory_human

# Количество подключений
redis-cli INFO clients | grep connected_clients

# Пиковое использование памяти
redis-cli INFO memory | grep peak
```

---

## Troubleshooting

### Connection refused

```
Error: dial tcp 127.0.0.1:6379: connect: connection refused
```

**Решение:**

```bash
# Проверить статус Redis
docker ps | grep redis

# Запустить если не работает
docker-compose up -d redis

# Или для brew
brew services start redis
```

### WRONGPASS invalid password

```
Error: WRONGPASS invalid username-password pair
```

**Решение:**

- Проверить `REDIS_PASSWORD` в `.env`
- Убедиться что пароль совпадает с `--requirepass` в docker-compose

### i/o timeout

```
Error: redis get error: i/o timeout
```

**Решение:**

```bash
# Проверить сетевую доступность
telnet localhost 6379

# Проверить настройки хоста/порта
echo $REDIS_HOST $REDIS_PORT
```

### Память растёт слишком быстро

**Диагностика:**

```bash
redis-cli INFO memory
redis-cli DBSIZE
redis-cli KEYS "*" | wc -l
```

**Решение:**

- Уменьшить TTL в конфигурации
- Настроить `maxmemory-policy` для автоматического удаления

### Ключ не найден

```bash
# Проверить существование
EXISTS queue:user:abc123

# Проверить TTL (если -2 — ключ не существует)
TTL queue:user:abc123

# Проверить все ключи пользователя
KEYS "*abc123*"
```

### Данные потеряны после перезапуска

**Причина:** Не настроена персистентность.

**Решение:** Убедиться что в docker-compose есть:

```yaml
command: redis-server --appendonly yes
volumes:
  - redis_data:/data
```

---

## Полезные ссылки

- [Redis Documentation](https://redis.io/documentation)
- [go-redis GitHub](https://github.com/redis/go-redis)
- [Redis Commands Reference](https://redis.io/commands)
- [Redis Data Types](https://redis.io/docs/data-types/)

---

**См. также:** [Redis Best Practices](./REDIS_BEST_PRACTICES.md)
