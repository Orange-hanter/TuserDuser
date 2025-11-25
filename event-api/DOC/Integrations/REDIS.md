# Redis Integration

## Описание

Redis интегрирован в проект для хранения временных данных:

1. **Коды верификации** - хранятся с TTL 10 минут
2. **Token Blacklist** - отозванные JWT токены с TTL равным времени жизни токена

## Преимущества использования Redis

### По сравнению с in-memory хранением

- ✅ Персистентность при перезапуске приложения
- ✅ Масштабируемость - несколько инстансов могут работать с одним Redis
- ✅ Автоматическое удаление истекших ключей (TTL)
- ✅ Высокая производительность

### По сравнению с PostgreSQL

- ✅ Быстрее для временных данных
- ✅ Встроенная поддержка TTL
- ✅ Меньше нагрузка на основную БД
- ✅ Оптимизирован для key-value операций

## Архитектура хранения

### Коды верификации

```bash
Key: verify:{email}
Value: {6-значный код}
TTL: 10 минут
```

Пример:

```bash
verify:user@example.com = "123456"
```

### Token Blacklist

```json
Key: blacklist:{jwt_token}
Value: "1"
TTL: время жизни токена (по умолчанию 1 час)
```

Пример:

```bash
blacklist:eyJhbGc... = "1"
```

## Настройка

### Переменные окружения (.env)

````env
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=devpass
REDIS_DB=0
```bash
### Docker Compose

Redis запускается автоматически:

```bash
docker-compose up -d redis
````

Параметры:

- Image: `redis:7-alpine`
- Persistence: AOF (Append Only File)
- Password защита
- Health check каждые 10 секунд

## Использование в коде

### Инициализация клиента

````go
redisConfig := &redisClient.Config{
    Host:     cfg.RedisHost,
    Port:     cfg.RedisPort,
    Password: cfg.RedisPassword,
    DB:       cfg.RedisDB,
}

redis, err := redisClient.NewClient(redisConfig, logger.Log)
if err != nil {
    log.Fatal(err)
}
defer redis.Close()
```bash
### Основные операции

#### Set с TTL

```go
ctx := context.Background()
err := redis.Set(ctx, "key", "value", 10*time.Minute)
````

#### Get

````go
value, err := redis.Get(ctx, "key")
if err != nil {
    // Ключ не найден или истек TTL
}
```bash
#### Delete

```go
err := redis.Del(ctx, "key")
````

#### Exists

````go
exists, err := redis.Exists(ctx, "key")
```bash
## API методы с Redis

### 1. POST /v1/api/auth/register

**Использует Redis для:**

- Сохранения кода верификации

**Поток:**

````

1. Создание пользователя в PostgreSQL
2. Генерация кода верификации
3. Сохранение кода в Redis: verify:{email} = {code} (TTL: 10 мин)
4. Асинхронная отправка кода

```bash
### 2. POST /v1/api/auth/verify

**Использует Redis для:**

- Проверки кода верификации

**Поток:**

```

1. Получение кода из Redis: verify:{email}
2. Проверка соответствия кодов
3. Обновление статуса пользователя в PostgreSQL
4. Удаление кода из Redis
5. Генерация JWT токена

```bash
### 3. POST /v1/api/auth/logout

**Использует Redis для:**

- Добавления токена в blacklist

**Поток:**

```

1. Извлечение токена из Authorization header
2. Добавление в blacklist: blacklist:{token} = "1" (TTL: время жизни токена)

```bash
### 4. GET /v1/api/auth/me (и другие protected endpoints)

**Использует Redis для:**

- Проверки токена в blacklist

**Поток:**

```

1. Проверка токена в blacklist: EXISTS blacklist:{token}
2. Если найден - отклонить запрос
3. Если нет - продолжить валидацию токена

````bash
## Мониторинг

### Подключение к Redis CLI

```bash
## Через Docker
## Через Docker
docker exec -it event_api_redis redis-cli -a devpass

## Локально (если установлен Redis CLI)
## Локально (если установлен Redis CLI)
redis-cli -h localhost -p 6379 -a devpass
````

### Полезные команды

````bash
## Посмотреть все ключи
## Посмотреть все ключи
KEYS *

## Посмотреть значение ключа
## Посмотреть значение ключа
GET verify:user@example.com

## Проверить TTL ключа
## Проверить TTL ключа
TTL verify:user@example.com

## Посмотреть информацию о Redis
## Посмотреть информацию о Redis
INFO

## Количество ключей в БД
## Количество ключей в БД
DBSIZE

## Очистить все ключи (осторожно!)
## Очистить все ключи (осторожно!)
FLUSHDB
```bash
### Мониторинг активности в реальном времени

```bash
## Мониторинг всех команд
## Мониторинг всех команд
MONITOR

## Статистика
## Статистика
INFO stats
````

## Производительность

### Текущие настройки

- Workers: 5
- Максимальное количество подключений: По умолчанию (определяется go-redis)
- Timeout: 5 секунд для проверки подключения

### Оптимизация

Для production рекомендуется:

1. **Connection Pool**

````go
redis.NewClient(&redis.Options{
    PoolSize:     100,
    MinIdleConns: 10,
})
```bash
1. **Timeouts**

```go
redis.NewClient(&redis.Options{
    DialTimeout:  5 * time.Second,
    ReadTimeout:  3 * time.Second,
    WriteTimeout: 3 * time.Second,
})
````

1. **Redis Sentinel** для high availability

## Миграция с базы данных

До Redis коды верификации хранились в PostgreSQL таблице `verification_codes`.

### Что изменилось

**Было (PostgreSQL):**

- Таблица `verification_codes` с полями: id, email, code, expires_at, created_at
- Ручная проверка expires_at при каждом запросе
- Необходимость периодической очистки истекших кодов

**Стало (Redis):**

- Key-value storage: `verify:{email}` = `{code}`
- Автоматическое удаление по TTL
- Быстрее в 10-100 раз для read/write операций
- Меньше нагрузка на PostgreSQL

### Откат на PostgreSQL (если нужно)

Таблица `verification_codes` все еще существует в схеме БД. Для отката:

1. В `internal/service/auth.go` вернуть использование PostgreSQL:

````go
// Вместо Redis
s.redis.Set(ctx, verifyKey, code, 10*time.Minute)

// Использовать БД
s.db.Exec("INSERT INTO verification_codes ...")
```bash
1. Перезапустить приложение

## Безопасность

### Текущие меры

1. ✅ Password защита Redis
2. ✅ Токены в blacklist автоматически удаляются после истечения
3. ✅ Коды верификации истекают через 10 минут

### Рекомендации для production

1. Использовать TLS для Redis соединения
2. Настроить firewall правила (только app → redis)
3. Использовать Redis ACL для ограничения команд
4. Регулярный backup Redis данных (если критично)

## Troubleshooting

### Redis не запускается

```bash
docker logs event_api_redis
docker-compose restart redis
````

### Ошибка подключения

````bash
## Проверить что Redis запущен
## Проверить что Redis запущен
docker ps | grep redis

## Проверить порт
## Проверить порт
netstat -an | grep 6379

## Проверить пароль в .env
## Проверить пароль в .env
cat .env | grep REDIS_PASSWORD
```bash
### Предупреждение "maint_notifications disabled"

Это не критично - просто информация о совместимости версий клиента и сервера
Redis.

### Коды верификации не работают

```bash
## Подключиться к Redis CLI
## Подключиться к Redis CLI
docker exec -it event_api_redis redis-cli -a devpass

## Проверить наличие ключа
## Проверить наличие ключа
KEYS verify:*

## Проверить TTL
## Проверить TTL
TTL verify:user@example.com
````

## Дальнейшие улучшения

1. **Rate Limiting** - ограничение количества запросов через Redis
2. **Session Storage** - хранение пользовательских сессий
3. **Caching** - кеширование часто запрашиваемых данных из PostgreSQL
4. **Pub/Sub** - для real-time уведомлений между инстансами
5. **Distributed Locks** - для синхронизации между инстансами

## Ссылки

- [Redis Documentation](https://redis.io/documentation)
- [go-redis GitHub](https://github.com/redis/go-redis)
- [Redis Best Practices](https://redis.io/docs/manual/patterns/)
