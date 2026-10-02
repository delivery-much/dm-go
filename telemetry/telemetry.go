package telemetry

import (
	"context"
	"errors"
	stdlog "log"
	"net/http"
	"os"
	"sync"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// initMu guards against concurrent or repeated initialization: a second Init
// call is rejected until the previous shutdown function has been run.
var (
	initMu      sync.Mutex
	initialized bool
)

// Init bootstraps all OpenTelemetry signals (traces, metrics, logs) using the
// default configuration (environment variables and library defaults) and returns
// a shutdown function. The shutdown function must be called before the application exits.
// Init may be called again only after the returned shutdown function has been run;
// until then, further calls return an error.
//
// Example:
//
//	shutdown, err := telemetry.Init(ctx)
//	if err != nil {
//	    log.Fatalf("failed to setup telemetry: %v", err)
//	}
//	defer shutdown(context.Background())
func Init(ctx context.Context) (shutdown func(context.Context) error, err error) {
	return InitWithConfig(ctx, Config{})
}

// InitWithConfig is like Init, but accepts an explicit Config.
// Zero-value fields fall back to environment variables and library defaults
// (see the Config field documentation).
//
// Example:
//
//	shutdown, err := telemetry.InitWithConfig(ctx, telemetry.Config{
//	    ServiceName:  "my-service",
//	    SamplerRatio: 0.1,
//	})
//	if err != nil {
//	    log.Fatalf("failed to setup telemetry: %v", err)
//	}
//	defer shutdown(context.Background())
func InitWithConfig(ctx context.Context, cfg Config) (shutdown func(context.Context) error, err error) {
	applyDefaults(&cfg)

	// Telemetry is active only when an OTLP endpoint is configured, typically
	// via OTEL_EXPORTER_OTLP_ENDPOINT. Without one, no providers are installed
	// and the OTel API stays a no-op.
	if cfg.OTLPEndpoint == "" {
		stdlog.Println("telemetry: no OTLP endpoint configured, OpenTelemetry disabled")
		return func(context.Context) error { return nil }, nil
	}

	// dm-go/logger gates its OTel emission on this exact env var. When the
	// endpoint comes from another source (signal-specific env vars or Config),
	// traces and metrics would flow while OTel logs silently stay off — warn
	// so the mismatch is visible at startup.
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		stdlog.Println("telemetry: OTLP endpoint configured without OTEL_EXPORTER_OTLP_ENDPOINT; dm-go/logger will not emit OTel logs unless that env var is set")
	}

	initMu.Lock()
	defer initMu.Unlock()

	if initialized {
		return func(context.Context) error { return nil },
			errors.New("telemetry: already initialized, run the previous shutdown function first")
	}

	var shutdownFuncs []func(context.Context) error

	runShutdown := func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	shutdown = func(ctx context.Context) error {
		initMu.Lock()
		defer initMu.Unlock()
		initialized = false
		return runShutdown(ctx)
	}

	handleErr := func(inErr error) {
		err = errors.Join(inErr, runShutdown(ctx))
	}

	// Resource
	res, err := buildResource(ctx, cfg)
	if err != nil {
		handleErr(err)
		return
	}

	// Build every enabled provider before touching any global, so a failed
	// Init never leaves a partially initialized (or already shut down)
	// provider registered globally.
	var tp *sdktrace.TracerProvider
	if *cfg.TracingEnabled {
		tp, err = newTraceProvider(ctx, cfg, res)
		if err != nil {
			handleErr(err)
			return
		}
		shutdownFuncs = append(shutdownFuncs, tp.Shutdown)
	}

	var mp *metric.MeterProvider
	if *cfg.MetricsEnabled {
		mp, err = newMeterProvider(ctx, cfg, res)
		if err != nil {
			handleErr(err)
			return
		}
		shutdownFuncs = append(shutdownFuncs, mp.Shutdown)
	}

	var lp *log.LoggerProvider
	if *cfg.LoggingEnabled {
		lp, err = newLoggerProvider(ctx, cfg, res)
		if err != nil {
			handleErr(err)
			return
		}
		shutdownFuncs = append(shutdownFuncs, lp.Shutdown)
	}

	// Every signal initialized successfully: install the globals.
	prop := cfg.Propagator
	if prop == nil {
		prop = propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		)
	}
	otel.SetTextMapPropagator(prop)

	if tp != nil {
		otel.SetTracerProvider(tp)
	}
	if mp != nil {
		otel.SetMeterProvider(mp)
	}
	if lp != nil {
		global.SetLoggerProvider(lp)
	}

	initialized = true
	return shutdown, nil
}

// hostname is os.Hostname, replaceable in tests.
var hostname = os.Hostname

func buildResource(ctx context.Context, cfg Config) (*resource.Resource, error) {
	var attrs []attribute.KeyValue

	if cfg.ServiceName != "" {
		attrs = append(attrs, semconv.ServiceName(cfg.ServiceName))
	}

	if cfg.ServiceVersion != "" {
		attrs = append(attrs, semconv.ServiceVersion(cfg.ServiceVersion))
	}

	if cfg.Environment != "" {
		attrs = append(attrs, semconv.DeploymentEnvironment(cfg.Environment))
	}

	if cfg.ServiceInstanceID != "" {
		attrs = append(attrs, semconv.ServiceInstanceID(cfg.ServiceInstanceID))
	}

	attrs = append(attrs, cfg.ResourceAttributes...)

	// Detectors are merged in order and later ones win, so the derived instance
	// id goes first: OTEL_RESOURCE_ATTRIBUTES overrides it, and the explicit
	// Config attributes override both.
	return resource.New(ctx,
		resource.WithAttributes(defaultInstanceAttributes(cfg.ServicePort)...),
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithContainer(),
		resource.WithDetectors(cfg.ResourceDetectors...),
		resource.WithAttributes(attrs...),
	)
}

// defaultInstanceAttributes derives service.instance.id as "hostname:port". The
// port keeps two processes of the same service on one host apart and, unlike a
// PID or a random id, stays the same across restarts, so no new series is born
// on every deploy. Without a hostname nothing is derived.
func defaultInstanceAttributes(port string) []attribute.KeyValue {
	host, err := hostname()
	if err != nil || host == "" {
		return nil
	}
	if port != "" {
		host += ":" + port
	}
	return []attribute.KeyValue{semconv.ServiceInstanceID(host)}
}

func newTraceProvider(ctx context.Context, cfg Config, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	opts := traceExporterOptions(cfg)

	exp, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	sampler := sdktrace.AlwaysSample()
	if *cfg.SamplerRatio < 1.0 {
		sampler = sdktrace.ParentBased(sdktrace.TraceIDRatioBased(*cfg.SamplerRatio))
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exp,
			sdktrace.WithBatchTimeout(cfg.BatchTimeout),
		),
	)

	for _, sp := range cfg.SpanProcessors {
		tp.RegisterSpanProcessor(sp)
	}

	return tp, nil
}

func newMeterProvider(ctx context.Context, cfg Config, res *resource.Resource) (*metric.MeterProvider, error) {
	exp, err := otlpmetrichttp.New(ctx, metricExporterOptions(cfg)...)
	if err != nil {
		return nil, err
	}

	opts := append(meterProviderOptions(cfg, res),
		metric.WithReader(metric.NewPeriodicReader(exp,
			metric.WithInterval(cfg.MetricInterval),
		)),
	)

	return metric.NewMeterProvider(opts...), nil
}

// meterProviderOptions returns the reader-independent MeterProvider options
// derived from cfg. A nil MetricCardinalityLimit leaves the SDK default in
// place (including the OTEL_GO_X_CARDINALITY_LIMIT env var).
func meterProviderOptions(cfg Config, res *resource.Resource) []metric.Option {
	opts := []metric.Option{metric.WithResource(res)}

	if cfg.MetricCardinalityLimit != nil {
		opts = append(opts, metric.WithCardinalityLimit(*cfg.MetricCardinalityLimit))
	}

	return opts
}

func newLoggerProvider(ctx context.Context, cfg Config, res *resource.Resource) (*log.LoggerProvider, error) {
	opts := logExporterOptions(cfg)

	exp, err := otlploghttp.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return log.NewLoggerProvider(
		log.WithResource(res),
		log.WithProcessor(log.NewBatchProcessor(exp,
			log.WithExportInterval(cfg.BatchTimeout),
		)),
	), nil
}

// Middleware returns an otelhttp middleware for instrumenting net/http servers.
// It automatically creates spans for incoming requests and records HTTP metrics.
// Requests to /health are not traced. Extra otelhttp options are applied after
// the defaults, so they can override them.
//
// Server spans follow the OpenTelemetry HTTP semantic conventions: they are
// named "{METHOD} {pattern}" when the router sets r.Pattern (net/http ServeMux
// on Go 1.22+) and "{METHOD}" otherwise. The operation string is kept for the
// otelhttp API but is not used as the span name. Wrap individual routes with
// Route to name their spans after the route pattern on any router.
//
// Example:
//
//	mux := http.NewServeMux()
//	mux.HandleFunc("GET /health", healthHandler)
//	mux.Handle("GET /api/hello", telemetry.Route("GET /api/hello", helloHandler))
//	handler := telemetry.Middleware("my-service")(mux)
func Middleware(operation string, opts ...otelhttp.Option) func(http.Handler) http.Handler {
	defaults := []otelhttp.Option{
		otelhttp.WithFilter(func(r *http.Request) bool {
			return r.URL.Path != "/health"
		}),
	}

	return func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(next, operation, append(defaults, opts...)...)
	}
}

// Route wraps a route's handler so the server span created by Middleware is
// named after the route pattern and carries the http.route attribute. Works
// with any router: the renaming happens inside the routed handler, where the
// matched pattern is known.
//
// Example:
//
//	mux.Handle("GET /api/hello", telemetry.Route("GET /api/hello", helloHandler))
func Route(pattern string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := trace.SpanFromContext(r.Context())
		span.SetName(pattern)
		span.SetAttributes(semconv.HTTPRoute(pattern))
		h.ServeHTTP(w, r)
	})
}

// Transport returns an instrumented http.RoundTripper for outbound HTTP calls.
// It automatically creates client spans and injects trace context headers.
//
// Example:
//
//	client := &http.Client{Transport: telemetry.Transport(nil)}
func Transport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return otelhttp.NewTransport(base)
}

// HTTPClient returns a pre-configured http.Client with OTel instrumentation.
//
// Example:
//
//	client := telemetry.HTTPClient()
//	resp, err := client.Get("https://api.example.com/data")
func HTTPClient() *http.Client {
	return &http.Client{
		Transport: Transport(nil),
	}
}
