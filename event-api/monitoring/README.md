# Monitoring Stack - Event API

Полный стек мониторинга включен в основной `docker-compose.yml` вместе с:

- PostgreSQL - база данных
- Redis - кеш и очереди
- Event API - основное приложение
- Prometheus - сбор метрик
- Grafana - визуализация
- Loki - логирование
- Promtail - сбор логов
- PgAdmin - управление БД

## Быстрый старт

```bash
# Из корня проекта
cd event-api

# Скопировать .env файл (опционально)
cp .env.example .env

# Запустить весь стек
docker-compose up -d

# Проверить статус
docker-compose ps
```

Или использовать скрипт:

```bash
../monitoring-stack.sh start
```

## Доступ к сервисам

| Сервис     | URL                   | Учетные данные            |
| ---------- | --------------------- | ------------------------- |
| Event API  | http://localhost:8080 | -                         |
| Prometheus | http://localhost:9090 | -                         |
| Grafana    | http://localhost:3000 | admin / admin             |
| Loki       | http://localhost:3100 | -                         |
| PgAdmin    | http://localhost:5050 | admin@example.com / admin |

## Структура конфигов

```
monitoring/
├── prometheus.yml          # Сбор метрик
├── alerts.yml              # Правила алертов
├── loki-config.yml         # Хранилище логов
├── promtail-config.yml     # Сбор логов
├── README.md               # Эта документация
└── grafana/
    └── provisioning/
        ├── datasources.yml # Prometheus + Loki
        ├── dashboards.yml  # Конфиг dashboards
        └── dashboards/     # JSON dashboards
            ├── redis-metrics.json
            ├── logs.json
            └── discovery.json
```

## Собираемые метрики

### Redis операции

- `redis_commands_total` - общее количество команд
- `redis_command_duration_seconds` - время выполнения
- `redis_command_errors_total` - ошибки

### Redis здоровье

- `redis_connections_active` - активные подключения
- `redis_connection_errors_total` - ошибки подключения

### Redis данные

- `redis_memory_usage_bytes` - использованная память
- `redis_keys_total` - количество ключей

### Discovery Engine

- `discovery_queue_size` - размер очередей
- `discovery_queue_ops_total` - операции
- `discovery_queue_error_rate` - ошибки

## Управление

```bash
# Остановить стек
docker-compose down

# Просмотр логов
docker-compose logs -f prometheus
docker-compose logs -f grafana
docker-compose logs -f loki

# Очистить все данные
docker-compose down -v
```

## Алерты

Настроены алерты для:

- Высокое использование памяти Redis (> 1GB)
- Высокий процент ошибок Redis
- Ошибки Discovery Queue
- API недоступен
- Loki недоступен

## Dashboards

Автоматически загружаются три dashboards:

1. **Redis Metrics** - команды, ошибки, память, подключения
2. **Logs** - логи приложения, ошибки, система
3. **Discovery Engine** - очереди, операции, ошибки

## Переменные окружения

Установить в `.env` файл (скопировать из `.env.example`):

```bash
GRAFANA_USER=admin
GRAFANA_PASSWORD=admin
REDIS_PASSWORD=devpass
```

## Кастомизация

Отредактировать конфиги в `monitoring/`:

- `prometheus.yml` - добавить новые job'ы для сбора метрик
- `alerts.yml` - добавить новые правила алертов
- `promtail-config.yml` - добавить новые источники логов
- `grafana/provisioning/dashboards/` - добавить новые dashboards

## Проблемы

**Prometheus не видит метрики:**

- Проверить: `curl http://localhost:8080/metrics`
- Targets: http://localhost:9090/targets

**Loki не собирает логи:**

- Логи: `docker-compose logs promtail`
- Проверить доступ к `/var/run/docker.sock`

**Grafana dashboards не загружаются:**

- Логи: `docker-compose logs grafana`
- Перезапустить: `docker-compose restart grafana`
