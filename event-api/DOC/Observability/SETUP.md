# Observability Setup Guide

> Руководство по настройке и запуску стека мониторинга.

## Содержание

- [Быстрый старт](#быстрый-старт)
- [Компоненты](#компоненты)
- [Конфигурация](#конфигурация)
- [Дашборды](#дашборды)
- [Troubleshooting](#troubleshooting)

---

## Быстрый старт

### Prerequisites

- Docker
- Docker Compose

### Запуск стека

```bash
# Из корня проекта
cd event-api

# Скопировать .env файл
cp .env.example .env

# Запустить весь стек
docker-compose up -d

# Или только мониторинг
docker-compose up -d prometheus grafana loki promtail tempo otel-collector
```

### Проверка статуса

```bash
docker-compose ps

# Ожидаемый результат: все сервисы Up (healthy)
```

---

## Компоненты

| Сервис             | URL              | Порт                     | Credentials       |
| ------------------ | ---------------- | ------------------------ | ----------------- |
| **Grafana**        | `localhost:3000` | 3000                     | `admin` / `admin` |
| **Prometheus**     | `localhost:9090` | 9090                     | —                 |
| **Loki**           | `localhost:3100` | 3100                     | —                 |
| **Tempo**          | `localhost:3200` | 3200                     | —                 |
| **OTel Collector** | —                | 4317 (gRPC), 4318 (HTTP) | —                 |

### Проверка endpoints

```bash
# Проверить метрики Event API
curl http://localhost:8080/metrics

# Проверить targets Prometheus
curl http://localhost:9090/api/v1/targets

# Проверить готовность Loki
curl http://localhost:3100/ready

# Проверить готовность Tempo
curl http://localhost:3200/ready
```

---

## Конфигурация

### Структура файлов

```text
monitoring/
├── prometheus.yml          # Scrape targets
├── alerts.yml/             # Alerting rules
├── promtail-config.yml     # Log collection
├── loki-config.yml/        # Log storage
├── tempo/                  # Tracing config
│   └── tempo.yml
├── otel/                   # OpenTelemetry
│   └── otel-collector.yml
└── grafana/
    └── provisioning/
        ├── datasources/    # Auto-configured sources
        └── dashboards/     # Pre-built dashboards
```

### Переменные окружения

```bash
# .env файл
GRAFANA_USER=admin
GRAFANA_PASSWORD=admin
REDIS_PASSWORD=devpass

# OpenTelemetry
OTEL_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
OTEL_SERVICE_NAME=event-api
```

### Добавление новых scrape targets

Отредактируйте `monitoring/prometheus.yml`:

```yaml
scrape_configs:
  - job_name: "event-api"
    static_configs:
      - targets: ["event-api:8080"]

  # Добавить новый сервис
  - job_name: "telegram-service"
    static_configs:
      - targets: ["telegram-service:8081"]
```

---

## Дашборды

Grafana поставляется с предустановленными дашбордами:

| Dashboard                 | Описание                                 |
| ------------------------- | ---------------------------------------- |
| **Event API - Overview**  | RPS, Error Rate, Latency p95, HTTP codes |
| **Event API - Redis**     | Команды/сек, память, ключи, соединения   |
| **Event API - Discovery** | Размеры очередей, операции, ошибки       |
| **Event API - Logs**      | Логи по уровням, топ ошибок, volume      |
| **Event API - Traces**    | Распределение трейсов, медленные запросы |

### Создание нового дашборда

1. **Через UI:**
   - Grafana → Dashboards → New Dashboard
   - Добавить панели
   - Export → JSON

2. **Через файл:**

```bash
# Положить JSON в provisioning
cp my-dashboard.json monitoring/grafana/provisioning/dashboards/

# Перезапустить Grafana
docker-compose restart grafana
```

---

## Troubleshooting

### "Target Down" в Prometheus

- Проверьте `http://localhost:9090/targets`
- Убедитесь, что сервис запущен и экспортирует метрики:

```bash
curl http://localhost:8080/metrics
```

- Проверьте сетевую связность между контейнерами

### Нет данных в Grafana

```bash
# Проверить что сервисы запущены
docker-compose ps

# Проверить логи Prometheus
docker-compose logs prometheus

# Проверить что метрики доступны
curl http://localhost:9090/api/v1/query?query=up
```

### Нет логов в Loki

```bash
# Проверить Promtail
docker-compose logs promtail

# Проверить что Loki ready
curl http://localhost:3100/ready

# Проверить labels в Loki
curl -G http://localhost:3100/loki/api/v1/labels
```

### Нет трейсов в Tempo

- Проверьте `OTEL_ENABLED=true` в `.env`
- Проверьте OTel Collector:

```bash
docker-compose logs otel-collector
```

- Проверьте Tempo:

```bash
curl http://localhost:3200/ready
```

### Grafana не загружает дашборды

```bash
# Проверить права на файлы
ls -la monitoring/grafana/provisioning/

# Проверить логи
docker-compose logs grafana

# Перезапустить
docker-compose restart grafana
```

---

## Управление стеком

```bash
# Остановить
docker-compose down

# Просмотр логов
docker-compose logs -f prometheus grafana loki

# Очистить все данные (volumes)
docker-compose down -v

# Перезапустить конкретный сервис
docker-compose restart grafana
```
