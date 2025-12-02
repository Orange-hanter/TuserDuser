// Package config реализует загрузку и хранение конфигурации приложения.
//
// internal/config/config.go
package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config структура для хранения конфигурации приложения.
type Config struct {
	// General config
	Port               string
	Env                string
	CORSAllowedOrigins []string
	ShutdownTimeout    int

	// JWT config
	JWTSecret     string
	JWTExpiration int64

	// Database config
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	DBMaxConn  int
	DBMinConn  int

	// Redis config
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int

	// SMS config
	SMSProvider string
	SMSAPIKey   string
	SMSAPIToken string
	SMSFrom     string

	// Email config
	EmailProvider string
	EmailAPIKey   string
	EmailFrom     string
	EmailFromName string
	SMTPHost      string
	SMTPPort      int
	SMTPUsername  string
	SMTPPassword  string
	SMTPUseSSL    bool

	// Telegram notification config
	TelegramEnabled          bool
	TelegramBotToken         string
	TelegramWebhookSecret    string
	TelegramBindingSecret    string
	TelegramBindingTTL       int
	TelegramRateLimitPerSec  int
	TelegramMaxAttempts      int
	TelegramRetryBaseSeconds int
	TelegramWebhookAlias     string
	TelegramBotUsername      string
	TelegramAPIBaseURL       string

	// Discovery streaming config
	DiscoveryUpdatesEnabled bool
	DiscoveryUpdatesChannel string

	// Discovery Redis config
	DiscoveryHistoryTTL int
	DiscoveryQueueTTL   int

	// OpenTelemetry config
	OTelEnabled     bool
	OTelEndpoint    string
	OTelServiceName string
}

// Load загружает конфигурацию из переменных окружения.
func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  .env не найден, используются системные переменные")
	}

	origins := strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}

	return &Config{
		Port:               getEnv("PORT", "8080"),
		Env:                getEnv("ENV", "development"),
		CORSAllowedOrigins: origins,
		JWTSecret:          getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		JWTExpiration:      getEnvAsDuration("JWT_EXPIRATION", 24*time.Hour), // по умолчанию 24 часа
		ShutdownTimeout:    getEnvAsInt("SHUTDOWN_TIMEOUT", 30),              // 30 секунд

		// Database config
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "event_api"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),
		DBMaxConn:  25,
		DBMinConn:  5,

		// Redis config
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 0),

		// SMS config
		SMSProvider: getEnv("SMS_PROVIDER", "mock"),
		SMSAPIKey:   getEnv("SMS_API_KEY", ""),
		SMSAPIToken: getEnv("SMS_API_TOKEN", ""),
		SMSFrom:     getEnv("SMS_FROM", ""),

		// Email config
		EmailProvider: getEnv("EMAIL_PROVIDER", "mock"),
		EmailAPIKey:   getEnv("EMAIL_API_KEY", ""),
		EmailFrom:     getEnv("EMAIL_FROM", "noreply@tuserduser.online"),
		EmailFromName: getEnv("EMAIL_FROM_NAME", "TuserDuser"),
		SMTPHost:      getEnv("SMTP_HOST", ""),
		SMTPPort:      getEnvAsInt("SMTP_PORT", 587),
		SMTPUsername:  getEnv("SMTP_USERNAME", ""),
		SMTPPassword:  getEnv("SMTP_PASSWORD", ""),
		SMTPUseSSL:    getEnvAsBool("SMTP_USE_SSL", false),

		TelegramEnabled:          getEnvAsBool("TELEGRAM_ENABLED", false),
		TelegramBotToken:         getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramWebhookSecret:    getEnv("TELEGRAM_WEBHOOK_SECRET", ""),
		TelegramBindingSecret:    getEnv("TELEGRAM_BINDING_SECRET", "change-me"),
		TelegramBindingTTL:       getEnvAsInt("TELEGRAM_BINDING_TTL_SECONDS", 600),
		TelegramRateLimitPerSec:  getEnvAsInt("TELEGRAM_RATE_LIMIT_PER_SEC", 30),
		TelegramMaxAttempts:      getEnvAsInt("TELEGRAM_MAX_ATTEMPTS", 5),
		TelegramRetryBaseSeconds: getEnvAsInt("TELEGRAM_RETRY_BASE_SECONDS", 5),
		TelegramWebhookAlias:     getEnv("TELEGRAM_WEBHOOK_ALIAS", "primary"),
		TelegramBotUsername:      getEnv("TELEGRAM_BOT_USERNAME", ""),
		TelegramAPIBaseURL:       getEnv("TELEGRAM_API_BASE_URL", "https://api.telegram.org"),

		DiscoveryUpdatesEnabled: getEnvAsBool("DISCOVERY_UPDATES_ENABLED", true),
		DiscoveryUpdatesChannel: getEnv("DISCOVERY_UPDATES_CHANNEL", "discovery:event_updates"),

		// Discovery Redis TTL config (in seconds)
		DiscoveryHistoryTTL: getEnvAsInt("DISCOVERY_HISTORY_TTL", 7*24*3600), // 7 days
		DiscoveryQueueTTL:   getEnvAsInt("DISCOVERY_QUEUE_TTL", 30*24*3600),  // 30 days

		// OpenTelemetry config
		OTelEnabled:     getEnvAsBool("OTEL_ENABLED", true),
		OTelEndpoint:    getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		OTelServiceName: getEnv("OTEL_SERVICE_NAME", "event-api"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}

func getEnvAsBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		switch strings.ToLower(value) {
		case "1", "true", "t", "yes", "y", "on":
			return true
		case "0", "false", "f", "no", "n", "off":
			return false
		}
	}
	return fallback
}

func getEnvAsDuration(key string, fallback time.Duration) int64 {
	if value := os.Getenv(key); value != "" {
		// Пробуем распарсить как duration (например, "24h", "720h")
		if duration, err := time.ParseDuration(value); err == nil {
			return int64(duration.Seconds())
		}
		// Если не получилось, пробуем как число секунд
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return int64(fallback.Seconds())
}
