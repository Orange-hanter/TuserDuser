// Package logger предоставляет функциональность для логирования с использованием zap.Logger
package logger

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log is the global zap.Logger used by the application.
var Log *zap.Logger

// includeTraceFields controls whether trace_id/span_id are added to log entries.
var includeTraceFields bool

// currentLevel stores parsed log level string (e.g., "trace", "debug", "info").

// ANSI цвета для консоли.
const (
	ColorReset   = "\033[0m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorWhite   = "\033[37m"
	ColorGray    = "\033[90m"
)

// Границы для форматирования сообщений.
const topLine = "╔════════════════════════════════════════════════════════════╗"
const midLine = "╠════════════════════════════════════════════════════════════╣"
const bottomLine = "╚════════════════════════════════════════════════════════════╝"
const template = "%s%s%s%s\n"

// drowHeaderLine рисует верхнюю часть рамки с заголовком.
func drowHeaderLine(output *strings.Builder, title, color string) {
	fmt.Fprintf(output, template, color, color, topLine, ColorReset)
	output.WriteString(title)
	fmt.Fprintf(output, template, color, color, midLine, ColorReset)
}

// Init инициализирует логгер с красивым форматированием.
func Init() {
	var err error

	// Определяем окружение
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	// Читаем уровень логирования: trace | debug | info | warn | error
	lvlStr := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL")))
	if lvlStr == "" {
		// По умолчанию не светим трасс-идентификаторы и логируем на info в проде, debug в деве
		if env == "development" {
			lvlStr = "debug"
		} else {
			lvlStr = "info"
		}
	}
	level := parseLevel(lvlStr)
	includeTraceFields = (lvlStr == "trace")

	if env == "development" {
		Log, err = createDevelopmentLogger(level)
	} else {
		// production config with selected level
		cfg := zap.NewProductionConfig()
		cfg.Level = zap.NewAtomicLevelAt(level)
		Log, err = cfg.Build()
	}

	if err != nil {
		panic(err)
	}
}

// createDevelopmentLogger создает логгер для разработки с красивым выводом.
func createDevelopmentLogger(level zapcore.Level) (*zap.Logger, error) {
	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(level)
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.TimeKey = "time"
	config.EncoderConfig.NameKey = "logger"
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.FunctionKey = "func"
	config.EncoderConfig.StacktraceKey = "stacktrace"
	config.EncoderConfig.LineEnding = zapcore.DefaultLineEnding
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	logger, err := config.Build(zap.AddCallerSkip(1))
	return logger, err
}

// Sync синхронизирует логгер (вызывать в defer).
func Sync() error {
	if Log != nil {
		return Log.Sync()
	}
	return nil
}

// GetLogger возвращает экземпляр логгера.
func GetLogger() *zap.Logger {
	if Log == nil {
		Init()
	}
	return Log
}

// WithContext returns a logger with trace_id and span_id from the context.
// This enables log correlation with traces in Grafana.
func WithContext(ctx context.Context) *zap.Logger {
	if Log == nil {
		Init()
	}

	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return Log
	}

	if !includeTraceFields {
		// Трасс-идентификаторы публикуем только на уровне trace
		return Log
	}

	return Log.With(
		zap.String("trace_id", span.SpanContext().TraceID().String()),
		zap.String("span_id", span.SpanContext().SpanID().String()),
	)
}

// InfoCtx logs an info message with trace context.
func InfoCtx(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Info(msg, fields...)
}

// ErrorCtx logs an error message with trace context.
func ErrorCtx(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Error(msg, fields...)
}

// WarnCtx logs a warning message with trace context.
func WarnCtx(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Warn(msg, fields...)
}

// DebugCtx logs a debug message with trace context.
func DebugCtx(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Debug(msg, fields...)
}

// parseLevel maps string level to zapcore.Level. "trace" maps to Debug.
func parseLevel(lvl string) zapcore.Level {
	switch strings.ToLower(lvl) {
	case "trace":
		return zapcore.DebugLevel
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// FormatError форматирует ошибку с красивым выводом.
func FormatError(title string, err error, details ...string) string {
	var output strings.Builder

	drowHeaderLine(&output, fmt.Sprintf("%s%s║ ❌ %s%s\n", ColorRed, ColorRed, title, ColorReset), ColorRed)

	if err != nil {
		output.WriteString(fmt.Sprintf("%s%s║ Error: %v%s\n", ColorRed, ColorRed, err, ColorReset))
	}

	for _, detail := range details {
		output.WriteString(fmt.Sprintf("%s%s║ → %s%s\n", ColorYellow, ColorYellow, detail, ColorReset))
	}

	output.WriteString(fmt.Sprintf("%s%s%s%s\n", ColorRed, ColorRed, bottomLine, ColorReset))

	return output.String()
}

// FormatSuccess форматирует успешное сообщение.
func FormatSuccess(message string, details ...string) string {
	var output strings.Builder

	drowHeaderLine(&output, fmt.Sprintf("%s%s║ ✅ %s%s\n", ColorGreen, ColorGreen, message, ColorReset), ColorGreen)
	for _, detail := range details {
		output.WriteString(fmt.Sprintf("%s%s║ → %s%s\n", ColorCyan, ColorCyan, detail, ColorReset))
	}

	output.WriteString(fmt.Sprintf("%s%s%s%s\n", ColorGreen, ColorGreen, bottomLine, ColorReset))

	return output.String()
}

// FormatWarning форматирует предупреждение.
func FormatWarning(message string, details ...string) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("%s%s╔════════════════════════════════════════════════════════════╗%s\n", ColorYellow, ColorYellow, ColorReset))
	output.WriteString(fmt.Sprintf("%s%s║ ⚠️  %s%s\n", ColorYellow, ColorYellow, message, ColorReset))
	output.WriteString(fmt.Sprintf("%s%s╠════════════════════════════════════════════════════════════╣%s\n", ColorYellow, ColorYellow, ColorReset))

	for _, detail := range details {
		output.WriteString(fmt.Sprintf("%s%s║ → %s%s\n", ColorYellow, ColorYellow, detail, ColorReset))
	}

	output.WriteString(fmt.Sprintf("%s%s╚════════════════════════════════════════════════════════════╝%s\n", ColorYellow, ColorYellow, ColorReset))

	return output.String()
}

// FormatInfo форматирует информационное сообщение.
func FormatInfo(message string, details ...string) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("%s%s╔════════════════════════════════════════════════════════════╗%s\n", ColorBlue, ColorBlue, ColorReset))
	output.WriteString(fmt.Sprintf("%s%s║ ℹ️  %s%s\n", ColorBlue, ColorBlue, message, ColorReset))
	output.WriteString(fmt.Sprintf("%s%s╠════════════════════════════════════════════════════════════╣%s\n", ColorBlue, ColorBlue, ColorReset))

	for _, detail := range details {
		output.WriteString(fmt.Sprintf("%s%s║ → %s%s\n", ColorCyan, ColorCyan, detail, ColorReset))
	}

	output.WriteString(fmt.Sprintf("%s%s╚════════════════════════════════════════════════════════════╝%s\n", ColorBlue, ColorBlue, ColorReset))

	return output.String()
}
