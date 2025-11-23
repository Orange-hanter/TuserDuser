package discovery

import "context"

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
