package logger

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log is the global zap.Logger used by the application.
var Log *zap.Logger

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
const sideLine = "║"
const template = "%s%s%s%s\n"

// drowHeaderLine рисует верхнюю часть рамки с заголовком.
func drowHeaderLine(output *strings.Builder, title string, color string) {
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

	if env == "development" {
		Log, err = createDevelopmentLogger()
	} else {
		Log, err = zap.NewProduction()
	}

	if err != nil {
		panic(err)
	}
}

// createDevelopmentLogger создает логгер для разработки с красивым выводом.
func createDevelopmentLogger() (*zap.Logger, error) {
	config := zap.NewDevelopmentConfig()
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
