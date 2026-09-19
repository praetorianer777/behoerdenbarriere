package telemetry

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

// WrapHandler gives every request a span and the HTTP server metrics otelhttp
// records. The health endpoints are left out: they are polled every few seconds and
// would bury the actual traffic.
func WrapHandler(h http.Handler, service string) http.Handler {
	return otelhttp.NewHandler(routeNamer(h), service,
		otelhttp.WithFilter(func(r *http.Request) bool {
			return r.URL.Path != "/healthz" && r.URL.Path != "/readyz"
		}),
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return r.Method
		}),
	)
}

// routeNamer names the span after the chi route rather than the URL, so that a
// thousand agency detail requests are one entry and not a thousand. The route is only
// known once chi has routed, hence the routing context handed in beforehand and read
// out afterwards.
func routeNamer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := trace.SpanFromContext(r.Context())
		if !span.IsRecording() {
			next.ServeHTTP(w, r)
			return
		}

		rctx := chi.NewRouteContext()
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx)))

		if pattern := rctx.RoutePattern(); pattern != "" {
			span.SetName(r.Method + " " + pattern)
			span.SetAttributes(semconv.HTTPRoute(pattern))
		}
	})
}
