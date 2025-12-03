-- Migration: Initialize telegram-service database schema
-- This creates all tables needed for the standalone telegram-service

BEGIN;

-- Schema version tracking
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert initial version
INSERT INTO schema_migrations (version) VALUES (1) ON CONFLICT DO NOTHING;

-- Binding tokens table
-- Stores single-use, time-limited tokens for binding verification
CREATE TABLE IF NOT EXISTS telegram_binding_tokens (
    id SERIAL PRIMARY KEY,
    nonce_hash VARCHAR(64) NOT NULL UNIQUE,
    user_id VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_telegram_binding_tokens_expires 
    ON telegram_binding_tokens(expires_at);

CREATE INDEX IF NOT EXISTS idx_telegram_binding_tokens_user 
    ON telegram_binding_tokens(user_id);

-- Main bindings table
-- Maps core service user_id to Telegram chat_id
CREATE TABLE IF NOT EXISTS telegram_bindings (
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
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT valid_status CHECK (status IN ('pending', 'active', 'blocked', 'revoked'))
);

CREATE INDEX IF NOT EXISTS idx_telegram_bindings_chat_id 
    ON telegram_bindings(chat_id);

CREATE INDEX IF NOT EXISTS idx_telegram_bindings_status 
    ON telegram_bindings(status);

-- Webhook events audit log
-- Stores all incoming webhook payloads for debugging and auditing
CREATE TABLE IF NOT EXISTS telegram_webhook_events (
    id SERIAL PRIMARY KEY,
    bot_alias VARCHAR(64) NOT NULL,
    update_id BIGINT NOT NULL,
    payload JSONB NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_telegram_webhook_events_update_id 
    ON telegram_webhook_events(update_id);

CREATE INDEX IF NOT EXISTS idx_telegram_webhook_events_received 
    ON telegram_webhook_events(received_at);

-- Message delivery tracking
-- Tracks outbound message delivery attempts and status
CREATE TABLE IF NOT EXISTS telegram_delivery (
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
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT valid_delivery_status CHECK (
        status IN ('scheduled', 'sending', 'sent', 'failed', 'blocked', 'abandoned')
    )
);

CREATE INDEX IF NOT EXISTS idx_telegram_delivery_status 
    ON telegram_delivery(status, next_attempt_at);

CREATE INDEX IF NOT EXISTS idx_telegram_delivery_user 
    ON telegram_delivery(user_id);

CREATE INDEX IF NOT EXISTS idx_telegram_delivery_chat 
    ON telegram_delivery(chat_id);

-- Delivery audit log
-- Records every status transition for debugging
CREATE TABLE IF NOT EXISTS telegram_delivery_log (
    id SERIAL PRIMARY KEY,
    delivery_id VARCHAR(36) NOT NULL,
    status VARCHAR(20) NOT NULL,
    attempt INT NOT NULL,
    error_code INT,
    error_msg TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_telegram_delivery_log_delivery 
    ON telegram_delivery_log(delivery_id);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for updated_at
DROP TRIGGER IF EXISTS trigger_telegram_bindings_updated ON telegram_bindings;
CREATE TRIGGER trigger_telegram_bindings_updated
    BEFORE UPDATE ON telegram_bindings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS trigger_telegram_delivery_updated ON telegram_delivery;
CREATE TRIGGER trigger_telegram_delivery_updated
    BEFORE UPDATE ON telegram_delivery
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Cleanup job helper: delete expired tokens
CREATE OR REPLACE FUNCTION cleanup_expired_tokens()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM telegram_binding_tokens WHERE expires_at < NOW();
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

COMMIT;
