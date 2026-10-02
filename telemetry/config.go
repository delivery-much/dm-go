package telemetry

import (
	"os"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Config holds the configuration for the OpenTelemetry SDK setup.
type Config struct {
	// ServiceName is the name of the service (maps to service.name resource attribute).
	ServiceName string

	// ServiceVersion is the version of the service (maps to service.version resource attribute).
	ServiceVersion string

	// Environment is the deployment environment (maps to the deployment.environment
	// resource attribute, per semconv v1.26).
	Environment string

	// ServiceInstanceID identifies this process among the replicas of the service
	// (maps to the service.instance.id resource attribute, which the Prometheus side
	// of the collector turns into the instance label). When empty, the value of
	// OTEL_RESOURCE_ATTRIBUTES is honored; when that has none either, the library
	// defaults to "hostname:ServicePort" — the pod name on Kubernetes, the host
	// name on EC2, the container id under Docker, plus the port so two processes
	// on one host stay apart. Without an instance id, counters from different
	// replicas or from consecutive deploys collapse into a single series and
	// rate/increase go wrong.
	ServiceInstanceID string

	// ServicePort is the port the service listens on, used only to build the
	// default ServiceInstanceID. Can also be set via the SERVICE_PORT env var.
	ServicePort string

	// OTLPEndpoint is the OTLP collector endpoint (e.g. "localhost:4318").
	// Can also be set via OTEL_EXPORTER_OTLP_ENDPOINT env var.
	// If no endpoint is configured (field or env vars), telemetry is disabled
	// entirely: Init installs no providers and returns a no-op shutdown.
	OTLPEndpoint string

	// OTLPTracesEndpoint is the OTLP traces endpoint.
	// Can also be set via the OTEL_EXPORTER_OTLP_TRACES_ENDPOINT env var, or —
	// only when no base endpoint is configured — the legacy
	// OTEL_TRACES_OTLP_ENDPOINT alias.
	OTLPTracesEndpoint string

	// OTLPMetricsEndpoint is the OTLP metrics endpoint.
	// Can also be set via the OTEL_EXPORTER_OTLP_METRICS_ENDPOINT env var, or —
	// only when no base endpoint is configured — the legacy
	// OTEL_METRICS_OTLP_ENDPOINT / OTEL_METRICS_HTTP_ENDPOINT aliases.
	OTLPMetricsEndpoint string

	// OTLPLogsEndpoint is the OTLP logs endpoint.
	// Can also be set via the OTEL_EXPORTER_OTLP_LOGS_ENDPOINT env var, or —
	// only when no base endpoint is configured — the legacy
	// OTEL_LOGS_OTLP_ENDPOINT alias.
	OTLPLogsEndpoint string

	// OTLPInsecure forces plain HTTP for the OTLP exporters.
	// Can also be set via the OTEL_EXPORTER_OTLP_INSECURE env var.
	// Scheme-less endpoints (e.g. "collector:4318") are already treated as
	// insecure by default — use an https:// endpoint URL to enable TLS.
	OTLPInsecure bool

	// OTLPHeaders are additional headers sent with every OTLP request.
	OTLPHeaders map[string]string

	// SamplerRatio controls trace sampling (0.0 to 1.0). Defaults to 1.0
	// (sample everything) when nil; 0.0 disables sampling (parent-based, so
	// spans with a sampled remote parent are still recorded).
	// Use the Float64 helper: SamplerRatio: telemetry.Float64(0.1).
	SamplerRatio *float64

	// MetricInterval is the interval between metric exports. Defaults to 60s.
	MetricInterval time.Duration

	// MetricCardinalityLimit caps the number of distinct attribute sets
	// collected per metric instrument in a single collect cycle. Once reached,
	// new attribute sets are aggregated into a single overflow series tagged
	// otel.metric.overflow=true. When nil, the SDK default applies (2000, or
	// the OTEL_GO_X_CARDINALITY_LIMIT env var). 0 or a negative value disables
	// the limit. Use the Int helper: MetricCardinalityLimit: telemetry.Int(0).
	MetricCardinalityLimit *int

	// BatchTimeout is the maximum time before a trace batch is exported. Defaults to 10s.
	BatchTimeout time.Duration

	// TracingEnabled enables the trace signal. Defaults to true.
	TracingEnabled *bool

	// MetricsEnabled enables the metrics signal. Defaults to true.
	MetricsEnabled *bool

	// LoggingEnabled enables the logs signal. Defaults to true.
	LoggingEnabled *bool

	// ResourceDetectors are additional resource detectors (e.g. AWS EC2, EKS).
	ResourceDetectors []resource.Detector

	// ResourceAttributes are additional resource attributes to include.
	ResourceAttributes []attribute.KeyValue

	// SpanProcessors are additional span processors to register.
	SpanProcessors []sdktrace.SpanProcessor

	// Propagator overrides the default W3C TraceContext + Baggage propagator.
	Propagator propagation.TextMapPropagator
}

func applyDefaults(cfg *Config) {
	if cfg.ServiceName == "" {
		cfg.ServiceName = os.Getenv("SERVICE_NAME")
	}

	if cfg.ServiceVersion == "" {
		cfg.ServiceVersion = os.Getenv("CODE_VERSION")
	}

	if cfg.Environment == "" {
		cfg.Environment = os.Getenv("ENVIRONMENT")
	}

	if cfg.ServicePort == "" {
		cfg.ServicePort = os.Getenv("SERVICE_PORT")
	}

	if cfg.OTLPEndpoint == "" {
		cfg.OTLPEndpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	}

	// Legacy alias env vars are honored only when no base endpoint is
	// configured (Config field or OTEL_EXPORTER_OTLP_ENDPOINT): a stale alias
	// left over from an older telemetry stack must not silently redirect one
	// signal away from an explicitly configured collector. The standard
	// signal-specific vars keep their spec-defined precedence over the base
	// endpoint.
	useLegacyAliases := cfg.OTLPEndpoint == ""

	if cfg.OTLPTracesEndpoint == "" {
		cfg.OTLPTracesEndpoint = os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
		if cfg.OTLPTracesEndpoint == "" && useLegacyAliases {
			cfg.OTLPTracesEndpoint = os.Getenv("OTEL_TRACES_OTLP_ENDPOINT")
		}
	}

	if cfg.OTLPMetricsEndpoint == "" {
		cfg.OTLPMetricsEndpoint = os.Getenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT")
		if cfg.OTLPMetricsEndpoint == "" && useLegacyAliases {
			cfg.OTLPMetricsEndpoint = firstEnv("OTEL_METRICS_OTLP_ENDPOINT", "OTEL_METRICS_HTTP_ENDPOINT")
		}
	}

	if cfg.OTLPLogsEndpoint == "" {
		cfg.OTLPLogsEndpoint = os.Getenv("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT")
		if cfg.OTLPLogsEndpoint == "" && useLegacyAliases {
			cfg.OTLPLogsEndpoint = os.Getenv("OTEL_LOGS_OTLP_ENDPOINT")
		}
	}

	if cfg.OTLPEndpoint == "" {
		cfg.OTLPEndpoint = firstDerivedBaseEndpoint(
			cfg.OTLPTracesEndpoint,
			cfg.OTLPMetricsEndpoint,
			cfg.OTLPLogsEndpoint,
		)
	}

	if cfg.SamplerRatio == nil {
		cfg.SamplerRatio = Float64(1.0)
	}

	if cfg.MetricInterval == 0 {
		cfg.MetricInterval = 60 * time.Second
	}

	if cfg.BatchTimeout == 0 {
		cfg.BatchTimeout = 10 * time.Second
	}

	if cfg.TracingEnabled == nil {
		cfg.TracingEnabled = boolPtr(true)
	}

	if cfg.MetricsEnabled == nil {
		cfg.MetricsEnabled = boolPtr(true)
	}

	if cfg.LoggingEnabled == nil {
		cfg.LoggingEnabled = boolPtr(true)
	}

	if !cfg.OTLPInsecure {
		if insecure, err := strconv.ParseBool(os.Getenv("OTEL_EXPORTER_OTLP_INSECURE")); err == nil {
			cfg.OTLPInsecure = insecure
		}
	}
}

func boolPtr(b bool) *bool {
	return &b
}

// Bool returns a pointer to b, for the Config *bool toggle fields.
func Bool(b bool) *bool {
	return &b
}

// Float64 returns a pointer to f, for Config.SamplerRatio.
func Float64(f float64) *float64 {
	return &f
}

// Int returns a pointer to i, for Config.MetricCardinalityLimit.
func Int(i int) *int {
	return &i
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	return ""
}

func firstDerivedBaseEndpoint(endpoints ...string) string {
	for _, endpoint := range endpoints {
		if baseEndpoint := deriveBaseEndpoint(endpoint); baseEndpoint != "" {
			return baseEndpoint
		}
	}
	return ""
}
