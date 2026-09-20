// Package telemetry wires the process up to OpenTelemetry: traces, metrics and
// trace-aware logging. Without an endpoint it does nothing at all — no provider is
// installed, so the global tracer and meter stay the no-op ones the SDK ships with
// and neither a local run nor the test suite needs a collector.
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	ProtocolGRPC = "grpc"
	ProtocolHTTP = "http/protobuf"

	scopeName = "github.com/praetorianer777/behoerdenbarriere"
)

type Config struct {
	ServiceName string
	Endpoint    string
	Protocol    string
	SampleRatio float64
}

func (c Config) Enabled() bool { return c.Endpoint != "" }

// Shutdown flushes what is still buffered. It is safe to call when telemetry is off.
type Shutdown func(context.Context) error

// Setup installs the providers and returns the matching shutdown. With telemetry off
// it returns a shutdown that does nothing, and no error: being off is not a fault.
func Setup(ctx context.Context, c Config) (Shutdown, error) {
	if !c.Enabled() {
		return func(context.Context) error { return nil }, nil
	}

	// The exporters only log an endpoint they cannot read and then export nowhere;
	// a misconfigured collector should stop the process instead.
	if _, err := url.Parse(c.Endpoint); err != nil {
		return nil, fmt.Errorf("otlp endpoint: %w", err)
	}

	res, err := resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL, semconv.ServiceName(c.ServiceName),
	))
	if err != nil {
		return nil, fmt.Errorf("telemetry resource: %w", err)
	}

	traceExporter, err := newTraceExporter(ctx, c)
	if err != nil {
		return nil, err
	}
	metricExporter, err := newMetricExporter(ctx, c)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(traceExporter),
		// ParentBased keeps a sampled request whole: once the API has decided to
		// record a trace, the worker does not drop half of it.
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(c.SampleRatio))),
	)
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
	)

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))

	return func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return errors.Join(tp.Shutdown(ctx), mp.Shutdown(ctx))
	}, nil
}

func newTraceExporter(ctx context.Context, c Config) (sdktrace.SpanExporter, error) {
	var (
		exp sdktrace.SpanExporter
		err error
	)
	switch c.Protocol {
	case ProtocolHTTP:
		exp, err = otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(c.Endpoint))
	default:
		exp, err = otlptracegrpc.New(ctx, otlptracegrpc.WithEndpointURL(c.Endpoint))
	}
	if err != nil {
		return nil, fmt.Errorf("otlp trace exporter: %w", err)
	}
	return exp, nil
}

func newMetricExporter(ctx context.Context, c Config) (sdkmetric.Exporter, error) {
	var (
		exp sdkmetric.Exporter
		err error
	)
	switch c.Protocol {
	case ProtocolHTTP:
		exp, err = otlpmetrichttp.New(ctx, otlpmetrichttp.WithEndpointURL(c.Endpoint))
	default:
		exp, err = otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithEndpointURL(c.Endpoint))
	}
	if err != nil {
		return nil, fmt.Errorf("otlp metric exporter: %w", err)
	}
	return exp, nil
}

// Tracer is the one tracer of this project; with telemetry off it is the no-op one.
func Tracer() trace.Tracer { return otel.Tracer(scopeName) }
