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

// MetricsEndpoint is a thin wrapper that serves Prometheus metrics and
// is annotated for Swagger/OpenAPI documentation.
// @Summary Prometheus metrics
// @Description Exposes Prometheus metrics in the text-based OpenMetrics format. Intended for Prometheus scraping.
// @Tags monitoring
// @Produce plain
// @Success 200 {string} string "Prometheus metrics payload"
// @Router /metrics [get]
func MetricsEndpoint(w http.ResponseWriter, r *http.Request) {
	MetricsHandler.ServeHTTP(w, r)
}
