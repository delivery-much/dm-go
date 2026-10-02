package telemetry

import (
	"context"
	"errors"
	"testing"

	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// stubHostname replaces the hostname lookup for one test.
func stubHostname(t *testing.T, host string, err error) {
	t.Helper()
	previous := hostname
	hostname = func() (string, error) { return host, err }
	t.Cleanup(func() { hostname = previous })
}

func instanceID(t *testing.T, cfg Config) (string, bool) {
	t.Helper()
	applyDefaults(&cfg)
	res, err := buildResource(context.Background(), cfg)
	if err != nil {
		t.Fatalf("buildResource: %v", err)
	}
	value, ok := res.Set().Value(semconv.ServiceInstanceIDKey)
	return value.AsString(), ok
}

func TestBuildResourceDefaultsInstanceIDToHostnameAndPort(t *testing.T) {
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "")
	t.Setenv("SERVICE_PORT", "")
	stubHostname(t, "ip-10-0-1-23", nil)

	id, ok := instanceID(t, Config{ServicePort: "3000"})
	if !ok || id != "ip-10-0-1-23:3000" {
		t.Fatalf("expected ip-10-0-1-23:3000, got %q (present=%v)", id, ok)
	}
}

func TestBuildResourceReadsServicePortFromEnv(t *testing.T) {
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "")
	t.Setenv("SERVICE_PORT", "8080")
	stubHostname(t, "muchcar-go-8446bdff76-llpbp", nil)

	id, _ := instanceID(t, Config{})
	if id != "muchcar-go-8446bdff76-llpbp:8080" {
		t.Fatalf("expected muchcar-go-8446bdff76-llpbp:8080, got %q", id)
	}
}

func TestBuildResourceInstanceIDWithoutPortIsHostnameOnly(t *testing.T) {
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "")
	t.Setenv("SERVICE_PORT", "")
	stubHostname(t, "ip-10-0-1-23", nil)

	id, _ := instanceID(t, Config{})
	if id != "ip-10-0-1-23" {
		t.Fatalf("expected ip-10-0-1-23, got %q", id)
	}
}

func TestBuildResourceEnvInstanceIDOverridesDefault(t *testing.T) {
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.instance.id=pod-from-env")
	t.Setenv("SERVICE_PORT", "")
	stubHostname(t, "ip-10-0-1-23", nil)

	id, _ := instanceID(t, Config{ServicePort: "3000"})
	if id != "pod-from-env" {
		t.Fatalf("expected OTEL_RESOURCE_ATTRIBUTES to win over the derived id, got %q", id)
	}
}

func TestBuildResourceConfigInstanceIDOverridesEnv(t *testing.T) {
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.instance.id=pod-from-env")
	stubHostname(t, "ip-10-0-1-23", nil)

	id, _ := instanceID(t, Config{ServiceInstanceID: "explicit-id", ServicePort: "3000"})
	if id != "explicit-id" {
		t.Fatalf("expected Config.ServiceInstanceID to win, got %q", id)
	}
}

func TestBuildResourceUnknownHostnameDerivesNothing(t *testing.T) {
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "")
	t.Setenv("SERVICE_PORT", "")
	stubHostname(t, "", errors.New("no hostname"))

	if id, ok := instanceID(t, Config{ServicePort: "3000"}); ok {
		t.Fatalf("expected no service.instance.id, got %q", id)
	}
}
