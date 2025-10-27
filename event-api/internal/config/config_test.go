package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Save original environment
	originalEnv := map[string]string{
		"PORT":                  os.Getenv("PORT"),
		"ENV":                   os.Getenv("ENV"),
		"JWT_SECRET":            os.Getenv("JWT_SECRET"),
		"CORS_ALLOWED_ORIGINS":  os.Getenv("CORS_ALLOWED_ORIGINS"),
		"DB_HOST":               os.Getenv("DB_HOST"),
		"DB_PORT":               os.Getenv("DB_PORT"),
		"DB_USER":               os.Getenv("DB_USER"),
		"DB_PASSWORD":           os.Getenv("DB_PASSWORD"),
		"DB_NAME":               os.Getenv("DB_NAME"),
		"DB_SSLMODE":            os.Getenv("DB_SSLMODE"),
	}

	defer func() {
		// Restore original environment
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	tests := []struct {
		name      string
		setupEnv  map[string]string
		checkFunc func(t *testing.T, cfg *Config)
	}{
		{
			name: "Load with all environment variables set",
			setupEnv: map[string]string{
				"PORT":                  "9000",
				"ENV":                   "production",
				"JWT_SECRET":            "test-secret-key",
				"CORS_ALLOWED_ORIGINS":  "http://localhost:3000, http://localhost:3001",
				"DB_HOST":               "db.example.com",
				"DB_PORT":               "5433",
				"DB_USER":               "testuser",
				"DB_PASSWORD":           "testpass",
				"DB_NAME":               "testdb",
				"DB_SSLMODE":            "require",
			},
			checkFunc: func(t *testing.T, cfg *Config) {
				if cfg.Port != "9000" {
					t.Errorf("expected Port=9000, got %s", cfg.Port)
				}
				if cfg.Env != "production" {
					t.Errorf("expected Env=production, got %s", cfg.Env)
				}
				if cfg.JWTSecret != "test-secret-key" {
					t.Errorf("expected JWTSecret=test-secret-key, got %s", cfg.JWTSecret)
				}
				if len(cfg.CORSAllowedOrigins) != 2 {
					t.Errorf("expected 2 CORS origins, got %d", len(cfg.CORSAllowedOrigins))
				}
				if cfg.CORSAllowedOrigins[0] != "http://localhost:3000" {
					t.Errorf("expected first origin=http://localhost:3000, got %s", cfg.CORSAllowedOrigins[0])
				}
				if cfg.DBHost != "db.example.com" {
					t.Errorf("expected DBHost=db.example.com, got %s", cfg.DBHost)
				}
				if cfg.DBPort != "5433" {
					t.Errorf("expected DBPort=5433, got %s", cfg.DBPort)
				}
				if cfg.DBSSLMode != "require" {
					t.Errorf("expected DBSSLMode=require, got %s", cfg.DBSSLMode)
				}
			},
		},
		{
			name:     "Load with default values when env vars not set",
			setupEnv: map[string]string{},
			checkFunc: func(t *testing.T, cfg *Config) {
				if cfg.Port != "8080" {
					t.Errorf("expected default Port=8080, got %s", cfg.Port)
				}
				if cfg.Env != "development" {
					t.Errorf("expected default Env=development, got %s", cfg.Env)
				}
				if cfg.JWTSecret != "your-secret-key-change-in-production" {
					t.Errorf("expected default JWTSecret, got %s", cfg.JWTSecret)
				}
				if cfg.DBHost != "localhost" {
					t.Errorf("expected default DBHost=localhost, got %s", cfg.DBHost)
				}
				if cfg.DBPort != "5432" {
					t.Errorf("expected default DBPort=5432, got %s", cfg.DBPort)
				}
				if cfg.DBUser != "postgres" {
					t.Errorf("expected default DBUser=postgres, got %s", cfg.DBUser)
				}
				if cfg.DBSSLMode != "disable" {
					t.Errorf("expected default DBSSLMode=disable, got %s", cfg.DBSSLMode)
				}
			},
		},
		{
			name: "Load with empty CORS origins",
			setupEnv: map[string]string{
				"CORS_ALLOWED_ORIGINS": "",
			},
			checkFunc: func(t *testing.T, cfg *Config) {
				if len(cfg.CORSAllowedOrigins) != 1 {
					t.Errorf("expected 1 CORS origin (empty string), got %d", len(cfg.CORSAllowedOrigins))
				}
				if cfg.CORSAllowedOrigins[0] != "" {
					t.Errorf("expected empty CORS origin, got %s", cfg.CORSAllowedOrigins[0])
				}
			},
		},
		{
			name: "Load with single CORS origin",
			setupEnv: map[string]string{
				"CORS_ALLOWED_ORIGINS": "http://example.com",
			},
			checkFunc: func(t *testing.T, cfg *Config) {
				if len(cfg.CORSAllowedOrigins) != 1 {
					t.Errorf("expected 1 CORS origin, got %d", len(cfg.CORSAllowedOrigins))
				}
				if cfg.CORSAllowedOrigins[0] != "http://example.com" {
					t.Errorf("expected CORS origin=http://example.com, got %s", cfg.CORSAllowedOrigins[0])
				}
			},
		},
		{
			name: "Load with multiple CORS origins with spaces",
			setupEnv: map[string]string{
				"CORS_ALLOWED_ORIGINS": "  http://localhost:3000  ,  http://example.com  , https://app.example.com  ",
			},
			checkFunc: func(t *testing.T, cfg *Config) {
				if len(cfg.CORSAllowedOrigins) != 3 {
					t.Errorf("expected 3 CORS origins, got %d", len(cfg.CORSAllowedOrigins))
				}
				expectedOrigins := []string{"http://localhost:3000", "http://example.com", "https://app.example.com"}
				for i, expected := range expectedOrigins {
					if cfg.CORSAllowedOrigins[i] != expected {
						t.Errorf("origin %d: expected %s, got %s", i, expected, cfg.CORSAllowedOrigins[i])
					}
				}
			},
		},
		{
			name: "Load database configuration",
			setupEnv: map[string]string{
				"DB_HOST":      "prod-db.example.com",
				"DB_PORT":      "5433",
				"DB_USER":      "appuser",
				"DB_PASSWORD":  "securepass123",
				"DB_NAME":      "production_db",
				"DB_SSLMODE":   "require",
			},
			checkFunc: func(t *testing.T, cfg *Config) {
				if cfg.DBHost != "prod-db.example.com" {
					t.Errorf("expected DBHost=prod-db.example.com, got %s", cfg.DBHost)
				}
				if cfg.DBPort != "5433" {
					t.Errorf("expected DBPort=5433, got %s", cfg.DBPort)
				}
				if cfg.DBUser != "appuser" {
					t.Errorf("expected DBUser=appuser, got %s", cfg.DBUser)
				}
				if cfg.DBPassword != "securepass123" {
					t.Errorf("expected DBPassword=securepass123, got %s", cfg.DBPassword)
				}
				if cfg.DBName != "production_db" {
					t.Errorf("expected DBName=production_db, got %s", cfg.DBName)
				}
				if cfg.DBSSLMode != "require" {
					t.Errorf("expected DBSSLMode=require, got %s", cfg.DBSSLMode)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all env vars
			clearEnv := []string{
				"PORT", "ENV", "JWT_SECRET", "CORS_ALLOWED_ORIGINS",
				"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE",
			}
			for _, key := range clearEnv {
				os.Unsetenv(key)
			}

			// Set test environment variables
			for key, value := range tt.setupEnv {
				os.Setenv(key, value)
			}

			cfg := Load()
			tt.checkFunc(t, cfg)
		})
	}
}

func TestGetEnv(t *testing.T) {
	// Save original environment
	originalValue := os.Getenv("TEST_ENV_VAR")
	defer func() {
		if originalValue == "" {
			os.Unsetenv("TEST_ENV_VAR")
		} else {
			os.Setenv("TEST_ENV_VAR", originalValue)
		}
	}()

	tests := []struct {
		name           string
		key            string
		fallback       string
		setValue       string
		shouldSet      bool
		expectedResult string
	}{
		{
			name:           "Return environment variable when set",
			key:            "TEST_ENV_VAR",
			fallback:       "default-value",
			setValue:       "custom-value",
			shouldSet:      true,
			expectedResult: "custom-value",
		},
		{
			name:           "Return fallback when environment variable not set",
			key:            "TEST_ENV_VAR_NOT_SET",
			fallback:       "fallback-value",
			shouldSet:      false,
			expectedResult: "fallback-value",
		},
		{
			name:           "Return empty string as environment variable if set",
			key:            "TEST_ENV_VAR",
			fallback:       "fallback-value",
			setValue:       "",
			shouldSet:      true,
			expectedResult: "fallback-value", // getEnv treats empty string as not set
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(tt.key)
			if tt.shouldSet {
				os.Setenv(tt.key, tt.setValue)
			}

			result := getEnv(tt.key, tt.fallback)
			if result != tt.expectedResult {
				t.Errorf("expected %s, got %s", tt.expectedResult, result)
			}
		})
	}
}

func TestConfigConstants(t *testing.T) {
	cfg := Load()

	// Test fixed values
	if cfg.JWTExpiration != 3600 {
		t.Errorf("expected JWTExpiration=3600, got %d", cfg.JWTExpiration)
	}
	if cfg.DBMaxConn != 25 {
		t.Errorf("expected DBMaxConn=25, got %d", cfg.DBMaxConn)
	}
	if cfg.DBMinConn != 5 {
		t.Errorf("expected DBMinConn=5, got %d", cfg.DBMinConn)
	}
}
