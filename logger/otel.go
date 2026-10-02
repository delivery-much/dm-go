package logger

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.uber.org/zap/zapcore"
)

const otelExporterOTLPEndpointEnv = "OTEL_EXPORTER_OTLP_ENDPOINT"

type otelLogger struct {
	minLevel zapcore.Level
}

func msgFromFormat(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}

func newOTelLogger(config Configuration) *otelLogger {
	if config.DisableOpenTelemetry {
		return nil
	}
	if os.Getenv(otelExporterOTLPEndpointEnv) == "" {
		return nil
	}

	return &otelLogger{
		minLevel: getZapLevel(config.Level),
	}
}

func (l *zapLogger) emitOTel(ctx context.Context, level string, msg string, keysAndValues ...any) {
	if l == nil || l.otel == nil {
		return
	}

	severity, severityText, zapLevel := otelSeverity(level)
	if zapLevel < l.otel.minLevel {
		return
	}

	now := time.Now()
	record := otellog.Record{}
	record.SetTimestamp(now)
	record.SetObservedTimestamp(now)
	record.SetSeverity(severity)
	record.SetSeverityText(severityText)
	record.SetBody(attribute.StringValue(msg))
	// Service metadata (name, version, environment) is not stamped on each
	// record: it comes from the LoggerProvider's resource, set up by
	// the telemetry package's Init.
	record.AddAttributes(otelKeyValues(keysAndValues...)...)
	record.AddAttributes(l.otel.contextAttributes(ctx, l.ctxFields)...)

	global.Logger("github.com/delivery-much/dm-go/logger").Emit(ctx, record)
}

func (l *otelLogger) contextAttributes(ctx context.Context, ctxFields map[any]string) []attribute.KeyValue {
	if ctx == nil || len(ctxFields) == 0 {
		return nil
	}

	attrs := make([]attribute.KeyValue, 0, len(ctxFields))
	for key, field := range ctxFields {
		if field == "" {
			continue
		}
		if val := ctx.Value(key); val != nil {
			attrs = append(attrs, otelAttribute(field, val))
		}
	}

	return attrs
}

func otelKeyValues(keysAndValues ...any) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, len(keysAndValues)/2)
	for i := 0; i+1 < len(keysAndValues); i += 2 {
		key := fmt.Sprint(keysAndValues[i])
		if key == "" {
			continue
		}
		attrs = append(attrs, otelAttribute(key, keysAndValues[i+1]))
	}

	return attrs
}

func otelAttribute(key string, value any) attribute.KeyValue {
	switch v := value.(type) {
	case string:
		return attribute.String(key, v)
	case bool:
		return attribute.Bool(key, v)
	case int:
		return attribute.Int(key, v)
	case int8:
		return attribute.Int64(key, int64(v))
	case int16:
		return attribute.Int64(key, int64(v))
	case int32:
		return attribute.Int64(key, int64(v))
	case int64:
		return attribute.Int64(key, v)
	case uint:
		return attribute.Int64(key, int64(v))
	case uint8:
		return attribute.Int64(key, int64(v))
	case uint16:
		return attribute.Int64(key, int64(v))
	case uint32:
		return attribute.Int64(key, int64(v))
	case uint64:
		return attribute.Int64(key, int64(v))
	case float32:
		return attribute.Float64(key, float64(v))
	case float64:
		return attribute.Float64(key, v)
	case []byte:
		return attribute.ByteSlice(key, v)
	case error:
		return attribute.String(key, v.Error())
	default:
		return attribute.String(key, fmt.Sprint(v))
	}
}

func otelSeverity(level string) (otellog.Severity, string, zapcore.Level) {
	switch level {
	case DEBUG:
		return otellog.SeverityDebug, "DEBUG", zapcore.DebugLevel
	case WARN:
		return otellog.SeverityWarn, "WARN", zapcore.WarnLevel
	case ERROR:
		return otellog.SeverityError, "ERROR", zapcore.ErrorLevel
	case FATAL:
		return otellog.SeverityFatal, "FATAL", zapcore.FatalLevel
	default:
		return otellog.SeverityInfo, "INFO", zapcore.InfoLevel
	}
}
