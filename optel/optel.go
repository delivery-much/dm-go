package optel

import (
	"context"
	"errors"
	"net/http"

	dmMiddleware "github.com/delivery-much/dm-go/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/riandyrn/otelchi"
	"go.mongodb.org/mongo-driver/event"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

type TraceConfiguration struct {
	CTXAttributes    map[any]string
	ExporterProtocol string
	Endpoint         string
}

type OptelConfiguration struct {
	Appname     string
	TraceConfig TraceConfiguration
}

var config OptelConfiguration
var requestIDField = "request_id"

var globalResource *resource.Resource
var globalTracer oteltrace.Tracer
var shutdownFuncs []func(context.Context) error

// StartOptelConnection creates all the necessary resources to send monitoring information to a OpenTelemetry colletor
// Even though this function returns an error if the setup fails, it is not recommended to kill the application in case of monitoring failure
// It is essential to run ShutdownOptelConnection when before stopping the application.
func StartOptelConnection(ctx context.Context, c OptelConfiguration) (err error) {
	config = c

	globalResource, err = resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(config.Appname),
		),
	)
	if err != nil {
		return
	}

	// Set up propagator.
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	// Set up trace provider.
	tracerProvider, err := newTraceProvider(ctx)
	if err != nil {
		err = errors.Join(err, ShutdownOptelConnection())
		return
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	// Set globalTracer to the default
	globalTracer = otel.Tracer("")
	return
}

// ShutdownOptelConnection runs every shutdown routined registered by the StartOptelConnection function.
// It is recommended to run this function using defer afte running StartOptelConnection
// Example:
//
//	err := StartOptelConnection(ctx, myConfig)
//	if err != nil {
//	  handleErrorFunction(err)
//	}
//	defer ShutdownOptelConnection()
func ShutdownOptelConnection() (err error) {
	// shutdown calls cleanup functions registered via shutdownFuncs.
	ctx := context.Background()
	for _, fn := range shutdownFuncs {
		err = errors.Join(err, fn(ctx))
	}
	shutdownFuncs = nil
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
	traceExporter, err := newTraceExporter(ctx)
	if err != nil {
		return nil, err
	}
	traceProvider := trace.NewTracerProvider(
		trace.WithResource(globalResource),
		trace.WithBatcher(traceExporter),
	)
	return traceProvider, nil
}

func newTraceExporter(ctx context.Context) (trace.SpanExporter, error) {
	if config.TraceConfig.ExporterProtocol == "http" {
		return otlptracehttp.New(ctx, otlptracehttp.WithInsecure(), otlptracehttp.WithEndpointURL(config.TraceConfig.Endpoint))
	}
	if config.TraceConfig.ExporterProtocol == "stdout" {
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	}
	return otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure(), otlptracegrpc.WithEndpointURL(config.TraceConfig.Endpoint))
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
		return r.URL.Path != "/health"
	}
	return otelchi.Middleware(
		appName, otelchi.WithChiRoutes(r),
		otelchi.WithFilter(healthFilter),
	)
}

// TraceMiddlewares returns a slice of Chi middlewares that implement OpenTelemetry tracing and integrate with the dm-go/middleware/request_id.
//
// It is best to apply this middleware right after the request_id middleware in the middleware chain.
//
// Example:
//
//	r := chi.NewRouter()
//	r.Use(optel.TraceMiddlewares("rochelle-coupon", r)...) // Unpack the slice before passing it as an argument.
func TraceMiddlewares(appName string, r chi.Routes) (middlewares []func(next http.Handler) http.Handler) {
	middlewares = append(middlewares, getFilteredOtelchi(appName, r))
	middlewares = append(middlewares, getReqIdMiddleware)
	return
}

func addCTXTraceAttributes(ctx context.Context, s oteltrace.Span) {
	// Add Request ID if found
	s.SetAttributes(attribute.String(requestIDField, dmMiddleware.GetReqID(ctx)))

	// Add configured CTXAttributes if found
	for contextKey, attributeKey := range config.TraceConfig.CTXAttributes {
		if value, ok := ctx.Value(contextKey).(string); ok {
			s.SetAttributes(attribute.String(attributeKey, value))
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

// StartTrack is used to start a new span (checkpoint) in the application trace.
//
// The function starts tracking as soon as it is called, but it is essential to call the returned end function using defer.
//
// One-liner example (preferred usage):
//
//	defer StartTrack(ctx, "myEventName")()
//
// Verbose example:
//
//	endFunction := StartTrack(ctx, "myEventName")
//	defer endFunction()
func StartTrack(ctx context.Context, n string) func() {
	if globalTracer == nil {
		print("Error: attempting to start span before initializing globalTracer")
		return func() {}
	}
	_, span := globalTracer.Start(ctx, n)

	addCTXTraceAttributes(ctx, span)

	return func() {
		span.End()
	}
}

// NewMongoMonitor returns a new *event.CommandMonitor for the mongodb client
//
// # The returned event monitor is meant to be set in the mongoDB clientOptions through the mongo.NewMonitor function
//
// Example:
//
//	 c := &Client{}
//	 clientOptions := options.Client().
//		 ApplyURI(uri)
//	 clientOptions.SetMonitor(optel.NewMongoMonitor())
//	 c.conn, err = mongo.Connect(ctx, clientOptions)
func NewMongoMonitor() *event.CommandMonitor {
	return otelmongo.NewMonitor()
}
