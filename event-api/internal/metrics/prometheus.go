// Package metrics provides Prometheus metrics for Redis operations and discovery engine monitoring.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// RedisMetrics tracks Redis operations and health
type RedisMetrics struct {
	// Queue metrics (for discovery engine)
	QueueSize      prometheus.GaugeVec
	QueueOpsTotal  prometheus.CounterVec
	QueueErrorRate prometheus.GaugeVec

	// Memory metrics
	RedisKeyCount    prometheus.Gauge
	RedisMemoryUsage prometheus.Gauge

	// Connection metrics
	RedisConnectionErrors prometheus.Counter
	RedisConnections      prometheus.Gauge

	// Command counters
	RedisCommandsTotal prometheus.CounterVec

	// Operation timings
	RedisCommandDuration prometheus.HistogramVec

	// Error tracking
	RedisCommandErrors prometheus.CounterVec
}

// NewRedisMetrics creates a new RedisMetrics instance
func NewRedisMetrics() *RedisMetrics {
	return &RedisMetrics{
		QueueSize: *promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "discovery_queue_size",
				Help: "Size of discovery engine queues",
			},
			[]string{"queue_name"},
		),
		QueueOpsTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "discovery_queue_ops_total",
				Help: "Total discovery queue operations",
			},
			[]string{"queue_name", "operation"},
		),
		QueueErrorRate: *promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "discovery_queue_error_rate",
				Help: "Error rate for discovery queue operations",
			},
			[]string{"queue_name"},
		),
		RedisKeyCount: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "redis_keys_total",
				Help: "Total number of keys in Redis",
			},
		),
		RedisMemoryUsage: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "redis_memory_usage_bytes",
				Help: "Redis memory usage in bytes",
			},
		),
		RedisConnectionErrors: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "redis_connection_errors_total",
				Help: "Total number of Redis connection errors",
			},
		),
		RedisConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "redis_connections_active",
				Help: "Number of active Redis connections",
			},
		),
		RedisCommandsTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "redis_commands_total",
				Help: "Total number of Redis commands executed",
			},
			[]string{"command"},
		),
		RedisCommandDuration: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "redis_command_duration_seconds",
				Help:    "Redis command execution time in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"command"},
		),
		RedisCommandErrors: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "redis_command_errors_total",
				Help: "Total number of Redis command errors",
			},
			[]string{"command", "error_type"},
		),
	}
}

// RecordCommand increments the command counter
func (m *RedisMetrics) RecordCommand(command string) {
	m.RedisCommandsTotal.WithLabelValues(command).Inc()
}

// RecordCommandDuration records command execution time
func (m *RedisMetrics) RecordCommandDuration(command string, duration float64) {
	m.RedisCommandDuration.WithLabelValues(command).Observe(duration)
}

// RecordCommandError records a command error
func (m *RedisMetrics) RecordCommandError(command string, errorType string) {
	m.RedisCommandErrors.WithLabelValues(command, errorType).Inc()
}

// SetKeyCount sets the current key count
func (m *RedisMetrics) SetKeyCount(count float64) {
	m.RedisKeyCount.Set(count)
}

// SetMemoryUsage sets the current memory usage
func (m *RedisMetrics) SetMemoryUsage(bytes float64) {
	m.RedisMemoryUsage.Set(bytes)
}

// RecordConnectionError increments the connection error counter
func (m *RedisMetrics) RecordConnectionError() {
	m.RedisConnectionErrors.Inc()
}

// SetActiveConnections sets the number of active connections
func (m *RedisMetrics) SetActiveConnections(count float64) {
	m.RedisConnections.Set(count)
}

// SetQueueSize updates queue size
func (m *RedisMetrics) SetQueueSize(queueName string, size float64) {
	m.QueueSize.WithLabelValues(queueName).Set(size)
}

// RecordQueueOperation records a queue operation
func (m *RedisMetrics) RecordQueueOperation(queueName string, operation string) {
	m.QueueOpsTotal.WithLabelValues(queueName, operation).Inc()
}

// SetQueueErrorRate sets the error rate for a queue
func (m *RedisMetrics) SetQueueErrorRate(queueName string, errorRate float64) {
	m.QueueErrorRate.WithLabelValues(queueName).Set(errorRate)
}
