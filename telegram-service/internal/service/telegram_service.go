// Package service implements the core business logic for telegram-service.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"telegram-service/internal/database"
	"telegram-service/internal/metrics"
	"telegram-service/internal/telegram"
)

// Common errors
var (
	ErrUserNotBound  = errors.New("user_not_bound")
	ErrBlocked       = errors.New("blocked")
	ErrRateLimited   = errors.New("rate_limited")
	ErrSendFailed    = errors.New("send_failed")
	ErrInvalidToken  = errors.New("invalid_token")
	ErrInvalidUserID = errors.New("invalid_user_id")
)

// TelegramService provides the core Telegram integration functionality.
type TelegramService struct {
	store       *database.Store
	client      telegram.Client
	encoder     *TokenEncoder
	botUsername string
	logger      *zap.Logger
}

// NewTelegramService creates a new service instance.
func NewTelegramService(
	store *database.Store,
	client telegram.Client,
	encoder *TokenEncoder,
	botUsername string,
	logger *zap.Logger,
) *TelegramService {
	return &TelegramService{
		store:       store,
		client:      client,
		encoder:     encoder,
		botUsername: botUsername,
		logger:      logger,
	}
}

// ChatMetadata contains Telegram chat information for binding.
type ChatMetadata struct {
	ChatID    int64
	Username  string
	FirstName string
	LastName  string
}

// BindingLinkResult contains the generated binding link data.
type BindingLinkResult struct {
	DeepLink  string
	Token     string
	ExpiresAt time.Time
}

// GenerateBindingLink creates a new binding deep link for a user.
func (s *TelegramService) GenerateBindingLink(ctx context.Context, userID string) (*BindingLinkResult, error) {
	if userID == "" {
		return nil, ErrInvalidUserID
	}

	token, nonce, expiresAt, err := s.encoder.Mint(userID)
	if err != nil {
		s.logger.Error("failed to mint token", zap.Error(err))
		return nil, fmt.Errorf("mint token: %w", err)
	}

	if err := s.store.SaveBindingToken(ctx, HashNonce(nonce), userID, expiresAt); err != nil {
		s.logger.Error("failed to save binding token", zap.Error(err))
		return nil, fmt.Errorf("save token: %w", err)
	}

	deepLink := fmt.Sprintf("https://t.me/%s?start=%s", s.botUsername, token)

	metrics.BindingLinksGenerated.Inc()

	return &BindingLinkResult{
		DeepLink:  deepLink,
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

// HandleStartCommand processes /start <token> binding from Telegram webhook.
func (s *TelegramService) HandleStartCommand(ctx context.Context, token string, chat ChatMetadata) (*database.Binding, error) {
	userID, nonce, _, err := s.encoder.Parse(token)
	if err != nil {
		return nil, ErrInvalidToken
	}

	if _, err := s.store.ConsumeBindingToken(ctx, HashNonce(nonce)); err != nil {
		if errors.Is(err, database.ErrTokenExpired) {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("consume token: %w", err)
	}

	binding := database.Binding{
		UserID:    userID,
		ChatID:    chat.ChatID,
		Status:    database.BindingStatusActive,
		Username:  chat.Username,
		FirstName: chat.FirstName,
		LastName:  chat.LastName,
	}

	if err := s.store.UpsertBinding(ctx, binding); err != nil {
		return nil, fmt.Errorf("upsert binding: %w", err)
	}

	metrics.BindingsTotal.WithLabelValues("active").Inc()
	s.logger.Info("telegram binding activated",
		zap.String("user_id", userID),
		zap.Int64("chat_id", chat.ChatID),
	)

	return &binding, nil
}

// HandleUnsubscribe processes /unsubscribe command.
func (s *TelegramService) HandleUnsubscribe(ctx context.Context, chatID int64) (*database.Binding, error) {
	binding, err := s.store.GetBindingByChatID(ctx, chatID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrUserNotBound
		}
		return nil, err
	}

	reason := "user requested via /unsubscribe"
	if err := s.store.SetBindingStatus(ctx, binding.UserID, database.BindingStatusRevoked, &reason, nil); err != nil {
		return nil, err
	}

	binding.Status = database.BindingStatusRevoked
	binding.BlockedReason = &reason
	binding.UpdatedAt = time.Now()

	metrics.BindingsTotal.WithLabelValues("revoked").Inc()
	s.logger.Info("telegram binding revoked",
		zap.String("user_id", binding.UserID),
		zap.Int64("chat_id", chatID),
	)

	return binding, nil
}

// HandleIncomingMessage processes an incoming message from Telegram (used by polling mode).
// This is a unified entry point for processing messages from long polling.
func (s *TelegramService) HandleIncomingMessage(ctx context.Context, chatID, telegramUserID int64, username, firstName, lastName, text string) error {
	s.logger.Debug("incoming message",
		zap.Int64("chat_id", chatID),
		zap.Int64("telegram_user_id", telegramUserID),
		zap.String("username", username),
		zap.String("text", text),
	)

	// Handle /start command with binding token
	if len(text) > 7 && text[:7] == "/start " {
		token := text[7:]
		chat := ChatMetadata{
			ChatID:    chatID,
			Username:  username,
			FirstName: firstName,
			LastName:  lastName,
		}

		binding, err := s.HandleStartCommand(ctx, token, chat)
		if err != nil {
			// Send error message to user
			errorMsg := "❌ Ссылка недействительна или истекла. Запросите новую."
			if errors.Is(err, ErrInvalidToken) {
				s.sendSimpleMessage(ctx, chatID, errorMsg)
			}
			return err
		}

		// Send success message
		successMsg := fmt.Sprintf("✅ Telegram успешно привязан!\n\nВы будете получать уведомления о событиях.\n\nДля отключения: /unsubscribe")
		s.sendSimpleMessage(ctx, chatID, successMsg)

		s.logger.Info("binding activated via polling",
			zap.String("user_id", binding.UserID),
			zap.Int64("chat_id", chatID),
		)
		return nil
	}

	// Handle /unsubscribe command
	if text == "/unsubscribe" {
		binding, err := s.HandleUnsubscribe(ctx, chatID)
		if err != nil {
			if errors.Is(err, ErrUserNotBound) {
				s.sendSimpleMessage(ctx, chatID, "⚠️ Вы не привязаны к системе.")
			}
			return err
		}

		s.sendSimpleMessage(ctx, chatID, "✅ Уведомления отключены.\n\nЧтобы включить снова, используйте ссылку привязки в приложении.")

		s.logger.Info("unsubscribed via polling",
			zap.String("user_id", binding.UserID),
			zap.Int64("chat_id", chatID),
		)
		return nil
	}

	// Handle /start without token
	if text == "/start" {
		msg := "👋 Привет!\n\nЭтот бот отправляет уведомления о событиях.\n\nДля привязки используйте специальную ссылку из приложения."
		s.sendSimpleMessage(ctx, chatID, msg)
		return nil
	}

	// Handle /help
	if text == "/help" {
		msg := "📖 *Доступные команды:*\n\n/start — начало работы\n/unsubscribe — отключить уведомления\n/help — эта справка"
		s.client.SendMessage(ctx, telegram.OutboundMessage{
			ChatID:    chatID,
			Text:      msg,
			ParseMode: "MarkdownV2",
		})
		return nil
	}

	// Unknown message - ignore or send help
	return nil
}

// sendSimpleMessage sends a plain text message without formatting.
func (s *TelegramService) sendSimpleMessage(ctx context.Context, chatID int64, text string) {
	_, err := s.client.SendMessage(ctx, telegram.OutboundMessage{
		ChatID: chatID,
		Text:   text,
	})
	if err != nil {
		s.logger.Warn("failed to send simple message",
			zap.Int64("chat_id", chatID),
			zap.Error(err),
		)
	}
}

// IsUserBound checks if a user has an active Telegram binding.
func (s *TelegramService) IsUserBound(ctx context.Context, userID string) (bool, string, error) {
	binding, err := s.store.GetBindingByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return false, "", nil
		}
		return false, "", err
	}
	return binding.Status == database.BindingStatusActive, string(binding.Status), nil
}

// GetBindingStatus returns detailed binding information for a user.
func (s *TelegramService) GetBindingStatus(ctx context.Context, userID string) (*database.Binding, error) {
	binding, err := s.store.GetBindingByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrUserNotBound
		}
		return nil, err
	}
	return binding, nil
}

// UnbindUser removes the Telegram binding for a user.
func (s *TelegramService) UnbindUser(ctx context.Context, userID string, reason string) error {
	binding, err := s.store.GetBindingByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return ErrUserNotBound
		}
		return err
	}

	if reason == "" {
		reason = "admin action"
	}
	if err := s.store.SetBindingStatus(ctx, userID, database.BindingStatusRevoked, &reason, nil); err != nil {
		return err
	}

	metrics.BindingsTotal.WithLabelValues("revoked").Inc()
	s.logger.Info("telegram binding unbound",
		zap.String("user_id", binding.UserID),
		zap.String("reason", reason),
	)

	return nil
}

// SendVerificationCode sends a verification code to a bound user.
func (s *TelegramService) SendVerificationCode(ctx context.Context, userID, code string, expiresInMinutes int32) (*telegram.SendResult, error) {
	binding, err := s.getActiveBinding(ctx, userID)
	if err != nil {
		return nil, err
	}

	text := fmt.Sprintf("🔐 *Код подтверждения*\n\n`%s`\n\n", code)
	if expiresInMinutes > 0 {
		text += fmt.Sprintf("Код действителен %d мин.", expiresInMinutes)
	}

	return s.sendMessage(ctx, userID, binding.ChatID, telegram.OutboundMessage{
		ChatID:    binding.ChatID,
		Text:      text,
		ParseMode: "MarkdownV2",
	})
}

// SendEventReminder sends an event reminder to a bound user.
func (s *TelegramService) SendEventReminder(ctx context.Context, userID, eventID, title, description string, startTime time.Time, location, deeplinkURL string) (*telegram.SendResult, error) {
	binding, err := s.getActiveBinding(ctx, userID)
	if err != nil {
		return nil, err
	}

	text := fmt.Sprintf("📅 *Напоминание о событии*\n\n*%s*\n", escapeMarkdownV2(title))
	if description != "" {
		text += fmt.Sprintf("\n%s\n", escapeMarkdownV2(description))
	}
	text += fmt.Sprintf("\n🕐 %s", startTime.Format("02.01.2006 15:04"))
	if location != "" {
		text += fmt.Sprintf("\n📍 %s", escapeMarkdownV2(location))
	}

	msg := telegram.OutboundMessage{
		ChatID:    binding.ChatID,
		Text:      text,
		ParseMode: "MarkdownV2",
	}

	if deeplinkURL != "" {
		msg.ReplyMarkup = []byte(fmt.Sprintf(`{"inline_keyboard":[[{"text":"Открыть событие","url":"%s"}]]}`, deeplinkURL))
	}

	return s.sendMessage(ctx, userID, binding.ChatID, msg)
}

// SendMessage sends a custom message to a bound user.
func (s *TelegramService) SendMessage(ctx context.Context, userID, text, parseMode string, silent bool, replyMarkup string) (*telegram.SendResult, error) {
	binding, err := s.getActiveBinding(ctx, userID)
	if err != nil {
		return nil, err
	}

	msg := telegram.OutboundMessage{
		ChatID:    binding.ChatID,
		Text:      text,
		ParseMode: parseMode,
		Silent:    silent,
	}
	if replyMarkup != "" {
		msg.ReplyMarkup = []byte(replyMarkup)
	}

	return s.sendMessage(ctx, userID, binding.ChatID, msg)
}

// getActiveBinding retrieves an active binding for a user.
func (s *TelegramService) getActiveBinding(ctx context.Context, userID string) (*database.Binding, error) {
	binding, err := s.store.GetBindingByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrUserNotBound
		}
		return nil, err
	}

	if binding.Status == database.BindingStatusBlocked {
		return nil, ErrBlocked
	}

	if binding.Status != database.BindingStatusActive {
		return nil, ErrUserNotBound
	}

	return binding, nil
}

// sendMessage sends a message and handles errors appropriately.
func (s *TelegramService) sendMessage(ctx context.Context, userID string, chatID int64, msg telegram.OutboundMessage) (*telegram.SendResult, error) {
	result, err := s.client.SendMessage(ctx, msg)
	if err != nil {
		var apiErr *telegram.APIError
		if errors.As(err, &apiErr) {
			// Handle blocked users (403)
			if apiErr.IsBlocked() {
				reason := "user blocked the bot"
				code := apiErr.Code
				_ = s.store.SetBindingStatus(ctx, userID, database.BindingStatusBlocked, &reason, &code)
				metrics.MessagesSent.WithLabelValues("failed", "blocked").Inc()
				s.logger.Warn("user blocked bot",
					zap.String("user_id", userID),
					zap.Int64("chat_id", chatID),
				)
				return nil, ErrBlocked
			}

			// Handle rate limiting (429)
			if apiErr.IsRateLimited() {
				metrics.MessagesSent.WithLabelValues("failed", "rate_limited").Inc()
				s.logger.Warn("rate limited by Telegram",
					zap.String("user_id", userID),
					zap.Duration("retry_after", apiErr.RetryAfter),
				)
				return nil, ErrRateLimited
			}

			metrics.MessagesSent.WithLabelValues("failed", "api_error").Inc()
			s.logger.Error("telegram api error",
				zap.String("user_id", userID),
				zap.Int("code", apiErr.Code),
				zap.String("description", apiErr.Description),
			)
		} else {
			metrics.MessagesSent.WithLabelValues("failed", "network").Inc()
			s.logger.Error("failed to send message",
				zap.String("user_id", userID),
				zap.Error(err),
			)
		}
		return nil, ErrSendFailed
	}

	metrics.MessagesSent.WithLabelValues("sent", "").Inc()
	return result, nil
}

// escapeMarkdownV2 escapes special characters for Telegram MarkdownV2.
func escapeMarkdownV2(text string) string {
	special := []byte{'_', '*', '[', ']', '(', ')', '~', '`', '>', '#', '+', '-', '=', '|', '{', '}', '.', '!'}
	result := make([]byte, 0, len(text)*2)
	for i := 0; i < len(text); i++ {
		for _, s := range special {
			if text[i] == s {
				result = append(result, '\\')
				break
			}
		}
		result = append(result, text[i])
	}
	return string(result)
}
