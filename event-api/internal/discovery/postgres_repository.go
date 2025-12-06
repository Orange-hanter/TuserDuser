package discovery

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// PostgresHistoryRepository persists discovery actions to PostgreSQL.
// Implements HistoryRepository interface for production use.
type PostgresHistoryRepository struct {
	db *sql.DB
}

// NewPostgresHistoryRepository creates a PostgreSQL-backed history repository.
func NewPostgresHistoryRepository(db *sql.DB) *PostgresHistoryRepository {
	return &PostgresHistoryRepository{db: db}
}

// Append stores a history entry in the database.
func (r *PostgresHistoryRepository) Append(ctx context.Context, entry HistoryEntry) error {
	contextJSON, err := json.Marshal(entry.Context)
	if err != nil {
		contextJSON = []byte("{}")
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO discovery_actions (user_id, event_id, action, context, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, entry.UserID, entry.EventID, string(entry.Action), contextJSON, entry.Timestamp)

	if err != nil {
		return fmt.Errorf("insert discovery action: %w", err)
	}
	return nil
}

// List returns user history in chronological order.
func (r *PostgresHistoryRepository) List(ctx context.Context, userID string) ([]HistoryEntry, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT event_id, action, context, created_at
		FROM discovery_actions
		WHERE user_id = $1
		ORDER BY created_at ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query discovery actions: %w", err)
	}
	defer rows.Close()

	var entries []HistoryEntry
	for rows.Next() {
		var (
			eventID     string
			action      string
			contextJSON []byte
			createdAt   time.Time
		)
		if err := rows.Scan(&eventID, &action, &contextJSON, &createdAt); err != nil {
			return nil, fmt.Errorf("scan discovery action: %w", err)
		}

		entry := HistoryEntry{
			UserID:    userID,
			EventID:   eventID,
			Action:    UserAction(action),
			Timestamp: createdAt,
		}

		if len(contextJSON) > 0 {
			_ = json.Unmarshal(contextJSON, &entry.Context)
		}

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// LastAction returns last recorded action for a specific event.
func (r *PostgresHistoryRepository) LastAction(ctx context.Context, userID, eventID string) (HistoryEntry, bool, error) {
	var (
		action      string
		contextJSON []byte
		createdAt   time.Time
	)

	err := r.db.QueryRowContext(ctx, `
		SELECT action, context, created_at
		FROM discovery_actions
		WHERE user_id = $1 AND event_id = $2
		ORDER BY created_at DESC
		LIMIT 1
	`, userID, eventID).Scan(&action, &contextJSON, &createdAt)

	if err == sql.ErrNoRows {
		return HistoryEntry{}, false, nil
	}
	if err != nil {
		return HistoryEntry{}, false, fmt.Errorf("query last action: %w", err)
	}

	entry := HistoryEntry{
		UserID:    userID,
		EventID:   eventID,
		Action:    UserAction(action),
		Timestamp: createdAt,
	}

	if len(contextJSON) > 0 {
		_ = json.Unmarshal(contextJSON, &entry.Context)
	}

	return entry, true, nil
}

// GetExcludedEventIDs returns event IDs that user has already actioned (like/dislike/book).
// Optimized query using partial index.
func (r *PostgresHistoryRepository) GetExcludedEventIDs(ctx context.Context, userID string) (map[string]bool, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT event_id
		FROM discovery_actions
		WHERE user_id = $1 AND action IN ('like', 'dislike', 'book')
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query excluded events: %w", err)
	}
	defer rows.Close()

	excluded := make(map[string]bool)
	for rows.Next() {
		var eventID string
		if err := rows.Scan(&eventID); err != nil {
			return nil, fmt.Errorf("scan excluded event: %w", err)
		}
		excluded[eventID] = true
	}

	return excluded, rows.Err()
}

// RemoveBooking removes the booking action for a user/event pair from history.
// This allows the event to reappear in discovery after unsubscribe.
func (r *PostgresHistoryRepository) RemoveBooking(ctx context.Context, userID, eventID string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM discovery_actions
		WHERE user_id = $1 AND event_id = $2 AND action = 'book'
	`, userID, eventID)

	if err != nil {
		return fmt.Errorf("delete booking action: %w", err)
	}
	return nil
}
