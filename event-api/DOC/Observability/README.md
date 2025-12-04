# Observability Documentation

> Документация по мониторингу, логированию и трейсингу TuserDuser.

## 🚀 Быстрый старт

```bash
# Запустить стек мониторинга
cd event-api && docker-compose up -d prometheus grafana loki

# Открыть Grafana
open http://localhost:3000  # admin / admin
```

## 📚 Содержание

| Документ                                                    | Описание                                                                          |
| ----------------------------------------------------------- | --------------------------------------------------------------------------------- |
| [**Metrics & Logging Reference**](./METRICS_AND_LOGGING.md) | Полный справочник метрик, логирования и примеры запросов (PromQL, LogQL, TraceQL) |
| [**Setup Guide**](./SETUP.md)                               | Руководство по запуску стека мониторинга и настройке компонентов                  |
| [**Best Practices**](./OBSERVABILITY_BEST_PRACTICES.md)     | Рекомендации по инструментированию, алертингу и организации дашбордов             |

## 🔗 Quick Links

| Сервис         | URL                     | Порт |
| -------------- | ----------------------- | ---- |
| **Grafana**    | <http://localhost:3000> | 3000 |
| **Prometheus** | <http://localhost:9090> | 9090 |
| **Loki**       | <http://localhost:3100> | 3100 |
| **Tempo**      | <http://localhost:3200> | 3200 |

## 📊 Метрики сервисов

| Сервис               | Endpoint        | Основные метрики            |
| -------------------- | --------------- | --------------------------- |
| **Event API**        | `:8080/metrics` | Redis ops, Discovery queues |
| **Telegram Service** | `:8081/metrics` | Messages, Bindings, gRPC    |

## 📁 Связанные файлы

- Конфигурация: `event-api/monitoring/`
- Дашборды: `event-api/monitoring/grafana/provisioning/dashboards/`
- Алерты: `event-api/monitoring/alerts.yml/`
