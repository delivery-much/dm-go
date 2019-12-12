package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// BaseFields represents the base fields for create the basic fields of logger.
type BaseFields struct {
	ServiceName string
	Env         string
	CodeVersion string
}

// NewLogger Create a logger with prod or dev config.
// isJSON for JSON output and production config and encoder
// serviceName for set the default field service name in logger
func NewLogger(isJSON bool, baseFields BaseFields) *zap.SugaredLogger {
	var config zap.Config

	if isJSON {
		encoderConfig := zap.NewProductionEncoderConfig()
		encoderConfig.MessageKey = "message"
		encoderConfig.TimeKey = "time"
		encoderConfig.StacktraceKey = "stack"
		encoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder

		config = zap.NewProductionConfig()
		config.InitialFields = getBaseFields(baseFields)
		config.EncoderConfig = encoderConfig
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig = zap.NewDevelopmentEncoderConfig()
	}

	logger, err := config.Build()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	return logger.Sugar()
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
