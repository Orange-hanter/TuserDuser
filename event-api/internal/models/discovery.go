package models

// DiscoveryActionRequest describes a user reaction to the currently served event.
type DiscoveryActionRequest struct {
	// EventID is the identifier of the event returned by GET /next.
	EventID string `json:"eventId" example:"evt_123"`
	// Action accepts "like", "dislike" or "neutral" and must match the current event.
	Action string `json:"action" example:"like"`
}

// DiscoveryBookRequest requests a commitment to a specific event.
type DiscoveryBookRequest struct {
	// EventID points to the event that should be booked.
	EventID string `json:"eventId" example:"evt_123"`
}

// DiscoveryFilter allows users to filter events in discovery queue.
type DiscoveryFilter struct {
	// Types filters events by type (e.g., "Конференция", "Мастер-класс")
	Types []string `json:"types,omitempty" example:"Конференция,Мастер-класс"`
	// PriceTypes filters by price type: "free", "paid", "donation"
	PriceTypes []string `json:"priceTypes,omitempty" example:"free,paid"`
	// DateFrom filters events starting from this date (RFC3339)
	DateFrom string `json:"dateFrom,omitempty" example:"2025-01-01T00:00:00Z"`
	// DateTo filters events ending before this date (RFC3339)
	DateTo string `json:"dateTo,omitempty" example:"2025-12-31T23:59:59Z"`
	// Places filters by venue
	Places []string `json:"places,omitempty" example:"Коворкинг"`
}
