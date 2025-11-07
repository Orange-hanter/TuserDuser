// internal/config/config.go
package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	RedisPort          string
	DBSSLMode          string
	SMSFrom            string
	JWTSecret          string
	SMSAPIToken        string
	SMSAPIKey          string
	DBHost             string
	DBPort             string
	DBUser             string
	SMSProvider        string
	Env                string
	DBName             string
	DBPassword         string
	RedisPassword      string
	RedisHost          string
	Port               string
	CORSAllowedOrigins []string
	DBMinConn          int
	RedisDB            int
	DBMaxConn          int
	ShutdownTimeout    int
	JWTExpiration      int64
}

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
		JWTExpiration:      3600,                                // 1 час
		ShutdownTimeout:    getEnvAsInt("SHUTDOWN_TIMEOUT", 30), // 30 секунд

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
