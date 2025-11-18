package telegram

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Service orchestrates binding tokens and webhook side effects.
type Service struct {
	store    *Store
	encoder  *TokenEncoder
	settings Settings
	logger   *zap.Logger
}

// ChatMetadata captures Telegram chat information needed for binding updates.
type ChatMetadata struct {
	ChatID    int64
	Username  string
	FirstName string
	LastName  string
}

// NewService builds a Service instance.
func NewService(store *Store, settings Settings, logger *zap.Logger) *Service {
	encoder := NewTokenEncoder(settings.BindingSecret, settings.BindingTTLSeconds)
	return &Service{
		store:    store,
		encoder:  encoder,
		settings: settings,
		logger:   logger,
	}
}

// IssueBindingLink mints a single-use binding token and deep link.
func (s *Service) IssueBindingLink(ctx context.Context, userID string) (*BindingLink, error) {
	if s.settings.BotUsername == "" {
		return nil, errors.New("telegram bot username not configured")
	}
	token, nonce, expiresAt, err := s.encoder.Mint(userID)
	if err != nil {
		return nil, err
	}
	if err := s.store.SaveBindingToken(ctx, HashNonce(nonce), userID, expiresAt); err != nil {
		return nil, err
	}
	deepLink := fmt.Sprintf("https://t.me/%s?start=%s", s.settings.BotUsername, token)
	return &BindingLink{Token: token, DeepLink: deepLink, ExpiresAt: expiresAt}, nil
}

// CurrentBinding fetches binding by user id.
func (s *Service) CurrentBinding(ctx context.Context, userID string) (*Binding, error) {
	return s.store.GetBindingByUserID(ctx, userID)
}

// HandleStartCommand processes /start <token> binding.
func (s *Service) HandleStartCommand(ctx context.Context, token string, chat ChatMetadata) (*Binding, error) {
	userID, nonce, _, err := s.encoder.Parse(token)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if _, err := s.store.ConsumeBindingToken(ctx, HashNonce(nonce)); err != nil {
		return nil, err
	}
	binding := Binding{
		UserID:    userID,
		ChatID:    chat.ChatID,
		Status:    BindingStatusActive,
		Username:  chat.Username,
		FirstName: chat.FirstName,
		LastName:  chat.LastName,
	}
	if err := s.store.UpsertBinding(ctx, binding); err != nil {
		return nil, err
	}
	s.logger.Info("telegram binding activated", zap.String("user_id", userID), zap.Int64("chat_id", chat.ChatID))
	return &binding, nil
}

// HandleUnsubscribe processes /unsubscribe command.
func (s *Service) HandleUnsubscribe(ctx context.Context, chatID int64) (*Binding, error) {
	binding, err := s.store.GetBindingByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	reason := "user requested via /unsubscribe"
	if err := s.store.SetBindingStatus(ctx, binding.UserID, BindingStatusRevoked, &reason, nil); err != nil {
		return nil, err
	}
	binding.Status = BindingStatusRevoked
	binding.BlockedReason = &reason
	binding.UpdatedAt = time.Now()
	return binding, nil
}

// BindingStatusForChat returns binding metadata for /status command.
func (s *Service) BindingStatusForChat(ctx context.Context, chatID int64) (*Binding, error) {
	return s.store.GetBindingByChatID(ctx, chatID)
}

// BlockBinding updates status to blocked along with error metadata.
func (s *Service) BlockBinding(ctx context.Context, userID string, reason string, code int) error {
	return s.store.SetBindingStatus(ctx, userID, BindingStatusBlocked, &reason, &code)
}

// ActivateBinding marks binding active (used after transient failures).
func (s *Service) ActivateBinding(ctx context.Context, userID string) error {
	return s.store.SetBindingStatus(ctx, userID, BindingStatusActive, nil, nil)
}
