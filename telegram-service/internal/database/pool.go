// Package database handles PostgreSQL connections and queries for telegram-service.
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// NewPool creates a new PostgreSQL connection pool.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	config.MaxConns = 10
	config.MinConns = 2

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}

// Migrate runs database migrations.
func Migrate(ctx context.Context, pool *pgxpool.Pool, logger *zap.Logger) error {
	migrations := []string{
		// Schema version tracking
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			version INT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,

		// Binding tokens (single-use, time-limited)
		`CREATE TABLE IF NOT EXISTS telegram_binding_tokens (
			id SERIAL PRIMARY KEY,
			nonce_hash VARCHAR(64) NOT NULL UNIQUE,
			user_id VARCHAR(255) NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,

		// Main bindings table
		`CREATE TABLE IF NOT EXISTS telegram_bindings (
			user_id VARCHAR(255) PRIMARY KEY,
			chat_id BIGINT NOT NULL UNIQUE,
			status VARCHAR(20) NOT NULL DEFAULT 'active',
			telegram_username VARCHAR(255),
			telegram_first_name VARCHAR(255),
			telegram_last_name VARCHAR(255),
			blocked_reason TEXT,
			last_error_code INT,
			last_error_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,

		// Webhook events for auditing
		`CREATE TABLE IF NOT EXISTS telegram_webhook_events (
			id SERIAL PRIMARY KEY,
			bot_alias VARCHAR(64) NOT NULL,
			update_id BIGINT NOT NULL,
			payload JSONB NOT NULL,
			received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,

		// Message delivery attempts
		`CREATE TABLE IF NOT EXISTS telegram_delivery (
			id VARCHAR(36) PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL,
			chat_id BIGINT NOT NULL,
			message_type VARCHAR(50) NOT NULL,
			payload JSONB NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'scheduled',
			attempt_count INT NOT NULL DEFAULT 0,
			last_error_code INT,
			last_error_msg TEXT,
			next_attempt_at TIMESTAMPTZ,
			message_id VARCHAR(64),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,

		// Delivery audit log
		`CREATE TABLE IF NOT EXISTS telegram_delivery_log (
			id SERIAL PRIMARY KEY,
			delivery_id VARCHAR(36) NOT NULL,
			status VARCHAR(20) NOT NULL,
			attempt INT NOT NULL,
			error_code INT,
			error_msg TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,

		// Short binding codes (user-friendly 6-char codes)
		`CREATE TABLE IF NOT EXISTS telegram_binding_codes (
			id SERIAL PRIMARY KEY,
			code VARCHAR(6) NOT NULL UNIQUE,
			user_id VARCHAR(255) NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,

		// Indexes
		`CREATE INDEX IF NOT EXISTS idx_telegram_binding_codes_expires ON telegram_binding_codes(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_telegram_binding_tokens_expires ON telegram_binding_tokens(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_telegram_bindings_chat_id ON telegram_bindings(chat_id)`,
		`CREATE INDEX IF NOT EXISTS idx_telegram_bindings_status ON telegram_bindings(status)`,
		`CREATE INDEX IF NOT EXISTS idx_telegram_delivery_status ON telegram_delivery(status, next_attempt_at)`,
		`CREATE INDEX IF NOT EXISTS idx_telegram_delivery_user ON telegram_delivery(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_telegram_delivery_log_delivery ON telegram_delivery_log(delivery_id)`,
	}

	for i, migration := range migrations {
		_, err := pool.Exec(ctx, migration)
		if err != nil {
			return fmt.Errorf("migration %d failed: %w", i, err)
		}
	}

	logger.Info("database migrations completed")
	return nil
}
