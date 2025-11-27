package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisHistoryRepository stores hot history data in Redis.
// Keeps recent user actions for fast access with automatic expiration.
type RedisHistoryRepository struct {
	client *redis.Client
	ttl    time.Duration
	limit  int // Maximum entries per user to keep in Redis
}

// NewRedisHistoryRepository creates a new Redis-backed history repository.
// ttl specifies how long to keep history entries (e.g., 7*24*time.Hour for 7 days).
// limit specifies maximum entries per user (older entries are trimmed).
func NewRedisHistoryRepository(client *redis.Client, ttl time.Duration, limit int) *RedisHistoryRepository {
	if limit <= 0 {
		limit = 100 // Default to 100 entries per user
	}
	return &RedisHistoryRepository{
		client: client,
		ttl:    ttl,
		limit:  limit,
	}
}

// Append adds a new history entry for a user.
func (r *RedisHistoryRepository) Append(ctx context.Context, entry HistoryEntry) error {
	historyKey := fmt.Sprintf("history:user:%s", entry.UserID)
	lastActionKey := fmt.Sprintf("last-action:user:%s:event:%s", entry.UserID, entry.EventID)

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal history entry: %w", err)
	}

	// Use Redis transaction to ensure consistency
	pipe := r.client.Pipeline()

	// Add to history list (LPUSH for newest first)
	pipe.LPush(ctx, historyKey, data)

	// Trim to keep only the latest entries
	pipe.LTrim(ctx, historyKey, 0, int64(r.limit-1))

	// Set expiration on history list
	pipe.Expire(ctx, historyKey, r.ttl)

	// Store last action for this event (overwrites previous)
	pipe.Set(ctx, lastActionKey, data, r.ttl)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("redis pipeline error: %w", err)
	}

	return nil
}

// List returns all history entries for a user in chronological order (newest first from Redis).
func (r *RedisHistoryRepository) List(ctx context.Context, userID string) ([]HistoryEntry, error) {
	key := fmt.Sprintf("history:user:%s", userID)

	// LRange gets entries from index 0 to limit (newest first)
	results, err := r.client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("redis lrange error: %w", err)
	}

	entries := make([]HistoryEntry, 0, len(results))
	for _, data := range results {
		var entry HistoryEntry
		if err := json.Unmarshal([]byte(data), &entry); err != nil {
			// Log but don't fail if single entry is corrupted
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// LastAction retrieves the most recent action for a specific event by a user.
func (r *RedisHistoryRepository) LastAction(ctx context.Context, userID, eventID string) (HistoryEntry, bool, error) {
	key := fmt.Sprintf("last-action:user:%s:event:%s", userID, eventID)

	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return HistoryEntry{}, false, nil
		}
		return HistoryEntry{}, false, fmt.Errorf("redis get error: %w", err)
	}

	var entry HistoryEntry
	if err := json.Unmarshal([]byte(data), &entry); err != nil {
		return HistoryEntry{}, false, fmt.Errorf("failed to unmarshal history entry: %w", err)
	}

	return entry, true, nil
}

// GetExcludedEventIDs returns a map of excluded event IDs for a user (for optimization).
func (r *RedisHistoryRepository) GetExcludedEventIDs(ctx context.Context, userID string) (map[string]bool, error) {
	entries, err := r.List(ctx, userID)
	if err != nil {
		return nil, err
	}

	excluded := make(map[string]bool)
	for _, entry := range entries {
		if entry.Action == "reject" || entry.Action == "book" {
			excluded[entry.EventID] = true
		}
	}

	return excluded, nil
}
