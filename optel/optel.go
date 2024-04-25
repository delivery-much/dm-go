package optel

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	dmMiddleware "github.com/delivery-much/dm-go/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/riandyrn/otelchi"
	"go.mongodb.org/mongo-driver/event"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"

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
var requestIDField = "request_id"

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

func getReqIdMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		defer StartTrack(ctx, "ReqID")()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getFilteredOtelchi(appName string, r chi.Routes) func(next http.Handler) http.Handler {
	healthFilter := func(r *http.Request) bool {
		if r.URL.Path == "/health" {
			return false
		}
		return true
	}
	return otelchi.Middleware(
		appName, otelchi.WithChiRoutes(r),
		otelchi.WithFilter(healthFilter),
	)
}

func TraceMiddlewares(appName string, r chi.Routes) []func(next http.Handler) http.Handler {
	var middlewares []func(next http.Handler) http.Handler
	middlewares = append(middlewares, getFilteredOtelchi(appName, r))
	middlewares = append(middlewares, getReqIdMiddleware)
	return middlewares
}

func addCTXTraceAttributes(ctx context.Context, s *oteltrace.Span) {
	// Add Request ID if found
	(*s).SetAttributes(attribute.String(requestIDField, dmMiddleware.GetReqID(ctx)))

	// Add configured CTXAttributes if found
	for k, v := range config.TraceConfig.CTXAttributes {
		if _v, ok := ctx.Value(k).(string); ok {
			(*s).SetAttributes(attribute.String(v, _v))
		}
	}
}

func TraceIdFromContext(ctx context.Context) string {
	sp := oteltrace.SpanFromContext(ctx)
	if sp.SpanContext().IsValid() {
		return sp.SpanContext().TraceID().String()
	}
	return ""
}

func StartTrack(ctx context.Context, n string) func() {
	if globalTracer == nil {
		print("Error, trying to start span before initializing globalTracer")
		return func() {
			return
		}
	}
	_, span := globalTracer.Start(ctx, n)

	addCTXTraceAttributes(ctx, &span)

	return func() {
		span.End()
	}
}

func NewMongoMonitor() *event.CommandMonitor {
	return otelmongo.NewMonitor()
}
