# Redis Best Practices

Рекомендации по использованию Redis в production для event-api.

## Содержание

1. [Production Configuration](#production-configuration)
2. [Персистентность](#персистентность)
3. [High Availability](#high-availability)
4. [Безопасность](#безопасность)
5. [Производительность](#производительность)
6. [Мониторинг](#мониторинг)
7. [Операционные задачи](#операционные-задачи)
8. [Чеклист для Production](#чеклист-для-production)

---

## Production Configuration

### Рекомендуемая конфигурация redis.conf

```conf
# Память
maxmemory 1gb
maxmemory-policy allkeys-lru

# Персистентность (выберите один вариант)
# Вариант 1: AOF (рекомендуется для надёжности)
appendonly yes
appendfsync everysec

# Вариант 2: RDB (для быстрых снапшотов)
save 900 1
save 300 10
save 60 10000

# Сеть
bind 127.0.0.1
port 6379
timeout 300
tcp-keepalive 60

# Безопасность
requirepass your_strong_password_here

# Логирование
loglevel notice
logfile /var/log/redis/redis.log
```

### Переменные окружения для Production

```bash
REDIS_HOST=redis.internal.yourcompany.com
REDIS_PORT=6379
REDIS_PASSWORD=strong_random_password_32_chars
REDIS_DB=0

# Увеличенные TTL для production
DISCOVERY_HISTORY_TTL=604800     # 7 дней
DISCOVERY_QUEUE_TTL=2592000      # 30 дней
```

### Docker Compose для Production

```yaml
redis:
  image: redis:7-alpine
  container_name: redis_prod
  restart: always
  ports:
    - "127.0.0.1:6379:6379" # Только localhost
  command: >
    redis-server
    --appendonly yes
    --appendfsync everysec
    --maxmemory 1gb
    --maxmemory-policy allkeys-lru
    --requirepass ${REDIS_PASSWORD}
  volumes:
    - redis_data:/data
  healthcheck:
    test: ["CMD", "redis-cli", "-a", "${REDIS_PASSWORD}", "ping"]
    interval: 10s
    timeout: 5s
    retries: 5
  deploy:
    resources:
      limits:
        memory: 1.5G
      reservations:
        memory: 512M
```

---

## Персистентность

Redis поддерживает два механизма сохранения данных на диск.

### AOF (Append-Only File)

**Рекомендуется для event-api** — максимальная надёжность данных.

```conf
appendonly yes
appendfsync everysec   # Оптимальный баланс производительности/надёжности
```

| appendfsync | Описание                          | Потеря данных  |
| ----------- | --------------------------------- | -------------- |
| `always`    | Синхронизация после каждой записи | 0              |
| `everysec`  | Синхронизация раз в секунду       | до 1 секунды   |
| `no`        | Синхронизация по решению ОС       | непредсказуемо |

**Оптимизация AOF:**

```bash
# Автоматическое переписывание AOF при росте
auto-aof-rewrite-percentage 100
auto-aof-rewrite-min-size 64mb
```

### RDB (Redis Database Snapshots)

Периодические снимки всей базы. Быстрее при восстановлении, но возможна потеря данных.

```conf
save 900 1      # Сохранять каждые 15 мин если была 1+ запись
save 300 10     # Каждые 5 мин если 10+ записей
save 60 10000   # Каждую минуту если 10000+ записей
```

### Гибридный подход (Redis 4.0+)

Комбинирует RDB + AOF:

```conf
aof-use-rdb-preamble yes
```

---

## High Availability

### Redis Sentinel

Автоматический failover при падении master.

```
┌─────────────┐
│  Sentinel 1 │
└──────┬──────┘
       │ мониторит
┌──────▼──────┐    репликация    ┌─────────────┐
│   Master    │◄─────────────────│   Slave 1   │
└─────────────┘                  └─────────────┘
                                       ▲
                                       │
                                 ┌─────┴─────┐
                                 │  Slave 2  │
                                 └───────────┘
```

**Конфигурация Sentinel:**

```conf
sentinel monitor mymaster 127.0.0.1 6379 2
sentinel down-after-milliseconds mymaster 5000
sentinel failover-timeout mymaster 30000
sentinel parallel-syncs mymaster 1
```

### Redis Cluster

Для горизонтального масштабирования (шардинг данных).

```
┌─────────────────────────────────────────────┐
│              Redis Cluster                   │
├─────────────┬─────────────┬─────────────────┤
│  Node 1     │  Node 2     │  Node 3         │
│  Slots 0-5k │  Slots 5k-10k│  Slots 10k-16k │
└─────────────┴─────────────┴─────────────────┘
```

⚠️ **Для event-api** достаточно Sentinel, если нагрузка < 100K ops/sec.

---

## Безопасность

### 1. Аутентификация

**Обязательно установите пароль:**

```conf
requirepass your_very_long_random_password_minimum_32_characters
```

**Генерация безопасного пароля:**

```bash
openssl rand -base64 32
```

### 2. Сетевая изоляция

```conf
# Привязка только к внутренним интерфейсам
bind 127.0.0.1 10.0.0.1

# Отключение опасных команд
rename-command FLUSHDB ""
rename-command FLUSHALL ""
rename-command DEBUG ""
rename-command CONFIG ""
```

### 3. TLS шифрование (Redis 6+)

```conf
tls-port 6379
port 0
tls-cert-file /path/to/redis.crt
tls-key-file /path/to/redis.key
tls-ca-cert-file /path/to/ca.crt
```

**Подключение с TLS в Go:**

```go
client := redis.NewClient(&redis.Options{
    Addr: "redis:6379",
    TLSConfig: &tls.Config{
        MinVersion: tls.VersionTLS12,
    },
})
```

### 4. Firewall

```bash
# Только app-серверы могут подключаться к Redis
iptables -A INPUT -p tcp --dport 6379 -s 10.0.0.0/24 -j ACCEPT
iptables -A INPUT -p tcp --dport 6379 -j DROP
```

---

## Производительность

### Connection Pooling

```go
client := redis.NewClient(&redis.Options{
    Addr:         "redis:6379",
    Password:     "password",
    DB:           0,

    // Pool settings
    PoolSize:     100,          // Максимум соединений
    MinIdleConns: 10,           // Минимум idle соединений
    PoolTimeout:  4 * time.Second,

    // Timeouts
    DialTimeout:  5 * time.Second,
    ReadTimeout:  3 * time.Second,
    WriteTimeout: 3 * time.Second,
})
```

### Pipelining

Выполняйте несколько команд за один round-trip:

```go
// ❌ Плохо: 3 round-trips
client.Set(ctx, "key1", "val1", 0)
client.Set(ctx, "key2", "val2", 0)
client.Set(ctx, "key3", "val3", 0)

// ✅ Хорошо: 1 round-trip
pipe := client.Pipeline()
pipe.Set(ctx, "key1", "val1", 0)
pipe.Set(ctx, "key2", "val2", 0)
pipe.Set(ctx, "key3", "val3", 0)
_, err := pipe.Exec(ctx)
```

### Избегайте KEYS

```bash
# ❌ Плохо: блокирует Redis при большом количестве ключей
KEYS "queue:user:*"

# ✅ Хорошо: неблокирующий итератор
SCAN 0 MATCH "queue:user:*" COUNT 100
```

### Сжатие значений

Для больших JSON объектов используйте сжатие:

```go
import "compress/gzip"

// Сжатие перед сохранением
var buf bytes.Buffer
gz := gzip.NewWriter(&buf)
gz.Write(jsonData)
gz.Close()
client.Set(ctx, key, buf.Bytes(), ttl)

// Распаковка при чтении
data, _ := client.Get(ctx, key).Bytes()
gz, _ := gzip.NewReader(bytes.NewReader(data))
decompressed, _ := io.ReadAll(gz)
```

### Memory Optimization

```conf
# Включить сжатие для мелких объектов
hash-max-ziplist-entries 512
hash-max-ziplist-value 64
list-max-ziplist-size -2
```

---

## Мониторинг

### Prometheus + Grafana

**Установка redis_exporter:**

```bash
docker run -d \
  --name redis_exporter \
  -p 9121:9121 \
  -e REDIS_ADDR=redis://redis:6379 \
  -e REDIS_PASSWORD=password \
  oliver006/redis_exporter
```

**prometheus.yml:**

```yaml
scrape_configs:
  - job_name: "redis"
    static_configs:
      - targets: ["redis_exporter:9121"]
```

### Ключевые метрики для мониторинга

| Метрика                            | Описание                | Alert threshold     |
| ---------------------------------- | ----------------------- | ------------------- |
| `redis_memory_used_bytes`          | Использование памяти    | > 80% maxmemory     |
| `redis_connected_clients`          | Активные подключения    | > 90% maxclients    |
| `redis_commands_processed_total`   | Ops/sec                 | зависит от нагрузки |
| `redis_keyspace_hits_total`        | Cache hits              | < 90% hit rate      |
| `redis_keyspace_misses_total`      | Cache misses            | отслеживать рост    |
| `redis_blocked_clients`            | Заблокированные клиенты | > 0                 |
| `redis_rejected_connections_total` | Отклонённые подключения | > 0                 |

### Алерты

```yaml
# alertmanager rules
groups:
  - name: redis
    rules:
      - alert: RedisHighMemory
        expr: redis_memory_used_bytes / redis_memory_max_bytes > 0.8
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Redis memory usage > 80%"

      - alert: RedisDown
        expr: redis_up == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Redis instance is down"
```

### Логирование команд

```bash
# Мониторинг в реальном времени (для отладки)
redis-cli MONITOR

# Slow log (команды дольше 10ms)
CONFIG SET slowlog-log-slower-than 10000
SLOWLOG GET 10
```

---

## Операционные задачи

### Backup

**Ручной backup:**

```bash
# RDB snapshot
redis-cli BGSAVE
cp /var/lib/redis/dump.rdb /backup/redis-$(date +%Y%m%d).rdb

# AOF
cp /var/lib/redis/appendonly.aof /backup/
```

**Автоматический backup (cron):**

```bash
0 */6 * * * redis-cli BGSAVE && cp /var/lib/redis/dump.rdb /backup/redis-$(date +\%Y\%m\%d-\%H).rdb
```

### Restore

```bash
# Остановить Redis
redis-cli SHUTDOWN

# Заменить файл данных
cp /backup/redis-20251204.rdb /var/lib/redis/dump.rdb

# Запустить Redis
redis-server /etc/redis/redis.conf
```

### Memory Management

**Очистка всех данных (осторожно!):**

```bash
redis-cli FLUSHDB      # Текущая БД
redis-cli FLUSHALL     # Все БД
```

**Удаление по паттерну:**

```bash
# Удалить все очереди пользователей
redis-cli --scan --pattern "queue:user:*" | xargs redis-cli DEL
```

**Анализ памяти:**

```bash
redis-cli MEMORY DOCTOR
redis-cli MEMORY USAGE key_name
redis-cli INFO memory
```

### Обновление Redis

1. Создать backup
2. Обновить slave первым
3. Failover на slave
4. Обновить бывший master
5. Failover обратно (опционально)

---

## Чеклист для Production

### Перед деплоем

- [ ] Установлен сильный пароль (минимум 32 символа)
- [ ] Redis слушает только на внутренних интерфейсах
- [ ] Настроен firewall
- [ ] Включена персистентность (AOF или RDB)
- [ ] Настроен maxmemory и maxmemory-policy
- [ ] Отключены опасные команды (FLUSHDB, DEBUG)
- [ ] Настроен connection pool в приложении

### Мониторинг

- [ ] Установлен redis_exporter
- [ ] Настроены алерты на память, downtime, rejected connections
- [ ] Включён slowlog
- [ ] Настроены дашборды в Grafana

### Операции

- [ ] Настроен автоматический backup
- [ ] Документирована процедура restore
- [ ] Протестирован failover (если используется Sentinel)
- [ ] Есть runbook для типичных инцидентов

### Безопасность

- [ ] TLS между приложением и Redis (production)
- [ ] Redis не доступен из интернета
- [ ] Пароли хранятся в secrets manager
- [ ] Логи не содержат sensitive данных

---

## Полезные команды для Production

```bash
# Проверить состояние
redis-cli INFO server
redis-cli INFO replication
redis-cli INFO persistence

# Диагностика проблем
redis-cli MEMORY DOCTOR
redis-cli SLOWLOG GET 10
redis-cli CLIENT LIST

# Статистика
redis-cli INFO stats
redis-cli INFO commandstats

# Graceful shutdown
redis-cli SHUTDOWN SAVE
```

---

**См. также:** [Redis Guide](./REDIS.md)
