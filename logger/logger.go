package logger

import "golang.org/x/net/context"

var (
	// A global variable so that log functions can be directly accessed
	log *zapLogger
)

const (
	// DEBUG has verbose message
	DEBUG = "debug"
	// INFO is default log level
	INFO = "info"
	// WARN is for logging messages about possible issues
	WARN = "warn"
	// ERROR is for logging errors
	ERROR = "error"
	// FATAL is for logging fatal messages. The sytem shutsdown after logging the message.
	FATAL = "fatal"
)

// BaseFields represents the base fields for create the basic fields of logger.
type BaseFields struct {
	ServiceName string
	Env         string
	CodeVersion string
}

// Configuration stores the config for the logger
// For some loggers there can only be one level across writers, for such the level of Console is picked by default
//
// The CTXFields value maps the fields that the logger should look for in the context to its correspondent field in the logger.
// Ex.:
//
//	Configuration{
//		CTXFields: map[string]any{
//			0: "request_id"
//		}
//	}
//
// when loggin, will look for the value stored in the 0 key in the provided context,
// and log it on the "request_id" field, alongside the log message and information.
//
// By default, if the CTX fields are specified or not, the lib will search for a request id in the context
type Configuration struct {
	IsJSON     bool
	Level      string
	BaseFields BaseFields
	CTXFields  map[any]string
}

// NewLogger returns an instance of logger
func NewLogger(config Configuration) (err error) {
	log, err = newZapLogger(config)
	return
}

// NoCTX allows access to log functions without the need to provide a context variable
func NoCTX() *zapLogger {
	if log == nil {
		return &zapLogger{}
	}

	return log
}

// getBaseFields returns the map of basic fields that should appear in every log output.
func getBaseFields(baseFields BaseFields) map[string]interface{} {
	initFields := make(map[string]interface{})
	if baseFields.ServiceName != "" {
		initFields["service_name"] = baseFields.ServiceName
	}
	if baseFields.Env != "" {
		initFields["env"] = baseFields.Env
	}
	if baseFields.CodeVersion != "" {
		initFields["code_version"] = baseFields.CodeVersion
	}

	return initFields
}

// Debug log a debug message.
func Debug(ctx context.Context, msg string) {
	log.addCTXFields(ctx).Debug(msg)
}

// Debugw logs a message with some additional context,
// With key and values, example: log.Debugw("message", "url", url, "attempt", 3)
// Keys in key-value pairs should be strings.
func Debugw(ctx context.Context, msg string, keysAndValues ...interface{}) {
	log.addCTXFields(ctx).Debugw(msg, keysAndValues...)
}

// Debugf uses fmt.Sprintf to log a templated message.
func Debugf(ctx context.Context, template string, args ...interface{}) {
	log.addCTXFields(ctx).Debugf(template, args...)
}

// Info log a info message.
func Info(ctx context.Context, msg string) {
	log.addCTXFields(ctx).Info(msg)
}

// Infow logs a message with some additional context,
// With key and values, example: log.Infow("message", "url", url, "attempt", 3)
// Keys in key-value pairs should be strings.
func Infow(ctx context.Context, msg string, keysAndValues ...interface{}) {
	log.addCTXFields(ctx).Infow(msg, keysAndValues...)
}

// Infof uses fmt.Sprintf to log a templated message.
func Infof(ctx context.Context, template string, args ...interface{}) {
	log.addCTXFields(ctx).Infof(template, args...)
}

// Warn log a warn message.
func Warn(ctx context.Context, msg string) {
	log.addCTXFields(ctx).Warn(msg)
}

// Warnw logs a message with some additional context,
// With key and values, example: log.Warnw("message", "url", url, "attempt", 3)
// Keys in key-value pairs should be strings.
func Warnw(ctx context.Context, msg string, keysAndValues ...interface{}) {
	log.addCTXFields(ctx).Warnw(msg, keysAndValues...)
}

// Warnf uses fmt.Sprintf to log a templated message.
func Warnf(ctx context.Context, template string, args ...interface{}) {
	log.addCTXFields(ctx).Warnf(template, args...)
}

// Error log a error message.
func Error(ctx context.Context, msg string) {
	log.addCTXFields(ctx).Error(msg)
}

// Errorw logs a message with some additional context,
// With key and values, example: log.Errorw("message", "url", url, "attempt", 3)
// Keys in key-value pairs should be strings.
func Errorw(ctx context.Context, msg string, keysAndValues ...interface{}) {
	log.addCTXFields(ctx).Errorw(msg, keysAndValues...)
}

// Errorf uses fmt.Sprintf to log a templated message.
func Errorf(ctx context.Context, template string, args ...interface{}) {
	log.addCTXFields(ctx).Errorf(template, args...)
}

// Fatal log a fatal message.
func Fatal(ctx context.Context, msg string) {
	log.addCTXFields(ctx).Fatal(msg)
}

// Fatalw logs a message with some additional context,
// With key and values, example: log.Fatalw("message", "url", url, "attempt", 3)
// Keys in key-value pairs should be strings.
func Fatalw(ctx context.Context, msg string, keysAndValues ...interface{}) {
	log.addCTXFields(ctx).Fatalw(msg, keysAndValues...)
}

// Fatalf uses fmt.Sprintf to log a templated message.
func Fatalf(ctx context.Context, template string, args ...interface{}) {
	log.addCTXFields(ctx).Fatalf(template, args...)
}

// Panic log a panic message.
func Panic(ctx context.Context, msg string) {
	log.addCTXFields(ctx).Panic(msg)
}

// Panicw logs a message with some additional context,
// With key and values, example: log.Panicw("message", "url", url, "attempt", 3)
// Keys in key-value pairs should be strings.
func Panicw(ctx context.Context, msg string, keysAndValues ...interface{}) {
	log.addCTXFields(ctx).Panicw(msg, keysAndValues...)
}

// Panicf uses fmt.Sprintf to log a templated message.
func Panicf(ctx context.Context, template string, args ...interface{}) {
	log.addCTXFields(ctx).Panicf(template, args...)
}
