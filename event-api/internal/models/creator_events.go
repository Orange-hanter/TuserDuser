package models

import "time"

// Дополнительные статусы событий для автора (основные в event.go)
const (
	EventStatusNeedsRevision = "needs_revision" // Требует доработки
	EventStatusBlocked       = "blocked"        // Заблокировано
)

// CreatorEvent представляет событие с точки зрения автора.
type CreatorEvent struct {
	ID               string         `json:"id"`
	Type             string         `json:"type"`
	StartTime        time.Time      `json:"startTime"`
	EndTime          time.Time      `json:"endTime"`
	Duration         int            `json:"duration"`
	Place            string         `json:"place,omitempty"`
	PriceType        string         `json:"priceType"`
	NeedRegistration bool           `json:"needRegistration"`
	Details          map[string]any `json:"details,omitempty"`
	Status           string         `json:"status"`
	ReviewComment    string         `json:"reviewComment,omitempty"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

// BlockedEvent представляет заблокированное событие.
type BlockedEvent struct {
	CreatorEvent
	BlockReason string    `json:"blockReason,omitempty"`
	BlockedAt   time.Time `json:"blockedAt"`
}

// ReviewComment представляет комментарий модерации.
type ReviewComment struct {
	ID         int       `json:"id"`
	EventID    string    `json:"eventId"`
	AuthorID   string    `json:"authorId"`
	AuthorRole string    `json:"authorRole"`
	Comment    string    `json:"comment"`
	CreatedAt  time.Time `json:"createdAt"`
}

// CreatorEventsResponse ответ со списком событий автора.
type CreatorEventsResponse struct {
	Pending  []CreatorEvent `json:"pending"`  // Ожидающие проверки
	Active   []CreatorEvent `json:"active"`   // Одобренные и еще не прошедшие
	Rejected []CreatorEvent `json:"rejected"` // Отклонённые
}

// AddReviewCommentRequest запрос на добавление комментария.
type AddReviewCommentRequest struct {
	EventID string `json:"eventId" validate:"required"`
	Comment string `json:"comment" validate:"required"`
}

// BlockEventRequest запрос на блокировку события.
type BlockEventRequest struct {
	EventID string `json:"eventId" validate:"required"`
	Reason  string `json:"reason" validate:"required"`
}
