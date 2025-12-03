// Package config handles configuration loading for telegram-service.
package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all configuration values for the service.
type Config struct {
	// Server settings
	GRPCPort string
	HTTPPort string
	Env      string

	// Database
	DatabaseURL string

	// Telegram Bot settings
	BotToken           string
	BotUsername        string
	WebhookSecret      string
	WebhookAlias       string
	TelegramAPIBaseURL string

	// Update mode: "webhook" or "polling"
	UpdateMode        string
	PollingTimeout    int // Long polling timeout in seconds (default: 30)
	PollingRetryDelay int // Retry delay in seconds on error (default: 3)

	// Token settings
	BindingSecret     string
	BindingTTLSeconds int

	// Rate limiting
	RateLimitPerSecond int
	MaxRetryAttempts   int
	RetryBaseSeconds   int
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	// Try to load .env file (ignore error if not exists)
	_ = godotenv.Load()

	cfg := &Config{
		GRPCPort:           getEnv("GRPC_PORT", "50051"),
		HTTPPort:           getEnv("HTTP_PORT", "8081"),
		Env:                getEnv("ENV", "development"),
		DatabaseURL:        getEnv("DATABASE_URL", ""),
		BotToken:           getEnv("TELEGRAM_BOT_TOKEN", ""),
		BotUsername:        getEnv("TELEGRAM_BOT_USERNAME", ""),
		WebhookSecret:      getEnv("TELEGRAM_WEBHOOK_SECRET", ""),
		WebhookAlias:       getEnv("TELEGRAM_WEBHOOK_ALIAS", "default"),
		TelegramAPIBaseURL: getEnv("TELEGRAM_API_BASE_URL", "https://api.telegram.org"),
		UpdateMode:         getEnv("TELEGRAM_UPDATE_MODE", "webhook"), // "webhook" or "polling"
		PollingTimeout:     getEnvInt("TELEGRAM_POLLING_TIMEOUT", 30),
		PollingRetryDelay:  getEnvInt("TELEGRAM_POLLING_RETRY_DELAY", 3),
		BindingSecret:      getEnv("TELEGRAM_BINDING_SECRET", ""),
		BindingTTLSeconds:  getEnvInt("TELEGRAM_BINDING_TTL", 3600), // 1 hour default
		RateLimitPerSecond: getEnvInt("TELEGRAM_RATE_LIMIT", 30),
		MaxRetryAttempts:   getEnvInt("TELEGRAM_MAX_RETRIES", 3),
		RetryBaseSeconds:   getEnvInt("TELEGRAM_RETRY_BASE_SECONDS", 5),
	}

	// Validate required fields
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.BotToken == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}
	if cfg.BotUsername == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_USERNAME is required")
	}
	if cfg.BindingSecret == "" {
		return nil, fmt.Errorf("TELEGRAM_BINDING_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
