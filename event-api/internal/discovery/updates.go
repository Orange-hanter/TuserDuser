package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// UpdateAction описывает тип события, которое передается по шине discovery обновлений.
type UpdateAction string

const (
	// UpdateActionEventApproved сигнализирует, что событие переведено в статус approved
	// и discovery движок должен обновить каталог событий.
	UpdateActionEventApproved UpdateAction = "event_approved"
)

// UpdateMessage описывает полезную нагрузку pub/sub уведомления о discovery обновлении.
type UpdateMessage struct {
	Action      UpdateAction `json:"action"`
	EventID     string       `json:"eventId,omitempty"`
	TriggeredAt time.Time    `json:"triggeredAt"`
}

// PublishUpdate сериализует и публикует сообщение в Redis канал.
func PublishUpdate(ctx context.Context, client *redis.Client, channel string, msg UpdateMessage) error {
	if client == nil {
		return fmt.Errorf("redis client is nil")
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal discovery update: %w", err)
	}
	if err := client.Publish(ctx, channel, payload).Err(); err != nil {
		return fmt.Errorf("publish discovery update: %w", err)
	}
	return nil
}

// DecodeUpdateMessage превращает строковое сообщение в UpdateMessage.
func DecodeUpdateMessage(raw string) (UpdateMessage, error) {
	var msg UpdateMessage
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		return msg, fmt.Errorf("decode discovery update: %w", err)
	}
	if msg.Action == "" {
		msg.Action = UpdateActionEventApproved
	}
	return msg, nil
}
