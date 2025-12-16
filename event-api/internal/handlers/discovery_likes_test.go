package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"event-api/internal/discovery"
	"event-api/internal/logger"

	"go.uber.org/zap"
)

func setupDiscoveryHandlerTest(t *testing.T) (*DiscoveryHandler, time.Time) {
	t.Helper()

	logger.Log = zap.NewNop()

	now := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	events := []discovery.Event{
		{
			ID:          "event-1",
			Title:       "Event 1",
			Description: "First",
			Slot: discovery.TimeSlot{
				Start: now.Add(24 * time.Hour),
				End:   now.Add(25 * time.Hour),
			},
			Metadata: map[string]any{},
		},
		{
			ID:          "event-2",
			Title:       "Event 2",
			Description: "Second",
			Slot: discovery.TimeSlot{
				Start: now.Add(48 * time.Hour),
				End:   now.Add(49 * time.Hour),
			},
			Metadata: map[string]any{},
		},
	}

	repo := discovery.NewInMemoryEventRepository(events)
	queues := discovery.NewInMemoryQueueRepository()
	history := discovery.NewInMemoryHistoryRepository()
	engine := discovery.NewEngine(repo, queues, history, discovery.EngineConfig{MaxQueueLength: 16, Now: func() time.Time { return now }})
	svc := discovery.NewService(engine)

	return NewDiscoveryHandler(svc, nil), now
}

func TestDiscoveryHandler_SessionLikes_Unauthorized(t *testing.T) {
	h, _ := setupDiscoveryHandlerTest(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/api/discovery/likes", nil)
	resp := httptest.NewRecorder()

	h.SessionLikes(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.Code)
	}
}

func TestDiscoveryHandler_SessionLikes_ReturnsLikedEventsForSession(t *testing.T) {
	h, now := setupDiscoveryHandlerTest(t)

	userID := "user-1"

	// Step 1: fetch next to initialize the queue/session.
	nextReq := httptest.NewRequest(http.MethodGet, "/v1/api/discovery/next", nil)
	nextReq.Header.Set("X-User-ID", userID)
	nextResp := httptest.NewRecorder()
	h.Next(nextResp, nextReq)
	if nextResp.Code != http.StatusOK {
		t.Fatalf("expected 200 from next, got %d: %s", nextResp.Code, nextResp.Body.String())
	}
	var next discovery.NextEvent
	if err := json.Unmarshal(nextResp.Body.Bytes(), &next); err != nil {
		t.Fatalf("failed to decode next response: %v", err)
	}
	if next.Event.ID == "" {
		t.Fatalf("expected event id")
	}

	// Step 2: like the event.
	actionBody := map[string]any{"eventId": next.Event.ID, "action": "like"}
	payload, err := json.Marshal(actionBody)
	if err != nil {
		t.Fatalf("failed to marshal action body: %v", err)
	}
	actionReq := httptest.NewRequest(http.MethodPost, "/v1/api/discovery/action", bytes.NewReader(payload))
	actionReq.Header.Set("X-User-ID", userID)
	actionResp := httptest.NewRecorder()
	h.Action(actionResp, actionReq)
	if actionResp.Code != http.StatusOK {
		t.Fatalf("expected 200 from action, got %d: %s", actionResp.Code, actionResp.Body.String())
	}

	// Step 3: fetch likes for current session.
	likesReq := httptest.NewRequest(http.MethodGet, "/v1/api/discovery/likes", nil)
	likesReq.Header.Set("X-User-ID", userID)
	likesResp := httptest.NewRecorder()
	h.SessionLikes(likesResp, likesReq)
	if likesResp.Code != http.StatusOK {
		t.Fatalf("expected 200 from likes, got %d: %s", likesResp.Code, likesResp.Body.String())
	}

	var out discovery.SessionLikes
	if err := json.Unmarshal(likesResp.Body.Bytes(), &out); err != nil {
		t.Fatalf("failed to decode likes response: %v", err)
	}
	if out.SessionID == "" {
		t.Fatalf("expected non-empty sessionId")
	}
	if len(out.Likes) != 1 {
		t.Fatalf("expected 1 like, got %d", len(out.Likes))
	}
	if out.Likes[0].Event.ID != next.Event.ID {
		t.Fatalf("expected liked event %s, got %s", next.Event.ID, out.Likes[0].Event.ID)
	}
	if !out.Likes[0].LikedAt.Equal(now) {
		t.Fatalf("expected likedAt %s, got %s", now.Format(time.RFC3339Nano), out.Likes[0].LikedAt.Format(time.RFC3339Nano))
	}
}
