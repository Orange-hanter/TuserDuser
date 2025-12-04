# Observability Best Practices

> Рекомендации по мониторингу, логированию и алертингу для TuserDuser.

## Содержание

- [Метрики и инструментирование](#1-метрики--инструментирование)
- [Стандарты логирования](#2-стандарты-логирования)
- [Стратегия алертинга](#3-стратегия-алертинга)
- [Организация дашбордов](#4-организация-дашбордов)
- [Обслуживание](#5-обслуживание)

---

## 1. Метрики & Инструментирование

### Соглашения об именовании

| Элемент                | Правило          | Пример                          |
| ---------------------- | ---------------- | ------------------------------- |
| **Prefix**             | Имя сервиса      | `event_api_`, `telegram_`       |
| **Suffix (\_total)**   | Для счётчиков    | `redis_commands_total`          |
| **Suffix (\_seconds)** | Для длительности | `http_request_duration_seconds` |
| **Suffix (\_bytes)**   | Для размеров     | `redis_memory_usage_bytes`      |

### Labels (Метки)

✅ **Хорошо:**

- `method` (GET, POST)
- `status_code` (200, 500)
- `endpoint` (/api/events)
- `command` (SET, GET)

❌ **Избегать (высокая кардинальность):**

- `user_id`
- `request_id`
- `trace_id`

### RED метрики

Каждый сервис должен экспортировать минимум три типа метрик:

| Метрика      | Что измеряет              | PromQL пример                                                 |
| ------------ | ------------------------- | ------------------------------------------------------------- |
| **Rate**     | Запросы в секунду         | `rate(http_requests_total[1m])`                               |
| **Errors**   | Процент ошибок            | `rate(http_errors_total[5m]) / rate(http_requests_total[5m])` |
| **Duration** | Латентность (гистограмма) | `histogram_quantile(0.99, rate(http_duration_bucket[5m]))`    |

### Redis специфика

| Метрика                        | Порог        | Действие                     |
| ------------------------------ | ------------ | ---------------------------- |
| **Memory Fragmentation Ratio** | > 1.5        | Перезапуск Redis             |
| **Evicted Keys**               | > 0          | Увеличить `maxmemory`        |
| **Slow Log**                   | > 10ms       | Оптимизировать запросы       |
| **Connected Clients**          | > 80% от max | Проверить connection pooling |

---

## 2. Стандарты логирования

### Формат логов

**Production:** JSON  
**Development:** Text (console)

### Обязательные поля

```json
{
  "level": "info",
  "ts": "2024-12-04T10:00:00.000Z",
  "caller": "handlers/auth.go:42",
  "msg": "user logged in",
  "trace_id": "abc123def456",
  "request_id": "req-789"
}
```

### Уровни логирования

| Уровень | Использование                                            | Включён в Prod |
| ------- | -------------------------------------------------------- | -------------- |
| `DEBUG` | Детали для отладки, значения переменных                  | ❌             |
| `INFO`  | Операционные события: startup, shutdown, важные действия | ✅             |
| `WARN`  | Некритичные проблемы: retries, deprecated API            | ✅             |
| `ERROR` | Критические сбои: DB down, panic, невосстановимые ошибки | ✅             |

### Что НЕ логировать

❌ Пароли и токены  
❌ Персональные данные (PII)  
❌ Номера карт  
❌ API ключи

### Контекстное логирование

```go
// ✅ Хорошо - включает контекст
logger.Info("event created",
    zap.String("event_id", event.ID),
    zap.String("creator_id", userID),
    zap.String("trace_id", traceID),
)

// ❌ Плохо - нет контекста
logger.Info("event created")
```

---

## 3. Стратегия алертинга

### Философия

> **"Page on Symptoms, not Causes"**  
> Алертить когда страдает пользователь, а не когда падает сервер.

### Правила эффективного алертинга

1. **Actionable** — Алерт требует действия человека
2. **Urgent** — Действие нужно сейчас
3. **No Flapping** — Стабильные пороги, минимум ложных срабатываний
4. **Documented** — Runbook для каждого алерта

### Приоритеты алертов

#### P1 — Критично (Немедленное действие)

| Алерт               | Условие              | SLA ответа |
| ------------------- | -------------------- | ---------- |
| **High Error Rate** | > 5% ошибок за 5 мин | 5 мин      |
| **High Latency**    | p99 > 2s за 5 мин    | 5 мин      |
| **Service Down**    | Target unreachable   | 1 мин      |
| **Redis Down**      | Connection failed    | 1 мин      |

#### P2 — Высокий (В рабочее время)

| Алерт                  | Условие            | SLA ответа |
| ---------------------- | ------------------ | ---------- |
| **Disk Space Low**     | < 15% свободно     | 4 часа     |
| **Memory High**        | > 85% использовано | 4 часа     |
| **Redis Memory**       | > 80% maxmemory    | 4 часа     |
| **Certificate Expiry** | < 14 дней          | 24 часа    |

#### P3 — Средний (Следующий рабочий день)

| Алерт                | Условие            |
| -------------------- | ------------------ |
| **Slow Queries**     | p95 > 500ms        |
| **Connection Pool**  | > 70% использовано |
| **Log Errors Spike** | +50% за час        |

### Пример Prometheus Alert Rule

```yaml
groups:
  - name: event-api-alerts
    rules:
      - alert: HighErrorRate
        expr: |
          sum(rate(http_server_request_duration_seconds_count{job="event-api",http_status_code=~"5.."}[5m])) 
          / sum(rate(http_server_request_duration_seconds_count{job="event-api"}[5m])) * 100 > 5
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate on Event API"
          description: 'Error rate is {{ $value | printf "%.2f" }}% (threshold: 5%)'
          runbook_url: "https://wiki.example.com/runbooks/high-error-rate"
```

---

## 4. Организация дашбордов

### Иерархия

```text
📊 Dashboards/
├── 🚦 Overview          # Светофор всех сервисов
├── 📈 Event API/
│   ├── Overview        # RED метрики, топ эндпоинтов
│   ├── Redis           # Команды, память, соединения
│   ├── Discovery       # Очереди, операции
│   └── Logs            # Ошибки, volume
├── 📱 Telegram Service/
│   ├── Overview        # Сообщения, bindings
│   └── gRPC            # Латентность по методам
└── 🖥️ Infrastructure/
    ├── Resources       # CPU, Memory, Disk
    └── Network         # Connections, Bandwidth
```

### Принципы построения дашбордов

1. **Сверху вниз** — От общего к частному
2. **Golden Signals** — Rate, Errors, Duration на каждом дашборде
3. **Time Range** — Использовать переменные для гибкости
4. **Links** — Связывать дашборды между собой

### Шаблон панели Overview

| Row         | Панели                   |
| ----------- | ------------------------ |
| **Status**  | Service Up/Down, Uptime  |
| **Traffic** | RPS, Active Users        |
| **Errors**  | Error Rate %, Top Errors |
| **Latency** | p50, p95, p99            |

---

## 5. Обслуживание

### Retention (Хранение данных)

| Данные                   | Срок хранения | Примечание                        |
| ------------------------ | ------------- | --------------------------------- |
| **Metrics (Prometheus)** | 15 дней       | Для долгосрочного — Thanos/Cortex |
| **Logs (Loki)**          | 7-14 дней     | Зависит от объёма                 |
| **Traces (Tempo)**       | 7 дней        | Семплирование для оптимизации     |

### Регулярные задачи

| Задача                          | Частота               |
| ------------------------------- | --------------------- |
| Проверка дискового пространства | Ежедневно (автоалерт) |
| Ревью алертов (false positives) | Еженедельно           |
| Обновление Grafana/Prometheus   | Ежемесячно            |
| Аудит дашбордов                 | Ежеквартально         |

### Чеклист перед Production

- [ ] Все сервисы экспортируют `/metrics`
- [ ] Prometheus scrape настроен для всех targets
- [ ] RED метрики присутствуют
- [ ] P1 алерты настроены и протестированы
- [ ] Runbooks написаны для каждого алерта
- [ ] Логи в JSON формате
- [ ] trace_id включён в логи
- [ ] Дашборды созданы и работают
