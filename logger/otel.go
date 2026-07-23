package logger

import (
	"context"
	"fmt"
	"time"

	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.uber.org/zap/zapcore"
)

type otelLogger struct {
	minLevel   zapcore.Level
	baseFields []otellog.KeyValue
}

func newOTelLogger(config Configuration) *otelLogger {
	if config.DisableOpenTelemetry {
		return nil
	}

	return &otelLogger{
		minLevel:   getZapLevel(config.Level),
		baseFields: baseFieldsToOTelAttributes(config.BaseFields),
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
	record.SetBody(otellog.StringValue(msg))
	record.AddAttributes(l.otel.baseFields...)
	record.AddAttributes(otelKeyValues(keysAndValues...)...)
	record.AddAttributes(l.otel.contextAttributes(ctx, l.ctxFields)...)

	global.Logger("github.com/delivery-much/dm-go/logger").Emit(ctx, record)
}

func (l *otelLogger) contextAttributes(ctx context.Context, ctxFields map[any]string) []otellog.KeyValue {
	if ctx == nil || len(ctxFields) == 0 {
		return nil
	}

	attrs := make([]otellog.KeyValue, 0, len(ctxFields))
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

func baseFieldsToOTelAttributes(baseFields BaseFields) []otellog.KeyValue {
	attrs := make([]otellog.KeyValue, 0, 3)
	if baseFields.ServiceName != "" {
		attrs = append(attrs, otellog.String("service_name", baseFields.ServiceName))
	}
	if baseFields.Env != "" {
		attrs = append(attrs, otellog.String("env", baseFields.Env))
	}
	if baseFields.CodeVersion != "" {
		attrs = append(attrs, otellog.String("code_version", baseFields.CodeVersion))
	}

	return attrs
}

func otelKeyValues(keysAndValues ...any) []otellog.KeyValue {
	attrs := make([]otellog.KeyValue, 0, len(keysAndValues)/2)
	for i := 0; i+1 < len(keysAndValues); i += 2 {
		key := fmt.Sprint(keysAndValues[i])
		if key == "" {
			continue
		}
		attrs = append(attrs, otelAttribute(key, keysAndValues[i+1]))
	}

	return attrs
}

func otelAttribute(key string, value any) otellog.KeyValue {
	switch v := value.(type) {
	case string:
		return otellog.String(key, v)
	case bool:
		return otellog.Bool(key, v)
	case int:
		return otellog.Int(key, v)
	case int8:
		return otellog.Int64(key, int64(v))
	case int16:
		return otellog.Int64(key, int64(v))
	case int32:
		return otellog.Int64(key, int64(v))
	case int64:
		return otellog.Int64(key, v)
	case uint:
		return otellog.Int64(key, int64(v))
	case uint8:
		return otellog.Int64(key, int64(v))
	case uint16:
		return otellog.Int64(key, int64(v))
	case uint32:
		return otellog.Int64(key, int64(v))
	case uint64:
		return otellog.Int64(key, int64(v))
	case float32:
		return otellog.Float64(key, float64(v))
	case float64:
		return otellog.Float64(key, v)
	case []byte:
		return otellog.Bytes(key, v)
	case error:
		return otellog.String(key, v.Error())
	default:
		return otellog.String(key, fmt.Sprint(v))
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
