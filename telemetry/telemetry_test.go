package telemetry

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
)

type failingDetector struct{}

func (failingDetector) Detect(context.Context) (*resource.Resource, error) {
	return nil, errors.New("detector failure")
}

func TestInitWithConfigFailureLeavesGlobalsUntouched(t *testing.T) {
	tpBefore := otel.GetTracerProvider()

	_, err := InitWithConfig(context.Background(), Config{
		OTLPEndpoint:      "localhost:4318",
		ResourceDetectors: []resource.Detector{failingDetector{}},
	})
	if err == nil {
		t.Fatal("expected error from failing resource detector")
	}

	if otel.GetTracerProvider() != tpBefore {
		t.Fatal("expected global tracer provider to be untouched after failed Init")
	}
}

func TestInitWithConfigDisabledWithoutEndpoint(t *testing.T) {
	clearEndpointEnv(t)

	tpBefore := otel.GetTracerProvider()

	shutdown, err := InitWithConfig(context.Background(), Config{})
	if err != nil {
		t.Fatalf("InitWithConfig() error = %v", err)
	}

	if otel.GetTracerProvider() != tpBefore {
		t.Fatal("expected global tracer provider to be untouched when telemetry is disabled")
	}

	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
}

func TestInitWithConfigRespectsSignalToggles(t *testing.T) {
	disabled := false

	shutdown, err := InitWithConfig(context.Background(), Config{
		OTLPEndpoint:   "localhost:4318",
		TracingEnabled: &disabled,
		MetricsEnabled: &disabled,
		LoggingEnabled: &disabled,
	})
	if err != nil {
		t.Fatalf("InitWithConfig() error = %v", err)
	}

	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
}

func TestInitWithConfigRejectsDoubleInit(t *testing.T) {
	disabled := false

	shutdown, err := InitWithConfig(context.Background(), Config{
		OTLPEndpoint:   "localhost:4318",
		TracingEnabled: &disabled,
		MetricsEnabled: &disabled,
		LoggingEnabled: &disabled,
	})
	if err != nil {
		t.Fatalf("InitWithConfig() error = %v", err)
	}

	if _, err := InitWithConfig(context.Background(), Config{OTLPEndpoint: "localhost:4318"}); err == nil {
		t.Fatal("expected error on second InitWithConfig before shutdown")
	}

	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}

	// After shutdown, initialization is allowed again.
	shutdown, err = InitWithConfig(context.Background(), Config{
		OTLPEndpoint:   "localhost:4318",
		TracingEnabled: &disabled,
		MetricsEnabled: &disabled,
		LoggingEnabled: &disabled,
	})
	if err != nil {
		t.Fatalf("InitWithConfig() after shutdown error = %v", err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
}

func TestInitWithConfigTracingOnly(t *testing.T) {
	disabled := false

	shutdown, err := InitWithConfig(context.Background(), Config{
		ServiceName:    "test-service",
		OTLPEndpoint:   "localhost:4318",
		MetricsEnabled: &disabled,
		LoggingEnabled: &disabled,
	})
	if err != nil {
		t.Fatalf("InitWithConfig() error = %v", err)
	}

	// No spans were recorded, so shutdown flushes nothing and must not error.
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
}
