package models

import "time"

// UserProfile represents the full user profile returned to the client.
type UserProfile struct {
	ID                 string        `json:"id" example:"usr_abc"`
	Email              string        `json:"email" example:"user@example.com"`
	Name               string        `json:"name" example:"Иван Петров"`
	CreatedAt          time.Time     `json:"created_at" example:"2025-01-10T08:30:00Z"`
	TelegramRegistered bool          `json:"telegram_registered" example:"true"`
	TelegramInfo       *TelegramInfo `json:"telegram_info,omitempty"`
}

// TelegramInfo represents the user's Telegram details.
type TelegramInfo struct {
	Username string `json:"username" example:"ivan_tg"`
	ChatID   int64  `json:"chat_id" example:"123456789"`
	Status   string `json:"status" example:"active"`
}

// SubscriptionStatus represents the status of an event subscription.
type SubscriptionStatus string

const (
	// SubscriptionStatusConfirmed indicates the user is confirmed for the event.
	SubscriptionStatusConfirmed SubscriptionStatus = "confirmed"
	// SubscriptionStatusWaitlisted indicates the user is on the waitlist.
	SubscriptionStatusWaitlisted SubscriptionStatus = "waitlisted"
	// SubscriptionStatusCancelled indicates the subscription was cancelled.
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled"
)

// EventSubscription represents a user's subscription to an event.
type EventSubscription struct {
	EventID      string             `json:"event_id" example:"evt_xyz"`
	Status       SubscriptionStatus `json:"status" example:"confirmed"`
	SubscribedAt time.Time          `json:"subscribed_at" example:"2025-11-20T15:30:00Z"`
	ExpiresAt    *time.Time         `json:"expires_at" example:"2025-11-21T15:30:00Z"` // Nullable
}

// EventWithSubscription extends Event with subscription details.
type EventWithSubscription struct {
	Event
	SubscriptionStatus SubscriptionStatus `json:"subscription_status" example:"confirmed"`
	SubscribedAt       time.Time          `json:"subscribed_at" example:"2025-11-20T15:30:00Z"`
	AttendanceStatus   string             `json:"attendance_status,omitempty" example:"attended"` // "attended", "no_show", "cancelled"
}

// SubscribeRequest represents the request body for subscribing to an event.
type SubscribeRequest struct {
	Metadata map[string]interface{} `json:"metadata" swaggertype:"object,string" example:"dietary_preferences:vegan"`
}

// Participant представляет участника события.
type Participant struct {
	UserID     string  `json:"user_id" example:"941b955e-ea57-dee3-565f-5684f81c4f14"`
	PublicName string  `json:"public_name" example:"Иван Петров"`
	AvatarURL  *string `json:"avatar_url,omitempty" example:"https://example.com/avatar.jpg"`
	Status     string  `json:"status" example:"confirmed"` // "confirmed", "waitlisted", "cancelled"
}
