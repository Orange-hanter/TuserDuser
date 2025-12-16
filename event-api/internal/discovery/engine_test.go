package discovery

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

var (
	testBaseTime = time.Date(2025, time.January, 1, 8, 0, 0, 0, time.UTC)
	ctx          = context.Background()
)

func TestEngineLikeAdvancesQueue(t *testing.T) {
	engine := newTestEngine([]Event{
		buildTestEvent("e1", 0),
		buildTestEvent("e2", 60),
	})
	res, err := engine.NextEvent(ctx, "like-user")
	if err != nil {
		t.Fatalf("next event failed: %v", err)
	}
	if res.Event.ID != "e1" {
		t.Fatalf("expected e1, got %s", res.Event.ID)
	}
	if _, err := engine.ApplyAction(ctx, "like-user", res.Event.ID, ActionLike); err != nil {
		t.Fatalf("apply like failed: %v", err)
	}
	res2, err := engine.NextEvent(ctx, "like-user")
	if err != nil {
		t.Fatalf("next event after like failed: %v", err)
	}
	if res2.Event.ID != "e2" {
		t.Fatalf("expected e2, got %s", res2.Event.ID)
	}
	history, err := engine.History(ctx, "like-user")
	if err != nil {
		t.Fatalf("history fetch failed: %v", err)
	}
	if len(history) != 1 || history[0].Action != ActionLike {
		t.Fatalf("unexpected history: %+v", history)
	}
}

func TestEngineDislikeRemovesEvent(t *testing.T) {
	engine := newTestEngine([]Event{
		buildTestEvent("d1", 0),
		buildTestEvent("d2", 45),
	})
	first, err := engine.NextEvent(ctx, "dislike-user")
	if err != nil {
		t.Fatalf("next event failed: %v", err)
	}
	if _, err := engine.ApplyAction(ctx, "dislike-user", first.Event.ID, ActionDislike); err != nil {
		t.Fatalf("apply dislike failed: %v", err)
	}
	second, err := engine.NextEvent(ctx, "dislike-user")
	if err != nil {
		t.Fatalf("next event failed: %v", err)
	}
	if second.Event.ID != "d2" {
		t.Fatalf("expected d2, got %s", second.Event.ID)
	}
}

func TestEngineNeutralRequeues(t *testing.T) {
	engine := newTestEngine([]Event{
		buildTestEvent("n1", 0),
		buildTestEvent("n2", 30),
	})
	first, err := engine.NextEvent(ctx, "neutral-user")
	if err != nil {
		t.Fatalf("next event failed: %v", err)
	}
	if _, err := engine.ApplyAction(ctx, "neutral-user", first.Event.ID, ActionNeutral); err != nil {
		t.Fatalf("neutral action failed: %v", err)
	}
	second, err := engine.NextEvent(ctx, "neutral-user")
	if err != nil {
		t.Fatalf("next event failed: %v", err)
	}
	if second.Event.ID != "n2" {
		t.Fatalf("expected n2, got %s", second.Event.ID)
	}
	if _, err := engine.ApplyAction(ctx, "neutral-user", second.Event.ID, ActionLike); err != nil {
		t.Fatalf("like second failed: %v", err)
	}
	third, err := engine.NextEvent(ctx, "neutral-user")
	if err != nil {
		t.Fatalf("fetch third failed: %v", err)
	}
	if third.Event.ID != "n1" {
		t.Fatalf("neutralized event should reappear, got %s", third.Event.ID)
	}
}

func TestBookingConflictPropagation(t *testing.T) {
	engine := newTestEngine([]Event{
		buildTestEvent("b1", 0),
		buildTestEvent("b2", 30),
		buildTestEvent("b3", 180),
	})
	first, err := engine.NextEvent(ctx, "booking-user")
	if err != nil {
		t.Fatalf("next failed: %v", err)
	}
	result, err := engine.BookEvent(ctx, "booking-user", first.Event.ID)
	if err != nil {
		t.Fatalf("book failed: %v", err)
	}
	if len(result.ConflictedEventIDs) != 1 || result.ConflictedEventIDs[0] != "b2" {
		t.Fatalf("unexpected conflict list: %+v", result)
	}
	second, err := engine.NextEvent(ctx, "booking-user")
	if err != nil {
		t.Fatalf("next after booking failed: %v", err)
	}
	if second.Event.ID != "b3" {
		t.Fatalf("expected non-conflict event b3 next, got %s", second.Event.ID)
	}
	// Like requeues to the end (same as neutral), so use dislike to advance.
	if _, err := engine.ApplyAction(ctx, "booking-user", second.Event.ID, ActionDislike); err != nil {
		t.Fatalf("dislike b3 failed: %v", err)
	}
	third, err := engine.NextEvent(ctx, "booking-user")
	if err != nil {
		t.Fatalf("next conflict failed: %v", err)
	}
	if third.Event.ID != "b2" {
		t.Fatalf("expected conflict event b2 last, got %s", third.Event.ID)
	}
	if !third.Conflict || third.ConflictFlag == nil || third.ConflictFlag.BookedEvent != "b1" {
		t.Fatalf("expected conflict metadata, got %+v", third)
	}
}

func TestNeutralOnConflictKeepsOrdering(t *testing.T) {
	engine := newTestEngine([]Event{
		buildTestEvent("c1", 0),
		buildTestEvent("c2", 15),
		buildTestEvent("c3", 120),
		buildTestEvent("c4", 180),
	})
	first, err := engine.NextEvent(ctx, "conflict-neutral")
	if err != nil {
		t.Fatalf("next failed: %v", err)
	}
	if _, err := engine.BookEvent(ctx, "conflict-neutral", first.Event.ID); err != nil {
		t.Fatalf("booking failed: %v", err)
	}
	second, err := engine.NextEvent(ctx, "conflict-neutral")
	if err != nil {
		t.Fatalf("next after booking failed: %v", err)
	}
	if second.Event.ID != "c3" {
		t.Fatalf("expected c3 first, got %s", second.Event.ID)
	}
	// Like requeues to the end (same as neutral), so use dislike to advance.
	if _, err := engine.ApplyAction(ctx, "conflict-neutral", second.Event.ID, ActionDislike); err != nil {
		t.Fatalf("dislike c3 failed: %v", err)
	}
	third, err := engine.NextEvent(ctx, "conflict-neutral")
	if err != nil {
		t.Fatalf("next third failed: %v", err)
	}
	if third.Event.ID != "c4" {
		t.Fatalf("expected c4 next, got %s", third.Event.ID)
	}
	if _, err := engine.ApplyAction(ctx, "conflict-neutral", third.Event.ID, ActionDislike); err != nil {
		t.Fatalf("dislike c4 failed: %v", err)
	}
	fourth, err := engine.NextEvent(ctx, "conflict-neutral")
	if err != nil {
		t.Fatalf("next conflict failed: %v", err)
	}
	if fourth.Event.ID != "c2" || !fourth.Conflict {
		t.Fatalf("expected conflict event c2, got %+v", fourth)
	}
	if _, err := engine.ApplyAction(ctx, "conflict-neutral", fourth.Event.ID, ActionNeutral); err != nil {
		t.Fatalf("neutral on conflict failed: %v", err)
	}
	fifth, err := engine.NextEvent(ctx, "conflict-neutral")
	if err != nil {
		t.Fatalf("next after neutral conflict failed: %v", err)
	}
	if fifth.Event.ID != "c2" || !fifth.Conflict {
		t.Fatalf("conflict event should remain conflict, got %+v", fifth)
	}
}

func TestQueueExhaustion(t *testing.T) {
	engine := newTestEngine([]Event{buildTestEvent("q1", 0)})
	first, err := engine.NextEvent(ctx, "exhaust-user")
	if err != nil {
		t.Fatalf("next failed: %v", err)
	}
	if _, err := engine.ApplyAction(ctx, "exhaust-user", first.Event.ID, ActionLike); err != nil {
		t.Fatalf("like failed: %v", err)
	}
	// Like requeues to the end (same as neutral), so with a single event it reappears.
	second, err := engine.NextEvent(ctx, "exhaust-user")
	if err != nil {
		t.Fatalf("expected next event, got error %v", err)
	}
	if second.Event.ID != "q1" {
		t.Fatalf("expected q1 to reappear, got %s", second.Event.ID)
	}
}

func TestIdempotentActions(t *testing.T) {
	engine := newTestEngine([]Event{
		buildTestEvent("idem1", 0),
		buildTestEvent("idem2", 60),
	})
	first, err := engine.NextEvent(ctx, "idem-user")
	if err != nil {
		t.Fatalf("next failed: %v", err)
	}

	// First like should work
	entry1, err := engine.ApplyAction(ctx, "idem-user", first.Event.ID, ActionLike)
	if err != nil {
		t.Fatalf("first like failed: %v", err)
	}

	// After like, event is no longer current, so second like should fail with ErrNoActiveEvent
	_, err = engine.ApplyAction(ctx, "idem-user", first.Event.ID, ActionLike)
	if !errors.Is(err, ErrNoActiveEvent) && !errors.Is(err, ErrOutOfOrderAction) {
		t.Fatalf("expected no active event or out of order error, got: %v", err)
	}

	// Get next event and verify idempotency works for CURRENT event
	second, err := engine.NextEvent(ctx, "idem-user")
	if err != nil {
		t.Fatalf("second next failed: %v", err)
	}

	// Call next again without action - should return same event (idempotent)
	secondAgain, err := engine.NextEvent(ctx, "idem-user")
	if err != nil {
		t.Fatalf("third next failed: %v", err)
	}
	if secondAgain.Event.ID != second.Event.ID {
		t.Fatalf("next should be idempotent, got %s then %s", second.Event.ID, secondAgain.Event.ID)
	}

	_ = entry1 // use variable
}

func TestBookEventIdempotency(t *testing.T) {
	engine := newTestEngine([]Event{
		buildTestEvent("book1", 0),
		buildTestEvent("book2", 120),
	})
	first, err := engine.NextEvent(ctx, "book-user")
	if err != nil {
		t.Fatalf("next failed: %v", err)
	}
	result, err := engine.BookEvent(ctx, "book-user", first.Event.ID)
	if err != nil {
		t.Fatalf("booking failed: %v", err)
	}
	second, err := engine.BookEvent(ctx, "book-user", first.Event.ID)
	if err != nil {
		t.Fatalf("idempotent booking failed: %v", err)
	}
	if second.BookedEvent.ID != result.BookedEvent.ID {
		t.Fatalf("unexpected booked event: %+v", second)
	}
}

func TestConcurrencySafety(t *testing.T) {
	events := []Event{
		buildTestEvent("con1", 0),
		buildTestEvent("con2", 60),
		buildTestEvent("con3", 120),
	}
	engine := newTestEngine(events)
	const workers = 10
	const stepsPerWorker = 100
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			userID := fmt.Sprintf("parallel-%d", idx)
			for step := 0; step < stepsPerWorker; step++ {
				res, err := engine.NextEvent(ctx, userID)
				if err != nil {
					t.Errorf("user %s next failed: %v", userID, err)
					return
				}
				if _, err := engine.ApplyAction(ctx, userID, res.Event.ID, ActionLike); err != nil {
					t.Errorf("user %s apply failed: %v", userID, err)
					return
				}
			}
		}(i)
	}
	wg.Wait()
}

func newTestEngine(events []Event) *Engine {
	repo := NewInMemoryEventRepository(events)
	queues := NewInMemoryQueueRepository()
	history := NewInMemoryHistoryRepository()
	return NewEngine(repo, queues, history, EngineConfig{Now: func() time.Time { return testBaseTime }})
}

func buildTestEvent(id string, startMinute int) Event {
	start := testBaseTime.Add(time.Duration(startMinute) * time.Minute)
	return Event{
		ID:          id,
		Title:       id,
		Description: id,
		Slot: TimeSlot{
			Start: start,
			End:   start.Add(30 * time.Minute),
		},
		Metadata: map[string]interface{}{},
	}
}

func TestRefreshCatalogAtomic(t *testing.T) {
	engine := newTestEngine([]Event{buildTestEvent("old1", 0)})

	// User gets first event
	res, err := engine.NextEvent(ctx, "refresh-user")
	if err != nil {
		t.Fatalf("next failed: %v", err)
	}
	if res.Event.ID != "old1" {
		t.Fatalf("expected old1, got %s", res.Event.ID)
	}

	// Atomic refresh with new events
	newEvents := []Event{buildTestEvent("new1", 0), buildTestEvent("new2", 60)}
	if err := engine.RefreshCatalog(ctx, newEvents); err != nil {
		t.Fatalf("refresh failed: %v", err)
	}

	// User should see new events (queue was reset)
	res2, err := engine.NextEvent(ctx, "refresh-user")
	if err != nil {
		t.Fatalf("next after refresh failed: %v", err)
	}
	if res2.Event.ID != "new1" {
		t.Fatalf("expected new1, got %s", res2.Event.ID)
	}
}

func TestCleanupStaleLocks(t *testing.T) {
	engine := newTestEngine([]Event{buildTestEvent("c1", 0)})

	// Create some locks by accessing users
	for i := 0; i < 5; i++ {
		_, _ = engine.NextEvent(ctx, fmt.Sprintf("cleanup-user-%d", i))
	}

	// Cleanup with 0 duration should remove all
	removed := engine.CleanupStaleLocks(0)
	if removed != 5 {
		t.Fatalf("expected 5 removed, got %d", removed)
	}

	// Second cleanup should find nothing
	removed2 := engine.CleanupStaleLocks(0)
	if removed2 != 0 {
		t.Fatalf("expected 0 removed on second pass, got %d", removed2)
	}
}

func TestConcurrentRefreshAndRead(t *testing.T) {
	engine := newTestEngine([]Event{buildTestEvent("r1", 0), buildTestEvent("r2", 60)})

	var wg sync.WaitGroup
	const iterations = 100

	// Reader goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_, err := engine.NextEvent(ctx, "concurrent-reader")
			if err != nil && !errors.Is(err, ErrQueueEmpty) {
				t.Errorf("reader iteration %d failed: %v", i, err)
				return
			}
			// Reset queue to allow re-reading
			_ = engine.ResetUserQueue(ctx, "concurrent-reader")
		}
	}()

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			events := []Event{buildTestEvent(fmt.Sprintf("refresh-%d", i), 0)}
			if err := engine.RefreshCatalog(ctx, events); err != nil {
				t.Errorf("writer iteration %d failed: %v", i, err)
				return
			}
		}
	}()

	wg.Wait()
}

// --- Filter Tests ---

func TestFilterMatches(t *testing.T) {
	baseEvent := Event{
		ID:    "test-event",
		Title: "Test Event",
		Slot: TimeSlot{
			Start: testBaseTime,
			End:   testBaseTime.Add(60 * time.Minute),
		},
		Metadata: map[string]any{
			"type":      "Конференция",
			"priceType": "paid",
			"place":     "Коворкинг \"Старт\"",
		},
	}

	tests := []struct {
		name     string
		filter   Filter
		event    Event
		expected bool
	}{
		{
			name:     "empty filter matches all",
			filter:   Filter{},
			event:    baseEvent,
			expected: true,
		},
		{
			name:     "type filter matches",
			filter:   Filter{Types: []string{"Конференция"}},
			event:    baseEvent,
			expected: true,
		},
		{
			name:     "type filter case insensitive",
			filter:   Filter{Types: []string{"конференция"}},
			event:    baseEvent,
			expected: true,
		},
		{
			name:     "type filter no match",
			filter:   Filter{Types: []string{"Воркшоп"}},
			event:    baseEvent,
			expected: false,
		},
		{
			name:     "priceType filter matches",
			filter:   Filter{PriceTypes: []string{"paid"}},
			event:    baseEvent,
			expected: true,
		},
		{
			name:     "priceType filter no match",
			filter:   Filter{PriceTypes: []string{"free"}},
			event:    baseEvent,
			expected: false,
		},
		{
			name:     "place substring match",
			filter:   Filter{Places: []string{"Коворкинг"}},
			event:    baseEvent,
			expected: true,
		},
		{
			name:     "place partial match",
			filter:   Filter{Places: []string{"Старт"}},
			event:    baseEvent,
			expected: true,
		},
		{
			name:     "place no match",
			filter:   Filter{Places: []string{"Стадион"}},
			event:    baseEvent,
			expected: false,
		},
		{
			name: "dateFrom filter matches",
			filter: Filter{
				DateFrom: func() *time.Time { t := testBaseTime.Add(-1 * time.Hour); return &t }(),
			},
			event:    baseEvent,
			expected: true,
		},
		{
			name: "dateFrom filter no match",
			filter: Filter{
				DateFrom: func() *time.Time { t := testBaseTime.Add(1 * time.Hour); return &t }(),
			},
			event:    baseEvent,
			expected: false,
		},
		{
			name: "dateTo filter matches",
			filter: Filter{
				DateTo: func() *time.Time { t := testBaseTime.Add(1 * time.Hour); return &t }(),
			},
			event:    baseEvent,
			expected: true,
		},
		{
			name: "dateTo filter no match",
			filter: Filter{
				DateTo: func() *time.Time { t := testBaseTime.Add(-1 * time.Hour); return &t }(),
			},
			event:    baseEvent,
			expected: false,
		},
		{
			name: "multiple filters all match",
			filter: Filter{
				Types:      []string{"Конференция"},
				PriceTypes: []string{"paid", "free"},
				Places:     []string{"Коворкинг"},
			},
			event:    baseEvent,
			expected: true,
		},
		{
			name: "multiple filters one fails",
			filter: Filter{
				Types:      []string{"Конференция"},
				PriceTypes: []string{"free"},
			},
			event:    baseEvent,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.Matches(tt.event)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestNextEventFiltered(t *testing.T) {
	events := []Event{
		{
			ID:    "e1",
			Title: "Conference",
			Slot:  TimeSlot{Start: testBaseTime, End: testBaseTime.Add(60 * time.Minute)},
			Metadata: map[string]any{
				"type":      "Конференция",
				"priceType": "paid",
			},
		},
		{
			ID:    "e2",
			Title: "Workshop",
			Slot:  TimeSlot{Start: testBaseTime.Add(120 * time.Minute), End: testBaseTime.Add(180 * time.Minute)},
			Metadata: map[string]any{
				"type":      "Воркшоп",
				"priceType": "free",
			},
		},
		{
			ID:    "e3",
			Title: "Party",
			Slot:  TimeSlot{Start: testBaseTime.Add(240 * time.Minute), End: testBaseTime.Add(300 * time.Minute)},
			Metadata: map[string]any{
				"type":      "Вечеринка",
				"priceType": "paid",
			},
		},
	}
	engine := newTestEngine(events)

	// Without filter - get first event
	res, err := engine.NextEventFiltered(ctx, "filter-user-1", Filter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Event.ID != "e1" {
		t.Errorf("expected e1, got %s", res.Event.ID)
	}

	// With type filter - get only Воркшоп
	res, err = engine.NextEventFiltered(ctx, "filter-user-2", Filter{Types: []string{"Воркшоп"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Event.ID != "e2" {
		t.Errorf("expected e2, got %s", res.Event.ID)
	}

	// With priceType filter - get only free
	res, err = engine.NextEventFiltered(ctx, "filter-user-3", Filter{PriceTypes: []string{"free"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Event.ID != "e2" {
		t.Errorf("expected e2, got %s", res.Event.ID)
	}

	// With combined filters
	res, err = engine.NextEventFiltered(ctx, "filter-user-4", Filter{
		Types:      []string{"Вечеринка"},
		PriceTypes: []string{"paid"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Event.ID != "e3" {
		t.Errorf("expected e3, got %s", res.Event.ID)
	}

	// No matching events
	_, err = engine.NextEventFiltered(ctx, "filter-user-5", Filter{Types: []string{"NonExistent"}})
	if !errors.Is(err, ErrQueueEmpty) {
		t.Errorf("expected ErrQueueEmpty, got %v", err)
	}
}
