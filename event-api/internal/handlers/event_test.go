package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"event-api/internal/logger"
	"event-api/internal/models"
	"event-api/internal/service"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func setupEventHandlerTest(t *testing.T) (*EventHandler, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	logger.Log = zap.NewNop()
	svc := service.NewEventService(db, zap.NewNop())
	handler := NewEventHandler(svc)
	return handler, mock, func() { _ = db.Close() }
}

func TestEventHandler_CreateEvent_ReturnsPendingEvent(t *testing.T) {
	handler, mock, cleanup := setupEventHandlerTest(t)
	defer cleanup()

	startTime := time.Date(2024, 8, 10, 9, 0, 0, 0, time.UTC)
	endTime := startTime.Add(2 * time.Hour)
	reqPayload := models.CreateEventRequest{
		Type:             "meetup",
		PriceType:        "free",
		Duration:         120,
		Place:            "Online",
		StartTime:        startTime,
		EndTime:          endTime,
		NeedRegistration: true,
		Details: map[string]interface{}{
			"lang": "en",
		},
	}

	body, err := json.Marshal(reqPayload)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	returnDetails, _ := json.Marshal(reqPayload.Details)
	createdAt := startTime
	updatedAt := endTime
	rows := sqlmock.NewRows([]string{
		"id", "type", "start_time", "end_time", "duration", "place",
		"price_type", "need_registration", "details", "status", "review_comment",
		"created_at", "updated_at", "reviewed_at",
	}).AddRow(
		"event-123",
		reqPayload.Type,
		reqPayload.StartTime,
		reqPayload.EndTime,
		reqPayload.Duration,
		reqPayload.Place,
		reqPayload.PriceType,
		reqPayload.NeedRegistration,
		returnDetails,
		models.EventStatusPending,
		"",
		createdAt,
		updatedAt,
		nil,
	)

	insertQuery := regexp.QuoteMeta(`
		INSERT INTO events_pending (type, start_time, end_time, duration, place, price_type, need_registration, details)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, type, start_time, end_time, duration, place, price_type, need_registration, details, status, review_comment, created_at, updated_at, reviewed_at
	`)

	mock.ExpectQuery(insertQuery).
		WithArgs(
			reqPayload.Type,
			reqPayload.StartTime,
			reqPayload.EndTime,
			reqPayload.Duration,
			reqPayload.Place,
			reqPayload.PriceType,
			reqPayload.NeedRegistration,
			sqlmock.AnyArg(),
		).
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodPost, "/v1/api/events", bytes.NewReader(body))
	resp := httptest.NewRecorder()
	handler.CreateEvent(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.Code)
	}

	var pending models.PendingEvent
	if err := json.Unmarshal(resp.Body.Bytes(), &pending); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if pending.ID != "event-123" {
		t.Fatalf("expected event id event-123, got %s", pending.ID)
	}
	if pending.Status != models.EventStatusPending {
		t.Fatalf("expected status pending, got %s", pending.Status)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestEventHandler_GetPendingEvents_ReturnsList(t *testing.T) {
	handler, mock, cleanup := setupEventHandlerTest(t)
	defer cleanup()

	createdAt := time.Now().UTC()
	reviewedAt := createdAt.Add(time.Hour)
	rows := sqlmock.NewRows([]string{
		"id", "type", "start_time", "end_time", "duration", "place",
		"price_type", "need_registration", "details", "status", "review_comment",
		"created_at", "updated_at", "reviewed_at",
	}).AddRow(
		"pending-1",
		"meetup",
		createdAt,
		createdAt.Add(2*time.Hour),
		120,
		"Online",
		"free",
		true,
		[]byte(`{"lang":"en"}`),
		models.EventStatusPending,
		"waiting",
		createdAt,
		createdAt,
		reviewedAt,
	)

	selectQuery := regexp.QuoteMeta(`
		SELECT id, type, start_time, end_time, duration, place,
		       price_type, need_registration, details, status, review_comment,
		       created_at, updated_at, reviewed_at
		FROM events_pending
		ORDER BY created_at DESC
	`)

	mock.ExpectQuery(selectQuery).WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/v1/api/events/pending", nil)
	resp := httptest.NewRecorder()
	handler.GetPendingEvents(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var pending []*models.PendingEvent
	if err := json.Unmarshal(resp.Body.Bytes(), &pending); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending event, got %d", len(pending))
	}
	if pending[0].ID != "pending-1" {
		t.Fatalf("unexpected pending event id: %s", pending[0].ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestEventHandler_ReviewPendingEvent_Approve(t *testing.T) {
	handler, mock, cleanup := setupEventHandlerTest(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{"id"}).AddRow("pending-1")
	updateQuery := regexp.QuoteMeta(`
		UPDATE events_pending
		SET status = $2,
		    review_comment = NULLIF($3, ''),
		    reviewed_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1 AND status = 'pending'
		RETURNING id
	`)

	mock.ExpectQuery(updateQuery).
		WithArgs("pending-1", models.EventStatusApproved, "").
		WillReturnRows(rows)

	reqBody := models.ReviewEventRequest{Action: "approve"}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal review body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/api/events/pending-1/review", bytes.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "pending-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	resp := httptest.NewRecorder()

	handler.ReviewPendingEvent(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var payload map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload["status"] != models.EventStatusApproved {
		t.Fatalf("expected approved status, got %s", payload["status"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestEventHandler_ReviewPendingEvent_RejectRequiresComment(t *testing.T) {
	handler, _, cleanup := setupEventHandlerTest(t)
	defer cleanup()

	reqBody := models.ReviewEventRequest{Action: "reject"}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/api/events/pending-1/review", bytes.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "pending-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	resp := httptest.NewRecorder()

	handler.ReviewPendingEvent(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}
