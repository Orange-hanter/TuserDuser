// Package database provides data access for telegram-service.
package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Common errors
var (
	ErrNotFound     = errors.New("record not found")
	ErrTokenExpired = errors.New("token expired or already used")
)

// BindingStatus represents the lifecycle state of a Telegram binding.
type BindingStatus string

const (
	BindingStatusPending BindingStatus = "pending"
	BindingStatusActive  BindingStatus = "active"
	BindingStatusBlocked BindingStatus = "blocked"
	BindingStatusRevoked BindingStatus = "revoked"
)

// Binding represents a user's Telegram chat binding.
type Binding struct {
	UserID        string
	ChatID        int64
	Status        BindingStatus
	Username      string
	FirstName     string
	LastName      string
	BlockedReason *string
	LastErrorCode *int
	LastErrorAt   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// WebhookEvent stores inbound webhook raw payloads for auditing.
type WebhookEvent struct {
	BotAlias   string
	UpdateID   int64
	Payload    json.RawMessage
	ReceivedAt time.Time
}

// Store provides data access operations.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore creates a new store instance.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// SaveBindingToken stores a binding token for later consumption.
func (s *Store) SaveBindingToken(ctx context.Context, nonceHash, userID string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO telegram_binding_tokens (nonce_hash, user_id, expires_at, created_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (nonce_hash) DO UPDATE
		 SET user_id = EXCLUDED.user_id, expires_at = EXCLUDED.expires_at, created_at = NOW()`,
		nonceHash, userID, expiresAt,
	)
	return err
}

// ConsumeBindingToken deletes the token and returns associated user if valid.
func (s *Store) ConsumeBindingToken(ctx context.Context, nonceHash string) (string, error) {
	row := s.pool.QueryRow(ctx,
		`DELETE FROM telegram_binding_tokens
		 WHERE nonce_hash = $1 AND expires_at > NOW()
		 RETURNING user_id`, nonceHash,
	)
	var userID string
	if err := row.Scan(&userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrTokenExpired
		}
		return "", err
	}
	return userID, nil
}

// UpsertBinding inserts or updates a telegram binding.
func (s *Store) UpsertBinding(ctx context.Context, binding Binding) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO telegram_bindings (user_id, chat_id, status, telegram_username, telegram_first_name, telegram_last_name, blocked_reason, last_error_code, last_error_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		 ON CONFLICT (user_id) DO UPDATE SET
			chat_id = EXCLUDED.chat_id,
			status = EXCLUDED.status,
			telegram_username = EXCLUDED.telegram_username,
			telegram_first_name = EXCLUDED.telegram_first_name,
			telegram_last_name = EXCLUDED.telegram_last_name,
			blocked_reason = EXCLUDED.blocked_reason,
			last_error_code = EXCLUDED.last_error_code,
			last_error_at = EXCLUDED.last_error_at,
			updated_at = NOW()`,
		binding.UserID,
		binding.ChatID,
		binding.Status,
		nullableString(binding.Username),
		nullableString(binding.FirstName),
		nullableString(binding.LastName),
		binding.BlockedReason,
		binding.LastErrorCode,
		binding.LastErrorAt,
	)
	return err
}

// GetBindingByUserID fetches a binding by user ID.
func (s *Store) GetBindingByUserID(ctx context.Context, userID string) (*Binding, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT user_id, chat_id, status, telegram_username, telegram_first_name, telegram_last_name,
		 blocked_reason, last_error_code, last_error_at, created_at, updated_at
		 FROM telegram_bindings WHERE user_id = $1`, userID)
	return scanBinding(row)
}

// GetBindingByChatID fetches a binding by Telegram chat ID.
func (s *Store) GetBindingByChatID(ctx context.Context, chatID int64) (*Binding, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT user_id, chat_id, status, telegram_username, telegram_first_name, telegram_last_name,
		 blocked_reason, last_error_code, last_error_at, created_at, updated_at
		 FROM telegram_bindings WHERE chat_id = $1`, chatID)
	return scanBinding(row)
}

// SetBindingStatus updates binding status and optional metadata.
func (s *Store) SetBindingStatus(ctx context.Context, userID string, status BindingStatus, reason *string, code *int) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE telegram_bindings
		 SET status = $2,
			blocked_reason = $3,
			last_error_code = $4,
			last_error_at = CASE WHEN $4 IS NULL THEN NULL ELSE NOW() END,
			updated_at = NOW()
		 WHERE user_id = $1`,
		userID, status, reason, code,
	)
	return err
}

// DeleteBinding removes a binding.
func (s *Store) DeleteBinding(ctx context.Context, userID string) error {
	result, err := s.pool.Exec(ctx, `DELETE FROM telegram_bindings WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// RecordWebhookEvent stores inbound webhook payload for auditing.
func (s *Store) RecordWebhookEvent(ctx context.Context, event WebhookEvent) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO telegram_webhook_events (bot_alias, update_id, payload, received_at)
		 VALUES ($1, $2, $3, $4)`,
		event.BotAlias, event.UpdateID, event.Payload, event.ReceivedAt,
	)
	return err
}

// CleanupExpiredTokens removes expired binding tokens.
func (s *Store) CleanupExpiredTokens(ctx context.Context) (int64, error) {
	result, err := s.pool.Exec(ctx,
		`DELETE FROM telegram_binding_tokens WHERE expires_at < NOW()`,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// CountActiveBindings returns the count of active bindings.
func (s *Store) CountActiveBindings(ctx context.Context) (int64, error) {
	var count int64
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM telegram_bindings WHERE status = 'active'`,
	).Scan(&count)
	return count, err
}

func scanBinding(row pgx.Row) (*Binding, error) {
	var b Binding
	var username, firstName, lastName sql.NullString
	err := row.Scan(
		&b.UserID,
		&b.ChatID,
		&b.Status,
		&username,
		&firstName,
		&lastName,
		&b.BlockedReason,
		&b.LastErrorCode,
		&b.LastErrorAt,
		&b.CreatedAt,
		&b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	b.Username = username.String
	b.FirstName = firstName.String
	b.LastName = lastName.String
	return &b, nil
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
