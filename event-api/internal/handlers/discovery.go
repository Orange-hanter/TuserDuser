package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"event-api/internal/discovery"
	"event-api/internal/logger"
	"event-api/internal/models"

	"go.uber.org/zap"
)

// DiscoveryHandler exposes the narrow time-slot discovery engine over HTTP.
type DiscoveryHandler struct {
	service *discovery.Service
}

// NewDiscoveryHandler instantiates the handler layer.
func NewDiscoveryHandler(service *discovery.Service) *DiscoveryHandler {
	return &DiscoveryHandler{service: service}
}

// Next returns the next event in the queue.
//
// Retrieves the next event for the authenticated user, taking into account
// existing bookings and schedule conflicts. Events are processed in FIFO order,
// but skipped if they conflict with prior reservations.
//
// @Summary Получить следующее событие окна
// @Description Возвращает следующее событие в очереди с учетом бронирований и конфликтов
// @Tags discovery
// @Produce json
// @Security BearerAuth
// @Success 200 {object} discovery.NextEvent
// @Failure 401 {object} models.ErrorResponse "Нет авторизации"
// @Failure 404 {object} models.ErrorResponse "Очередь пуста"
// @Failure 409 {object} models.ErrorResponse "Конфликт действий"
// @Failure 500 {object} models.ErrorResponse "Внутренняя ошибка"
// @Router /v1/api/discovery/next [get]
func (h *DiscoveryHandler) Next(w http.ResponseWriter, r *http.Request) {
	userID, ok := extractUserID(w, r)
	if !ok {
		return
	}
	result, err := h.service.NextEvent(r.Context(), userID)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}
	respondWithJSON(w, http.StatusOK, result)
}

// Action applies like/dislike/neutral feedback.
// @Summary Отправить реакцию на событие
// @Description Обрабатывает действия like/dislike/neutral для текущего события в очереди
// @Tags discovery
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.DiscoveryActionRequest true "Действие пользователя"
// @Success 200 {object} discovery.HistoryEntry
// @Failure 400 {object} models.ErrorResponse "Некорректные данные"
// @Failure 401 {object} models.ErrorResponse "Нет авторизации"
// @Failure 404 {object} models.ErrorResponse "Событие не найдено"
// @Failure 409 {object} models.ErrorResponse "Конфликт очереди"
// @Failure 500 {object} models.ErrorResponse "Внутренняя ошибка"
// @Router /v1/api/discovery/action [post]
func (h *DiscoveryHandler) Action(w http.ResponseWriter, r *http.Request) {
	userID, ok := extractUserID(w, r)
	if !ok {
		return
	}
	var req models.DiscoveryActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "bad_request", "Неверный формат запроса")
		return
	}
	if strings.TrimSpace(req.EventID) == "" {
		respondWithError(w, http.StatusBadRequest, "validation_error", "eventId обязателен")
		return
	}
	action := discovery.UserAction(strings.ToLower(strings.TrimSpace(req.Action)))
	if action == discovery.ActionBook {
		respondWithError(w, http.StatusBadRequest, "validation_error", "Используйте /book для бронирования")
		return
	}
	entry, err := h.service.ApplyAction(r.Context(), userID, req.EventID, action)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}
	respondWithJSON(w, http.StatusOK, entry)
}

// Book commits to an event and propagates conflicts.
// @Summary Забронировать событие
// @Description Подтверждает участие в текущем событии, откладывая конфликтующие события в конец очереди
// @Tags discovery
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.DiscoveryBookRequest true "Запрос на бронирование"
// @Success 200 {object} discovery.BookingResult
// @Failure 400 {object} models.ErrorResponse "Некорректные данные"
// @Failure 401 {object} models.ErrorResponse "Нет авторизации"
// @Failure 404 {object} models.ErrorResponse "Событие не найдено"
// @Failure 409 {object} models.ErrorResponse "Конфликт очереди"
// @Failure 500 {object} models.ErrorResponse "Внутренняя ошибка"
// @Router /v1/api/discovery/book [post]
func (h *DiscoveryHandler) Book(w http.ResponseWriter, r *http.Request) {
	userID, ok := extractUserID(w, r)
	if !ok {
		return
	}
	var req models.DiscoveryBookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "bad_request", "Неверный формат запроса")
		return
	}
	if strings.TrimSpace(req.EventID) == "" {
		respondWithError(w, http.StatusBadRequest, "validation_error", "eventId обязателен")
		return
	}
	result, err := h.service.BookEvent(r.Context(), userID, req.EventID)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}
	respondWithJSON(w, http.StatusOK, result)
}

// History returns chronological user actions.
// @Summary История действий пользователя
// @Description Возвращает детальную историю действий discovery-движка для пользователя
// @Tags discovery
// @Produce json
// @Security BearerAuth
// @Success 200 {array} discovery.HistoryEntry
// @Failure 401 {object} models.ErrorResponse "Нет авторизации"
// @Failure 500 {object} models.ErrorResponse "Внутренняя ошибка"
// @Router /v1/api/discovery/history [get]
func (h *DiscoveryHandler) History(w http.ResponseWriter, r *http.Request) {
	userID, ok := extractUserID(w, r)
	if !ok {
		return
	}
	entries, err := h.service.History(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondWithJSON(w, http.StatusOK, entries)
}

func (h *DiscoveryHandler) handleDomainError(w http.ResponseWriter, err error) {
	switch err {
	case discovery.ErrQueueEmpty:
		respondWithError(w, http.StatusNotFound, "queue_empty", "События для окна не найдены")
	case discovery.ErrInvalidAction:
		respondWithError(w, http.StatusBadRequest, "invalid_action", err.Error())
	case discovery.ErrOutOfOrderAction, discovery.ErrNoActiveEvent:
		respondWithError(w, http.StatusConflict, "queue_conflict", err.Error())
	case discovery.ErrEventNotFound:
		respondWithError(w, http.StatusNotFound, "not_found", err.Error())
	default:
		logger.Log.Error("discovery handler error", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "internal_error", "Не удалось обработать запрос")
	}
}

func extractUserID(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "Требуется аутентификация")
		return "", false
	}
	return userID, true
}
