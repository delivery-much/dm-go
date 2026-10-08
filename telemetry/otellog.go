package telemetry

import (
	"fmt"

	"github.com/go-logr/logr"

	"github.com/delivery-much/dm-go/logger"
)

// structuredLogger is the subset of dm-go/logger used to forward the
// OpenTelemetry SDK's own diagnostics.
type structuredLogger interface {
	Debugw(msg string, keysAndValues ...any)
	Infow(msg string, keysAndValues ...any)
	Warnw(msg string, keysAndValues ...any)
	Errorw(msg string, keysAndValues ...any)
}

// sdkLogger resolves the destination of SDK diagnostics at log time, so the
// logger configured later by logger.NewLogger is picked up. Replaceable in tests.
var sdkLogger = func() structuredLogger { return logger.NoCTX() }

// Verbosity levels used by the OpenTelemetry SDK internal logger
// (go.opentelemetry.io/otel/internal/global): Warn is V(1), Info is V(4), Debug is V(8).
const (
	otelVerbosityInfo  = 4
	otelVerbosityDebug = 8
)

// otelErrorHandler receives the errors the SDK reports through otel.Handle,
// such as failed OTLP exports, and writes them as structured error logs.
type otelErrorHandler struct{}

func (otelErrorHandler) Handle(err error) {
	if err == nil {
		return
	}
	sdkLogger().Errorw("OpenTelemetry SDK error", "error", err.Error())
}

// otelLogSink is a logr.LogSink that routes the SDK internal logger (dropped
// spans, attribute limits, provider lifecycle) to dm-go/logger.
type otelLogSink struct {
	name   string
	values []any
}

func (otelLogSink) Init(logr.RuntimeInfo) {}

// Enabled accepts every verbosity; the configured dm-go/logger level decides what is written.
func (otelLogSink) Enabled(int) bool { return true }

func (s otelLogSink) Info(level int, msg string, keysAndValues ...any) {
	fields := s.fields(keysAndValues)

	switch {
	case level >= otelVerbosityDebug:
		sdkLogger().Debugw(msg, fields...)
	case level >= otelVerbosityInfo:
		sdkLogger().Infow(msg, fields...)
	default:
		sdkLogger().Warnw(msg, fields...)
	}
}

func (s otelLogSink) Error(err error, msg string, keysAndValues ...any) {
	fields := s.fields(keysAndValues)
	if err != nil {
		fields = append(fields, "error", err.Error())
	}
	sdkLogger().Errorw(msg, fields...)
}

func (s otelLogSink) WithValues(keysAndValues ...any) logr.LogSink {
	values := make([]any, 0, len(s.values)+len(keysAndValues))
	values = append(values, s.values...)
	values = append(values, keysAndValues...)
	return otelLogSink{name: s.name, values: values}
}

func (s otelLogSink) WithName(name string) logr.LogSink {
	if s.name != "" {
		name = s.name + "/" + name
	}
	return otelLogSink{name: name, values: s.values}
}

// fields merges the sink name, the accumulated values and the call's pairs,
// forcing every key to a string so the zap-backed logger never drops a pair.
func (s otelLogSink) fields(keysAndValues []any) []any {
	fields := make([]any, 0, 2+len(s.values)+len(keysAndValues)+2)
	if s.name != "" {
		fields = append(fields, "otel_logger", s.name)
	}
	fields = appendPairs(fields, s.values)
	return appendPairs(fields, keysAndValues)
}

func appendPairs(dst, pairs []any) []any {
	for i := 0; i < len(pairs); i += 2 {
		key, ok := pairs[i].(string)
		if !ok {
			key = fmt.Sprint(pairs[i])
		}

		var value any
		if i+1 < len(pairs) {
			value = pairs[i+1]
		}
		dst = append(dst, key, value)
	}
	return dst
}
