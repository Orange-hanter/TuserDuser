package discovery

import (
	"context"
	"errors"
	"sort"
	"sync"
)

// EventRepository abstracts event storage.
type EventRepository interface {
	ReplaceAll(ctx context.Context, events []Event) error
	List(ctx context.Context) ([]Event, error)
	Get(ctx context.Context, id string) (Event, error)
}

// QueueRepository stores per-user queue states.
type QueueRepository interface {
	Get(ctx context.Context, userID string) (QueueState, error)
	Save(ctx context.Context, userID string, state QueueState) error
	Clear(ctx context.Context, userID string) error
	ClearAll(ctx context.Context) error
}

// HistoryRepository persists chronological logs of user actions.
type HistoryRepository interface {
	Append(ctx context.Context, entry HistoryEntry) error
	List(ctx context.Context, userID string) ([]HistoryEntry, error)
	LastAction(ctx context.Context, userID, eventID string) (HistoryEntry, bool, error)
}

// ExcludedEventsProvider is an optional interface for optimized excluded events lookup.
// PostgresHistoryRepository implements this for better performance.
type ExcludedEventsProvider interface {
	GetExcludedEventIDs(ctx context.Context, userID string) (map[string]bool, error)
}

// ErrQueueStateNotFound indicates missing state for provided user.
var ErrQueueStateNotFound = errors.New("queue state not found")

// InMemoryEventRepository provides deterministic event data during tests and development.
type InMemoryEventRepository struct {
	mu     sync.RWMutex
	events map[string]Event
}

// NewInMemoryEventRepository returns a repository prefilled with events.
func NewInMemoryEventRepository(seed []Event) *InMemoryEventRepository {
	repo := &InMemoryEventRepository{events: make(map[string]Event)}
	_ = repo.ReplaceAll(context.Background(), seed)
	return repo
}

// ReplaceAll swaps entire dataset atomically.
func (r *InMemoryEventRepository) ReplaceAll(_ context.Context, events []Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = make(map[string]Event, len(events))
	for _, e := range events {
		// ensure metadata map is not nil to avoid nil map panics during JSON marshaling
		if e.Metadata == nil {
			e.Metadata = map[string]any{}
		}
		r.events[e.ID] = e
	}
	return nil
}

// List returns events sorted by start time then ID to guarantee determinism.
//
// TODO: Write test case for this function
func (r *InMemoryEventRepository) List(_ context.Context) ([]Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	events := make([]Event, 0, len(r.events))
	for _, e := range r.events {
		events = append(events, e)
	}
	sort.Slice(events, func(i, j int) bool {
		if !events[i].Slot.Start.Equal(events[j].Slot.Start) {
			return events[i].Slot.Start.Before(events[j].Slot.Start)
		}
		return events[i].ID < events[j].ID
	})
	return events, nil
}

// Get returns a single event by ID.
func (r *InMemoryEventRepository) Get(_ context.Context, id string) (Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	event, ok := r.events[id]
	if !ok {
		return Event{}, ErrEventNotFound
	}
	return event, nil
}

// InMemoryQueueRepository stores queue states per user safely.
//
// TODO: write extended documentation. How it used, for what purpose
type InMemoryQueueRepository struct {
	mu     sync.RWMutex
	states map[string]QueueState
}

// NewInMemoryQueueRepository allocates a repository.
func NewInMemoryQueueRepository() *InMemoryQueueRepository {
	return &InMemoryQueueRepository{states: make(map[string]QueueState)}
}

// Get fetches queue state for a user.
func (r *InMemoryQueueRepository) Get(_ context.Context, userID string) (QueueState, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	state, ok := r.states[userID]
	if !ok {
		return QueueState{}, ErrQueueStateNotFound
	}
	return state.Clone(), nil
}

// Save persists queue state replacing previous value.
func (r *InMemoryQueueRepository) Save(_ context.Context, userID string, state QueueState) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	state.EnsureConflictRegistry()
	r.states[userID] = state.Clone()
	return nil
}

// Clear removes stored state.
func (r *InMemoryQueueRepository) Clear(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.states, userID)
	return nil
}

// ClearAll remove all stored states.
func (r *InMemoryQueueRepository) ClearAll(_ context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.states = make(map[string]QueueState)
	return nil
}

// InMemoryHistoryRepository stores chronological logs.
type InMemoryHistoryRepository struct {
	mu        sync.RWMutex
	entries   map[string][]HistoryEntry
	lastByKey map[string]map[string]HistoryEntry
}

// NewInMemoryHistoryRepository allocates repository.
func NewInMemoryHistoryRepository() *InMemoryHistoryRepository {
	return &InMemoryHistoryRepository{
		entries:   make(map[string][]HistoryEntry),
		lastByKey: make(map[string]map[string]HistoryEntry),
	}
}

// Append stores a history entry while keeping idempotency lookup tables.
func (r *InMemoryHistoryRepository) Append(_ context.Context, entry HistoryEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copyEntry := cloneHistoryEntry(entry)
	r.entries[entry.UserID] = append(r.entries[entry.UserID], copyEntry)
	if _, ok := r.lastByKey[entry.UserID]; !ok {
		r.lastByKey[entry.UserID] = make(map[string]HistoryEntry)
	}
	r.lastByKey[entry.UserID][entry.EventID] = copyEntry
	return nil
}

// List returns user history in chronological order.
func (r *InMemoryHistoryRepository) List(_ context.Context, userID string) ([]HistoryEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entries := r.entries[userID]
	copyEntries := make([]HistoryEntry, len(entries))
	for i, entry := range entries {
		copyEntries[i] = cloneHistoryEntry(entry)
	}
	return copyEntries, nil
}

// LastAction returns last recorded action for event.
func (r *InMemoryHistoryRepository) LastAction(_ context.Context, userID, eventID string) (HistoryEntry, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	perUser, ok := r.lastByKey[userID]
	if !ok {
		return HistoryEntry{}, false, nil
	}
	entry, ok := perUser[eventID]
	if !ok {
		return HistoryEntry{}, false, nil
	}
	return cloneHistoryEntry(entry), true, nil
}

func cloneHistoryEntry(entry HistoryEntry) HistoryEntry {
	clone := entry
	if entry.Context != nil {
		clone.Context = make(map[string]interface{}, len(entry.Context))
		for k, v := range entry.Context {
			switch typed := v.(type) {
			case []string:
				dup := make([]string, len(typed))
				copy(dup, typed)
				clone.Context[k] = dup
			default:
				clone.Context[k] = typed
			}
		}
	}
	return clone
}
