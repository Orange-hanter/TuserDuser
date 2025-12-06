package discovery

import (
	"context"
	"time"
)

// Service exposes application-level operations for the discovery engine.
type Service struct {
	engine *Engine
}

// NewService wires an engine behind a transport-friendly façade.
func NewService(engine *Engine) *Service {
	return &Service{engine: engine}
}

// NextEvent proxies to the underlying engine.
func (s *Service) NextEvent(ctx context.Context, userID string) (NextEvent, error) {
	return s.engine.NextEvent(ctx, userID)
}

// NextEventFiltered returns next event matching filter criteria.
func (s *Service) NextEventFiltered(ctx context.Context, userID string, filter Filter) (NextEvent, error) {
	return s.engine.NextEventFiltered(ctx, userID, filter)
}

// ApplyAction proxies action handling.
func (s *Service) ApplyAction(ctx context.Context, userID, eventID string, action UserAction) (HistoryEntry, error) {
	return s.engine.ApplyAction(ctx, userID, eventID, action)
}

// BookEvent commits an event.
func (s *Service) BookEvent(ctx context.Context, userID, eventID string) (BookingResult, error) {
	return s.engine.BookEvent(ctx, userID, eventID)
}

// RegisterBooking proxies external booking handling.
func (s *Service) RegisterBooking(ctx context.Context, userID, eventID string) (BookingResult, error) {
	return s.engine.RegisterBooking(ctx, userID, eventID)
}

// CancelBooking removes a booking record, allowing the event to reappear in discovery.
func (s *Service) CancelBooking(ctx context.Context, userID, eventID string) error {
	return s.engine.CancelBooking(ctx, userID, eventID)
}

// History returns chronological user actions.
func (s *Service) History(ctx context.Context, userID string) ([]HistoryEntry, error) {
	return s.engine.History(ctx, userID)
}

// ReplaceEvents refreshes runtime event catalog.
func (s *Service) ReplaceEvents(ctx context.Context, events []Event) error {
	return s.engine.ReplaceEvents(ctx, events)
}

// ResetQueue forces a queue rebuild on next request.
func (s *Service) ResetQueue(ctx context.Context, userID string) error {
	return s.engine.ResetUserQueue(ctx, userID)
}

// ResetAllQueues очищает очереди всех пользователей.
func (s *Service) ResetAllQueues(ctx context.Context) error {
	return s.engine.ResetAllQueues(ctx)
}

// RefreshCatalog atomically replaces events and resets all queues.
func (s *Service) RefreshCatalog(ctx context.Context, events []Event) error {
	return s.engine.RefreshCatalog(ctx, events)
}

// CleanupStaleLocks removes old lock entries to prevent memory leaks.
func (s *Service) CleanupStaleLocks(maxAge time.Duration) int {
	return s.engine.CleanupStaleLocks(maxAge)
}
