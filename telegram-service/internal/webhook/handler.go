// Package webhook handles Telegram webhook requests.
package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"telegram-service/internal/database"
	"telegram-service/internal/metrics"
	"telegram-service/internal/service"
)

// Handler processes Telegram webhook requests.
type Handler struct {
	service       *service.TelegramService
	store         *database.Store
	botToken      string
	webhookSecret string
	webhookAlias  string
	logger        *zap.Logger
	httpClient    *http.Client
}

// NewHandler creates a new webhook handler.
func NewHandler(svc *service.TelegramService, store *database.Store, botToken, secret, alias string, logger *zap.Logger) *Handler {
	return &Handler{
		service:       svc,
		store:         store,
		botToken:      botToken,
		webhookSecret: secret,
		webhookAlias:  alias,
		logger:        logger,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// telegramUpdate represents the Telegram update structure.
type telegramUpdate struct {
	UpdateID int64            `json:"update_id"`
	Message  *telegramMessage `json:"message,omitempty"`
}

type telegramMessage struct {
	MessageID int64         `json:"message_id"`
	From      *telegramUser `json:"from,omitempty"`
	Chat      telegramChat  `json:"chat"`
	Date      int64         `json:"date"`
	Text      string        `json:"text,omitempty"`
}

type telegramUser struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

type telegramChat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title,omitempty"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

// HandleWebhook processes incoming Telegram webhook requests.
func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Validate bot alias
	alias := chi.URLParam(r, "botAlias")
	if alias == "" || alias != h.webhookAlias {
		metrics.WebhookRequests.WithLabelValues("invalid_alias").Inc()
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Validate webhook secret
	if h.webhookSecret != "" {
		secret := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
		if secret != h.webhookSecret {
			metrics.WebhookRequests.WithLabelValues("invalid_secret").Inc()
			w.WriteHeader(http.StatusForbidden)
			return
		}
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		metrics.WebhookRequests.WithLabelValues("read_error").Inc()
		h.logger.Error("failed to read webhook body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Parse update
	var update telegramUpdate
	if err := json.Unmarshal(body, &update); err != nil {
		metrics.WebhookRequests.WithLabelValues("parse_error").Inc()
		h.logger.Error("failed to parse webhook", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Store webhook event for auditing
	_ = h.store.RecordWebhookEvent(r.Context(), database.WebhookEvent{
		BotAlias:   alias,
		UpdateID:   update.UpdateID,
		Payload:    json.RawMessage(body),
		ReceivedAt: time.Now(),
	})

	// Process message
	if update.Message != nil {
		h.handleMessage(r.Context(), update.Message)
	}

	metrics.WebhookRequests.WithLabelValues("success").Inc()
	w.WriteHeader(http.StatusOK)
}

// handleMessage processes incoming Telegram messages.
func (h *Handler) handleMessage(ctx context.Context, msg *telegramMessage) {
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return
	}

	command, arg := parseCommand(text)

	switch command {
	case "/start":
		h.handleStartCommand(ctx, msg, arg)
	case "/unsubscribe":
		h.handleUnsubscribeCommand(ctx, msg)
	case "/status":
		h.handleStatusCommand(ctx, msg)
	case "/help":
		h.handleHelpCommand(ctx, msg)
	default:
		// Ignore unknown commands/messages
	}
}

// handleStartCommand processes /start <token> binding.
func (h *Handler) handleStartCommand(ctx context.Context, msg *telegramMessage, token string) {
	if token == "" {
		h.sendReply(ctx, msg.Chat.ID, "⚠️ Токен не найден. Откройте ссылку из приложения снова.")
		return
	}

	binding, err := h.service.HandleStartCommand(ctx, token, service.ChatMetadata{
		ChatID:    msg.Chat.ID,
		Username:  msg.Chat.Username,
		FirstName: msg.Chat.FirstName,
		LastName:  msg.Chat.LastName,
	})
	if err != nil {
		h.sendReply(ctx, msg.Chat.ID, "❌ Не удалось привязать Telegram. Ссылка устарела или уже использована.")
		h.logger.Warn("telegram start failed",
			zap.Int64("chat_id", msg.Chat.ID),
			zap.Error(err),
		)
		return
	}

	h.sendReply(ctx, msg.Chat.ID, "✅ Уведомления подключены!\n\nВы можете отправить /unsubscribe чтобы остановить уведомления или /status чтобы проверить статус.")
	h.logger.Info("telegram binding confirmed",
		zap.String("user_id", binding.UserID),
		zap.Int64("chat_id", msg.Chat.ID),
	)
}

// handleUnsubscribeCommand processes /unsubscribe command.
func (h *Handler) handleUnsubscribeCommand(ctx context.Context, msg *telegramMessage) {
	binding, err := h.service.HandleUnsubscribe(ctx, msg.Chat.ID)
	if err != nil {
		if err == service.ErrUserNotBound {
			h.sendReply(ctx, msg.Chat.ID, "ℹ️ У вас нет активной привязки.")
			return
		}
		h.sendReply(ctx, msg.Chat.ID, "❌ Не удалось отключить уведомления.")
		h.logger.Warn("unsubscribe failed",
			zap.Int64("chat_id", msg.Chat.ID),
			zap.Error(err),
		)
		return
	}

	h.sendReply(ctx, msg.Chat.ID, "🛑 Вы больше не будете получать напоминания.\n\nЧтобы повторно включить, вернитесь в приложение и привяжите Telegram заново.")
	h.logger.Info("telegram binding revoked",
		zap.String("user_id", binding.UserID),
		zap.Int64("chat_id", msg.Chat.ID),
	)
}

// handleStatusCommand processes /status command.
func (h *Handler) handleStatusCommand(ctx context.Context, msg *telegramMessage) {
	binding, err := h.store.GetBindingByChatID(ctx, msg.Chat.ID)
	if err != nil {
		if err == database.ErrNotFound {
			h.sendReply(ctx, msg.Chat.ID, "ℹ️ У вас нет активной привязки к аккаунту.")
			return
		}
		h.sendReply(ctx, msg.Chat.ID, "❌ Не удалось получить статус.")
		return
	}

	var statusText string
	switch binding.Status {
	case database.BindingStatusActive:
		statusText = "✅ Активна"
	case database.BindingStatusBlocked:
		statusText = "🚫 Заблокирована"
	case database.BindingStatusRevoked:
		statusText = "🛑 Отключена"
	default:
		statusText = "⏳ В обработке"
	}

	reply := "📊 *Статус привязки*\n\n"
	reply += "Статус: " + statusText + "\n"
	reply += "Привязан: " + binding.CreatedAt.Format("02.01.2006 15:04") + "\n"

	h.sendReply(ctx, msg.Chat.ID, reply)
}

// handleHelpCommand processes /help command.
func (h *Handler) handleHelpCommand(ctx context.Context, msg *telegramMessage) {
	help := "🤖 *Доступные команды*\n\n"
	help += "/status — Проверить статус привязки\n"
	help += "/unsubscribe — Отключить уведомления\n"
	help += "/help — Показать эту справку\n"

	h.sendReply(ctx, msg.Chat.ID, help)
}

// sendReply sends a text reply to a chat using Telegram Bot API directly.
func (h *Handler) sendReply(ctx context.Context, chatID int64, text string) {
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		h.logger.Error("failed to marshal reply", zap.Error(err))
		return
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", h.botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		h.logger.Error("failed to create reply request", zap.Error(err))
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		h.logger.Error("failed to send reply", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		h.logger.Warn("telegram reply failed",
			zap.Int64("chat_id", chatID),
			zap.Int("status_code", resp.StatusCode),
		)
	}
}

// parseCommand extracts command and argument from message text.
func parseCommand(text string) (command, arg string) {
	parts := strings.SplitN(text, " ", 2)
	command = strings.ToLower(parts[0])

	// Handle commands with @botname suffix
	if idx := strings.Index(command, "@"); idx > 0 {
		command = command[:idx]
	}

	if len(parts) > 1 {
		arg = strings.TrimSpace(parts[1])
	}

	return command, arg
}
