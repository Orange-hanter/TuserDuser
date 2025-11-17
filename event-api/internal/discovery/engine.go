package discovery

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
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
	locks   sync.Map // map[userID]*sync.Mutex
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

// NextEvent returns the next item for a user, lazily initializing their queue.
func (e *Engine) NextEvent(ctx context.Context, userID string) (NextEvent, error) {
	unlock := e.lock(userID)
	defer unlock()

	state, err := e.ensureState(ctx, userID)
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
	unlock := e.lock(userID)
	defer unlock()

	if last, ok, _ := e.history.LastAction(ctx, userID, eventID); ok && last.Action == action {
		return last, nil
	}

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
	if action != ActionLike && action != ActionDislike && action != ActionNeutral {
		return HistoryEntry{}, ErrInvalidAction
	}
	state.EnsureConflictRegistry()
	switch action {
	case ActionLike:
		state.DropCurrent()
		delete(state.ConflictRegistry, eventID)
	case ActionDislike:
		state.DropCurrent()
		delete(state.ConflictRegistry, eventID)
	case ActionNeutral:
		conflict := state.CurrentIsConflict
		if !conflict {
			if flag, ok := state.ConflictFlagFor(eventID); ok && flag.Active {
				conflict = true
			}
		}
		state.DropCurrent()
		state.AppendNeutral(eventID, conflict)
	}
	entry := HistoryEntry{
		UserID:    userID,
		EventID:   eventID,
		Action:    action,
		Timestamp: e.cfg.Now(),
	}
	if err := e.persist(ctx, userID, state, entry); err != nil {
		return HistoryEntry{}, err
	}
	return entry, nil
}

// BookEvent commits to the current event and propagates conflicts.
func (e *Engine) BookEvent(ctx context.Context, userID, eventID string) (BookingResult, error) {
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

// History returns chronological entries for the user.
func (e *Engine) History(ctx context.Context, userID string) ([]HistoryEntry, error) {
	return e.history.List(ctx, userID)
}

// ReplaceEvents swaps entire candidate pool atomically.
func (e *Engine) ReplaceEvents(ctx context.Context, events []Event) error {
	return e.events.ReplaceAll(ctx, events)
}

// ResetUserQueue drops queue state forcing it to be reinitialized on next request.
func (e *Engine) ResetUserQueue(ctx context.Context, userID string) error {
	unlock := e.lock(userID)
	defer unlock()
	return e.queues.Clear(ctx, userID)
}

func (e *Engine) ensureState(ctx context.Context, userID string) (QueueState, error) {
	state, err := e.queues.Get(ctx, userID)
	if err == nil {
		return state, nil
	}
	if !errors.Is(err, ErrQueueStateNotFound) {
		return QueueState{}, err
	}
	events, err := e.events.List(ctx)
	if err != nil {
		return QueueState{}, err
	}
	if len(events) == 0 {
		return QueueState{}, ErrQueueEmpty
	}
	limit := e.cfg.MaxQueueLength
	if limit > len(events) {
		limit = len(events)
	}
	state = QueueState{
		Primary:           make([]string, 0, limit),
		Conflicts:         []string{},
		ConflictRegistry:  map[string]ConflictFlag{},
		CurrentEventID:    "",
		CurrentIsConflict: false,
	}
	for i := 0; i < limit; i++ {
		state.Primary = append(state.Primary, events[i].ID)
	}
	if err := e.queues.Save(ctx, userID, state); err != nil {
		return QueueState{}, err
	}
	return state, nil
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
	val, _ := e.locks.LoadOrStore(userID, &sync.Mutex{})
	mu := val.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}
