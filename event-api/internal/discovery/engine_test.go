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
	if _, err := engine.ApplyAction(ctx, "booking-user", second.Event.ID, ActionLike); err != nil {
		t.Fatalf("like b3 failed: %v", err)
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
	if _, err := engine.ApplyAction(ctx, "conflict-neutral", second.Event.ID, ActionLike); err != nil {
		t.Fatalf("like c3 failed: %v", err)
	}
	third, err := engine.NextEvent(ctx, "conflict-neutral")
	if err != nil {
		t.Fatalf("next third failed: %v", err)
	}
	if third.Event.ID != "c4" {
		t.Fatalf("expected c4 next, got %s", third.Event.ID)
	}
	if _, err := engine.ApplyAction(ctx, "conflict-neutral", third.Event.ID, ActionLike); err != nil {
		t.Fatalf("like c4 failed: %v", err)
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
	if _, err := engine.NextEvent(ctx, "exhaust-user"); !errors.Is(err, ErrQueueEmpty) {
		t.Fatalf("expected queue empty error, got %v", err)
	}
}

func TestIdempotentActions(t *testing.T) {
	engine := newTestEngine([]Event{buildTestEvent("idem", 0)})
	first, err := engine.NextEvent(ctx, "idem-user")
	if err != nil {
		t.Fatalf("next failed: %v", err)
	}
	if _, err := engine.ApplyAction(ctx, "idem-user", first.Event.ID, ActionLike); err != nil {
		t.Fatalf("first like failed: %v", err)
	}
	if _, err := engine.ApplyAction(ctx, "idem-user", first.Event.ID, ActionLike); err != nil {
		t.Fatalf("idempotent like failed: %v", err)
	}
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
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			userID := fmt.Sprintf("parallel-%d", idx)
			for {
				res, err := engine.NextEvent(ctx, userID)
				if errors.Is(err, ErrQueueEmpty) {
					return
				}
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
