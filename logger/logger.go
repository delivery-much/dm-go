package logger

// A global variable so that log functions can be directly accessed
var log Logger

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

//Logger is our contract for the logger
type Logger interface {
	Debug(msg string)
	Debugw(msg string, keysAndValues ...interface{})
	Debugf(template string, args ...interface{})

	Info(msg string)
	Infow(msg string, keysAndValues ...interface{})
	Infof(template string, args ...interface{})

	Warn(msg string)
	Warnw(msg string, keysAndValues ...interface{})
	Warnf(template string, args ...interface{})

	Error(msg string)
	Errorw(msg string, keysAndValues ...interface{})
	Errorf(template string, args ...interface{})

	Fatal(msg string)
	Fatalw(msg string, keysAndValues ...interface{})
	Fatalf(template string, args ...interface{})

	Panic(msg string)
	Panicw(msg string, keysAndValues ...interface{})
	Panicf(template string, args ...interface{})
}

func init() {
	log, _ = newZapLogger(Configuration{})
}

// BaseFields represents the base fields for create the basic fields of logger.
type BaseFields struct {
	ServiceName string
	Env         string
	CodeVersion string
}

// Configuration stores the config for the logger
// For some loggers there can only be one level across writers, for such the level of Console is picked by default
type Configuration struct {
	IsJSON     bool
	Level      string
	BaseFields BaseFields
}

//NewLogger returns an instance of logger
func NewLogger(config Configuration) error {
	logger, err := newZapLogger(config)
	if err != nil {
		return err
	}
	log = logger
	return nil
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
func Debug(msg string) {
	log.Debug(msg)
}

// Debugw logs a message with some additional context,
// With key and values, example: log.Debugw("message", "url", url, "attempt", 3)
// Keys in key-value pairs should be strings.
func Debugw(msg string, keysAndValues ...interface{}) {
	log.Debugw(msg, keysAndValues...)
}

// Debugf uses fmt.Sprintf to log a templated message.
func Debugf(template string, args ...interface{}) {
	log.Debugf(template, args...)
}

// Info log a info message.
func Info(msg string) {
	log.Info(msg)
}

// Infow logs a message with some additional context,
// With key and values, example: log.Infow("message", "url", url, "attempt", 3)
// Keys in key-value pairs should be strings.
func Infow(msg string, keysAndValues ...interface{}) {
	log.Infow(msg, keysAndValues...)
}

// Infof uses fmt.Sprintf to log a templated message.
func Infof(template string, args ...interface{}) {
	log.Infof(template, args...)
}

// Warn log a warn message.
func Warn(msg string) {
	log.Warn(msg)
}

// Warnw logs a message with some additional context,
// With key and values, example: log.Warnw("message", "url", url, "attempt", 3)
// Keys in key-value pairs should be strings.
func Warnw(msg string, keysAndValues ...interface{}) {
	log.Warnw(msg, keysAndValues...)
}

// Warnf uses fmt.Sprintf to log a templated message.
func Warnf(template string, args ...interface{}) {
	log.Warnf(template, args...)
}

// Error log a error message.
func Error(msg string) {
	log.Error(msg)
}

// Errorw logs a message with some additional context,
// With key and values, example: log.Errorw("message", "url", url, "attempt", 3)
// Keys in key-value pairs should be strings.
func Errorw(msg string, keysAndValues ...interface{}) {
	log.Errorw(msg, keysAndValues...)
}

// Errorf uses fmt.Sprintf to log a templated message.
func Errorf(template string, args ...interface{}) {
	log.Errorf(template, args...)
}

// Fatal log a fatal message.
func Fatal(msg string) {
	log.Fatal(msg)
}

// Fatalw logs a message with some additional context,
// With key and values, example: log.Fatalw("message", "url", url, "attempt", 3)
// Keys in key-value pairs should be strings.
func Fatalw(msg string, keysAndValues ...interface{}) {
	log.Fatalw(msg, keysAndValues...)
}

// Fatalf uses fmt.Sprintf to log a templated message.
func Fatalf(template string, args ...interface{}) {
	log.Fatalf(template, args...)
}

// Panic log a panic message.
func Panic(msg string) {
	log.Panic(msg)
}

// Panicw logs a message with some additional context,
// With key and values, example: log.Panicw("message", "url", url, "attempt", 3)
// Keys in key-value pairs should be strings.
func Panicw(msg string, keysAndValues ...interface{}) {
	log.Panicw(msg, keysAndValues...)
}

// Panicf uses fmt.Sprintf to log a templated message.
func Panicf(template string, args ...interface{}) {
	log.Panicf(template, args...)
}
