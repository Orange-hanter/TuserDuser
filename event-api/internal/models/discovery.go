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
