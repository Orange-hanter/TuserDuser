package handlers

import (
	"errors"
	"net/http"

	"event-api/internal/telegramclient"

	"go.uber.org/zap"
)

// TelegramGRPCHandler provides HTTP endpoints that delegate to telegram-service via gRPC.
// This handler should be used when TELEGRAM_SERVICE_ENABLED=true.
type TelegramGRPCHandler struct {
	client *telegramclient.Client
	logger *zap.Logger
}

// NewTelegramGRPCHandler creates a new handler that uses the gRPC client.
func NewTelegramGRPCHandler(client *telegramclient.Client, logger *zap.Logger) *TelegramGRPCHandler {
	return &TelegramGRPCHandler{
		client: client,
		logger: logger,
	}
}

// IssueLink handles POST /api/notifications/telegram/link via gRPC.
// @Summary Issue Telegram binding link
// @Description Generates a short-lived token, deep link URL, and 6-character short code for binding
// @Description a user's Telegram account. The deep link format is https://t.me/BotName?start=TOKEN.
// @Description Users can either click the deep link or manually send the 6-character code to the bot.
// @Description Short codes are recommended for users who have previously interacted with the bot.
// @Tags notifications, telegram
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} TelegramLinkResponse "Binding credentials generated successfully"
// @Failure 401 {object} ErrorResponse "Unauthorized - user id missing from X-User-ID header"
// @Failure 503 {object} ErrorResponse "Telegram service unavailable"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /v1/api/notifications/telegram/link [post]
func (h *TelegramGRPCHandler) IssueLink(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "user id missing")
		return
	}
	result, err := h.client.GenerateBindingLink(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to generate binding link via gRPC",
			zap.String("user_id", userID),
			zap.Error(err),
		)

		// Check if it's a service error
		var svcErr *telegramclient.ServiceError
		if errors.As(err, &svcErr) {
			respondWithError(w, http.StatusServiceUnavailable, svcErr.Code, svcErr.Message)
			return
		}

		respondWithError(w, http.StatusInternalServerError, "telegram_error", "Попробуйте позже")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"token":      result.Token,
		"deeplink":   result.DeepLink,
		"code":       result.Code, // Short 6-character code for manual entry
		"expires_at": result.ExpiresAt,
	})
}

// BindingStatus returns binding info for authenticated user via gRPC.
// @Summary Get Telegram binding status
// @Description Returns detailed information about the user's Telegram binding including
// @Description chat_id, username, first/last name, status (active/blocked/unsubscribed),
// @Description and the timestamp of the last update.
// @Tags notifications, telegram
// @Security BearerAuth
// @Produce json
// @Success 200 {object} TelegramBindingStatusResponse "Binding details"
// @Failure 401 {object} ErrorResponse "Unauthorized - user id missing"
// @Failure 404 {object} ErrorResponse "User has no Telegram binding"
// @Failure 503 {object} ErrorResponse "Telegram service unavailable"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /v1/api/notifications/telegram/status [get]
func (h *TelegramGRPCHandler) BindingStatus(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "user id missing")
		return
	}

	status, err := h.client.GetBindingStatus(r.Context(), userID)
	if err != nil {
		var svcErr *telegramclient.ServiceError
		if errors.As(err, &svcErr) {
			if svcErr.IsUserNotBound() {
				respondWithError(w, http.StatusNotFound, "telegram_not_bound", "Telegram chat not linked")
				return
			}
			respondWithError(w, http.StatusServiceUnavailable, svcErr.Code, svcErr.Message)
			return
		}

		h.logger.Error("failed to get binding status via gRPC",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		respondWithError(w, http.StatusInternalServerError, "telegram_error", "failed to fetch binding")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status":     status.Status,
		"chat_id":    status.ChatID,
		"username":   status.Username,
		"first_name": status.FirstName,
		"last_name":  status.LastName,
		"updated_at": status.UpdatedAt,
	})
}

// IsUserBound checks if user has an active Telegram binding via gRPC.
// @Summary Check if user has Telegram binding
// @Description Lightweight check to determine if the user has an active Telegram binding.
// @Description Returns is_bound=true only if status is "active". On service errors,
// @Description gracefully returns is_bound=false to avoid blocking UI.
// @Tags notifications, telegram
// @Security BearerAuth
// @Produce json
// @Success 200 {object} TelegramBoundResponse "Binding check result"
// @Failure 401 {object} ErrorResponse "Unauthorized - user id missing"
// @Router /v1/api/notifications/telegram/bound [get]
func (h *TelegramGRPCHandler) IsUserBound(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "user id missing")
		return
	}

	isBound, status, err := h.client.IsUserBound(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to check binding via gRPC",
			zap.String("user_id", userID),
			zap.Error(err),
		)

		// On gRPC error, return false (non-critical)
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"is_bound": false,
			"status":   "",
		})
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"is_bound": isBound,
		"status":   status,
	})
}

// Unbind removes the Telegram binding for the authenticated user via gRPC.
// @Summary Remove Telegram binding
// @Description Removes the user's Telegram binding, allowing them to re-bind a different
// @Description account or stop receiving Telegram notifications. The user can also unbind
// @Description by sending /unsubscribe command to the bot.
// @Tags notifications, telegram
// @Security BearerAuth
// @Produce json
// @Success 200 {object} TelegramUnbindResponse "Binding removed successfully"
// @Failure 401 {object} ErrorResponse "Unauthorized - user id missing"
// @Failure 404 {object} ErrorResponse "User has no Telegram binding to remove"
// @Failure 503 {object} ErrorResponse "Telegram service unavailable"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /v1/api/notifications/telegram/unbind [post]
func (h *TelegramGRPCHandler) Unbind(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "user id missing")
		return
	}

	err := h.client.UnbindUser(r.Context(), userID, "user requested via API")
	if err != nil {
		var svcErr *telegramclient.ServiceError
		if errors.As(err, &svcErr) {
			if svcErr.IsUserNotBound() {
				respondWithError(w, http.StatusNotFound, "telegram_not_bound", "Telegram chat not linked")
				return
			}
			respondWithError(w, http.StatusServiceUnavailable, svcErr.Code, svcErr.Message)
			return
		}

		h.logger.Error("failed to unbind via gRPC",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		respondWithError(w, http.StatusInternalServerError, "telegram_error", "failed to unbind")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Telegram binding removed",
	})
}

// Swagger response models for Telegram endpoints

// TelegramLinkResponse represents the response for IssueLink endpoint.
// @Description Response containing binding credentials for Telegram connection
type TelegramLinkResponse struct {
	// JWT token for deep link authentication
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIs..."`
	// Deep link URL to open Telegram bot
	DeepLink string `json:"deeplink" example:"https://t.me/EventBot?start=TOKEN"`
	// 6-character alphanumeric code for manual entry
	Code string `json:"code" example:"A3X9K2"`
	// Token expiration time
	ExpiresAt string `json:"expires_at" example:"2025-01-15T12:00:00Z"`
}

// TelegramBindingStatusResponse represents detailed binding information.
// @Description Response containing user's Telegram binding details
type TelegramBindingStatusResponse struct {
	// Binding status: active, blocked, or unsubscribed
	Status string `json:"status" example:"active"`
	// Telegram chat ID
	ChatID int64 `json:"chat_id" example:"123456789"`
	// Telegram username (without @)
	Username string `json:"username" example:"johndoe"`
	// User's first name in Telegram
	FirstName string `json:"first_name" example:"John"`
	// User's last name in Telegram
	LastName string `json:"last_name" example:"Doe"`
	// Last update timestamp
	UpdatedAt string `json:"updated_at" example:"2025-01-15T10:30:00Z"`
}

// TelegramBoundResponse represents the binding check result.
// @Description Response for lightweight binding check
type TelegramBoundResponse struct {
	// Whether user has active Telegram binding
	IsBound bool `json:"is_bound" example:"true"`
	// Binding status if bound
	Status string `json:"status" example:"active"`
}

// TelegramUnbindResponse represents the unbind operation result.
// @Description Response for unbind operation
type TelegramUnbindResponse struct {
	// Whether unbind was successful
	Success bool `json:"success" example:"true"`
	// Confirmation message
	Message string `json:"message" example:"Telegram binding removed"`
}
