package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Tracer returns a named tracer from the global TracerProvider.
//
// Example:
//
//	tracer := telemetry.Tracer("my-package")
//	ctx, span := tracer.Start(ctx, "operation-name")
//	defer span.End()
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// StartSpan starts a new span and returns the updated context and an end function.
// Always defer the end function.
//
// Example:
//
//	ctx, end := telemetry.StartSpan(ctx, "process-order")
//	defer end()
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, func()) {
	ctx, span := otel.Tracer("").Start(ctx, name, opts...)
	return ctx, func() { span.End() }
}

// SpanFromContext returns the current span from the context.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// TraceIDFromContext returns the trace ID string from the current span, or empty if none.
func TraceIDFromContext(ctx context.Context) string {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if sc.IsValid() {
		return sc.TraceID().String()
	}
	return ""
}

// RecordError records an error on the current span and sets the span status to Error.
func RecordError(ctx context.Context, err error, opts ...trace.EventOption) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err, opts...)
	span.SetStatus(codes.Error, err.Error())
}

// SetAttributes sets attributes on the current span.
func SetAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	trace.SpanFromContext(ctx).SetAttributes(attrs...)
}
