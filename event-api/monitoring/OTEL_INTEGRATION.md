# OpenTelemetry Integration for Event-API

## ✅ Integrated Features

The event-api is now integrated with OpenTelemetry for:

- **Distributed Tracing** — Automatic spans for all HTTP requests
- **Log Correlation** — Logs include `trace_id` and `span_id` for Grafana correlation
- **Metrics** — Prometheus metrics (existing) + OTel Collector export

## Monitoring Stack Overview

This monitoring setup uses:

- **Prometheus** - Metrics collection and storage
- **Loki** - Log aggregation
- **Tempo** - Distributed tracing
- **Grafana** - Visualization and dashboards
- **OpenTelemetry Collector** - Central telemetry hub

## Directory Structure

```
monitoring/
├── prometheus/
│   └── prometheus.yml
├── loki/
│   └── loki-config.yml
├── tempo/
│   └── tempo.yml
├── otel/
│   └── otel-collector.yml
└── grafana/
    └── provisioning/
        └── datasources/
            └── datasources.yml
```

## Starting the Stack

```bash
docker-compose up -d
```

## Access Points

| Service    | URL                   | Description        |
| ---------- | --------------------- | ------------------ |
| Grafana    | http://localhost:3000 | Dashboards & Viz   |
| Prometheus | http://localhost:9090 | Metrics            |
| Loki       | http://localhost:3100 | Logs               |
| Tempo      | http://localhost:3200 | Traces             |
| OTLP gRPC  | localhost:4317        | OpenTelemetry gRPC |
| OTLP HTTP  | localhost:4318        | OpenTelemetry HTTP |

## Environment Variables

```bash
# Enable OpenTelemetry (default: true)
OTEL_ENABLED=true

# OTel Collector endpoint (default: localhost:4317)
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317

# Service name for traces (default: event-api)
OTEL_SERVICE_NAME=event-api
```

## Code Usage

### Automatic HTTP Tracing

All HTTP requests are automatically traced via middleware in `routes.go`:

```go
r.Use(telemetry.HTTPMiddleware)
```

### Manual Span Creation

```go
import "event-api/internal/telemetry"

func MyHandler(w http.ResponseWriter, r *http.Request) {
    ctx, span := telemetry.StartSpan(r.Context(), "my-operation")
    defer span.End()

    // Your code here...

    telemetry.AddSpanAttributes(ctx,
        attribute.String("user.id", userID),
    )
}
```

### Logging with Trace Context

Use context-aware logging for automatic trace correlation:

```go
import "event-api/internal/logger"

func MyHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // These logs will include trace_id and span_id
    logger.InfoCtx(ctx, "processing request", zap.String("path", r.URL.Path))
    logger.ErrorCtx(ctx, "something failed", zap.Error(err))
}
```

### Get Trace/Span IDs

```go
import "event-api/internal/telemetry"

traceID := telemetry.TraceIDFromContext(ctx)
spanID := telemetry.SpanIDFromContext(ctx)
```

## Grafana Default Credentials

- **Username**: admin
- **Password**: admin (or value of `GRAFANA_PASSWORD`)

## Features

- ✅ Automatic trace correlation between logs and traces
- ✅ Service map visualization in Tempo
- ✅ Logs-to-traces navigation in Grafana
- ✅ Metrics exposed via Prometheus
- ✅ All telemetry flows through OTel Collector
- ✅ Context-aware logging with trace IDs
