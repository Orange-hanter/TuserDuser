package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"go.uber.org/zap"
)

func TestTelegramRegistrationAndSend(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO telegram_binding_tokens")).
		WithArgs(sqlmock.AnyArg(), "user-123", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta("DELETE FROM telegram_binding_tokens")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow("user-123"))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO telegram_bindings")).
		WithArgs(
			sqlmock.AnyArg(), // user_id
			sqlmock.AnyArg(), // chat_id
			sqlmock.AnyArg(), // status
			sqlmock.AnyArg(), // blocked_reason
			sqlmock.AnyArg(), // last_error_code
			sqlmock.AnyArg(), // last_error_at
			sqlmock.AnyArg(), // telegram_username
			sqlmock.AnyArg(), // telegram_first_name
			sqlmock.AnyArg(), // telegram_last_name
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	settings := Settings{
		BotUsername:       "BrestEvents_bot",
		BindingSecret:     "test-secret",
		BindingTTLSeconds: 600,
	}
	store := NewStore(db)
	svc := NewService(store, settings, zap.NewNop())

	link, err := svc.IssueBindingLink(context.Background(), "user-123")
	if err != nil {
		t.Fatalf("issue binding link: %v", err)
	}

	chat := ChatMetadata{ChatID: 42, Username: "testuser"}
	binding, err := svc.HandleStartCommand(context.Background(), link.Token, chat)
	if err != nil {
		t.Fatalf("handle /start: %v", err)
	}
	if binding.UserID != "user-123" || binding.ChatID != chat.ChatID {
		t.Fatalf("binding mismatch: %+v", binding)
	}

	var payload map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode telegram request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":     true,
			"result": map[string]any{"message_id": 99},
		})
	}))
	defer ts.Close()

	client := NewHTTPClient("token", ts.URL)
	result, err := client.SendMessage(context.Background(), OutboundMessage{ChatID: chat.ChatID, Text: "Hello from system"})
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	if result.MessageID != "99" {
		t.Fatalf("unexpected message id: %s", result.MessageID)
	}
	if payload == nil {
		t.Fatalf("no payload captured")
	}
	chatID, ok := payload["chat_id"].(float64)
	if !ok || int64(chatID) != chat.ChatID {
		t.Fatalf("unexpected chat_id in payload: %+v", payload)
	}
	if payload["text"] != "Hello from system" {
		t.Fatalf("unexpected text payload: %+v", payload)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
