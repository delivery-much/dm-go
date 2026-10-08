package telemetry

import (
	"errors"
	"reflect"
	"testing"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel"
)

type recordedLog struct {
	level  string
	msg    string
	fields []any
}

type recorderLogger struct {
	logs []recordedLog
}

func (r *recorderLogger) Debugw(msg string, kv ...any) { r.record("debug", msg, kv) }
func (r *recorderLogger) Infow(msg string, kv ...any)  { r.record("info", msg, kv) }
func (r *recorderLogger) Warnw(msg string, kv ...any)  { r.record("warn", msg, kv) }
func (r *recorderLogger) Errorw(msg string, kv ...any) { r.record("error", msg, kv) }

func (r *recorderLogger) record(level, msg string, kv []any) {
	r.logs = append(r.logs, recordedLog{level: level, msg: msg, fields: kv})
}

func useRecorder(t *testing.T) *recorderLogger {
	t.Helper()

	rec := &recorderLogger{}
	previous := sdkLogger
	sdkLogger = func() structuredLogger { return rec }
	t.Cleanup(func() { sdkLogger = previous })

	return rec
}

func TestOTelErrorHandlerLogsAsError(t *testing.T) {
	rec := useRecorder(t)

	otelErrorHandler{}.Handle(errors.New("traces export: connection refused"))
	otelErrorHandler{}.Handle(nil)

	want := []recordedLog{{
		level:  "error",
		msg:    "OpenTelemetry SDK error",
		fields: []any{"error", "traces export: connection refused"},
	}}
	if !reflect.DeepEqual(rec.logs, want) {
		t.Fatalf("logs = %#v, want %#v", rec.logs, want)
	}
}

func TestOTelLogSinkMapsVerbosityToLevel(t *testing.T) {
	rec := useRecorder(t)

	l := logr.New(otelLogSink{})
	l.V(1).Info("warn message")
	l.V(4).Info("info message")
	l.V(8).Info("debug message")
	l.Error(errors.New("boom"), "error message", "count", 3)

	want := []recordedLog{
		{level: "warn", msg: "warn message", fields: []any{}},
		{level: "info", msg: "info message", fields: []any{}},
		{level: "debug", msg: "debug message", fields: []any{}},
		{level: "error", msg: "error message", fields: []any{"count", 3, "error", "boom"}},
	}
	if !reflect.DeepEqual(rec.logs, want) {
		t.Fatalf("logs = %#v, want %#v", rec.logs, want)
	}
}

func TestOTelLogSinkCarriesNameAndValues(t *testing.T) {
	rec := useRecorder(t)

	l := logr.New(otelLogSink{}).WithName("sdk").WithName("trace").WithValues("component", "bsp")
	l.V(4).Info("exporting spans", "count", 10)

	want := []recordedLog{{
		level:  "info",
		msg:    "exporting spans",
		fields: []any{"otel_logger", "sdk/trace", "component", "bsp", "count", 10},
	}}
	if !reflect.DeepEqual(rec.logs, want) {
		t.Fatalf("logs = %#v, want %#v", rec.logs, want)
	}
}

func TestOTelLogSinkNormalizesKeys(t *testing.T) {
	rec := useRecorder(t)

	logr.New(otelLogSink{}).V(4).Info("odd pairs", 42, "answer", "dangling")

	want := []recordedLog{{
		level:  "info",
		msg:    "odd pairs",
		fields: []any{"42", "answer", "dangling", nil},
	}}
	if !reflect.DeepEqual(rec.logs, want) {
		t.Fatalf("logs = %#v, want %#v", rec.logs, want)
	}
}

func TestInitWithConfigInstallsSDKDiagnostics(t *testing.T) {
	rec := useRecorder(t)
	disabled := false

	shutdown, err := InitWithConfig(t.Context(), Config{
		OTLPEndpoint:   "localhost:4318",
		TracingEnabled: &disabled,
		MetricsEnabled: &disabled,
		LoggingEnabled: &disabled,
	})
	if err != nil {
		t.Fatalf("InitWithConfig() error = %v", err)
	}
	t.Cleanup(func() { _ = shutdown(t.Context()) })

	rec.logs = nil
	otel.Handle(errors.New("export failed"))

	if len(rec.logs) != 1 || rec.logs[0].level != "error" {
		t.Fatalf("expected otel.Handle to reach the structured logger, got %#v", rec.logs)
	}
}
