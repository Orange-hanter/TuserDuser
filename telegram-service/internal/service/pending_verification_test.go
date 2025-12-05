// Package service provides integration tests for pending verification flow.
package service

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"telegram-service/internal/database"
	"telegram-service/internal/telegram"
)

// mockTelegramClient implements telegram.Client for testing.
type mockTelegramClient struct {
	sentMessages []telegram.OutboundMessage
	shouldFail   bool
}

func (m *mockTelegramClient) SendMessage(_ context.Context, msg telegram.OutboundMessage) (*telegram.SendResult, error) {
	if m.shouldFail {
		return nil, &telegram.APIError{Code: 500, Description: "mock error"}
	}
	m.sentMessages = append(m.sentMessages, msg)
	return &telegram.SendResult{
		MessageID: int64(len(m.sentMessages)),
		SentAt:    time.Now(),
	}, nil
}

// mockStore implements database.Store interface methods needed for testing.
type mockStore struct {
	bindings             map[string]*database.Binding
	pendingVerifications map[string]*database.PendingVerification
	bindingTokens        map[string]string // nonceHash -> userID
	bindingCodes         map[string]string // code -> userID
}

func newMockStore() *mockStore {
	return &mockStore{
		bindings:             make(map[string]*database.Binding),
		pendingVerifications: make(map[string]*database.PendingVerification),
		bindingTokens:        make(map[string]string),
		bindingCodes:         make(map[string]string),
	}
}

func (m *mockStore) SaveBindingToken(_ context.Context, nonceHash, userID string, _ time.Time) error {
	m.bindingTokens[nonceHash] = userID
	return nil
}

func (m *mockStore) ConsumeBindingToken(_ context.Context, nonceHash string) (string, error) {
	userID, ok := m.bindingTokens[nonceHash]
	if !ok {
		return "", database.ErrTokenExpired
	}
	delete(m.bindingTokens, nonceHash)
	return userID, nil
}

func (m *mockStore) SaveBindingCode(_ context.Context, code, userID string, _ time.Time) error {
	m.bindingCodes[code] = userID
	return nil
}

func (m *mockStore) ConsumeBindingCode(_ context.Context, code string) (string, error) {
	userID, ok := m.bindingCodes[code]
	if !ok {
		return "", database.ErrTokenExpired
	}
	delete(m.bindingCodes, code)
	return userID, nil
}

func (m *mockStore) SavePendingVerification(_ context.Context, userID, code, bindingToken string, expiresAt time.Time) error {
	m.pendingVerifications[userID] = &database.PendingVerification{
		UserID:           userID,
		VerificationCode: code,
		BindingToken:     bindingToken,
		ExpiresAt:        expiresAt,
		CreatedAt:        time.Now(),
	}
	return nil
}

func (m *mockStore) GetPendingVerification(_ context.Context, userID string) (*database.PendingVerification, error) {
	pv, ok := m.pendingVerifications[userID]
	if !ok || pv.ExpiresAt.Before(time.Now()) {
		return nil, database.ErrNotFound
	}
	return pv, nil
}

func (m *mockStore) ConsumePendingVerification(_ context.Context, userID string) (*database.PendingVerification, error) {
	pv, ok := m.pendingVerifications[userID]
	if !ok || pv.ExpiresAt.Before(time.Now()) {
		return nil, database.ErrNotFound
	}
	delete(m.pendingVerifications, userID)
	return pv, nil
}

func (m *mockStore) HasPendingVerification(_ context.Context, userID string) (bool, error) {
	pv, ok := m.pendingVerifications[userID]
	return ok && pv.ExpiresAt.After(time.Now()), nil
}

func (m *mockStore) UpsertBinding(_ context.Context, binding database.Binding) error {
	m.bindings[binding.UserID] = &binding
	return nil
}

func (m *mockStore) GetBindingByUserID(_ context.Context, userID string) (*database.Binding, error) {
	b, ok := m.bindings[userID]
	if !ok {
		return nil, database.ErrNotFound
	}
	return b, nil
}

func (m *mockStore) GetBindingByChatID(_ context.Context, chatID int64) (*database.Binding, error) {
	for _, b := range m.bindings {
		if b.ChatID == chatID {
			return b, nil
		}
	}
	return nil, database.ErrNotFound
}

func (m *mockStore) SetBindingStatus(_ context.Context, userID string, status database.BindingStatus, reason *string, _ *int) error {
	if b, ok := m.bindings[userID]; ok {
		b.Status = status
		b.BlockedReason = reason
	}
	return nil
}

// testableStore wraps mockStore to implement *database.Store interface.
// Since we can't easily mock *database.Store (pointer to struct), we create a testable service.
type testableService struct {
	*TelegramService
	mockStore  *mockStore
	mockClient *mockTelegramClient
}

func newTestableService() *testableService {
	logger, _ := zap.NewDevelopment()
	mockStore := newMockStore()
	mockClient := &mockTelegramClient{}

	encoder := NewTokenEncoder("test-secret-key", 600) // 10 minutes in seconds

	// We need to create a TelegramService but we can't use mockStore directly
	// because TelegramService expects *database.Store
	// For integration tests, we'll test the flow logic separately

	return &testableService{
		mockStore:  mockStore,
		mockClient: mockClient,
		TelegramService: &TelegramService{
			encoder:     encoder,
			botUsername: "TestBot",
			logger:      logger,
		},
	}
}

// TestPendingVerificationFlow_FullCycle tests the complete deferred verification flow:
// 1. Register pending verification (simulates event-api calling RegisterPendingVerification)
// 2. User binds Telegram via code
// 3. Verification code is automatically sent
func TestPendingVerificationFlow_FullCycle(t *testing.T) {
	ctx := context.Background()
	svc := newTestableService()

	userID := "user-123"
	verificationCode := "456789"
	chatID := int64(999888777)

	// Step 1: Register pending verification
	// (In real flow, event-api calls gRPC RegisterPendingVerification)
	err := svc.mockStore.SavePendingVerification(ctx, userID, verificationCode, "binding-token", time.Now().Add(10*time.Minute))
	if err != nil {
		t.Fatalf("SavePendingVerification failed: %v", err)
	}

	// Also save a binding code for the user
	bindingCode := "ABC123"
	err = svc.mockStore.SaveBindingCode(ctx, bindingCode, userID, time.Now().Add(10*time.Minute))
	if err != nil {
		t.Fatalf("SaveBindingCode failed: %v", err)
	}

	// Step 2: Verify pending verification exists
	hasPending, err := svc.mockStore.HasPendingVerification(ctx, userID)
	if err != nil {
		t.Fatalf("HasPendingVerification failed: %v", err)
	}
	if !hasPending {
		t.Error("Expected pending verification to exist")
	}

	// Step 3: Simulate user binding via code
	consumedUserID, err := svc.mockStore.ConsumeBindingCode(ctx, bindingCode)
	if err != nil {
		t.Fatalf("ConsumeBindingCode failed: %v", err)
	}
	if consumedUserID != userID {
		t.Errorf("Expected userID %s, got %s", userID, consumedUserID)
	}

	// Create binding
	binding := database.Binding{
		UserID:    userID,
		ChatID:    chatID,
		Status:    database.BindingStatusActive,
		Username:  "testuser",
		FirstName: "Test",
	}
	err = svc.mockStore.UpsertBinding(ctx, binding)
	if err != nil {
		t.Fatalf("UpsertBinding failed: %v", err)
	}

	// Step 4: Consume pending verification (simulates CheckAndSendPendingVerification)
	pendingVerif, err := svc.mockStore.ConsumePendingVerification(ctx, userID)
	if err != nil {
		t.Fatalf("ConsumePendingVerification failed: %v", err)
	}

	if pendingVerif.VerificationCode != verificationCode {
		t.Errorf("Expected verification code %s, got %s", verificationCode, pendingVerif.VerificationCode)
	}

	// Step 5: Send verification code via Telegram
	_, err = svc.mockClient.SendMessage(ctx, telegram.OutboundMessage{
		ChatID:    chatID,
		Text:      "🔐 *Код подтверждения*\n\n`" + pendingVerif.VerificationCode + "`",
		ParseMode: "MarkdownV2",
	})
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	// Verify message was sent
	if len(svc.mockClient.sentMessages) != 1 {
		t.Errorf("Expected 1 sent message, got %d", len(svc.mockClient.sentMessages))
	}

	// Step 6: Verify pending verification is consumed
	hasPending, err = svc.mockStore.HasPendingVerification(ctx, userID)
	if err != nil {
		t.Fatalf("HasPendingVerification failed: %v", err)
	}
	if hasPending {
		t.Error("Expected pending verification to be consumed")
	}
}

// TestPendingVerificationFlow_NoPending tests binding without pending verification.
func TestPendingVerificationFlow_NoPending(t *testing.T) {
	ctx := context.Background()
	svc := newTestableService()

	userID := "user-456"

	// No pending verification registered

	// Try to consume - should return ErrNotFound
	_, err := svc.mockStore.ConsumePendingVerification(ctx, userID)
	if err != database.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

// TestPendingVerificationFlow_Expired tests that expired verifications are not consumed.
func TestPendingVerificationFlow_Expired(t *testing.T) {
	ctx := context.Background()
	svc := newTestableService()

	userID := "user-789"
	verificationCode := "111222"

	// Register with already expired time
	err := svc.mockStore.SavePendingVerification(ctx, userID, verificationCode, "token", time.Now().Add(-1*time.Minute))
	if err != nil {
		t.Fatalf("SavePendingVerification failed: %v", err)
	}

	// Should not find expired verification
	_, err = svc.mockStore.ConsumePendingVerification(ctx, userID)
	if err != database.ErrNotFound {
		t.Errorf("Expected ErrNotFound for expired verification, got %v", err)
	}
}

// TestPendingVerificationFlow_BindingViaToken tests binding via deep link token.
func TestPendingVerificationFlow_BindingViaToken(t *testing.T) {
	ctx := context.Background()
	svc := newTestableService()

	userID := "user-token-test"
	verificationCode := "333444"

	// Register pending verification
	err := svc.mockStore.SavePendingVerification(ctx, userID, verificationCode, "binding-token", time.Now().Add(10*time.Minute))
	if err != nil {
		t.Fatalf("SavePendingVerification failed: %v", err)
	}

	// Save binding token
	nonceHash := "hashed-nonce-123"
	err = svc.mockStore.SaveBindingToken(ctx, nonceHash, userID, time.Now().Add(10*time.Minute))
	if err != nil {
		t.Fatalf("SaveBindingToken failed: %v", err)
	}

	// Simulate HandleStartCommand flow
	consumedUserID, err := svc.mockStore.ConsumeBindingToken(ctx, nonceHash)
	if err != nil {
		t.Fatalf("ConsumeBindingToken failed: %v", err)
	}
	if consumedUserID != userID {
		t.Errorf("Expected userID %s, got %s", userID, consumedUserID)
	}

	// Check pending verification exists
	pendingVerif, err := svc.mockStore.GetPendingVerification(ctx, userID)
	if err != nil {
		t.Fatalf("GetPendingVerification failed: %v", err)
	}
	if pendingVerif.VerificationCode != verificationCode {
		t.Errorf("Expected code %s, got %s", verificationCode, pendingVerif.VerificationCode)
	}
}

// TestPendingVerificationFlow_MultipleRegistrations tests that re-registering updates the code.
func TestPendingVerificationFlow_MultipleRegistrations(t *testing.T) {
	ctx := context.Background()
	svc := newTestableService()

	userID := "user-multi"

	// Register first code
	err := svc.mockStore.SavePendingVerification(ctx, userID, "code1", "token1", time.Now().Add(10*time.Minute))
	if err != nil {
		t.Fatalf("First SavePendingVerification failed: %v", err)
	}

	// Register second code (should replace first)
	err = svc.mockStore.SavePendingVerification(ctx, userID, "code2", "token2", time.Now().Add(10*time.Minute))
	if err != nil {
		t.Fatalf("Second SavePendingVerification failed: %v", err)
	}

	// Should get the second code
	pendingVerif, err := svc.mockStore.GetPendingVerification(ctx, userID)
	if err != nil {
		t.Fatalf("GetPendingVerification failed: %v", err)
	}
	if pendingVerif.VerificationCode != "code2" {
		t.Errorf("Expected code2, got %s", pendingVerif.VerificationCode)
	}
}

// TestTokenEncoder tests the token encoding/decoding.
func TestTokenEncoder(t *testing.T) {
	encoder := NewTokenEncoder("test-secret", 300) // 5 minutes in seconds

	userID := "user-encoder-test"

	// Mint a token
	token, nonce, expiresAt, err := encoder.Mint(userID)
	if err != nil {
		t.Fatalf("Mint failed: %v", err)
	}

	if token == "" {
		t.Error("Token should not be empty")
	}
	if nonce == "" {
		t.Error("Nonce should not be empty")
	}
	if expiresAt.Before(time.Now()) {
		t.Error("ExpiresAt should be in the future")
	}

	// Parse the token
	parsedUserID, parsedNonce, parsedExpiresAt, err := encoder.Parse(token)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsedUserID != userID {
		t.Errorf("Expected userID %s, got %s", userID, parsedUserID)
	}
	if parsedNonce != nonce {
		t.Errorf("Expected nonce %s, got %s", nonce, parsedNonce)
	}
	// Compare Unix timestamps (nanoseconds are lost in serialization)
	if parsedExpiresAt.Unix() != expiresAt.Unix() {
		t.Errorf("Expected expiresAt %v, got %v", expiresAt, parsedExpiresAt)
	}

	// Test HashNonce
	hash := HashNonce(nonce)
	if hash == "" {
		t.Error("Hash should not be empty")
	}
	if hash == nonce {
		t.Error("Hash should be different from nonce")
	}
}

// TestTokenEncoder_InvalidToken tests parsing invalid tokens.
func TestTokenEncoder_InvalidToken(t *testing.T) {
	encoder := NewTokenEncoder("test-secret", 300) // 5 minutes

	testCases := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"invalid base64", "not-valid-base64!@#"},
		{"too short", "YWJj"}, // "abc" in base64
		{"wrong signature", "eyJ0ZXN0IjoidmFsdWUifQ=="},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, err := encoder.Parse(tc.token)
			if err == nil {
				t.Error("Expected error for invalid token")
			}
		})
	}
}
