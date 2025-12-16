package discovery

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// HistoryStreamConsumer consumes Redis Stream events and persists them to Postgres.
// Delivery semantics:
// - at-least-once from Redis Streams
// - idempotent writes to Postgres via op_id unique constraint
//
// Stream message fields:
// - kind: "append" | "remove_booking"
// - op_id: unique idempotency key
// - user_id, event_id
// - action (for append)
// - ts_unix_nano (optional)
// - context_json (optional, for append)
//
// Note: this consumer is intentionally small and synchronous; scale by running multiple consumers
// in the same consumer group.

type HistoryStreamConsumer struct {
	rdb       *redis.Client
	db        *sql.DB
	logger    *zap.Logger
	stream    string
	group     string
	consumer  string
	blockTime time.Duration
	count     int64
}

func NewHistoryStreamConsumer(rdb *redis.Client, db *sql.DB, logger *zap.Logger, stream, group, consumer string) *HistoryStreamConsumer {
	if consumer == "" {
		consumer = "consumer-1"
	}
	return &HistoryStreamConsumer{
		rdb:       rdb,
		db:        db,
		logger:    logger,
		stream:    stream,
		group:     group,
		consumer:  consumer,
		blockTime: 2 * time.Second,
		count:     64,
	}
}

func (c *HistoryStreamConsumer) ensureGroup(ctx context.Context) error {
	if c.stream == "" || c.group == "" {
		return fmt.Errorf("stream and group must be set")
	}
	// Create group if missing. Start from 0 to allow processing pending after restarts.
	err := c.rdb.XGroupCreateMkStream(ctx, c.stream, c.group, "0").Err()
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "BUSYGROUP") {
		return nil
	}
	return err
}

func (c *HistoryStreamConsumer) Run(ctx context.Context) error {
	if c.rdb == nil || c.db == nil {
		return fmt.Errorf("redis and db must be set")
	}
	if c.logger == nil {
		c.logger = zap.NewNop()
	}

	if err := c.ensureGroup(ctx); err != nil {
		return fmt.Errorf("ensure stream group: %w", err)
	}

	repo := NewPostgresHistoryRepository(c.db)

	c.logger.Info("✅ discovery history stream consumer started",
		zap.String("stream", c.stream),
		zap.String("group", c.group),
		zap.String("consumer", c.consumer),
	)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("discovery history stream consumer stopping", zap.Error(ctx.Err()))
			return nil
		default:
		}

		res, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.group,
			Consumer: c.consumer,
			Streams:  []string{c.stream, ">"},
			Count:    c.count,
			Block:    c.blockTime,
		}).Result()
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				continue
			}
			if errors.Is(err, redis.Nil) {
				continue
			}
			c.logger.Warn("history consumer: XREADGROUP failed", zap.Error(err))
			time.Sleep(500 * time.Millisecond)
			continue
		}

		for _, stream := range res {
			for _, msg := range stream.Messages {
				if err := c.handleMessage(ctx, repo, msg); err != nil {
					c.logger.Warn("history consumer: failed to handle message (will retry)",
						zap.String("id", msg.ID),
						zap.Error(err),
					)
					// Don't ACK on error -> stays pending and can be retried.
					continue
				}

				if err := c.rdb.XAck(ctx, c.stream, c.group, msg.ID).Err(); err != nil {
					c.logger.Warn("history consumer: XACK failed", zap.String("id", msg.ID), zap.Error(err))
				}
			}
		}
	}
}

func (c *HistoryStreamConsumer) handleMessage(ctx context.Context, repo *PostgresHistoryRepository, msg redis.XMessage) error {
	kind, _ := asString(msg.Values["kind"])
	opID, _ := asString(msg.Values["op_id"])
	userID, _ := asString(msg.Values["user_id"])
	eventID, _ := asString(msg.Values["event_id"])

	if kind == "" || userID == "" || eventID == "" {
		return fmt.Errorf("invalid message: missing kind/user_id/event_id")
	}

	switch kind {
	case "append":
		action, _ := asString(msg.Values["action"])
		if action == "" {
			return fmt.Errorf("append missing action")
		}

		var ts time.Time
		if raw, ok := msg.Values["ts_unix_nano"]; ok {
			if n, ok := asInt64(raw); ok && n > 0 {
				ts = time.Unix(0, n).UTC()
			}
		}
		if ts.IsZero() {
			ts = time.Now().UTC()
		}

		contextJSON, _ := asString(msg.Values["context_json"])
		var contextMap map[string]any
		if contextJSON != "" {
			_ = json.Unmarshal([]byte(contextJSON), &contextMap)
		}
		entry := HistoryEntry{
			UserID:    userID,
			EventID:   eventID,
			Action:    UserAction(action),
			Timestamp: ts,
			Context:   contextMap,
		}

		if opID != "" {
			return repo.AppendWithOpID(ctx, opID, entry)
		}
		return repo.Append(ctx, entry)

	case "remove_booking":
		// Delete is idempotent.
		return repo.RemoveBooking(ctx, userID, eventID)

	default:
		return fmt.Errorf("unknown kind: %s", kind)
	}
}

func asString(v any) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case []byte:
		return string(t), true
	default:
		return fmt.Sprintf("%v", v), v != nil
	}
}

func asInt64(v any) (int64, bool) {
	switch t := v.(type) {
	case int64:
		return t, true
	case int:
		return int64(t), true
	case string:
		n, err := strconv.ParseInt(t, 10, 64)
		return n, err == nil
	case []byte:
		n, err := strconv.ParseInt(string(t), 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}
