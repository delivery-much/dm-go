package telemetry

import (
	"net/url"
	"strings"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
)

func traceExporterOptions(cfg Config) []otlptracehttp.Option {
	endpoint := signalEndpoint(cfg.OTLPTracesEndpoint, cfg.OTLPEndpoint, "/v1/traces")
	opts := traceEndpointOptions(endpoint)

	if insecureEndpoint(cfg, endpoint) {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	if len(cfg.OTLPHeaders) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(cfg.OTLPHeaders))
	}

	return opts
}

func metricExporterOptions(cfg Config) []otlpmetrichttp.Option {
	endpoint := signalEndpoint(cfg.OTLPMetricsEndpoint, cfg.OTLPEndpoint, "/v1/metrics")
	opts := metricEndpointOptions(endpoint)

	if insecureEndpoint(cfg, endpoint) {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	}

	if len(cfg.OTLPHeaders) > 0 {
		opts = append(opts, otlpmetrichttp.WithHeaders(cfg.OTLPHeaders))
	}

	return opts
}

func logExporterOptions(cfg Config) []otlploghttp.Option {
	endpoint := signalEndpoint(cfg.OTLPLogsEndpoint, cfg.OTLPEndpoint, "/v1/logs")
	opts := logEndpointOptions(endpoint)

	if insecureEndpoint(cfg, endpoint) {
		opts = append(opts, otlploghttp.WithInsecure())
	}

	if len(cfg.OTLPHeaders) > 0 {
		opts = append(opts, otlploghttp.WithHeaders(cfg.OTLPHeaders))
	}

	return opts
}

func traceEndpointOptions(endpoint string) []otlptracehttp.Option {
	if hasScheme(endpoint) {
		return []otlptracehttp.Option{otlptracehttp.WithEndpointURL(endpoint)}
	}

	return []otlptracehttp.Option{otlptracehttp.WithEndpoint(endpoint)}
}

func metricEndpointOptions(endpoint string) []otlpmetrichttp.Option {
	if hasScheme(endpoint) {
		return []otlpmetrichttp.Option{otlpmetrichttp.WithEndpointURL(endpoint)}
	}

	return []otlpmetrichttp.Option{otlpmetrichttp.WithEndpoint(endpoint)}
}

func logEndpointOptions(endpoint string) []otlploghttp.Option {
	if hasScheme(endpoint) {
		return []otlploghttp.Option{otlploghttp.WithEndpointURL(endpoint)}
	}

	return []otlploghttp.Option{otlploghttp.WithEndpoint(endpoint)}
}

func hasScheme(endpoint string) bool {
	return strings.Contains(endpoint, "://")
}

// insecureEndpoint reports whether the exporter should use plain HTTP.
// Scheme-less endpoints (host:port) are treated as insecure — the typical
// local collector agent; use an https:// endpoint URL for TLS.
func insecureEndpoint(cfg Config, endpoint string) bool {
	return cfg.OTLPInsecure || !hasScheme(endpoint)
}

func signalEndpoint(signalEndpoint, baseEndpoint, signalPath string) string {
	endpoint := baseEndpoint
	if signalEndpoint != "" {
		endpoint = signalEndpoint
	}

	if !hasScheme(endpoint) {
		return endpoint
	}

	u, err := url.Parse(endpoint)
	if err != nil || (u.Path != "" && u.Path != "/") {
		return endpoint
	}

	u.Path = signalPath
	return u.String()
}

func deriveBaseEndpoint(endpoint string) string {
	if endpoint == "" || !hasScheme(endpoint) {
		return endpoint
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}

	if strings.HasPrefix(u.Path, "/v1/") {
		u.Path = ""
		u.RawPath = ""
	}

	return u.String()
}
