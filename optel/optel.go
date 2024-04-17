package optel

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/riandyrn/otelchi"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"

	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

type TraceConfiguration struct {
	CTXAttributes map[any]string
}

type OptelConfiguration struct {
	Appname     string
	TraceConfig TraceConfiguration
}

var config OptelConfiguration

var globalResource *resource.Resource
var globalTracer oteltrace.Tracer

// setupOTelSDK bootstraps the OpenTelemetry pipeline.
// If it does not return an error, make sure to call shutdown for proper cleanup.
func StartOptelConnection(ctx context.Context, c OptelConfiguration) (sd func()) {
	config = c
	var shutdownFuncs []func(context.Context) error
	var err error

	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	shutdown := func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	sd = func() {
		err = errors.Join(err, shutdown(context.Background()))
	}

	// handleErr calls shutdown for cleanup and makes sure that all errors are returned.
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	globalResource, err = resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(config.Appname),
		),
	)
	if err != nil {
		handleErr(err)
		return
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

func TraceMiddleware(appName string, r chi.Routes) func(next http.Handler) http.Handler {
	healthFilter := func(r *http.Request) bool {
		if r.URL.Path == "/health" {
			return false
		}
		return true
	}
	return otelchi.Middleware(appName, otelchi.WithChiRoutes(r), otelchi.WithFilter(healthFilter))
}

func addCTXTraceAttributes(ctx context.Context, s *oteltrace.Span) {
	for k, v := range config.TraceConfig.CTXAttributes {
		fmt.Printf("Add %s context as  %s attributes", k, v)
		_v := ctx.Value(k)
		if _v != nil {
			(*s).SetAttributes(attribute.String(v, _v.(string)))
			(*s).SetAttributes(attribute.String("This", "that"))
			fmt.Printf("%s:%s", v, k)
		}
	}
}

func StartTrack(ctx context.Context, n string) func() {
	if globalTracer == nil {
		print("Error, trying to start span before initializing globalTracer")
		return func() {
			return
		}
	}
	fmt.Printf("Starting Span")
	_, span := globalTracer.Start(ctx, n)
	fmt.Printf("Adding ctx attributes")
	addCTXTraceAttributes(ctx, &span)

	return func() {
		span.End()
	}
}
