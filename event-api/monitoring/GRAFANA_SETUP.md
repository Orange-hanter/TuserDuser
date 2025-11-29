# Настройка Grafana для Event API

## Обзор

Grafana автоматически настраивается при запуске через provisioning. Дашборды и источники данных загружаются из `monitoring/grafana/provisioning/`.

## Доступ

- **URL**: http://localhost:3000
- **Логин**: admin
- **Пароль**: admin (или значение `GRAFANA_PASSWORD` из .env)

## Источники данных (Datasources)

Автоматически настроены 3 источника:

| Datasource     | URL                    | Назначение        |
| -------------- | ---------------------- | ----------------- |
| **Prometheus** | http://prometheus:9090 | Метрики (default) |
| **Loki**       | http://loki:3100       | Логи              |
| **Tempo**      | http://tempo:3200      | Трейсы            |

### Корреляция логов и трейсов

Loki настроен для автоматического извлечения `trace_id` из JSON логов:

```yaml
derivedFields:
  - datasourceUid: tempo
    matcherRegex: '"trace_id":"([a-f0-9]+)"'
    name: TraceID
    url: "${__value.raw}"
```

Кликните на `trace_id` в логах → откроется трейс в Tempo.

## Дашборды

### 1. Event API - Overview (`event-api-overview`)

**Главный дашборд мониторинга сервиса**

Панели:

- **Service Status** - статус сервиса (UP/DOWN)
- **Request Rate (RPS)** - запросов в секунду
- **Error Rate** - процент ошибок (5xx)
- **Avg Response Time (p95)** - 95-й перцентиль латентности
- **HTTP Requests by Endpoint** - RPS по эндпоинтам
- **Response Time by Endpoint** - латентность по эндпоинтам
- **HTTP Status Codes** - распределение кодов ответа
- **HTTP Methods Distribution** - GET/POST/PUT/DELETE
- **Redis Connections** - активные подключения к Redis
- **Redis Memory** - использование памяти Redis
- **Response Time Heatmap** - тепловая карта времени ответа

### 2. Event API - Traces (`event-api-traces`)

**Дашборд для анализа распределённых трейсов**

Панели:

- **Trace Search** - поиск трейсов по service.name
- **Trace Duration Distribution** - p50/p90/p99 латентности
- **Slow Traces (>500ms)** - медленные запросы
- **Error Traces** - запросы с ошибками
- **Spans by Operation** - распределение по операциям

### 3. Event API - Logs (`event-api-logs`)

**Централизованное логирование с Loki**

Панели:

- **Log Volume by Level** - количество логов по уровням
- **Application Logs** - все логи приложения (с trace_id)
- **Error Logs** - только ошибки
- **Logs with Trace ID** - логи с корреляцией трейсов
- **Top Error Messages** - топ ошибок
- **Log Rate Over Time** - скорость логирования

### 4. Event API - Redis Metrics (`redis-metrics`)

**Мониторинг Redis**

Панели:

- **Redis Commands Per Second** - команды в секунду
- **Redis Command Errors** - ошибки команд
- **Redis Memory Usage** - использование памяти
- **Redis Keys Count** - количество ключей
- **Active Connections** - активные подключения
- **Command Duration (p95)** - латентность команд

### 5. Event API - Discovery Engine (`discovery`)

**Discovery Engine очереди**

Панели:

- **Queue Sizes** - размеры очередей
- **Queue Operations Per Second** - операции с очередями
- **Queue Error Rates** - частота ошибок

## Метрики приложения

### HTTP метрики (от otelhttp)

```promql
# Запросы в секунду
sum(rate(http_server_request_duration_seconds_count{job="event-api"}[1m]))

# Латентность p95
histogram_quantile(0.95, sum(rate(http_server_request_duration_seconds_bucket{job="event-api"}[5m])) by (le))

# Error rate
sum(rate(http_server_request_duration_seconds_count{job="event-api",http_status_code=~"5.."}[5m]))
  / sum(rate(http_server_request_duration_seconds_count{job="event-api"}[5m])) * 100

# По эндпоинту
sum by (http_route) (rate(http_server_request_duration_seconds_count{job="event-api"}[1m]))
```

### Redis метрики

```promql
# Команды
rate(redis_commands_total[1m])

# Ошибки
rate(redis_command_errors_total[5m])

# Память
redis_memory_usage_bytes

# Ключи
redis_keys_total

# Соединения
redis_connections_active
```

### Discovery Engine метрики

```promql
# Размер очереди
discovery_queue_size

# Операции
rate(discovery_queue_ops_total[1m])

# Ошибки
discovery_queue_error_rate
```

## TraceQL запросы (Tempo)

```traceql
# Все трейсы сервиса
{ resource.service.name = "event-api" }

# Медленные запросы (>500ms)
{ resource.service.name = "event-api" } | duration > 500ms

# Ошибки
{ resource.service.name = "event-api" && status = error }

# По HTTP методу
{ resource.service.name = "event-api" && span.http.method = "POST" }

# По эндпоинту
{ resource.service.name = "event-api" && name =~ ".*events.*" }
```

## LogQL запросы (Loki)

```logql
# Все логи
{job="event-api"}

# JSON парсинг
{job="event-api"} | json

# Фильтр по уровню
{job="event-api"} | json | level = "error"

# Поиск по сообщению
{job="event-api"} | json | msg =~ ".*database.*"

# Логи с trace_id
{job="event-api"} | json | trace_id != ""

# Статистика по ошибкам
sum by (msg) (count_over_time({job="event-api"} | json | level = "error" [1h]))
```

## Создание своих дашбордов

### Через UI

1. Зайти в Grafana → Dashboards → New Dashboard
2. Добавить панели
3. Сохранить (Export → JSON)
4. Положить файл в `monitoring/grafana/provisioning/dashboards/`

### Через JSON

1. Создать файл `monitoring/grafana/provisioning/dashboards/my-dashboard.json`
2. Перезапустить Grafana:
   ```bash
   docker compose restart grafana
   ```

## Алерты

Для настройки алертов:

1. Создать Contact Point (Settings → Alerting → Contact Points)
2. Создать Alert Rule на панели:
   - Выбрать панель → Edit → Alert
   - Настроить условие и threshold
   - Выбрать Contact Point

Пример alert rule для error rate:

```yaml
- alert: HighErrorRate
  expr: |
    sum(rate(http_server_request_duration_seconds_count{job="event-api",http_status_code=~"5.."}[5m])) 
    / sum(rate(http_server_request_duration_seconds_count{job="event-api"}[5m])) * 100 > 5
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "High error rate detected"
    description: "Error rate is above 5% for 5 minutes"
```

## Troubleshooting

### Дашборды не загружаются

```bash
# Проверить права
ls -la monitoring/grafana/provisioning/

# Проверить логи Grafana
docker compose logs grafana

# Перезапустить
docker compose restart grafana
```

### Нет данных в дашбордах

1. Проверить что сервисы запущены:

   ```bash
   docker compose ps
   ```

2. Проверить метрики Prometheus:

   ```bash
   curl http://localhost:9090/api/v1/targets
   ```

3. Проверить что event-api экспортирует метрики:
   ```bash
   curl http://localhost:8080/metrics
   ```

### Нет трейсов

1. Проверить что OTEL_ENABLED=true в .env
2. Проверить что otel-collector запущен
3. Проверить Tempo:
   ```bash
   curl http://localhost:3200/ready
   ```

### Нет логов

1. Проверить Loki:

   ```bash
   curl http://localhost:3100/ready
   ```

2. Проверить что логи пишутся в stdout:
   ```bash
   docker compose logs event-api
   ```
