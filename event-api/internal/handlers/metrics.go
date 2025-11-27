package handlers

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsHandler provides Prometheus metrics endpoint
var MetricsHandler = promhttp.Handler()

// RegisterMetricsRoute registers the Prometheus metrics endpoint
func RegisterMetricsRoute(mux *http.ServeMux) {
	mux.Handle("/metrics", MetricsHandler)
}

// HealthcheckMetrics provides a simple health check with metrics info
func HealthcheckMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
