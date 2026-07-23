package logger

import (
	"context"
	"errors"
	"sync"
	"testing"

	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

type memoryExporter struct {
	mu      sync.Mutex
	records []sdklog.Record
}

func (e *memoryExporter) Export(_ context.Context, records []sdklog.Record) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i := range records {
		e.records = append(e.records, records[i].Clone())
	}

	return nil
}

func (e *memoryExporter) Shutdown(context.Context) error {
	return nil
}

func (e *memoryExporter) ForceFlush(context.Context) error {
	return nil
}

func (e *memoryExporter) Records() []sdklog.Record {
	e.mu.Lock()
	defer e.mu.Unlock()

	records := make([]sdklog.Record, len(e.records))
	copy(records, e.records)
	return records
}

func setupOTelTestProvider(t *testing.T) *memoryExporter {
	t.Helper()

	exporter := &memoryExporter{}
	provider := sdklog.NewLoggerProvider(sdklog.WithProcessor(sdklog.NewSimpleProcessor(exporter)))
	global.SetLoggerProvider(provider)
	t.Cleanup(func() {
		_ = provider.Shutdown(context.Background())
		log = nil
	})

	return exporter
}

func TestOTelSeverityConversion(t *testing.T) {
	tests := []struct {
		level        string
		severity     otellog.Severity
		severityText string
	}{
		{DEBUG, otellog.SeverityDebug, "DEBUG"},
		{INFO, otellog.SeverityInfo, "INFO"},
		{WARN, otellog.SeverityWarn, "WARN"},
		{ERROR, otellog.SeverityError, "ERROR"},
		{FATAL, otellog.SeverityFatal, "FATAL"},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			severity, severityText, _ := otelSeverity(tt.level)
			if severity != tt.severity {
				t.Fatalf("severity = %v, want %v", severity, tt.severity)
			}
			if severityText != tt.severityText {
				t.Fatalf("severity text = %q, want %q", severityText, tt.severityText)
			}
		})
	}
}

func TestOTelKeyValuesConversion(t *testing.T) {
	attrs := otelKeyValues(
		"string", "value",
		"int", 10,
		"bool", true,
		"float", 1.5,
		"err", errors.New("boom"),
		"odd",
	)

	got := attributesByKey(attrs)
	if len(got) != 5 {
		t.Fatalf("attributes length = %d, want 5", len(got))
	}
	if got["string"].Value.AsString() != "value" {
		t.Fatalf("string attr = %q, want value", got["string"].Value.AsString())
	}
	if got["int"].Value.AsInt64() != 10 {
		t.Fatalf("int attr = %d, want 10", got["int"].Value.AsInt64())
	}
	if !got["bool"].Value.AsBool() {
		t.Fatal("bool attr = false, want true")
	}
	if got["float"].Value.AsFloat64() != 1.5 {
		t.Fatalf("float attr = %f, want 1.5", got["float"].Value.AsFloat64())
	}
	if got["err"].Value.AsString() != "boom" {
		t.Fatalf("err attr = %q, want boom", got["err"].Value.AsString())
	}
}

func TestOTelEmissionIncludesContextAndBaseFields(t *testing.T) {
	exporter := setupOTelTestProvider(t)

	const requestIDKey = "request-id"
	err := NewLogger(Configuration{
		IsJSON: true,
		Level:  DEBUG,
		BaseFields: BaseFields{
			ServiceName: "orders",
			Env:         "test",
			CodeVersion: "abc123",
		},
		CTXFields: map[any]string{
			requestIDKey: "request_id",
		},
	})
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	ctx := context.WithValue(context.Background(), requestIDKey, "req-1")
	Infow(ctx, "created", "order_id", 123)

	records := exporter.Records()
	if len(records) != 1 {
		t.Fatalf("records length = %d, want 1", len(records))
	}

	record := records[0]
	if record.Severity() != otellog.SeverityInfo {
		t.Fatalf("severity = %v, want %v", record.Severity(), otellog.SeverityInfo)
	}
	if record.SeverityText() != "INFO" {
		t.Fatalf("severity text = %q, want INFO", record.SeverityText())
	}
	if record.Body().AsString() != "created" {
		t.Fatalf("body = %q, want created", record.Body().AsString())
	}

	attrs := recordAttributes(record)
	assertAttrString(t, attrs, "service_name", "orders")
	assertAttrString(t, attrs, "env", "test")
	assertAttrString(t, attrs, "code_version", "abc123")
	assertAttrString(t, attrs, "request_id", "req-1")
	if attrs["order_id"].Value.AsInt64() != 123 {
		t.Fatalf("order_id = %d, want 123", attrs["order_id"].Value.AsInt64())
	}
}

func TestOTelEmissionRespectsLevelAndOptOut(t *testing.T) {
	exporter := setupOTelTestProvider(t)

	err := NewLogger(Configuration{
		IsJSON: true,
		Level:  ERROR,
	})
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	Debug(context.Background(), "debug")
	Error(context.Background(), "error")

	records := exporter.Records()
	if len(records) != 1 {
		t.Fatalf("records length = %d, want 1", len(records))
	}
	if records[0].Severity() != otellog.SeverityError {
		t.Fatalf("severity = %v, want %v", records[0].Severity(), otellog.SeverityError)
	}

	err = NewLogger(Configuration{
		IsJSON:               true,
		Level:                DEBUG,
		DisableOpenTelemetry: true,
	})
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	Info(context.Background(), "disabled")
	if len(exporter.Records()) != 1 {
		t.Fatalf("records length changed after opt-out: got %d, want 1", len(exporter.Records()))
	}
}

func recordAttributes(record sdklog.Record) map[string]otellog.KeyValue {
	attrs := make(map[string]otellog.KeyValue)
	record.WalkAttributes(func(attr otellog.KeyValue) bool {
		attrs[attr.Key] = attr
		return true
	})
	return attrs
}

func attributesByKey(attrs []otellog.KeyValue) map[string]otellog.KeyValue {
	byKey := make(map[string]otellog.KeyValue, len(attrs))
	for _, attr := range attrs {
		byKey[attr.Key] = attr
	}
	return byKey
}

func assertAttrString(t *testing.T, attrs map[string]otellog.KeyValue, key, want string) {
	t.Helper()

	got, ok := attrs[key]
	if !ok {
		t.Fatalf("missing attr %q", key)
	}
	if got.Value.AsString() != want {
		t.Fatalf("%s = %q, want %q", key, got.Value.AsString(), want)
	}
}
