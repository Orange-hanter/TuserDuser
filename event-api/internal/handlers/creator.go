package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"event-api/internal/service"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// CreatorHandler обрабатывает запросы автора событий.
type CreatorHandler struct {
	service *service.CreatorService
	logger  *zap.Logger
}

// NewCreatorHandler создаёт handler.
func NewCreatorHandler(svc *service.CreatorService, logger *zap.Logger) *CreatorHandler {
	return &CreatorHandler{service: svc, logger: logger}
}

// GetMyEvents возвращает события автора по категориям.
// @Summary Получить мои события
// @Description Возвращает события автора: ожидающие проверки, активные и отклонённые
// @Tags creator
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.CreatorEventsResponse
// @Failure 401 {object} models.ErrorResponse "Нет авторизации"
// @Failure 500 {object} models.ErrorResponse "Внутренняя ошибка"
// @Router /v1/api/creator/events [get]
func (h *CreatorHandler) GetMyEvents(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "Требуется авторизация")
		return
	}

	events, err := h.service.GetMyEvents(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get creator events", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "internal_error", "Ошибка получения событий")
		return
	}

	respondWithJSON(w, http.StatusOK, events)
}

// GetBlockedEvents возвращает заблокированные события автора.
// @Summary Получить заблокированные события
// @Description Возвращает список заблокированных событий автора с причинами блокировки
// @Tags creator
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.BlockedEvent
// @Failure 401 {object} models.ErrorResponse "Нет авторизации"
// @Failure 500 {object} models.ErrorResponse "Внутренняя ошибка"
// @Router /v1/api/creator/events/blocked [get]
func (h *CreatorHandler) GetBlockedEvents(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "Требуется авторизация")
		return
	}

	events, err := h.service.GetBlockedEvents(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get blocked events", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "internal_error", "Ошибка получения событий")
		return
	}

	respondWithJSON(w, http.StatusOK, events)
}

// GetEventComments возвращает комментарии модерации для события.
// @Summary Получить комментарии к событию
// @Description Возвращает историю комментариев модерации для указанного события
// @Tags creator
// @Produce json
// @Security BearerAuth
// @Param eventId path string true "ID события"
// @Success 200 {array} models.ReviewComment
// @Failure 401 {object} models.ErrorResponse "Нет авторизации"
// @Failure 404 {object} models.ErrorResponse "Событие не найдено"
// @Failure 500 {object} models.ErrorResponse "Внутренняя ошибка"
// @Router /v1/api/creator/events/{eventId}/comments [get]
func (h *CreatorHandler) GetEventComments(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "Требуется авторизация")
		return
	}

	eventID := chi.URLParam(r, "eventId")
	if eventID == "" {
		respondWithError(w, http.StatusBadRequest, "validation_error", "eventId обязателен")
		return
	}

	comments, err := h.service.GetEventComments(r.Context(), eventID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondWithError(w, http.StatusNotFound, "not_found", "Событие не найдено")
			return
		}
		h.logger.Error("failed to get event comments", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "internal_error", "Ошибка получения комментариев")
		return
	}

	respondWithJSON(w, http.StatusOK, comments)
}

// GetEventCommentsAsAdmin возвращает комментарии модерации для события (для админа).
// @Summary Получить комментарии к событию (админ)
// @Description Возвращает историю комментариев модерации для указанного события без проверки владельца
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param eventId path string true "ID события"
// @Success 200 {array} models.ReviewComment
// @Failure 401 {object} models.ErrorResponse "Нет авторизации"
// @Failure 404 {object} models.ErrorResponse "Событие не найдено"
// @Failure 500 {object} models.ErrorResponse "Внутренняя ошибка"
// @Router /v1/api/admin/events/{eventId}/comments [get]
func (h *CreatorHandler) GetEventCommentsAsAdmin(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventId")
	if eventID == "" {
		respondWithError(w, http.StatusBadRequest, "validation_error", "eventId обязателен")
		return
	}

	comments, err := h.service.GetEventCommentsForAdmin(r.Context(), eventID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondWithError(w, http.StatusNotFound, "not_found", "Событие не найдено")
			return
		}
		h.logger.Error("failed to get event comments", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "internal_error", "Ошибка получения комментариев")
		return
	}

	respondWithJSON(w, http.StatusOK, comments)
}

// AddComment добавляет комментарий к событию.
// @Summary Добавить комментарий
// @Description Позволяет автору добавить комментарий к своему событию
// @Tags creator
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param eventId path string true "ID события"
// @Param request body models.AddReviewCommentRequest true "Комментарий"
// @Success 200 {object} map[string]string
// @Failure 400 {object} models.ErrorResponse "Ошибка валидации"
// @Failure 401 {object} models.ErrorResponse "Нет авторизации"
// @Failure 500 {object} models.ErrorResponse "Внутренняя ошибка"
// @Router /v1/api/creator/events/{eventId}/comments [post]
func (h *CreatorHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	userRole := r.Header.Get("X-User-Role")
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "Требуется авторизация")
		return
	}

	eventID := chi.URLParam(r, "eventId")
	if eventID == "" {
		respondWithError(w, http.StatusBadRequest, "validation_error", "eventId обязателен")
		return
	}

	var req struct {
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "bad_request", "Неверный формат запроса")
		return
	}
	if strings.TrimSpace(req.Comment) == "" {
		respondWithError(w, http.StatusBadRequest, "validation_error", "comment обязателен")
		return
	}

	role := userRole
	if role == "" {
		role = "creator"
	}

	err := h.service.AddComment(r.Context(), eventID, userID, role, req.Comment)
	if err != nil {
		h.logger.Error("failed to add comment", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "internal_error", "Ошибка добавления комментария")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// RequestRevision запрашивает доработку события (для админа).
// @Summary Запросить доработку события
// @Description Переводит событие в статус needs_revision с комментарием
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param eventId path string true "ID события"
// @Param request body models.AddReviewCommentRequest true "Комментарий"
// @Success 200 {object} map[string]string
// @Failure 400 {object} models.ErrorResponse "Ошибка валидации"
// @Failure 401 {object} models.ErrorResponse "Нет авторизации"
// @Failure 404 {object} models.ErrorResponse "Событие не найдено"
// @Failure 500 {object} models.ErrorResponse "Внутренняя ошибка"
// @Router /v1/api/admin/events/{eventId}/request-revision [post]
func (h *CreatorHandler) RequestRevision(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "Требуется авторизация")
		return
	}

	eventID := chi.URLParam(r, "eventId")
	if eventID == "" {
		respondWithError(w, http.StatusBadRequest, "validation_error", "eventId обязателен")
		return
	}

	var req struct {
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "bad_request", "Неверный формат запроса")
		return
	}
	if strings.TrimSpace(req.Comment) == "" {
		respondWithError(w, http.StatusBadRequest, "validation_error", "comment обязателен")
		return
	}

	err := h.service.RequestRevision(r.Context(), eventID, userID, req.Comment)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondWithError(w, http.StatusNotFound, "not_found", err.Error())
			return
		}
		h.logger.Error("failed to request revision", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "internal_error", "Ошибка обновления события")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// BlockEvent блокирует событие (для админа).
// @Summary Заблокировать событие
// @Description Переносит событие в заблокированные с указанием причины
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param eventId path string true "ID события"
// @Param request body models.BlockEventRequest true "Причина блокировки"
// @Success 200 {object} map[string]string
// @Failure 400 {object} models.ErrorResponse "Ошибка валидации"
// @Failure 401 {object} models.ErrorResponse "Нет авторизации"
// @Failure 404 {object} models.ErrorResponse "Событие не найдено"
// @Failure 500 {object} models.ErrorResponse "Внутренняя ошибка"
// @Router /v1/api/admin/events/{eventId}/block [post]
func (h *CreatorHandler) BlockEvent(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "Требуется авторизация")
		return
	}

	eventID := chi.URLParam(r, "eventId")
	if eventID == "" {
		respondWithError(w, http.StatusBadRequest, "validation_error", "eventId обязателен")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "bad_request", "Неверный формат запроса")
		return
	}
	if strings.TrimSpace(req.Reason) == "" {
		respondWithError(w, http.StatusBadRequest, "validation_error", "reason обязателен")
		return
	}

	err := h.service.BlockEvent(r.Context(), eventID, userID, req.Reason)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondWithError(w, http.StatusNotFound, "not_found", err.Error())
			return
		}
		h.logger.Error("failed to block event", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "internal_error", "Ошибка блокировки события")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
