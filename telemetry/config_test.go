package telemetry

import "testing"

// clearEndpointEnv blanks every endpoint env var so tests are isolated from
// the host environment. t.Setenv restores the originals automatically.
func clearEndpointEnv(t *testing.T) {
	t.Helper()

	for _, name := range []string{
		"OTEL_EXPORTER_OTLP_ENDPOINT",
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
		"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT",
		"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT",
		"OTEL_TRACES_OTLP_ENDPOINT",
		"OTEL_METRICS_OTLP_ENDPOINT",
		"OTEL_METRICS_HTTP_ENDPOINT",
		"OTEL_LOGS_OTLP_ENDPOINT",
	} {
		t.Setenv(name, "")
	}
}

func TestApplyDefaultsUsesOTLPEndpointFromEnv(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://collector:4318/otlp")

	cfg := Config{}

	applyDefaults(&cfg)

	if cfg.OTLPEndpoint != "http://collector:4318/otlp" {
		t.Fatalf("expected OTLP endpoint from env, got %q", cfg.OTLPEndpoint)
	}
}

func TestApplyDefaultsKeepsConfiguredOTLPEndpoint(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://collector:4318/otlp")

	cfg := Config{
		OTLPEndpoint: "custom:4318",
	}

	applyDefaults(&cfg)

	if cfg.OTLPEndpoint != "custom:4318" {
		t.Fatalf("expected configured OTLP endpoint, got %q", cfg.OTLPEndpoint)
	}
}

func TestApplyDefaultsLeavesEndpointEmptyWithoutEnv(t *testing.T) {
	clearEndpointEnv(t)

	cfg := Config{}

	applyDefaults(&cfg)

	if cfg.OTLPEndpoint != "" {
		t.Fatalf("expected empty OTLP endpoint (telemetry disabled), got %q", cfg.OTLPEndpoint)
	}
}

func TestApplyDefaultsReadsInsecureFromEnv(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")

	cfg := Config{}

	applyDefaults(&cfg)

	if !cfg.OTLPInsecure {
		t.Fatal("expected OTLPInsecure from env, got false")
	}
}

func TestApplyDefaultsKeepsInsecureFalseWithoutEnv(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "")

	cfg := Config{}

	applyDefaults(&cfg)

	if cfg.OTLPInsecure {
		t.Fatal("expected OTLPInsecure false without env, got true")
	}
}

func TestInsecureEndpointDefaultsSchemelessToInsecure(t *testing.T) {
	if !insecureEndpoint(Config{}, "collector:4318") {
		t.Fatal("expected scheme-less endpoint to be insecure")
	}
}

func TestInsecureEndpointKeepsHTTPSSecure(t *testing.T) {
	if insecureEndpoint(Config{}, "https://collector:4318/v1/traces") {
		t.Fatal("expected https endpoint to be secure")
	}
}

func TestInsecureEndpointHonorsExplicitFlag(t *testing.T) {
	if !insecureEndpoint(Config{OTLPInsecure: true}, "https://collector:4318/v1/traces") {
		t.Fatal("expected explicit OTLPInsecure to force insecure")
	}
}

func TestApplyDefaultsSamplerRatioNilBecomesOne(t *testing.T) {
	cfg := Config{}

	applyDefaults(&cfg)

	if cfg.SamplerRatio == nil || *cfg.SamplerRatio != 1.0 {
		t.Fatalf("expected default sampler ratio 1.0, got %v", cfg.SamplerRatio)
	}
}

func TestApplyDefaultsKeepsExplicitZeroSamplerRatio(t *testing.T) {
	cfg := Config{SamplerRatio: Float64(0)}

	applyDefaults(&cfg)

	if *cfg.SamplerRatio != 0 {
		t.Fatalf("expected explicit sampler ratio 0 to be kept, got %v", *cfg.SamplerRatio)
	}
}

func TestSignalEndpointAppendsSignalPathToTrailingSlashURL(t *testing.T) {
	endpoint := signalEndpoint("", "http://collector:4318/", "/v1/traces")

	if endpoint != "http://collector:4318/v1/traces" {
		t.Fatalf("expected signal path appended to trailing-slash endpoint, got %q", endpoint)
	}
}

func TestApplyDefaultsUsesSignalSpecificEndpointsFromEnv(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "http://collector:4318/v1/traces")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", "http://collector:4318/v1/metrics")
	t.Setenv("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT", "http://collector:4318/v1/logs")

	cfg := Config{}

	applyDefaults(&cfg)

	if cfg.OTLPTracesEndpoint != "http://collector:4318/v1/traces" {
		t.Fatalf("expected traces endpoint from env, got %q", cfg.OTLPTracesEndpoint)
	}
	if cfg.OTLPMetricsEndpoint != "http://collector:4318/v1/metrics" {
		t.Fatalf("expected metrics endpoint from env, got %q", cfg.OTLPMetricsEndpoint)
	}
	if cfg.OTLPLogsEndpoint != "http://collector:4318/v1/logs" {
		t.Fatalf("expected logs endpoint from env, got %q", cfg.OTLPLogsEndpoint)
	}
}

func TestApplyDefaultsUsesLegacySignalSpecificEndpointsFromEnv(t *testing.T) {
	clearEndpointEnv(t)
	t.Setenv("OTEL_TRACES_OTLP_ENDPOINT", "http://collector:4318/v1/traces")
	t.Setenv("OTEL_METRICS_OTLP_ENDPOINT", "http://collector:4318/v1/metrics")
	t.Setenv("OTEL_LOGS_OTLP_ENDPOINT", "http://collector:4318/v1/logs")

	cfg := Config{}

	applyDefaults(&cfg)

	if cfg.OTLPTracesEndpoint != "http://collector:4318/v1/traces" {
		t.Fatalf("expected traces endpoint from env, got %q", cfg.OTLPTracesEndpoint)
	}
	if cfg.OTLPMetricsEndpoint != "http://collector:4318/v1/metrics" {
		t.Fatalf("expected metrics endpoint from env, got %q", cfg.OTLPMetricsEndpoint)
	}
	if cfg.OTLPLogsEndpoint != "http://collector:4318/v1/logs" {
		t.Fatalf("expected logs endpoint from env, got %q", cfg.OTLPLogsEndpoint)
	}
}

func TestApplyDefaultsUsesLegacyMetricsHTTPEndpointFromEnv(t *testing.T) {
	clearEndpointEnv(t)
	t.Setenv("OTEL_METRICS_HTTP_ENDPOINT", "http://prometheus:9090/api/v1/otlp/v1/metrics")

	cfg := Config{}

	applyDefaults(&cfg)

	if cfg.OTLPMetricsEndpoint != "http://prometheus:9090/api/v1/otlp/v1/metrics" {
		t.Fatalf("expected metrics HTTP endpoint from env, got %q", cfg.OTLPMetricsEndpoint)
	}
}

func TestApplyDefaultsIgnoresLegacyAliasesWhenBaseEndpointEnvSet(t *testing.T) {
	clearEndpointEnv(t)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://collector:4318")
	t.Setenv("OTEL_TRACES_OTLP_ENDPOINT", "http://stale:4318/v1/traces")
	t.Setenv("OTEL_METRICS_OTLP_ENDPOINT", "http://stale:4318/v1/metrics")
	t.Setenv("OTEL_METRICS_HTTP_ENDPOINT", "http://prometheus:9090/api/v1/otlp/v1/metrics")
	t.Setenv("OTEL_LOGS_OTLP_ENDPOINT", "http://stale:4318/v1/logs")

	cfg := Config{}

	applyDefaults(&cfg)

	if cfg.OTLPTracesEndpoint != "" {
		t.Fatalf("expected legacy traces alias ignored with base endpoint set, got %q", cfg.OTLPTracesEndpoint)
	}
	if cfg.OTLPMetricsEndpoint != "" {
		t.Fatalf("expected legacy metrics aliases ignored with base endpoint set, got %q", cfg.OTLPMetricsEndpoint)
	}
	if cfg.OTLPLogsEndpoint != "" {
		t.Fatalf("expected legacy logs alias ignored with base endpoint set, got %q", cfg.OTLPLogsEndpoint)
	}
}

func TestApplyDefaultsIgnoresLegacyAliasesWhenConfigEndpointSet(t *testing.T) {
	clearEndpointEnv(t)
	t.Setenv("OTEL_METRICS_HTTP_ENDPOINT", "http://prometheus:9090/api/v1/otlp/v1/metrics")

	cfg := Config{OTLPEndpoint: "http://collector:4318"}

	applyDefaults(&cfg)

	if cfg.OTLPMetricsEndpoint != "" {
		t.Fatalf("expected legacy metrics alias ignored with configured endpoint, got %q", cfg.OTLPMetricsEndpoint)
	}
}

func TestApplyDefaultsKeepsStandardSignalEndpointsWithBaseEndpointSet(t *testing.T) {
	clearEndpointEnv(t)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://collector:4318")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", "http://other-collector:4318/v1/metrics")

	cfg := Config{}

	applyDefaults(&cfg)

	if cfg.OTLPMetricsEndpoint != "http://other-collector:4318/v1/metrics" {
		t.Fatalf("expected standard metrics endpoint to win over base endpoint, got %q", cfg.OTLPMetricsEndpoint)
	}
}

func TestApplyDefaultsDerivesBaseEndpointFromLogsEndpoint(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT", "http://collector:4318/v1/logs")

	cfg := Config{}

	applyDefaults(&cfg)

	if cfg.OTLPEndpoint != "http://collector:4318" {
		t.Fatalf("expected base OTLP endpoint derived from logs endpoint, got %q", cfg.OTLPEndpoint)
	}
}

func TestSignalEndpointAppendsSignalPathToBaseHTTPURL(t *testing.T) {
	endpoint := signalEndpoint("", "http://collector:4318", "/v1/traces")

	if endpoint != "http://collector:4318/v1/traces" {
		t.Fatalf("expected signal path appended to base endpoint, got %q", endpoint)
	}
}

func TestSignalEndpointKeepsSpecificEndpoint(t *testing.T) {
	endpoint := signalEndpoint("http://collector:4318/custom-traces", "http://collector:4318", "/v1/traces")

	if endpoint != "http://collector:4318/custom-traces" {
		t.Fatalf("expected specific endpoint, got %q", endpoint)
	}
}

func TestSignalEndpointAppendsSignalPathToSpecificHTTPURL(t *testing.T) {
	endpoint := signalEndpoint("http://collector:4318", "http://fallback:4318", "/v1/logs")

	if endpoint != "http://collector:4318/v1/logs" {
		t.Fatalf("expected signal path appended to specific endpoint, got %q", endpoint)
	}
}

func TestDeriveBaseEndpointRemovesSignalPath(t *testing.T) {
	endpoint := deriveBaseEndpoint("http://collector:4318/v1/logs")

	if endpoint != "http://collector:4318" {
		t.Fatalf("expected signal path removed from endpoint, got %q", endpoint)
	}
}
