// Package handlers provides HTTP handlers that delegate Telegram operations to telegram-service via gRPC.
package handlers

import (
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
// @Summary Issue Telegram binding deep link (gRPC)
// @Description Issues a short-lived deep-link token via telegram-service gRPC
// @Tags notifications, telegram
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Returns the token and deeplink"
// @Failure 401 {object} map[string]interface{} "Unauthorized or missing user id"
// @Failure 500 {object} map[string]interface{} "Internal server error"
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
		if svcErr, ok := err.(*telegramclient.ServiceError); ok {
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
// @Summary Get Telegram binding status (gRPC)
// @Description Returns the current Telegram binding status via telegram-service gRPC
// @Tags notifications, telegram
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{} "status, chat_id, username, updated_at"
// @Failure 401 {object} map[string]interface{} "Unauthorized or missing user id"
// @Failure 404 {object} map[string]interface{} "Telegram not bound"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /v1/api/notifications/telegram/status [get]
func (h *TelegramGRPCHandler) BindingStatus(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "user id missing")
		return
	}

	status, err := h.client.GetBindingStatus(r.Context(), userID)
	if err != nil {
		if svcErr, ok := err.(*telegramclient.ServiceError); ok {
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
// @Summary Check if user is bound (gRPC)
// @Description Checks if user has active Telegram binding via telegram-service gRPC
// @Tags notifications, telegram
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{} "is_bound, status"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
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
// @Summary Unbind Telegram (gRPC)
// @Description Removes Telegram binding via telegram-service gRPC
// @Tags notifications, telegram
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{} "success"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Not bound"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /v1/api/notifications/telegram/unbind [post]
func (h *TelegramGRPCHandler) Unbind(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "user id missing")
		return
	}

	err := h.client.UnbindUser(r.Context(), userID, "user requested via API")
	if err != nil {
		if svcErr, ok := err.(*telegramclient.ServiceError); ok {
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
