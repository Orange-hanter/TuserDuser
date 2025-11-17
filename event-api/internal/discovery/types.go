package discovery

import (
	"errors"
	"time"
)

// UserAction represents every supported user intent the engine can process.
type UserAction string

const (
	ActionLike    UserAction = "like"
	ActionDislike UserAction = "dislike"
	ActionNeutral UserAction = "neutral"
	ActionBook    UserAction = "book"
)

// TimeSlot defines a closed-open interval for an event.
type TimeSlot struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// Overlaps detects whether two time slots intersect.
func (ts TimeSlot) Overlaps(other TimeSlot) bool {
	if ts.End.Before(ts.Start) || other.End.Before(other.Start) {
		return false
	}
	return !ts.End.Before(other.Start) && !other.End.Before(ts.Start)
}

// Event represents a candidate event inside the discovery queue.
type Event struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Slot        TimeSlot `json:"slot"`
	// unstructured, user-defined. Prefer typed fields for core logic.
	Metadata map[string]any `json:"metadata"`
}

// ConflictFlag keeps metadata that explains why an event is delayed.
type ConflictFlag struct {
	Active      bool      `json:"active"`
	BookedEvent string    `json:"bookedEvent"`
	Reason      string    `json:"reason"`
	ActivatedAt time.Time `json:"activatedAt"`
}

// HistoryEntry captures every user interaction with the engine.
type HistoryEntry struct {
	UserID    string         `json:"userId"`
	EventID   string         `json:"eventId"`
	Action    UserAction     `json:"action"`
	Timestamp time.Time      `json:"timestamp"`
	Context   map[string]any `json:"context,omitempty"`
}

// NextEvent wraps queue metadata returned to callers.
type NextEvent struct {
	Event              Event         `json:"event"`
	Conflict           bool          `json:"conflict"`
	ConflictFlag       *ConflictFlag `json:"conflictFlag,omitempty"`
	RemainingPrimary   int           `json:"remainingPrimary"`
	RemainingConflicts int           `json:"remainingConflicts"`
}

// BookingResult reports deterministic booking outcome.
type BookingResult struct {
	BookedEvent        Event    `json:"bookedEvent"`
	ConflictedEventIDs []string `json:"conflictedEventIds"`
}

// QueueState represents the deterministic structure used by the engine.
//
// QueueState maintains the following invariants:
// 1. CurrentEventID is either empty or equals Primary[0] or Conflicts[0].
// 2. ConflictRegistry contains only entries for events in Conflicts (or recently removed).
// 3. No duplicate event IDs across Primary and Conflicts.
// Violations indicate a logic bug — check DropCurrent/Remove/Append paths.
type QueueState struct {
	Primary           []string                `json:"primary"`
	Conflicts         []string                `json:"conflicts"`
	ConflictRegistry  map[string]ConflictFlag `json:"conflictRegistry"`
	CurrentEventID    string                  `json:"currentEventId"`
	CurrentIsConflict bool                    `json:"currentIsConflict"`
}

// Clone returns a deep copy so repositories can protect their state.
func (s QueueState) Clone() QueueState {
	clone := QueueState{
		Primary:           append([]string(nil), s.Primary...),
		Conflicts:         append([]string(nil), s.Conflicts...),
		ConflictRegistry:  make(map[string]ConflictFlag, len(s.ConflictRegistry)),
		CurrentEventID:    s.CurrentEventID,
		CurrentIsConflict: s.CurrentIsConflict,
	}
	for k, v := range s.ConflictRegistry {
		clone.ConflictRegistry[k] = v
	}
	return clone
}

// EnsureConflictRegistry lazily initializes the registry map.
func (s *QueueState) EnsureConflictRegistry() {
	if s.ConflictRegistry == nil {
		s.ConflictRegistry = make(map[string]ConflictFlag)
	}
}

// NextCandidate returns the event at the front without mutating slices.
func (s *QueueState) NextCandidate() (string, bool, error) {
	if s.CurrentEventID != "" {
		return s.CurrentEventID, s.CurrentIsConflict, nil
	}
	if len(s.Primary) > 0 {
		s.CurrentEventID = s.Primary[0]
		s.CurrentIsConflict = false
		return s.CurrentEventID, false, nil
	}
	if len(s.Conflicts) > 0 {
		s.CurrentEventID = s.Conflicts[0]
		s.CurrentIsConflict = true
		return s.CurrentEventID, true, nil
	}
	return "", false, ErrQueueEmpty
}

// ReleaseCurrent resets the currently assigned event pointer.
func (s *QueueState) ReleaseCurrent() {
	s.CurrentEventID = ""
	s.CurrentIsConflict = false
}

// DropCurrent removes the currently assigned element from the respective queue.
func (s *QueueState) DropCurrent() {
	if s.CurrentEventID == "" {
		return
	}
	if s.CurrentIsConflict {
		s.Conflicts = dropHeadIfMatch(s.Conflicts, s.CurrentEventID)
	} else {
		s.Primary = dropHeadIfMatch(s.Primary, s.CurrentEventID)
	}
	s.ReleaseCurrent()
}

// Remove removes the provided event from both queues.
func (s *QueueState) Remove(eventID string) bool {
	var removedPrimary bool
	s.Primary, removedPrimary = removeByID(s.Primary, eventID)
	var removedConflict bool
	s.Conflicts, removedConflict = removeByID(s.Conflicts, eventID)
	found := removedPrimary || removedConflict
	if s.CurrentEventID == eventID {
		s.ReleaseCurrent()
	}
	return found
}

// AppendNeutral appends event to the tail of the relevant queue.
func (s *QueueState) AppendNeutral(eventID string, conflict bool) {
	if conflict {
		s.Conflicts = append(s.Conflicts, eventID)
	} else {
		s.Primary = append(s.Primary, eventID)
	}
}

// ConflictFlagFor returns conflict metadata when available.
func (s QueueState) ConflictFlagFor(eventID string) (ConflictFlag, bool) {
	if s.ConflictRegistry == nil {
		return ConflictFlag{}, false
	}
	flag, ok := s.ConflictRegistry[eventID]
	return flag, ok && flag.Active
}

// IsEmpty returns true if both queues do not contain items.
func (s QueueState) IsEmpty() bool {
	return len(s.Primary) == 0 && len(s.Conflicts) == 0 && s.CurrentEventID == ""
}

func dropHeadIfMatch(src []string, expected string) []string {
	if len(src) == 0 {
		return src
	}
	if src[0] == expected {
		return append([]string(nil), src[1:]...)
	}
	return src
}

func removeByID(src []string, target string) ([]string, bool) {
	if len(src) == 0 {
		return src, false
	}
	res := make([]string, 0, len(src))
	found := false
	for _, candidate := range src {
		if !found && candidate == target {
			found = true
			continue
		}
		res = append(res, candidate)
	}
	return res, found
}

var (
	// ErrQueueEmpty indicates there is no event left to serve.
	ErrQueueEmpty = errors.New("queue is empty")
	// ErrEventNotFound signals missing events in repository.
	ErrEventNotFound = errors.New("event not found")
	// ErrInvalidAction indicates an unsupported user action was requested.
	ErrInvalidAction = errors.New("invalid action")
	// ErrOutOfOrderAction is returned when user tries to act on a non-current event.
	ErrOutOfOrderAction = errors.New("out of order action")
	// ErrNoActiveEvent is returned when ApplyAction is invoked without requesting Next first.
	ErrNoActiveEvent = errors.New("no active event")
)
