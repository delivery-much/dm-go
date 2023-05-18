package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/net/context"

	middlewareLib "github.com/delivery-much/dm-go/middleware"
)

// represents the request id field on the log body
const requestIDField = "request_id"

type zapLogger struct {
	sugaredLogger *zap.SugaredLogger
	ctxFields     map[any]string
}

func getZapLevel(level string) zapcore.Level {
	switch level {
	case INFO:
		return zapcore.InfoLevel
	case WARN:
		return zapcore.WarnLevel
	case DEBUG:
		return zapcore.DebugLevel
	case ERROR:
		return zapcore.ErrorLevel
	case FATAL:
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// newZapLogger Create a logger with prod or dev config.
// isJSON for JSON output and production config and encoder
// serviceName for set the default field service name in logger
func newZapLogger(config Configuration) (*zapLogger, error) {
	var c zap.Config

	if config.IsJSON {
		encoderConfig := zap.NewProductionEncoderConfig()
		encoderConfig.MessageKey = "message"
		encoderConfig.TimeKey = "time"
		encoderConfig.StacktraceKey = "stack"
		encoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder

		c = zap.NewProductionConfig()
		c.DisableCaller = true
		c.InitialFields = getBaseFields(config.BaseFields)
		c.EncoderConfig = encoderConfig
		c.Level = zap.NewAtomicLevelAt(getZapLevel(config.Level))
	} else {
		c = zap.NewDevelopmentConfig()
		c.EncoderConfig = zap.NewDevelopmentEncoderConfig()
		c.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	logger, err := c.Build()
	if err != nil {
		return nil, err
	}
	defer logger.Sync()

	if config.CTXFields == nil {
		config.CTXFields = map[any]string{}
	}

	config.CTXFields[middlewareLib.RequestIDKey] = requestIDField

	return &zapLogger{
		sugaredLogger: logger.Sugar(),
		ctxFields:     config.CTXFields,
	}, nil
}

// addCTXFields adds context related information in a given logger,
// and then returns a pointer to a copy of the original logger with the new information
func (l zapLogger) addCTXFields(ctx context.Context) (zl *zapLogger) {
	if l.sugaredLogger == nil {
		return
	}

	zl = &l

	// add context values to log
	fieldsAndVals := []any{}
	for key, field := range l.ctxFields {
		val := ctx.Value(key)
		if val == nil {
			continue
		}

		fieldsAndVals = append(fieldsAndVals, field)
		fieldsAndVals = append(fieldsAndVals, val)
	}

	l.sugaredLogger = l.sugaredLogger.With(fieldsAndVals...)
	return
}

func (l *zapLogger) AddRequestID(requestID string) *zapLogger {
	return &zapLogger{
		sugaredLogger: l.sugaredLogger.With(requestIDField, requestID),
	}
}

func (l *zapLogger) Debug(msg string) {
	l.sugaredLogger.Debug(msg)
}

func (l *zapLogger) Debugw(msg string, keysAndValues ...interface{}) {
	l.sugaredLogger.Debugw(msg, keysAndValues...)
}

func (l *zapLogger) Debugf(format string, args ...interface{}) {
	l.sugaredLogger.Debugf(format, args...)
}

func (l *zapLogger) Info(msg string) {
	l.sugaredLogger.Info(msg)
}

func (l *zapLogger) Infow(msg string, keysAndValues ...interface{}) {
	l.sugaredLogger.Infow(msg, keysAndValues...)
}

func (l *zapLogger) Infof(format string, args ...interface{}) {
	l.sugaredLogger.Infof(format, args...)
}

func (l *zapLogger) Warn(msg string) {
	l.sugaredLogger.Warn(msg)
}

func (l *zapLogger) Warnw(msg string, keysAndValues ...interface{}) {
	l.sugaredLogger.Warnw(msg, keysAndValues...)
}

func (l *zapLogger) Warnf(format string, args ...interface{}) {
	l.sugaredLogger.Warnf(format, args...)
}

func (l *zapLogger) Error(msg string) {
	l.sugaredLogger.Error(msg)
}

func (l *zapLogger) Errorw(msg string, keysAndValues ...interface{}) {
	l.sugaredLogger.Errorw(msg, keysAndValues...)
}

func (l *zapLogger) Errorf(format string, args ...interface{}) {
	l.sugaredLogger.Errorf(format, args...)
}

func (l *zapLogger) Fatal(msg string) {
	l.sugaredLogger.Fatal(msg)
}

func (l *zapLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	l.sugaredLogger.Fatalw(msg, keysAndValues...)
}

func (l *zapLogger) Fatalf(format string, args ...interface{}) {
	l.sugaredLogger.Fatalf(format, args...)
}

func (l *zapLogger) Panic(msg string) {
	l.sugaredLogger.Panic(msg)
}

func (l *zapLogger) Panicw(msg string, keysAndValues ...interface{}) {
	l.sugaredLogger.Panicw(msg, keysAndValues...)
}

func (l *zapLogger) Panicf(format string, args ...interface{}) {
	l.sugaredLogger.Panicf(format, args...)
}
