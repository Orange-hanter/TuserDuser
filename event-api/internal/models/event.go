package models

import "time"

// Event представляет событие в системе
type Event struct {
	ID               string                 `json:"id" db:"id"`
	Type             string                 `json:"type" db:"type"`
	StartTime        time.Time              `json:"start" db:"start_time"`
	EndTime          time.Time              `json:"end" db:"end_time"`
	Duration         int                    `json:"duration" db:"duration"` // в минутах
	Place            string                 `json:"place" db:"place"`
	PriceType        string                 `json:"priceType" db:"price_type"`
	NeedRegistration bool                   `json:"needReg" db:"need_registration"`
	Details          map[string]interface{} `json:"details" db:"details"`
	CreatedAt        time.Time              `json:"time" db:"created_at"`
	UpdatedAt        time.Time              `json:"-" db:"updated_at"`
}

// CreateEventRequest - запрос для создания события
type CreateEventRequest struct {
	Type             string                 `json:"type" binding:"required"`
	StartTime        time.Time              `json:"start" binding:"required"`
	EndTime          time.Time              `json:"end" binding:"required"`
	Duration         int                    `json:"duration" binding:"required"`
	Place            string                 `json:"place"`
	PriceType        string                 `json:"priceType" binding:"required"`
	NeedRegistration bool                   `json:"needReg"`
	Details          map[string]interface{} `json:"details"`
}

// UpdateEventRequest - запрос для обновления события
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
