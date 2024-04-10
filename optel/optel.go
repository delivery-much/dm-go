package optel

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	// "time"

	"github.com/go-chi/chi/v5"
	"github.com/riandyrn/otelchi"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	oteltrace "go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"

	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

var globalResource *resource.Resource
var globalTracer oteltrace.Tracer
var globalBatchTimeout int = 10

// setupOTelSDK bootstraps the OpenTelemetry pipeline.
// If it does not return an error, make sure to call shutdown for proper cleanup.
func SetupOTelSDK(appName string, ctx context.Context) (shutdown func(context.Context) error, err error) {
	fmt.Println("Initializing tracing")
	var shutdownFuncs []func(context.Context) error

	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	// handleErr calls shutdown for cleanup and makes sure that all errors are returned.
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	globalResource, err = resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(appName),
		),
	)
	if err != nil {
		return nil, err
	}

	// Set up propagator.
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	// Set up trace provider.
	tracerProvider, err := newTraceProvider(ctx)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)
	globalTracer = otel.Tracer("")
	fmt.Println("Tracer provider set")

	return
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
	)
}

func newStdoutTraceProvider(ctx context.Context) (*trace.TracerProvider, error) {
	traceExporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, err
	}
	traceProvider := trace.NewTracerProvider(
		trace.WithResource(globalResource),
		trace.WithBatcher(traceExporter),
	)
	return traceProvider, nil
}

func newTraceProvider(ctx context.Context) (*trace.TracerProvider, error) {
	traceExporter, err := otlptracehttp.New(ctx, otlptracehttp.WithInsecure())
	if err != nil {
		return nil, err
	}
	traceProvider := trace.NewTracerProvider(
		trace.WithResource(globalResource),
		trace.WithBatcher(traceExporter),
	)
	return traceProvider, nil
}

func WrapHandleFunc(route string, handler http.Handler) http.HandlerFunc {
	return http.HandlerFunc(otelhttp.WithRouteTag(route, handler).ServeHTTP)
}

func TraceMiddleware(appName string, r chi.Routes) func(next http.Handler) http.Handler {
	healthFilter := func(r *http.Request) bool {
		if r.URL.Path == "/health" {
			return false
		}
		return true
	}
	return otelchi.Middleware(appName, otelchi.WithChiRoutes(r), otelchi.WithFilter(healthFilter))
}

func StartTrack(ctx context.Context, n string) func() {
	if globalTracer == nil {
		print("Error, trying to start span before initializing globalTracer")
		return nil
	}
	_, span := globalTracer.Start(ctx, n)
	return func() {
		span.End()
	}
}
