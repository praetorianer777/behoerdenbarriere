package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/praetorianer777/behoerdenbarriere/internal/scanner"
)

// PageScanner is what the crawler expects of a scanner.
type PageScanner interface {
	Scan(ctx context.Context, url string) scanner.PageScan
}

// TracePageScanner puts a span around every page check. It sits between crawler and
// scanner so that neither has to know about telemetry.
func TracePageScanner(s PageScanner) PageScanner { return tracedScanner{s} }

type tracedScanner struct{ inner PageScanner }

func (t tracedScanner) Scan(ctx context.Context, url string) scanner.PageScan {
	ctx, span := Tracer().Start(ctx, "page scan", trace.WithAttributes(attribute.String("url", url)))
	defer span.End()

	out := t.inner.Scan(ctx, url)

	span.SetAttributes(
		attribute.Int("http.status", out.Result.HTTPStatus),
		attribute.Int("dom.nodes", out.Result.DOMNodes),
		attribute.Int("violations", len(out.Result.Violations)),
		attribute.Int("links", len(out.Links)),
	)
	if out.Result.Failed() {
		span.SetStatus(codes.Error, out.Result.Err)
	}
	return out
}
