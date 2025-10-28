package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	Env                string
	CORSAllowedOrigins []string
	JWTSecret          string
	JWTExpiration      int64 // в секундах
	ShutdownTimeout    int   // в секундах

	// Database config
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	DBMaxConn  int
	DBMinConn  int
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
