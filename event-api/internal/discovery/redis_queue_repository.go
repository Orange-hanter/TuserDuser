package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisQueueRepository stores per-user queue states in Redis.
// Provides persistent storage with automatic TTL expiration.
type RedisQueueRepository struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisQueueRepository creates a new Redis-backed queue repository.
// ttl specifies how long to keep queue states (e.g., 30*24*time.Hour for 30 days).
func NewRedisQueueRepository(client *redis.Client, ttl time.Duration) *RedisQueueRepository {
	return &RedisQueueRepository{
		client: client,
		ttl:    ttl,
	}
}

// Get retrieves the queue state for a user.
func (r *RedisQueueRepository) Get(ctx context.Context, userID string) (QueueState, error) {
	key := fmt.Sprintf("queue:user:%s", userID)

	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return QueueState{}, ErrQueueStateNotFound
		}
		return QueueState{}, fmt.Errorf("redis get error: %w", err)
	}

	var state QueueState
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		return QueueState{}, fmt.Errorf("failed to unmarshal queue state: %w", err)
	}

	return state, nil
}

// Save persists the queue state for a user with TTL expiration.
func (r *RedisQueueRepository) Save(ctx context.Context, userID string, state QueueState) error {
	key := fmt.Sprintf("queue:user:%s", userID)

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal queue state: %w", err)
	}

	if err := r.client.Set(ctx, key, data, r.ttl).Err(); err != nil {
		return fmt.Errorf("redis set error: %w", err)
	}

	return nil
}

// Clear removes the queue state for a user.
func (r *RedisQueueRepository) Clear(ctx context.Context, userID string) error {
	key := fmt.Sprintf("queue:user:%s", userID)

	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis del error: %w", err)
	}

	return nil
}

// ClearAll removes all queue states from Redis.
// WARNING: This uses SCAN to avoid blocking on large datasets, but still requires iteration.
func (r *RedisQueueRepository) ClearAll(ctx context.Context) error {
	pattern := "queue:user:*"

	var cursor uint64
	for {
		keys, cursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("redis scan error: %w", err)
		}

		if len(keys) > 0 {
			if err := r.client.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("redis del error: %w", err)
			}
		}

		if cursor == 0 {
			break
		}
	}

	return nil
}
