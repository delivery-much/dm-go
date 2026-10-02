package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
)

// collectCounterPoints builds a MeterProvider from cfg with a ManualReader,
// records distinct attribute sets on a counter and returns the collected
// data points.
func collectCounterPoints(t *testing.T, cfg Config, distinctSets int) []metricdata.DataPoint[int64] {
	t.Helper()

	reader := metric.NewManualReader()
	opts := append(meterProviderOptions(cfg, resource.Empty()), metric.WithReader(reader))
	mp := metric.NewMeterProvider(opts...)
	t.Cleanup(func() { _ = mp.Shutdown(context.Background()) })

	counter, err := mp.Meter("test").Int64Counter("requests")
	if err != nil {
		t.Fatalf("Int64Counter() error = %v", err)
	}

	for i := 0; i < distinctSets; i++ {
		counter.Add(context.Background(), 1, otelmetric.WithAttributes(attribute.Int("id", i)))
	}

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if len(rm.ScopeMetrics) != 1 || len(rm.ScopeMetrics[0].Metrics) != 1 {
		t.Fatalf("expected exactly one metric, got %+v", rm.ScopeMetrics)
	}

	sum, ok := rm.ScopeMetrics[0].Metrics[0].Data.(metricdata.Sum[int64])
	if !ok {
		t.Fatalf("expected Sum[int64], got %T", rm.ScopeMetrics[0].Metrics[0].Data)
	}

	return sum.DataPoints
}

func hasOverflowPoint(points []metricdata.DataPoint[int64]) bool {
	for _, dp := range points {
		if v, ok := dp.Attributes.Value("otel.metric.overflow"); ok && v.AsBool() {
			return true
		}
	}
	return false
}

func TestMeterProviderOptionsAppliesCardinalityLimit(t *testing.T) {
	const limit = 3

	points := collectCounterPoints(t, Config{MetricCardinalityLimit: Int(limit)}, 10)

	if len(points) > limit {
		t.Fatalf("expected at most %d data points, got %d", limit, len(points))
	}

	if !hasOverflowPoint(points) {
		t.Fatalf("expected an otel.metric.overflow data point, got %+v", points)
	}
}

func TestMeterProviderOptionsZeroDisablesCardinalityLimit(t *testing.T) {
	t.Setenv("OTEL_GO_X_CARDINALITY_LIMIT", "")

	const distinct = 10

	points := collectCounterPoints(t, Config{MetricCardinalityLimit: Int(0)}, distinct)

	if len(points) != distinct {
		t.Fatalf("expected %d data points with unlimited cardinality, got %d", distinct, len(points))
	}

	if hasOverflowPoint(points) {
		t.Fatalf("expected no overflow data point, got %+v", points)
	}
}

func TestMeterProviderOptionsNilKeepsSDKDefault(t *testing.T) {
	// The SDK reads this env var when no explicit limit is configured; a nil
	// field must leave that behavior untouched.
	t.Setenv("OTEL_GO_X_CARDINALITY_LIMIT", "2")

	points := collectCounterPoints(t, Config{}, 10)

	if len(points) > 2 {
		t.Fatalf("expected SDK env var limit of 2 to apply, got %d data points", len(points))
	}

	if !hasOverflowPoint(points) {
		t.Fatalf("expected an otel.metric.overflow data point, got %+v", points)
	}
}

func TestInitWithConfigAcceptsMetricCardinalityLimit(t *testing.T) {
	// The periodic reader flushes on shutdown, so point the metrics exporter
	// at a local collector stub instead of the default endpoint.
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(collector.Close)

	disabled := false

	shutdown, err := InitWithConfig(context.Background(), Config{
		OTLPEndpoint:           collector.URL,
		TracingEnabled:         &disabled,
		LoggingEnabled:         &disabled,
		MetricCardinalityLimit: Int(500),
	})
	if err != nil {
		t.Fatalf("InitWithConfig() error = %v", err)
	}

	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
}
