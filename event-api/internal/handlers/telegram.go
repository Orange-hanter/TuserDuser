package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"event-api/internal/telegram"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// TelegramHandler provides HTTP endpoints for Telegram integration.
type TelegramHandler struct {
	service  *telegram.Service
	store    *telegram.Store
	settings telegram.Settings
	client   telegram.Client
	logger   *zap.Logger
}

// NewTelegramHandler constructs a handler.
func NewTelegramHandler(service *telegram.Service, store *telegram.Store, settings telegram.Settings, client telegram.Client, logger *zap.Logger) *TelegramHandler {
	return &TelegramHandler{
		service:  service,
		store:    store,
		settings: settings,
		client:   client,
		logger:   logger,
	}
}

// IssueLink handles POST /api/notifications/telegram/link.
func (h *TelegramHandler) IssueLink(w http.ResponseWriter, r *http.Request) {
	if !h.settings.Enabled {
		respondWithError(w, http.StatusNotFound, "telegram_disabled", "Telegram notifications disabled")
		return
	}
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "user id missing")
		return
	}
	link, err := h.service.IssueBindingLink(r.Context(), userID)
	if err != nil {
		h.logger.Error("issue telegram link", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "telegram_error", "failed to issue link")
		return
	}
	respondWithJSON(w, http.StatusOK, link)
}

// @Summary Issue Telegram binding deep link
// @Description Issues a short-lived deep-link token that the client app can open to bind a Telegram chat to the current user. Requires authentication (X-User-ID / Bearer token via middleware).
// @Tags notifications, telegram
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} telegram.BindingLink "Returns the token and deeplink"
// @Failure 401 {object} map[string]interface{} "Unauthorized or missing user id"
// @Failure 404 {object} map[string]interface{} "Telegram integration disabled"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /v1/api/notifications/telegram/link [post]

// BindingStatus returns binding info for authenticated user.
func (h *TelegramHandler) BindingStatus(w http.ResponseWriter, r *http.Request) {
	if !h.settings.Enabled {
		respondWithError(w, http.StatusNotFound, "telegram_disabled", "Telegram notifications disabled")
		return
	}
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "user id missing")
		return
	}
	binding, err := h.service.CurrentBinding(r.Context(), userID)
	if err != nil {
		if err == telegram.ErrBindingNotFound {
			respondWithError(w, http.StatusNotFound, "telegram_not_bound", "Telegram chat not linked")
			return
		}
		h.logger.Error("get telegram binding", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "telegram_error", "failed to fetch binding")
		return
	}
	resp := map[string]any{
		"status":     binding.Status,
		"chat_id":    binding.ChatID,
		"username":   binding.Username,
		"updated_at": binding.UpdatedAt,
	}
	respondWithJSON(w, http.StatusOK, resp)
}

// @Summary Get Telegram binding status
// @Description Returns the current Telegram binding status for the authenticated user (status, chat_id, username, updated_at).
// @Tags notifications, telegram
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{} "status, chat_id, username, updated_at"
// @Failure 401 {object} map[string]interface{} "Unauthorized or missing user id"
// @Failure 404 {object} map[string]interface{} "Telegram not bound or integration disabled"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /v1/api/notifications/telegram/status [get]

// Webhook handles Telegram webhook callbacks.
func (h *TelegramHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	if !h.settings.Enabled {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	alias := chi.URLParam(r, "botAlias")
	if alias == "" || alias != h.settings.WebhookAlias {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if secret := r.Header.Get("X-Telegram-Bot-Api-Secret-Token"); h.settings.WebhookSecret != "" && secret != h.settings.WebhookSecret {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("read telegram webhook", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var update telegramUpdate
	if err := json.Unmarshal(body, &update); err != nil {
		h.logger.Error("decode telegram webhook", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	_ = h.store.RecordWebhookEvent(r.Context(), telegram.WebhookEvent{
		BotAlias:   alias,
		UpdateID:   update.UpdateID,
		Payload:    json.RawMessage(body),
		ReceivedAt: time.Now(),
	})

	if update.Message != nil {
		h.handleMessage(r.Context(), update.Message)
	}

	w.WriteHeader(http.StatusOK)
}

// @Summary Telegram webhook endpoint
// @Description Receives updates from Telegram Bot API for the configured bot alias. Validates optional secret header `X-Telegram-Bot-Api-Secret-Token` when configured. Persists raw payload for auditing and triggers handling of `/start`, `/unsubscribe`, `/status` commands.
// @Tags webhooks, telegram
// @Accept json
// @Produce json
// @Param botAlias path string true "Bot alias (path segment)"
// @Param X-Telegram-Bot-Api-Secret-Token header string false "Secret token for webhook verification"
// @Success 200 {string} string "ok"
// @Failure 400 {object} map[string]interface{} "Bad request / invalid payload"
// @Failure 403 {object} map[string]interface{} "Invalid secret token"
// @Router /webhooks/telegram/{botAlias} [post]

func (h *TelegramHandler) handleMessage(ctx context.Context, msg *telegramMessage) {
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return
	}
	command, arg := parseCommand(text)
	switch command {
	case "/start":
		if arg == "" {
			h.reply(ctx, msg.Chat.ID, "⚠️ Токен не найден. Откройте ссылку из приложения снова.", true)
			return
		}
		binding, err := h.service.HandleStartCommand(ctx, arg, telegram.ChatMetadata{
			ChatID:    msg.Chat.ID,
			Username:  msg.Chat.Username,
			FirstName: msg.Chat.FirstName,
			LastName:  msg.Chat.LastName,
		})
		if err != nil {
			h.reply(ctx, msg.Chat.ID, "❌ Не удалось привязать Telegram. Ссылка устарела.", true)
			h.logger.Warn("telegram start failed", zap.Error(err))
			return
		}
		h.reply(ctx, msg.Chat.ID, "✅ Уведомления подключены. Вы всегда можете отправить /unsubscribe чтобы остановить их.", false)
		h.logger.Info("telegram binding confirmed", zap.String("user_id", binding.UserID))
	case "/unsubscribe":
		binding, err := h.service.HandleUnsubscribe(ctx, msg.Chat.ID)
		if err != nil {
			h.reply(ctx, msg.Chat.ID, "❌ Не удалось отключить уведомления.", true)
			h.logger.Warn("unsubscribe failed", zap.Error(err))
			return
		}
		h.reply(ctx, msg.Chat.ID, "🛑 Вы больше не будете получать напоминания. Чтобы повторно включить, вернитесь в приложение.", false)
		h.logger.Info("telegram binding revoked", zap.String("user_id", binding.UserID))
	case "/status":
		binding, err := h.service.BindingStatusForChat(ctx, msg.Chat.ID)
		if err != nil {
			h.reply(ctx, msg.Chat.ID, "ℹ️ Этот чат ещё не привязан. Используйте ссылку из приложения.", false)
			return
		}
		statusMsg := fmt.Sprintf("Текущее состояние: %s", binding.Status)
		h.reply(ctx, msg.Chat.ID, statusMsg, false)
	default:
		// ignore other commands
	}
}

func (h *TelegramHandler) reply(ctx context.Context, chatID int64, text string, silent bool) {
	if h.client == nil {
		return
	}
	_, err := h.client.SendMessage(ctx, telegram.OutboundMessage{
		ChatID: chatID,
		Text:   text,
		Silent: silent,
	})
	if err != nil {
		h.logger.Warn("telegram reply failed", zap.Error(err))
	}
}

func parseCommand(text string) (string, string) {
	parts := strings.SplitN(text, " ", 2)
	cmd := strings.ToLower(strings.TrimSpace(parts[0]))
	var arg string
	if len(parts) > 1 {
		arg = strings.TrimSpace(parts[1])
	}
	return cmd, arg
}

type telegramUpdate struct {
	UpdateID int64            `json:"update_id"`
	Message  *telegramMessage `json:"message"`
}

type telegramMessage struct {
	MessageID int64        `json:"message_id"`
	Text      string       `json:"text"`
	Chat      telegramChat `json:"chat"`
}

type telegramChat struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}
