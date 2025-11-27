# Redis Monitoring with Prometheus

## Installation

### 1. Add Prometheus dependency to Go project

```bash
cd event-api
go get github.com/prometheus/client_golang
```

This adds the Prometheus client library to `go.mod`.

### 2. Files Created

- `internal/metrics/prometheus.go` - Metrics definitions and recorders
- `internal/handlers/metrics.go` - HTTP handler for metrics endpoint

### 3. Integration in main.go

Add to the `buildHTTPHandler` function or create a separate metrics router:

```go
// Import metrics
import "event-api/internal/metrics"

// In main() function, after creating http handler:
metricsHandler := handlers.MetricsHandler

// Register the /metrics endpoint
http.Handle("/metrics", metricsHandler)

// Or in chi router:
r.Handle("/metrics", metricsHandler)
```

### 4. Usage

Record Redis operations:

```go
import "event-api/internal/metrics"

// Create metrics instance (singleton)
redisMetrics := metrics.NewRedisMetrics()

// Record a command
redisMetrics.RecordCommand("SET")

// Record execution time
redisMetrics.RecordCommandDuration("GET", 0.025)

// Update memory stats
redisMetrics.SetMemoryUsage(1024000)
redisMetrics.SetKeyCount(5000)

// Track queue operations
redisMetrics.RecordQueueOperation("event-queue", "push")
redisMetrics.SetQueueSize("event-queue", 150)
```

### 5. View Metrics

Access metrics at:

```
http://localhost:8080/metrics
```

### 6. Example Metrics

```
# HELP redis_commands_total Total number of Redis commands executed
# TYPE redis_commands_total counter
redis_commands_total{command="SET"} 1500
redis_commands_total{command="GET"} 3200
redis_commands_total{command="DEL"} 450

# HELP redis_command_duration_seconds Redis command execution time in seconds
# TYPE redis_command_duration_seconds histogram
redis_command_duration_seconds_bucket{command="GET",le="0.005"} 2800
redis_command_duration_seconds_bucket{command="GET",le="0.01"} 3100
redis_command_duration_seconds_sum{command="GET"} 25.5
redis_command_duration_seconds_count{command="GET"} 3200

# HELP redis_memory_usage_bytes Redis memory usage in bytes
# TYPE redis_memory_usage_bytes gauge
redis_memory_usage_bytes 5242880

# HELP redis_keys_total Total number of keys in Redis
# TYPE redis_keys_total gauge
redis_keys_total 5000

# HELP discovery_queue_size Size of discovery engine queues
# TYPE discovery_queue_size gauge
discovery_queue_size{queue_name="confirmed"} 150
discovery_queue_size{queue_name="waitlisted"} 25
```

### 7. Prometheus Configuration

Create `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: "event-api"
    static_configs:
      - targets: ["localhost:8080"]
    metrics_path: "/metrics"
    scrape_interval: 10s
```

### 8. Docker Compose Integration

```yaml
prometheus:
  image: prom/prometheus:latest
  ports:
    - "9090:9090"
  volumes:
    - ./prometheus.yml:/etc/prometheus/prometheus.yml
  command:
    - "--config.file=/etc/prometheus/prometheus.yml"
```

### 9. Grafana Visualization

1. Add Prometheus data source: `http://prometheus:9090`
2. Create dashboard with panels:
   - Redis Command Rate (commands_total)
   - Command Duration (p95/p99)
   - Memory Usage (gauge)
   - Key Count (gauge)
   - Error Rate (errors_total / commands_total)

### 10. Alerts

Example alert rules for `alerts.yml`:

```yaml
groups:
  - name: redis
    rules:
      - alert: RedisHighMemoryUsage
        expr: redis_memory_usage_bytes > 1073741824 # 1GB
        for: 5m

      - alert: RedisHighCommandErrorRate
        expr: rate(redis_command_errors_total[5m]) > 0.05
        for: 10m

      - alert: RedisConnectionErrors
        expr: rate(redis_connection_errors_total[5m]) > 0
        for: 5m
```

## Metrics Exposed

### Redis Operations

- `redis_commands_total` - Counter of all Redis commands
- `redis_command_errors_total` - Counter of command errors by type
- `redis_command_duration_seconds` - Histogram of command durations

### Redis Health

- `redis_connections_active` - Gauge of active connections
- `redis_connection_errors_total` - Counter of connection failures

### Redis Data

- `redis_memory_usage_bytes` - Current memory usage
- `redis_keys_total` - Total number of keys

### Discovery Engine

- `discovery_queue_size` - Size of each queue
- `discovery_queue_ops_total` - Operations per queue
- `discovery_queue_error_rate` - Error rate per queue

## Next Steps

1. Install Prometheus client: `go get github.com/prometheus/client_golang`
2. Integrate metrics into Redis repositories
3. Add metrics collection to service layer
4. Deploy Prometheus and Grafana
5. Create alerting rules
