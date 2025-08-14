package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger

func InitLogger() error {
	config := zap.NewProductionConfig()

	// Configure log levels based on environment variable
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel != "" {
		level, err := zapcore.ParseLevel(logLevel)
		if err == nil {
			config.Level = zap.NewAtomicLevelAt(level)
		}
	}

	// Configure output
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	// Configure encoding
	config.Encoding = "json"
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.StacktraceKey = "stacktrace"

	var err error
	logger, err = config.Build()
	if err != nil {
		return err
	}

	// Replace zap global logger
	zap.ReplaceGlobals(logger)

	return nil
}

// GetLogger returns the configured logger
func GetLogger() *zap.Logger {
	if logger == nil {
		// Fallback to basic logger if not initialized
		logger, _ = zap.NewProduction()
	}
	return logger
}

// Sync synchronizes the logger (must be called on shutdown)
func Sync() error {
	if logger != nil {
		return logger.Sync()
	}
	return nil
}

// LogRequest logs information about an HTTP request
func LogRequest(method, path, traceID, spanID string, duration float64, statusCode int, errorMsg string) {
	log := GetLogger()

	fields := []zap.Field{
		zap.String("method", method),
		zap.String("path", path),
		zap.Float64("duration_seconds", duration),
		zap.Int("status_code", statusCode),
		zap.String("component", "http_server"),
	}

	if traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}

	if spanID != "" {
		fields = append(fields, zap.String("span_id", spanID))
	}

	if errorMsg != "" {
		fields = append(fields, zap.String("error", errorMsg))
		log.Error("HTTP request completed with error", fields...)
	} else {
		log.Info("HTTP request completed", fields...)
	}
}

// LogCEPMetrics logs CEP metrics
func LogCEPMetrics(cep, traceID, spanID string, isValid bool, format, errorType string) {
	log := GetLogger()

	fields := []zap.Field{
		zap.String("cep", cep),
		zap.Bool("is_valid", isValid),
		zap.String("format", format),
		zap.String("component", "cep_validation"),
	}

	if traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}

	if spanID != "" {
		fields = append(fields, zap.String("span_id", spanID))
	}

	if errorType != "" {
		fields = append(fields, zap.String("error_type", errorType))
		log.Warn("CEP validation failed", fields...)
	} else {
		log.Info("CEP validation successful", fields...)
	}
}

// LogServiceBCall logs Service B calls
func LogServiceBCall(cep, traceID, spanID string, duration float64, statusCode int, errorMsg string) {
	log := GetLogger()

	fields := []zap.Field{
		zap.String("cep", cep),
		zap.Float64("duration_seconds", duration),
		zap.Int("status_code", statusCode),
		zap.String("target_service", "service-b"),
		zap.String("component", "service_integration"),
	}

	if traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}

	if spanID != "" {
		fields = append(fields, zap.String("span_id", spanID))
	}

	if errorMsg != "" {
		fields = append(fields, zap.String("error", errorMsg))
		log.Error("Service B call failed", fields...)
	} else {
		log.Info("Service B call successful", fields...)
	}
}

// LogBusinessEvent logs business events
func LogBusinessEvent(eventType, cep, traceID, spanID string, additionalFields map[string]interface{}) {
	log := GetLogger()

	fields := []zap.Field{
		zap.String("event_type", eventType),
		zap.String("cep", cep),
		zap.String("component", "business_logic"),
	}

	if traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}

	if spanID != "" {
		fields = append(fields, zap.String("span_id", spanID))
	}

	for key, value := range additionalFields {
		switch v := value.(type) {
		case string:
			fields = append(fields, zap.String(key, v))
		case int:
			fields = append(fields, zap.Int(key, v))
		case float64:
			fields = append(fields, zap.Float64(key, v))
		case bool:
			fields = append(fields, zap.Bool(key, v))
		default:
			fields = append(fields, zap.Any(key, v))
		}
	}

	log.Info("Business event occurred", fields...)
}

// LogError logs errors with structured context
func LogError(message, traceID, spanID string, err error, additionalFields map[string]interface{}) {
	log := GetLogger()

	fields := []zap.Field{
		zap.Error(err),
	}

	if traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}

	if spanID != "" {
		fields = append(fields, zap.String("span_id", spanID))
	}

	for key, value := range additionalFields {
		switch v := value.(type) {
		case string:
			fields = append(fields, zap.String(key, v))
		case int:
			fields = append(fields, zap.Int(key, v))
		case float64:
			fields = append(fields, zap.Float64(key, v))
		case bool:
			fields = append(fields, zap.Bool(key, v))
		default:
			fields = append(fields, zap.Any(key, v))
		}
	}

	log.Error(message, fields...)
}

// LogInfo logs information with structured context
func LogInfo(message, traceID, spanID string, additionalFields map[string]interface{}) {
	log := GetLogger()

	fields := []zap.Field{}

	if traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}

	if spanID != "" {
		fields = append(fields, zap.String("span_id", spanID))
	}

	for key, value := range additionalFields {
		switch v := value.(type) {
		case string:
			fields = append(fields, zap.String(key, v))
		case int:
			fields = append(fields, zap.Int(key, v))
		case float64:
			fields = append(fields, zap.Float64(key, v))
		case bool:
			fields = append(fields, zap.Bool(key, v))
		default:
			fields = append(fields, zap.Any(key, v))
		}
	}

	log.Info(message, fields...)
}
