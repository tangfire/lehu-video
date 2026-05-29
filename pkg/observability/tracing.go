package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// InitTracing installs a lightweight in-process tracer provider so Kratos can
// create trace IDs even when no external collector is configured.
func InitTracing() func() {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	return func() {
		_ = tp.Shutdown(context.Background())
	}
}
