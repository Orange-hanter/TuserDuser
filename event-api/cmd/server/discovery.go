package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"event-api/internal/config"
	"event-api/internal/discovery"
	"event-api/internal/logger"
	"event-api/internal/models"
	redisClient "event-api/internal/redis"
	"event-api/internal/service"

	"go.uber.org/zap"
)

var errRedisPubsubClosed = errors.New("redis pubsub channel closed")

// StartDiscoveryWorkers initializes and starts all discovery-related workers.
func (c *AppContainer) StartDiscoveryWorkers(ctx context.Context) {
	// Start periodic lock cleanup
	go c.startLockCleanupWorker(ctx)

	// Start discovery update worker (if enabled)
	startDiscoveryUpdateWorker(ctx, c.Config, c.Redis, c.EventService, c.DiscoveryService)
}

// startLockCleanupWorker periodically removes stale locks from discovery service.
func (c *AppContainer) startLockCleanupWorker(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			removed := c.DiscoveryService.CleanupStaleLocks(24 * time.Hour)
			if removed > 0 {
				logger.Log.Info("discovery locks cleaned up", zap.Int("removed", removed))
			}
		}
	}
}

// startDiscoveryUpdateWorker starts the Redis pub/sub worker for discovery updates.
func startDiscoveryUpdateWorker(
	ctx context.Context,
	cfg *config.Config,
	redis *redisClient.Client,
	eventService *service.EventService,
	discoveryService *discovery.Service,
) {
	if !cfg.DiscoveryUpdatesEnabled {
		logger.Log.Info("discovery updates disabled")
		return
	}

	if redis == nil {
		logger.Log.Warn("discovery updates enabled but Redis client is nil")
		return
	}

	channel := cfg.DiscoveryUpdatesChannel
	if channel == "" {
		logger.Log.Warn("discovery updates channel is empty")
		return
	}

	go func() {
		backoff := 5 * time.Second
		for {
			if err := runDiscoveryUpdateLoop(ctx, redis, channel, eventService, discoveryService); err != nil {
				logger.Log.Error("discovery update worker exited", zap.Error(err))
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
		}
	}()
}

// runDiscoveryUpdateLoop listens to discovery update messages and refreshes state.
func runDiscoveryUpdateLoop(
	ctx context.Context,
	redis *redisClient.Client,
	channel string,
	eventService *service.EventService,
	discoveryService *discovery.Service,
) error {
	sub := redis.GetClient().Subscribe(ctx, channel)
	defer func() {
		if err := sub.Close(); err != nil {
			logger.Log.Warn("failed to close redis subscription", zap.Error(err))
		}
	}()

	logger.Log.Info("discovery update worker subscribed", zap.String("channel", channel))
	ch := sub.Channel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("discovery update loop context done: %w", ctx.Err())
		case msg, ok := <-ch:
			if !ok {
				return errRedisPubsubClosed
			}

			update, err := discovery.DecodeUpdateMessage(msg.Payload)
			if err != nil {
				logger.Log.Warn("invalid discovery update message", zap.Error(err))
				continue
			}

			refreshCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			err = refreshDiscoveryState(refreshCtx, eventService, discoveryService)
			cancel()

			if err != nil {
				logger.Log.Error("failed to refresh discovery state", zap.Error(err))
				continue
			}

			logger.Log.Info("discovery state refreshed",
				zap.String("action", string(update.Action)),
				zap.String("event_id", update.EventID),
			)
		}
	}
}

// refreshDiscoveryState reloads approved events into discovery service.
func refreshDiscoveryState(
	ctx context.Context,
	eventService *service.EventService,
	discoveryService *discovery.Service,
) error {
	events, err := eventService.GetApprovedEvents(ctx)
	if err != nil {
		return fmt.Errorf("load approved events: %w", err)
	}

	converted := toDiscoveryEvents(time.Now(), events)

	// Use atomic RefreshCatalog to prevent race conditions
	if err := discoveryService.RefreshCatalog(ctx, converted); err != nil {
		return fmt.Errorf("refresh discovery catalog: %w", err)
	}

	return nil
}

// toDiscoveryEvents converts internal Event models to discovery Event models.
func toDiscoveryEvents(now time.Time, src []*models.Event) []discovery.Event {
	result := make([]discovery.Event, 0, len(src))

	for _, evt := range src {
		if evt == nil {
			continue
		}

		// Skip past events
		if evt.EndTime.Before(now) {
			continue
		}

		// Build metadata from event details
		metadata := map[string]interface{}{
			"type":             evt.Type,
			"place":            evt.Place,
			"priceType":        evt.PriceType,
			"needRegistration": evt.NeedRegistration,
		}
		for k, v := range evt.Details {
			metadata[k] = v
		}

		result = append(result, discovery.Event{
			ID:          evt.ID,
			Title:       evt.Type,
			Description: fmt.Sprintf("%s @ %s", evt.Type, evt.Place),
			Slot: discovery.TimeSlot{
				Start: evt.StartTime,
				End:   evt.EndTime,
			},
			Metadata: metadata,
		})
	}

	return result
}

// publishDiscoveryUpdate publishes an event update to Redis pub/sub channel.
func publishDiscoveryUpdate(
	ctx context.Context,
	redis *redisClient.Client,
	channel, eventID string,
) {
	if redis == nil || channel == "" || eventID == "" {
		return
	}

	msg := discovery.UpdateMessage{
		Action:      discovery.UpdateActionEventApproved,
		EventID:     eventID,
		TriggeredAt: time.Now().UTC(),
	}

	if err := discovery.PublishUpdate(ctx, redis.GetClient(), channel, msg); err != nil {
		logger.Log.Warn("failed to publish discovery update",
			zap.Error(err),
			zap.String("event_id", eventID),
		)
	}
}
