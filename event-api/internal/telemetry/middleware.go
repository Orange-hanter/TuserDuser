// Package telemetry provides HTTP middleware for OpenTelemetry tracing.
package telemetry

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// HTTPMiddleware returns an HTTP middleware that wraps handlers with OpenTelemetry tracing.
// It automatically creates spans for each HTTP request and propagates trace context.
func HTTPMiddleware(next http.Handler) http.Handler {
	return otelhttp.NewHandler(next, "http-server",
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return r.Method + " " + r.URL.Path
		}),
	)
}

// WrapHandler wraps an http.Handler with OpenTelemetry tracing for a specific operation.
func WrapHandler(handler http.Handler, operation string) http.Handler {
	return otelhttp.NewHandler(handler, operation)
}

// WrapHandlerFunc wraps an http.HandlerFunc with OpenTelemetry tracing for a specific operation.
func WrapHandlerFunc(handler http.HandlerFunc, operation string) http.Handler {
	return otelhttp.NewHandler(handler, operation)
}
