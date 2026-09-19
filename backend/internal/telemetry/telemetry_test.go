package telemetry

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// Nothing configured has to mean nothing installed: no collector, no delay and not a
// single warning, otherwise every local run and every test would pay for telemetry.
func TestSetupWithoutEndpointStaysOut(t *testing.T) {
	var logs bytes.Buffer
	slog.SetDefault(NewLogger(&logs, "info"))

	started := time.Now()
	shutdown, err := Setup(context.Background(), Config{ServiceName: "test"})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Errorf("Setup took %v without an endpoint", elapsed)
	}

	_, span := Tracer().Start(context.Background(), "scan")
	defer span.End()
	if span.IsRecording() {
		t.Error("span is recorded although telemetry is off")
	}

	queueRead := false
	if err := ObserveQueue(func(context.Context) (QueueStats, error) {
		queueRead = true
		return QueueStats{}, nil
	}); err != nil {
		t.Fatalf("ObserveQueue: %v", err)
	}
	if queueRead {
		t.Error("the queue was read although telemetry is off")
	}

	if logs.Len() != 0 {
		t.Errorf("Setup logged without an endpoint: %s", logs.String())
	}
}

// An endpoint that no one is listening on must not keep the process from starting:
// the exporters connect lazily.
func TestSetupWithEndpointInstallsProviders(t *testing.T) {
	restore := swapProviders(t)
	defer restore()

	for _, protocol := range []string{ProtocolGRPC, ProtocolHTTP} {
		t.Run(protocol, func(t *testing.T) {
			shutdown, err := Setup(context.Background(), Config{
				ServiceName: "test",
				Endpoint:    "http://127.0.0.1:1/",
				Protocol:    protocol,
				SampleRatio: 1,
			})
			if err != nil {
				t.Fatalf("Setup: %v", err)
			}
			t.Cleanup(func() {
				// Nobody is listening, so the flush would wait out its whole timeout.
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()
				_ = shutdown(ctx)
			})

			_, span := Tracer().Start(context.Background(), "scan")
			defer span.End()
			if !span.IsRecording() {
				t.Error("span is not recorded although telemetry is on")
			}
		})
	}
}

func TestSetupRejectsAnUnusableEndpoint(t *testing.T) {
	restore := swapProviders(t)
	defer restore()

	if _, err := Setup(context.Background(), Config{
		ServiceName: "test", Endpoint: "://nonsense", Protocol: ProtocolHTTP, SampleRatio: 1,
	}); err == nil {
		t.Fatal("an unusable endpoint was accepted")
	}
}

func TestLoggerCarriesTheTraceID(t *testing.T) {
	provider := sdktrace.NewTracerProvider()
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })

	var logs bytes.Buffer
	log := NewLogger(&logs, "info").With("component", "worker")

	ctx, span := provider.Tracer("test").Start(context.Background(), "scan")
	log.InfoContext(ctx, "scan finished")
	span.End()

	if got := logs.String(); !bytes.Contains([]byte(got), []byte(span.SpanContext().TraceID().String())) {
		t.Fatalf("trace id missing: %s", got)
	}

	logs.Reset()
	log.InfoContext(context.Background(), "scan finished")
	if bytes.Contains(logs.Bytes(), []byte("trace_id")) {
		t.Fatalf("trace id without a span: %s", logs.String())
	}
}

func TestLoggerHonoursTheLevel(t *testing.T) {
	var logs bytes.Buffer
	NewLogger(&logs, "warn").Info("quiet")
	if logs.Len() != 0 {
		t.Fatalf("info was logged at level warn: %s", logs.String())
	}
}

// The span is named after the chi route, not after the URL — otherwise every agency
// would be a trace of its own.
func TestWrapHandlerNamesTheSpanAfterTheRoute(t *testing.T) {
	restore := swapProviders(t)
	defer restore()

	spans := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spans)))

	r := chi.NewRouter()
	r.Get("/api/v1/agencies/{slug}", func(w http.ResponseWriter, r *http.Request) {
		if got := chi.URLParam(r, "slug"); got != "bmi" {
			t.Errorf("slug = %q", got)
		}
		w.WriteHeader(http.StatusOK)
	})
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	srv := httptest.NewServer(WrapHandler(r, "api"))
	defer srv.Close()

	for _, path := range []string{"/api/v1/agencies/bmi", "/healthz"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET %s = %d", path, resp.StatusCode)
		}
	}

	ended := spans.Ended()
	if len(ended) != 1 {
		t.Fatalf("%d spans, expected only the one for the route", len(ended))
	}
	if got := ended[0].Name(); got != "GET /api/v1/agencies/{slug}" {
		t.Fatalf("span name = %q", got)
	}
	if !hasAttribute(ended[0].Attributes(), "http.route", "/api/v1/agencies/{slug}") {
		t.Fatalf("route attribute missing: %v", ended[0].Attributes())
	}
	if !hasAttribute(ended[0].Attributes(), "http.response.status_code", int64(200)) {
		t.Fatalf("status code missing: %v", ended[0].Attributes())
	}
}

// With telemetry off the wrapped handler still has to route and answer as before.
func TestWrapHandlerServesWithoutTelemetry(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/api/v1/agencies/{slug}", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(chi.URLParam(r, "slug")))
	})

	rec := httptest.NewRecorder()
	WrapHandler(r, "api").ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/agencies/bmi", nil))

	if rec.Code != http.StatusOK || rec.Body.String() != "bmi" {
		t.Fatalf("%d %q", rec.Code, rec.Body.String())
	}
}

func TestObserveQueueReportsTheQueue(t *testing.T) {
	restore := swapProviders(t)
	defer restore()

	reader := sdkmetric.NewManualReader()
	otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))

	if err := ObserveQueue(func(context.Context) (QueueStats, error) {
		return QueueStats{Queued: 3, Running: 1, OldestAge: 90 * time.Second}, nil
	}); err != nil {
		t.Fatalf("ObserveQueue: %v", err)
	}

	var collected metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &collected); err != nil {
		t.Fatalf("collect: %v", err)
	}

	want := map[string]float64{"jobs.queued": 3, "jobs.running": 1, "jobs.oldest_age": 90}
	for _, scope := range collected.ScopeMetrics {
		for _, m := range scope.Metrics {
			switch data := m.Data.(type) {
			case metricdata.Gauge[int64]:
				got := float64(data.DataPoints[0].Value)
				if want[m.Name] != got {
					t.Errorf("%s = %v, want %v", m.Name, got, want[m.Name])
				}
			case metricdata.Gauge[float64]:
				if got := data.DataPoints[0].Value; want[m.Name] != got {
					t.Errorf("%s = %v, want %v", m.Name, got, want[m.Name])
				}
			}
			delete(want, m.Name)
		}
	}
	if len(want) != 0 {
		t.Fatalf("metrics missing: %v", want)
	}
}

// RecordScan is the only test that may use the scan instruments: they are built once
// and would then hold on to whichever meter provider was installed first.
func TestRecordScanCountsByStatus(t *testing.T) {
	restore := swapProviders(t)
	defer restore()

	reader := sdkmetric.NewManualReader()
	otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))

	RecordScan(context.Background(), StatusDone, 12, 30*time.Second)
	RecordScan(context.Background(), StatusFailed, 0, time.Second)

	var collected metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &collected); err != nil {
		t.Fatalf("collect: %v", err)
	}

	found := map[string]bool{}
	for _, scope := range collected.ScopeMetrics {
		for _, m := range scope.Metrics {
			found[m.Name] = true
			if sum, ok := m.Data.(metricdata.Sum[int64]); ok && m.Name == "scans.total" {
				if len(sum.DataPoints) != 2 {
					t.Errorf("scans.total has %d series, want one per status", len(sum.DataPoints))
				}
			}
		}
	}
	for _, name := range []string{"scans.total", "scan.duration", "scan.pages"} {
		if !found[name] {
			t.Errorf("%s was not recorded", name)
		}
	}
}

func swapProviders(t *testing.T) func() {
	t.Helper()
	tracer, meter := otel.GetTracerProvider(), otel.GetMeterProvider()
	// Setting a provider back to the very same value makes the SDK complain, so the
	// providers only go back when a test actually replaced them.
	return func() {
		if otel.GetTracerProvider() != tracer {
			otel.SetTracerProvider(tracer)
		}
		if otel.GetMeterProvider() != meter {
			otel.SetMeterProvider(meter)
		}
	}
}

func hasAttribute(attrs []attribute.KeyValue, key string, value any) bool {
	for _, a := range attrs {
		if string(a.Key) == key && a.Value.AsInterface() == value {
			return true
		}
	}
	return false
}
