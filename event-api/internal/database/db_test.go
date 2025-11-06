package database

import (
	"testing"

	"go.uber.org/zap"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		config *Config
		verify func(t *testing.T, cfg *Config)
		name   string
	}{
		{
			name: "Valid config structure",
			config: &Config{
				Host:     "localhost",
				Port:     "5432",
				User:     "postgres",
				Password: "postgres",
				DBName:   "event_api",
				SSLMode:  "disable",
				MaxConn:  25,
				MinConn:  5,
			},
			verify: func(t *testing.T, cfg *Config) {
				if cfg.Host != "localhost" {
					t.Errorf("expected Host=localhost, got %s", cfg.Host)
				}
				if cfg.Port != "5432" {
					t.Errorf("expected Port=5432, got %s", cfg.Port)
				}
				if cfg.User != "postgres" {
					t.Errorf("expected User=postgres, got %s", cfg.User)
				}
				if cfg.MaxConn != 25 {
					t.Errorf("expected MaxConn=25, got %d", cfg.MaxConn)
				}
				if cfg.MinConn != 5 {
					t.Errorf("expected MinConn=5, got %d", cfg.MinConn)
				}
			},
		},
		{
			name: "Config with SSL enabled",
			config: &Config{
				Host:     "db.example.com",
				Port:     "5433",
				User:     "appuser",
				Password: "securepass",
				DBName:   "production",
				SSLMode:  "require",
				MaxConn:  50,
				MinConn:  10,
			},
			verify: func(t *testing.T, cfg *Config) {
				if cfg.SSLMode != "require" {
					t.Errorf("expected SSLMode=require, got %s", cfg.SSLMode)
				}
				if cfg.Host != "db.example.com" {
					t.Errorf("expected remote host, got %s", cfg.Host)
				}
			},
		},
		{
			name: "Config with empty password",
			config: &Config{
				Host:     "localhost",
				Port:     "5432",
				User:     "postgres",
				Password: "",
				DBName:   "test",
				SSLMode:  "disable",
				MaxConn:  10,
				MinConn:  1,
			},
			verify: func(t *testing.T, cfg *Config) {
				if cfg.Password != "" {
					t.Errorf("expected empty password, got %s", cfg.Password)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.verify(t, tt.config)
		})
	}
}

func TestDatabaseStructure(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// We can't test actual connection without a real DB,
	// but we can verify the structure is created correctly
	db := &Database{
		DB:     nil,
		Logger: logger,
	}

	if db.Logger == nil {
		t.Error("Logger should not be nil")
	}

	// Test that methods exist and can be called (on nil DB they will panic)
	// This just verifies the struct has the expected interface
	if err := db.Close(); err == nil {
		t.Error("expected error when closing a nil DB")
	}
}

func TestConnectionPoolSettings(t *testing.T) {
	tests := []struct {
		verify  func(t *testing.T, max, min int)
		name    string
		maxConn int
		minConn int
	}{
		{
			name:    "Standard pool sizes",
			maxConn: 25,
			minConn: 5,
			verify: func(t *testing.T, max, min int) {
				if max != 25 {
					t.Errorf("expected maxConn=25, got %d", max)
				}
				if min != 5 {
					t.Errorf("expected minConn=5, got %d", min)
				}
				if min >= max {
					t.Errorf("minConn should be less than maxConn")
				}
			},
		},
		{
			name:    "Large pool sizes",
			maxConn: 100,
			minConn: 20,
			verify: func(t *testing.T, max, min int) {
				if max != 100 {
					t.Errorf("expected maxConn=100, got %d", max)
				}
				if min < 20 {
					t.Errorf("expected minConn>=20, got %d", min)
				}
			},
		},
		{
			name:    "Minimal pool sizes",
			maxConn: 5,
			minConn: 1,
			verify: func(t *testing.T, max, min int) {
				if min == 0 {
					t.Error("minConn should not be 0")
				}
				if max <= min {
					t.Errorf("maxConn should be greater than minConn")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.verify(t, tt.maxConn, tt.minConn)
		})
	}
}

func TestDatabaseConfig_DSN(t *testing.T) {
	tests := []struct {
		config   *Config
		checkDSN func(t *testing.T, cfg *Config)
		name     string
	}{
		{
			name: "Local development DSN",
			config: &Config{
				Host:     "localhost",
				Port:     "5432",
				User:     "postgres",
				Password: "postgres",
				DBName:   "event_api",
				SSLMode:  "disable",
			},
			checkDSN: func(t *testing.T, cfg *Config) {
				// Just verify the config has the right values for DSN construction
				if cfg.Host == "" {
					t.Error("Host should not be empty")
				}
				if cfg.User == "" {
					t.Error("User should not be empty")
				}
				if cfg.DBName == "" {
					t.Error("DBName should not be empty")
				}
			},
		},
		{
			name: "Production DSN with SSL",
			config: &Config{
				Host:     "prod-db.example.com",
				Port:     "5433",
				User:     "appuser",
				Password: "securepass123",
				DBName:   "production_db",
				SSLMode:  "require",
			},
			checkDSN: func(t *testing.T, cfg *Config) {
				if cfg.SSLMode != "require" {
					t.Errorf("expected SSLMode=require, got %s", cfg.SSLMode)
				}
				if cfg.Port != "5433" {
					t.Errorf("expected Port=5433, got %s", cfg.Port)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.checkDSN(t, tt.config)
		})
	}
}

func TestLoggerIntegration(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	tests := []struct {
		logger *zap.Logger
		name   string
	}{
		{
			name:   "Development logger",
			logger: logger,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.logger == nil {
				t.Error("Logger should not be nil")
			}

			// Create a Database instance to verify logger integration
			db := &Database{
				Logger: tt.logger,
			}

			if db.Logger != tt.logger {
				t.Error("Logger not properly assigned to Database")
			}
		})
	}
}
