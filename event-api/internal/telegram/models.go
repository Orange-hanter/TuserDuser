package telegram

import (
	"encoding/json"
	"time"
)

// BindingStatus represents the lifecycle state of a Telegram chat binding.
type BindingStatus string

const (
	BindingStatusPending BindingStatus = "pending"
	BindingStatusActive  BindingStatus = "active"
	BindingStatusBlocked BindingStatus = "blocked"
	BindingStatusRevoked BindingStatus = "revoked"
)

// DeliveryStatus describes the send attempt lifecycle.
type DeliveryStatus string

const (
	DeliveryStatusScheduled DeliveryStatus = "scheduled"
	DeliveryStatusSending   DeliveryStatus = "sending"
	DeliveryStatusSent      DeliveryStatus = "sent"
	DeliveryStatusFailed    DeliveryStatus = "failed"
	DeliveryStatusBlocked   DeliveryStatus = "blocked"
	DeliveryStatusAbandoned DeliveryStatus = "abandoned"
)

// Binding aggregates binding metadata persisted in telegram_bindings.
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

// BindingLink represents a minted deep link token for the client app.
type BindingLink struct {
	Token     string    `json:"token"`
	DeepLink  string    `json:"deeplink"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ReminderJob models an inbound reminder payload from the ReminderWorker.
type ReminderJob struct {
	ReminderID  string          `json:"reminder_id"`
	UserID      string          `json:"user_id"`
	Message     string          `json:"message"`
	ParseMode   string          `json:"parse_mode"`
	Silent      bool            `json:"silent"`
	Metadata    map[string]any  `json:"metadata,omitempty"`
	ReplyMarkup json.RawMessage `json:"reply_markup,omitempty"`
}

// DeliveryAttempt persists a pending delivery attempt row.
type DeliveryAttempt struct {
	ID            string
	UserID        string
	ChatID        int64
	ReminderID    string
	Payload       json.RawMessage
	Status        DeliveryStatus
	AttemptCount  int
	LastErrorCode *int
	LastErrorMsg  *string
	NextAttemptAt *time.Time
	MessageID     *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// DeliveryLogEntry is appended for every transition.
type DeliveryLogEntry struct {
	DeliveryID string
	Status     DeliveryStatus
	Attempt    int
	ErrorCode  *int
	ErrorMsg   *string
	CreatedAt  time.Time
}

// OutboundMessage encapsulates a message sent to Telegram API.
type OutboundMessage struct {
	ChatID      int64
	Text        string
	ParseMode   string
	Silent      bool
	ReplyMarkup json.RawMessage
}

// SendResult captures Telegram Bot API response metadata.
type SendResult struct {
	MessageID string
}

// WebhookEvent stores inbound webhook raw payloads.
type WebhookEvent struct {
	BotAlias   string
	UpdateID   int64
	Payload    json.RawMessage
	ReceivedAt time.Time
}
