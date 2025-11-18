package telegram

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// Store persists telegram binding and delivery data in SQL database.
type Store struct {
	db *sql.DB
}

// NewStore creates a new telegram store.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// SaveBindingToken stores hashed nonce metadata for later consumption.
func (s *Store) SaveBindingToken(ctx context.Context, nonceHash, userID string, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO telegram_binding_tokens (nonce_hash, user_id, expires_at, created_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (nonce_hash) DO UPDATE
		 SET user_id = EXCLUDED.user_id, expires_at = EXCLUDED.expires_at, created_at = NOW()`,
		nonceHash, userID, expiresAt,
	)
	return err
}

// ConsumeBindingToken deletes the nonce and returns associated user if valid.
func (s *Store) ConsumeBindingToken(ctx context.Context, nonceHash string) (string, error) {
	row := s.db.QueryRowContext(ctx,
		`DELETE FROM telegram_binding_tokens
		 WHERE nonce_hash = $1 AND expires_at > NOW()
		 RETURNING user_id`, nonceHash,
	)
	var userID string
	if err := row.Scan(&userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrInvalidToken
		}
		return "", err
	}
	return userID, nil
}

// UpsertBinding inserts or updates a telegram binding.
func (s *Store) UpsertBinding(ctx context.Context, binding Binding) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO telegram_bindings (user_id, chat_id, status, blocked_reason, last_error_code, last_error_at, telegram_username, telegram_first_name, telegram_last_name, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW(),NOW())
		 ON CONFLICT (user_id) DO UPDATE SET
			chat_id = EXCLUDED.chat_id,
			status = EXCLUDED.status,
			blocked_reason = EXCLUDED.blocked_reason,
			last_error_code = EXCLUDED.last_error_code,
			last_error_at = EXCLUDED.last_error_at,
			telegram_username = EXCLUDED.telegram_username,
			telegram_first_name = EXCLUDED.telegram_first_name,
			telegram_last_name = EXCLUDED.telegram_last_name,
			updated_at = NOW()`,
		binding.UserID,
		binding.ChatID,
		binding.Status,
		binding.BlockedReason,
		binding.LastErrorCode,
		binding.LastErrorAt,
		binding.Username,
		binding.FirstName,
		binding.LastName,
	)
	return err
}

// GetBindingByUserID fetches an active binding.
func (s *Store) GetBindingByUserID(ctx context.Context, userID string) (*Binding, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT user_id, chat_id, status, telegram_username, telegram_first_name, telegram_last_name,
		 blocked_reason, last_error_code, last_error_at, created_at, updated_at
		 FROM telegram_bindings WHERE user_id = $1`, userID)
	return scanBinding(row)
}

// GetBindingByChatID fetches binding by chat id.
func (s *Store) GetBindingByChatID(ctx context.Context, chatID int64) (*Binding, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT user_id, chat_id, status, telegram_username, telegram_first_name, telegram_last_name,
		 blocked_reason, last_error_code, last_error_at, created_at, updated_at
		 FROM telegram_bindings WHERE chat_id = $1`, chatID)
	return scanBinding(row)
}

// SetBindingStatus updates binding status/blocked reason.
func (s *Store) SetBindingStatus(ctx context.Context, userID string, status BindingStatus, reason *string, code *int) error {
	_, err := s.db.ExecContext(ctx,
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

// RecordWebhookEvent stores inbound webhook payload for auditing.
func (s *Store) RecordWebhookEvent(ctx context.Context, event WebhookEvent) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO telegram_webhook_events (bot_alias, update_id, payload, received_at)
		 VALUES ($1,$2,$3,$4)`,
		event.BotAlias,
		event.UpdateID,
		event.Payload,
		event.ReceivedAt,
	)
	return err
}

// EnqueueDelivery inserts a delivery attempt.
func (s *Store) EnqueueDelivery(ctx context.Context, attempt DeliveryAttempt) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO telegram_delivery (id, user_id, chat_id, reminder_id, payload, status, attempt_count, next_attempt_at, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW(),NOW())`,
		attempt.ID,
		attempt.UserID,
		attempt.ChatID,
		attempt.ReminderID,
		attempt.Payload,
		attempt.Status,
		attempt.AttemptCount,
		attempt.NextAttemptAt,
	)
	return err
}

// FetchDueDeliveries returns due rows ready for processing.
func (s *Store) FetchDueDeliveries(ctx context.Context, limit int) ([]DeliveryAttempt, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, chat_id, reminder_id, payload, status, attempt_count, last_error_code, last_error_msg, next_attempt_at, message_id, created_at, updated_at
		 FROM telegram_delivery
		 WHERE status IN ('scheduled','failed')
		   AND (next_attempt_at IS NULL OR next_attempt_at <= NOW())
		 ORDER BY COALESCE(next_attempt_at, created_at)
		 LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DeliveryAttempt
	for rows.Next() {
		var attempt DeliveryAttempt
		var payload []byte
		var nextAttempt sql.NullTime
		var lastErrMsg sql.NullString
		var lastErrCode sql.NullInt64
		var messageID sql.NullString
		if err := rows.Scan(
			&attempt.ID,
			&attempt.UserID,
			&attempt.ChatID,
			&attempt.ReminderID,
			&payload,
			&attempt.Status,
			&attempt.AttemptCount,
			&lastErrCode,
			&lastErrMsg,
			&nextAttempt,
			&messageID,
			&attempt.CreatedAt,
			&attempt.UpdatedAt,
		); err != nil {
			return nil, err
		}
		attempt.Payload = json.RawMessage(payload)
		if nextAttempt.Valid {
			attempt.NextAttemptAt = &nextAttempt.Time
		}
		if lastErrMsg.Valid {
			msg := lastErrMsg.String
			attempt.LastErrorMsg = &msg
		}
		if lastErrCode.Valid {
			code := int(lastErrCode.Int64)
			attempt.LastErrorCode = &code
		}
		if messageID.Valid {
			mid := messageID.String
			attempt.MessageID = &mid
		}
		results = append(results, attempt)
	}
	return results, rows.Err()
}

// MarkDeliverySending transitions attempt to sending status.
func (s *Store) MarkDeliverySending(ctx context.Context, id string) (*DeliveryAttempt, error) {
	row := s.db.QueryRowContext(ctx,
		`UPDATE telegram_delivery
		 SET status = 'sending', attempt_count = attempt_count + 1, updated_at = NOW()
		 WHERE id = $1 AND status IN ('scheduled','failed')
		 RETURNING id, user_id, chat_id, reminder_id, payload, status, attempt_count, created_at, updated_at`, id)
	var attempt DeliveryAttempt
	var payload []byte
	if err := row.Scan(
		&attempt.ID,
		&attempt.UserID,
		&attempt.ChatID,
		&attempt.ReminderID,
		&payload,
		&attempt.Status,
		&attempt.AttemptCount,
		&attempt.CreatedAt,
		&attempt.UpdatedAt,
	); err != nil {
		return nil, err
	}
	attempt.Payload = json.RawMessage(payload)
	return &attempt, nil
}

// MarkDeliverySent finalizes successful attempts.
func (s *Store) MarkDeliverySent(ctx context.Context, id string, messageID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE telegram_delivery
		 SET status = 'sent', message_id = $2, updated_at = NOW()
		 WHERE id = $1`,
		id, messageID,
	)
	return err
}

// MarkDeliveryFailed updates failure/blocked status.
func (s *Store) MarkDeliveryFailed(ctx context.Context, id string, status DeliveryStatus, code *int, msg *string, nextAttempt *time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE telegram_delivery
		 SET status = $2,
			last_error_code = $3,
			last_error_msg = $4,
			next_attempt_at = $5,
			updated_at = NOW()
		 WHERE id = $1`,
		id, status, code, msg, nextAttempt,
	)
	return err
}

// AppendDeliveryLog appends immutable audit rows.
func (s *Store) AppendDeliveryLog(ctx context.Context, log DeliveryLogEntry) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO telegram_delivery_log (delivery_id, status, attempt, error_code, error_msg, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		log.DeliveryID,
		log.Status,
		log.Attempt,
		log.ErrorCode,
		log.ErrorMsg,
		log.CreatedAt,
	)
	return err
}

func scanBinding(row *sql.Row) (*Binding, error) {
	var binding Binding
	var blockedReason sql.NullString
	var username sql.NullString
	var firstName sql.NullString
	var lastName sql.NullString
	var lastErrorCode sql.NullInt64
	var lastErrorAt sql.NullTime

	if err := row.Scan(
		&binding.UserID,
		&binding.ChatID,
		&binding.Status,
		&username,
		&firstName,
		&lastName,
		&blockedReason,
		&lastErrorCode,
		&lastErrorAt,
		&binding.CreatedAt,
		&binding.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBindingNotFound
		}
		return nil, err
	}

	if username.Valid {
		binding.Username = username.String
	}
	if firstName.Valid {
		binding.FirstName = firstName.String
	}
	if lastName.Valid {
		binding.LastName = lastName.String
	}
	if blockedReason.Valid {
		reason := blockedReason.String
		binding.BlockedReason = &reason
	}
	if lastErrorCode.Valid {
		code := int(lastErrorCode.Int64)
		binding.LastErrorCode = &code
	}
	if lastErrorAt.Valid {
		t := lastErrorAt.Time
		binding.LastErrorAt = &t
	}
	return &binding, nil
}
