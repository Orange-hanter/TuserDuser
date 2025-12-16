package discovery

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EngineConfig configures runtime behavior.
type EngineConfig struct {
	MaxQueueLength int
	Now            func() time.Time
}

// Engine orchestrates deterministic queue execution.
type Engine struct {
	events  EventRepository
	queues  QueueRepository
	history HistoryRepository
	cfg     EngineConfig
	locks   sync.Map // map[userID]*lockEntry

	// mu protects atomic operations like ReplaceEvents + ResetAllQueues
	mu sync.RWMutex
}

// lockEntry wraps a mutex with last-access tracking for cleanup.
type lockEntry struct {
	mu         sync.Mutex
	lastAccess int64 // unix nano
}

// NewEngine builds a ready-to-use deterministic engine.
func NewEngine(events EventRepository, queues QueueRepository, history HistoryRepository, cfg EngineConfig) *Engine {
	if cfg.MaxQueueLength <= 0 {
		cfg.MaxQueueLength = 512
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Engine{
		events:  events,
		queues:  queues,
		history: history,
		cfg:     cfg,
	}
}

// NewEngineWithRedis builds engine with optional Redis repositories for queues and history.
// Falls back to in-memory repositories if Redis clients are nil.
func NewEngineWithRedis(
	events EventRepository,
	queueRepo QueueRepository,
	historyRepo HistoryRepository,
	cfg EngineConfig,
) *Engine {
	return NewEngine(events, queueRepo, historyRepo, cfg)
}

// NextEvent returns the next item for a user, lazily initializing their queue.
// Deprecated:
// Use NextEventFiltered with empty filter instead.
func (e *Engine) NextEvent(ctx context.Context, userID string) (NextEvent, error) {
	return e.NextEventFiltered(ctx, userID, Filter{})
}

// NextEventFiltered returns the next event matching the filter criteria.
// If the current queue doesn't match the filter, a new filtered queue is built.
func (e *Engine) NextEventFiltered(ctx context.Context, userID string, filter Filter) (NextEvent, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	unlock := e.lock(userID)
	defer unlock()

	state, err := e.ensureStateFiltered(ctx, userID, filter)
	if err != nil {
		return NextEvent{}, err
	}
	eventID, conflict, err := state.NextCandidate()
	if err != nil {
		return NextEvent{}, err
	}
	event, err := e.events.Get(ctx, eventID)
	if err != nil {
		return NextEvent{}, err
	}
	var flag *ConflictFlag
	if stored, ok := state.ConflictFlagFor(eventID); ok {
		copyFlag := stored
		flag = &copyFlag
	}
	primaryRemaining := len(state.Primary)
	conflictRemaining := len(state.Conflicts)
	if !conflict && primaryRemaining > 0 {
		primaryRemaining--
	}
	if conflict && conflictRemaining > 0 {
		conflictRemaining--
	}
	if err := e.queues.Save(ctx, userID, state); err != nil {
		return NextEvent{}, err
	}
	return NextEvent{
		Event:              event,
		Conflict:           conflict,
		ConflictFlag:       flag,
		RemainingPrimary:   primaryRemaining,
		RemainingConflicts: conflictRemaining,
	}, nil
}

// ApplyAction applies like/dislike/neutral behavior to the current event.
func (e *Engine) ApplyAction(ctx context.Context, userID, eventID string, action UserAction) (HistoryEntry, error) {
	if action == ActionBook {
		return HistoryEntry{}, ErrInvalidAction
	}
	e.mu.RLock()
	defer e.mu.RUnlock()

	unlock := e.lock(userID)
	defer unlock()

	state, err := e.ensureState(ctx, userID)
	if err != nil {
		return HistoryEntry{}, err
	}
	if state.CurrentEventID == "" {
		return HistoryEntry{}, ErrNoActiveEvent
	}
	if state.CurrentEventID != eventID {
		return HistoryEntry{}, ErrOutOfOrderAction
	}

	// Idempotency check AFTER verifying event is current.
	// Important: even if the same action was already recorded, we still must
	// advance the queue state to avoid getting stuck on the same current event.
	last, ok, _ := e.history.LastAction(ctx, userID, eventID)
	skipHistoryAppend := ok && last.Action == action

	if action != ActionLike && action != ActionDislike && action != ActionNeutral {
		return HistoryEntry{}, ErrInvalidAction
	}
	state.EnsureConflictRegistry()
	switch action {
	case ActionDislike:
		state.DropCurrent()
		delete(state.ConflictRegistry, eventID)
	case ActionLike, ActionNeutral:
		conflict := state.CurrentIsConflict
		if !conflict {
			if flag, ok := state.ConflictFlagFor(eventID); ok && flag.Active {
				conflict = true
			}
		}
		state.DropCurrent()
		state.AppendNeutral(eventID, conflict)
	}

	// Always persist queue state; append to history only when not idempotent.
	if skipHistoryAppend {
		if err := e.queues.Save(ctx, userID, state); err != nil {
			return HistoryEntry{}, err
		}
		return last, nil
	}

	entry := HistoryEntry{
		UserID:    userID,
		EventID:   eventID,
		Action:    action,
		Timestamp: e.cfg.Now(),
	}
	if action == ActionLike {
		entry.Context = map[string]any{"session_id": state.SessionID}
	}
	if err := e.persist(ctx, userID, state, entry); err != nil {
		return HistoryEntry{}, err
	}
	return entry, nil
}

// BookEvent commits to the current event and propagates conflicts.
func (e *Engine) BookEvent(ctx context.Context, userID, eventID string) (BookingResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	unlock := e.lock(userID)
	defer unlock()

	event, err := e.events.Get(ctx, eventID)
	if err != nil {
		return BookingResult{}, err
	}
	if last, ok, _ := e.history.LastAction(ctx, userID, eventID); ok && last.Action == ActionBook {
		result := BookingResult{BookedEvent: event}
		if ids, ok := last.Context["conflictedEventIds"].([]string); ok {
			result.ConflictedEventIDs = append([]string(nil), ids...)
		}
		return result, nil
	}
	state, err := e.ensureState(ctx, userID)
	if err != nil {
		return BookingResult{}, err
	}
	if state.CurrentEventID == "" {
		return BookingResult{}, ErrNoActiveEvent
	}
	if state.CurrentEventID != eventID {
		return BookingResult{}, ErrOutOfOrderAction
	}
	conflictedIDs, err := e.markConflicts(ctx, &state, event)
	if err != nil {
		return BookingResult{}, err
	}
	state.DropCurrent()
	entry := HistoryEntry{
		UserID:    userID,
		EventID:   eventID,
		Action:    ActionBook,
		Timestamp: e.cfg.Now(),
		Context: map[string]interface{}{
			"conflictedEventIds": conflictedIDs,
		},
	}
	if err := e.persist(ctx, userID, state, entry); err != nil {
		return BookingResult{}, err
	}
	return BookingResult{BookedEvent: event, ConflictedEventIDs: conflictedIDs}, nil
}

// RegisterBooking records a booking from an external source (e.g. direct subscription)
// and updates conflicts in the discovery queue.
func (e *Engine) RegisterBooking(ctx context.Context, userID, eventID string) (BookingResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	unlock := e.lock(userID)
	defer unlock()

	event, err := e.events.Get(ctx, eventID)
	if err != nil {
		return BookingResult{}, err
	}

	// Check if already booked
	if last, ok, _ := e.history.LastAction(ctx, userID, eventID); ok && last.Action == ActionBook {
		result := BookingResult{BookedEvent: event}
		if ids, ok := last.Context["conflictedEventIds"].([]string); ok {
			result.ConflictedEventIDs = append([]string(nil), ids...)
		}
		return result, nil
	}

	state, err := e.ensureState(ctx, userID)
	if err != nil {
		return BookingResult{}, err
	}

	// If the booked event was the current one, drop it.
	if state.CurrentEventID == eventID {
		state.DropCurrent()
	} else {
		// If it was in the queue (primary or conflicts), remove it.
		state.Primary, _ = removeByID(state.Primary, eventID)
		state.Conflicts, _ = removeByID(state.Conflicts, eventID)
		delete(state.ConflictRegistry, eventID)
	}

	conflictedIDs, err := e.markConflicts(ctx, &state, event)
	if err != nil {
		return BookingResult{}, err
	}

	entry := HistoryEntry{
		UserID:    userID,
		EventID:   eventID,
		Action:    ActionBook,
		Timestamp: e.cfg.Now(),
		Context: map[string]interface{}{
			"conflictedEventIds": conflictedIDs,
			"source":             "external_subscription",
		},
	}

	if err := e.persist(ctx, userID, state, entry); err != nil {
		return BookingResult{}, err
	}

	return BookingResult{BookedEvent: event, ConflictedEventIDs: conflictedIDs}, nil
}

// CancelBooking removes the booking record for a user/event pair.
// This allows the event to reappear in the user's discovery queue.
func (e *Engine) CancelBooking(ctx context.Context, userID, eventID string) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	unlock := e.lock(userID)
	defer unlock()

	// Remove booking from history if repository supports it
	if remover, ok := e.history.(BookingRemover); ok {
		if err := remover.RemoveBooking(ctx, userID, eventID); err != nil {
			return fmt.Errorf("failed to remove booking: %w", err)
		}
	}

	// Reset user's queue to force rebuild on next request
	// This ensures the event will appear again
	if err := e.queues.Clear(ctx, userID); err != nil {
		return fmt.Errorf("failed to clear queue: %w", err)
	}

	return nil
}

// History returns chronological entries for the user.
func (e *Engine) History(ctx context.Context, userID string) ([]HistoryEntry, error) {
	return e.history.List(ctx, userID)
}

// SessionLikes returns liked events for the user's current queue session.
// A "session" is defined by the persisted queue state: once it expires (TTL) or is cleared,
// a new session starts and likes are collected separately.
func (e *Engine) SessionLikes(ctx context.Context, userID string) (SessionLikes, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	unlock := e.lock(userID)
	defer unlock()

	state, err := e.ensureState(ctx, userID)
	if err != nil {
		return SessionLikes{}, err
	}
	sessionID := state.SessionID
	if sessionID == "" {
		// Should not happen (ensureState backfills), but keep it safe.
		sessionID = uuid.NewString()
		state.SessionID = sessionID
		_ = e.queues.Save(ctx, userID, state)
	}

	var entries []HistoryEntry
	if provider, ok := e.history.(SessionLikesProvider); ok {
		entries, err = provider.ListLikesBySession(ctx, userID, sessionID)
	} else {
		var history []HistoryEntry
		history, err = e.history.List(ctx, userID)
		if err == nil {
			entries = make([]HistoryEntry, 0)
			for _, entry := range history {
				if entry.Action != ActionLike {
					continue
				}
				if entry.Context == nil {
					continue
				}
				if sid, ok := entry.Context["session_id"].(string); ok && sid == sessionID {
					entries = append(entries, entry)
				}
			}
		}
	}
	if err != nil {
		return SessionLikes{}, err
	}

	// Sort by time desc to keep stable across repositories.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})

	likes := make([]LikedEvent, 0, len(entries))
	for _, entry := range entries {
		event, err := e.events.Get(ctx, entry.EventID)
		if err != nil {
			// Skip missing events (e.g., catalog refresh) to keep API resilient.
			continue
		}
		likes = append(likes, LikedEvent{Event: event, LikedAt: entry.Timestamp})
	}

	return SessionLikes{SessionID: sessionID, Likes: likes}, nil
}

// ReplaceEvents swaps entire candidate pool atomically.
func (e *Engine) ReplaceEvents(ctx context.Context, events []Event) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.events.ReplaceAll(ctx, events)
}

// ResetUserQueue drops queue state forcing it to be reinitialized on next request.
func (e *Engine) ResetUserQueue(ctx context.Context, userID string) error {
	unlock := e.lock(userID)
	defer unlock()
	return e.queues.Clear(ctx, userID)
}

// ResetAllQueues очищает очереди всех пользователей.
func (e *Engine) ResetAllQueues(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.queues.ClearAll(ctx)
}

// RefreshCatalog atomically replaces events and resets all queues.
// Use this instead of separate ReplaceEvents + ResetAllQueues calls.
func (e *Engine) RefreshCatalog(ctx context.Context, events []Event) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.events.ReplaceAll(ctx, events); err != nil {
		return err
	}
	return e.queues.ClearAll(ctx)
}

func (e *Engine) ensureState(ctx context.Context, userID string) (QueueState, error) {
	return e.ensureStateFiltered(ctx, userID, Filter{})
}

// ensureStateFiltered builds queue with applied filter criteria.
// When filter is empty, behaves like ensureState (no filtering).
// When filter is provided, rebuilds queue with matching events only.
func (e *Engine) ensureStateFiltered(ctx context.Context, userID string, filter Filter) (QueueState, error) {
	// If no filter, try to use cached state
	if filter.IsEmpty() {
		state, err := e.queues.Get(ctx, userID)
		if err == nil {
			// Backfill session id for older persisted states.
			if state.SessionID == "" {
				state.SessionID = uuid.NewString()
				_ = e.queues.Save(ctx, userID, state)
			}
			return state, nil
		}
		if !errors.Is(err, ErrQueueStateNotFound) {
			return QueueState{}, err
		}
	}

	// Always rebuild queue when filter is provided (filter might have changed)
	events, err := e.events.List(ctx)
	if err != nil {
		return QueueState{}, err
	}
	if len(events) == 0 {
		return QueueState{}, ErrQueueEmpty
	}

	// Build set of events user already actioned
	excluded, err := e.getExcludedEvents(ctx, userID)
	if err != nil {
		return QueueState{}, err
	}

	limit := e.cfg.MaxQueueLength
	state := QueueState{
		Primary:           make([]string, 0, limit),
		Conflicts:         []string{},
		ConflictRegistry:  map[string]ConflictFlag{},
		CurrentEventID:    "",
		CurrentIsConflict: false,
	}
	if filter.IsEmpty() {
		state.SessionID = uuid.NewString()
	}

	for _, evt := range events {
		if excluded[evt.ID] {
			continue
		}
		// Apply filter if provided
		if !filter.IsEmpty() && !filter.Matches(evt) {
			continue
		}
		if len(state.Primary) >= limit {
			break
		}
		state.Primary = append(state.Primary, evt.ID)
	}

	if len(state.Primary) == 0 {
		return QueueState{}, ErrQueueEmpty
	}

	// Only save state for non-filtered queues to preserve original behavior
	if filter.IsEmpty() {
		if err := e.queues.Save(ctx, userID, state); err != nil {
			return QueueState{}, err
		}
	}
	return state, nil
}

// getExcludedEvents returns event IDs user has already definitively actioned.
// Uses optimized query if history repo supports ExcludedEventsProvider interface.
func (e *Engine) getExcludedEvents(ctx context.Context, userID string) (map[string]bool, error) {
	// Try optimized path first (PostgresHistoryRepository implements this)
	if provider, ok := e.history.(ExcludedEventsProvider); ok {
		return provider.GetExcludedEventIDs(ctx, userID)
	}

	// Fallback: load full history and filter
	excluded := make(map[string]bool)
	history, err := e.history.List(ctx, userID)
	if err != nil {
		return excluded, nil // Don't fail, just return empty
	}
	for _, entry := range history {
		if entry.Action == ActionDislike || entry.Action == ActionBook {
			excluded[entry.EventID] = true
		}
	}
	return excluded, nil
}

func (e *Engine) markConflicts(ctx context.Context, state *QueueState, booked Event) ([]string, error) {
	state.EnsureConflictRegistry()
	queueSnapshot := append([]string(nil), state.Primary...)
	queueSnapshot = append(queueSnapshot, state.Conflicts...)
	conflicted := make([]string, 0)
	for _, candidateID := range queueSnapshot {
		candidate, err := e.events.Get(ctx, candidateID)
		if err != nil {
			return nil, fmt.Errorf("failed to load candidate %s: %w", candidateID, err)
		}
		if candidate.ID == booked.ID {
			continue
		}
		if !candidate.Slot.Overlaps(booked.Slot) {
			continue
		}
		conflicted = append(conflicted, candidate.ID)
		flag := ConflictFlag{
			Active:      true,
			BookedEvent: booked.ID,
			Reason:      "conflict_with_booking",
			ActivatedAt: e.cfg.Now(),
		}
		state.ConflictRegistry[candidate.ID] = flag
		state.Primary, _ = removeByID(state.Primary, candidate.ID)
		state.Conflicts, _ = removeByID(state.Conflicts, candidate.ID)
		state.Conflicts = append(state.Conflicts, candidate.ID)
	}
	return conflicted, nil
}

func (e *Engine) persist(ctx context.Context, userID string, state QueueState, entry HistoryEntry) error {
	if err := e.queues.Save(ctx, userID, state); err != nil {
		return err
	}
	return e.history.Append(ctx, entry)
}

func (e *Engine) lock(userID string) func() {
	val, _ := e.locks.LoadOrStore(userID, &lockEntry{})
	entry := val.(*lockEntry)
	entry.mu.Lock()
	entry.lastAccess = time.Now().UnixNano()
	return entry.mu.Unlock
}

// CleanupStaleLocks removes lock entries not accessed for the given duration.
// Call periodically (e.g., every hour) to prevent memory leaks.
func (e *Engine) CleanupStaleLocks(maxAge time.Duration) int {
	threshold := time.Now().Add(-maxAge).UnixNano()
	removed := 0
	e.locks.Range(func(key, value any) bool {
		entry := value.(*lockEntry)
		if entry.lastAccess < threshold {
			e.locks.Delete(key)
			removed++
		}
		return true
	})
	return removed
}
