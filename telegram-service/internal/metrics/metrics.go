// Package metrics provides Prometheus metrics for telegram-service.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// MessagesSent tracks sent messages by status and reason.
	MessagesSent = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "telegram_messages_total",
			Help: "Total number of Telegram messages sent",
		},
		[]string{"status", "reason"},
	)

	// BindingsTotal tracks binding state changes.
	BindingsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "telegram_bindings_total",
			Help: "Total number of Telegram binding state changes",
		},
		[]string{"status"},
	)

	// BindingLinksGenerated tracks generated binding links.
	BindingLinksGenerated = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "telegram_binding_links_generated_total",
			Help: "Total number of binding links generated",
		},
	)

	// WebhookRequests tracks webhook requests by status.
	WebhookRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "telegram_webhook_requests_total",
			Help: "Total number of webhook requests received",
		},
		[]string{"status"},
	)

	// GRPCRequestDuration tracks gRPC request latency.
	GRPCRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "telegram_grpc_request_duration_seconds",
			Help:    "Duration of gRPC requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "status"},
	)
)

// Register ensures all metrics are registered with Prometheus.
func Register() {
	// Metrics are auto-registered via promauto, this is a no-op placeholder
	// for any additional initialization if needed.
}
