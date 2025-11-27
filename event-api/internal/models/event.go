package models

import "time"

// Event представляет событие в системе.
type Event struct {
	StartTime        time.Time              `json:"start" db:"start_time"`
	EndTime          time.Time              `json:"end" db:"end_time"`
	CreatedAt        time.Time              `json:"time" db:"created_at"`
	UpdatedAt        time.Time              `json:"-" db:"updated_at"`
	Details          map[string]interface{} `json:"details" db:"details"`
	ID               string                 `json:"id" db:"id"`
	Type             string                 `json:"type" db:"type"`
	Place            string                 `json:"place" db:"place"`
	PriceType        string                 `json:"priceType" db:"price_type"`
	Duration         int                    `json:"duration" db:"duration"`
	NeedRegistration bool                   `json:"needReg" db:"need_registration"`
}

// PendingEvent представляет событие в очереди модерации.
type PendingEvent struct {
	Event
	Status        string     `json:"status" db:"status"`
	ReviewComment string     `json:"reviewComment,omitempty" db:"review_comment"`
	ReviewedAt    *time.Time `json:"reviewedAt,omitempty" db:"reviewed_at"`
}

const (
	EventStatusPending  = "pending"
	EventStatusApproved = "approved"
	EventStatusRejected = "rejected"
)

// CreateEventRequest - запрос для создания события.
type CreateEventRequest struct {
	StartTime        time.Time              `json:"start" binding:"required"`
	EndTime          time.Time              `json:"end" binding:"required"`
	Details          map[string]interface{} `json:"details"`
	Type             string                 `json:"type" binding:"required"`
	Place            string                 `json:"place"`
	PriceType        string                 `json:"priceType" binding:"required"`
	Duration         int                    `json:"duration" binding:"required"`
	NeedRegistration bool                   `json:"needReg"`
	CreatorID        string                 `json:"-"` // Заполняется из авторизации
}

// ReviewEventRequest описывает действие по одобрению/отклонению события.
type ReviewEventRequest struct {
	Action  string `json:"action" binding:"required"`
	Comment string `json:"comment"`
}

// UpdateEventRequest - запрос для обновления события.
type UpdateEventRequest struct {
	Type             *string                 `json:"type"`
	StartTime        *time.Time              `json:"start"`
	EndTime          *time.Time              `json:"end"`
	Duration         *int                    `json:"duration"`
	Place            *string                 `json:"place"`
	PriceType        *string                 `json:"priceType"`
	NeedRegistration *bool                   `json:"needReg"`
	Details          *map[string]interface{} `json:"details"`
}
