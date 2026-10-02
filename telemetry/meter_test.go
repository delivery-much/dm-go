package telemetry

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestMeterUsesGlobalMeterProvider(t *testing.T) {
	prev := otel.GetMeterProvider()
	t.Cleanup(func() { otel.SetMeterProvider(prev) })

	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))
	t.Cleanup(func() { _ = mp.Shutdown(context.Background()) })
	otel.SetMeterProvider(mp)

	counter, err := Meter("test-scope").Int64Counter("requests")
	if err != nil {
		t.Fatalf("Int64Counter() error = %v", err)
	}
	counter.Add(context.Background(), 1)

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if len(rm.ScopeMetrics) != 1 {
		t.Fatalf("expected exactly one scope, got %+v", rm.ScopeMetrics)
	}
	if got := rm.ScopeMetrics[0].Scope.Name; got != "test-scope" {
		t.Fatalf("scope name = %q, want %q", got, "test-scope")
	}
}

func TestMeterIsNoopWithoutProvider(t *testing.T) {
	prev := otel.GetMeterProvider()
	t.Cleanup(func() { otel.SetMeterProvider(prev) })
	otel.SetMeterProvider(noop.NewMeterProvider())

	counter, err := Meter("test-scope").Int64Counter("requests")
	if err != nil {
		t.Fatalf("Int64Counter() error = %v", err)
	}

	// must not panic
	counter.Add(context.Background(), 1)
}

func TestWithHTTPDurationBucketsAppliesBoundaries(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))
	t.Cleanup(func() { _ = mp.Shutdown(context.Background()) })

	hist, err := mp.Meter("test").Float64Histogram("duration", WithHTTPDurationBuckets())
	if err != nil {
		t.Fatalf("Float64Histogram() error = %v", err)
	}
	hist.Record(context.Background(), 0.2)

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if len(rm.ScopeMetrics) != 1 || len(rm.ScopeMetrics[0].Metrics) != 1 {
		t.Fatalf("expected exactly one metric, got %+v", rm.ScopeMetrics)
	}
	data, ok := rm.ScopeMetrics[0].Metrics[0].Data.(metricdata.Histogram[float64])
	if !ok {
		t.Fatalf("expected Histogram[float64], got %T", rm.ScopeMetrics[0].Metrics[0].Data)
	}
	if len(data.DataPoints) != 1 {
		t.Fatalf("expected one data point, got %d", len(data.DataPoints))
	}

	bounds := data.DataPoints[0].Bounds
	if len(bounds) != len(HTTPDurationBucketBoundaries) {
		t.Fatalf("bounds = %v, want %v", bounds, HTTPDurationBucketBoundaries)
	}
	for i := range bounds {
		if bounds[i] != HTTPDurationBucketBoundaries[i] {
			t.Fatalf("bounds[%d] = %v, want %v", i, bounds[i], HTTPDurationBucketBoundaries[i])
		}
	}
}
