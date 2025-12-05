package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"event-api/internal/logger"
	"event-api/internal/models"
	"event-api/internal/telegramclient"

	"go.uber.org/zap"
)

// Use existing mockAuthService from auth_test.go which provides function hooks.

// mockTelegramClient implements TelegramClient for testing.
type mockTelegramClient struct{}

func (m *mockTelegramClient) RegisterPendingVerification(ctx context.Context, userID, verificationCode string, ttlMinutes int32) (*telegramclient.PendingVerificationResult, error) {
	return &telegramclient.PendingVerificationResult{
		DeepLink:  "https://t.me/TestBot?start=tok",
		Token:     "tok",
		Code:      "A1B2C3",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}, nil
}

func (m *mockTelegramClient) GetBindingStatus(ctx context.Context, userID string) (*telegramclient.BindingStatus, error) {
	return &telegramclient.BindingStatus{IsBound: false, Status: "not_bound"}, nil
}

// testAuthService embeds the existing mockAuthService and implements missing methods
type testAuthService struct{ *mockAuthService }

func (t *testAuthService) CheckUserExists(email, phone string) (bool, bool, error) {
	return false, false, nil
}
func (t *testAuthService) ResendCode(email, verificationType string) (string, int, error) {
	return "", 0, nil
}

func TestRegister_ReturnsTelegramBinding(t *testing.T) {
	logger.Log = zap.NewNop()
	// create a test auth service that embeds the existing mock and implements missing methods
	baseMock := &mockAuthService{}
	authSvc := &testAuthService{mockAuthService: baseMock}
	// provide default register behavior
	authSvc.registerFn = func(req *models.RegisterRequest) (*models.User, string, error) {
		user := &models.User{ID: "usr_test_1", Email: req.Email, Phone: req.Phone}
		return user, "123456", nil
	}
	tgClient := &mockTelegramClient{}

	handler := NewAuthHandler(authSvc, nil)
	// inject mock via interface field
	handler.telegramClient = tgClient

	body := map[string]interface{}{
		"email":             "test+tg@example.com",
		"password":          "password123",
		"phone":             "+70000000000",
		"verification_type": "telegram",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/api/auth/register", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Register(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var resp models.RegisterResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.TelegramBinding == nil {
		t.Fatalf("expected telegram_binding in response, got nil; body: %s", rr.Body.String())
	}
	if resp.TelegramBinding.Deeplink == "" || resp.TelegramBinding.Code == "" {
		t.Fatalf("telegram_binding missing fields: %+v", resp.TelegramBinding)
	}
}
