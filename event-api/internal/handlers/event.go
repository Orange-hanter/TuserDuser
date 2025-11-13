package handlers

import (
	"encoding/json"
	"net/http"

	"event-api/internal/logger"
	"event-api/internal/models"
	"event-api/internal/service"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// EventHandler управляет всеми event endpoints.
type EventHandler struct {
	eventService *service.EventService
}

// NewEventHandler создает новый event handler.
func NewEventHandler(eventService *service.EventService) *EventHandler {
	return &EventHandler{
		eventService: eventService,
	}
}

// GetAllEvents получает все события
// GET /v1/api/events
// @Summary Получить все события
// @Description Возвращает список всех событий
// @Tags events
// @Produce json
// @Success 200 {array} models.Event "Список событий"
// @Failure 500 {object} models.ErrorResponse "Внутренняя ошибка сервера"
// @Router /v1/api/events [get].
func (h *EventHandler) GetAllEvents(w http.ResponseWriter, r *http.Request) {
	events, err := h.eventService.GetAllEvents(r.Context())
	if err != nil {
		logger.Log.Error("Ошибка при получении событий", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "internal_error", "Ошибка при получении событий")
		return
	}

	respondWithJSON(w, http.StatusOK, events)
}

// GetEventByID получает событие по ID
// GET /v1/api/events/{id}
// @Summary Получить событие по ID
// @Description Возвращает событие с указанным ID
// @Tags events
// @Produce json
// @Param id path string true "ID события"
// @Success 200 {object} models.Event "Событие"
// @Failure 404 {object} models.ErrorResponse "Событие не найдено"
// @Failure 500 {object} models.ErrorResponse "Внутренняя ошибка сервера"
// @Router /v1/api/events/{id} [get].
func (h *EventHandler) GetEventByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondWithError(w, http.StatusBadRequest, "bad_request", "ID события обязателен")
		return
	}

	event, err := h.eventService.GetEventByID(r.Context(), id)
	if err != nil {
		logger.Log.Error("Ошибка при получении события", zap.String("id", id), zap.Error(err))
		respondWithError(w, http.StatusNotFound, "not_found", "Событие не найдено")
		return
	}

	respondWithJSON(w, http.StatusOK, event)
}

// CreateEvent создает новое событие
// POST /v1/api/events
// @Summary Создать событие
// @Description Создает новое событие
// @Tags events
// @Accept json
// @Produce json
// @Param request body models.CreateEventRequest true "Данные события"
// @Success 201 {object} models.Event "Созданное событие"
// @Failure 400 {object} models.ErrorResponse "Неверный формат запроса"
// @Failure 500 {object} models.ErrorResponse "Внутренняя ошибка сервера"
// @Router /v1/api/events [post].
func (h *EventHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var req models.CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Error("Ошибка при парсинге CreateEventRequest", zap.Error(err))
		respondWithError(w, http.StatusBadRequest, "bad_request", "Неверный формат запроса")
		return
	}

	// Валидация
	if req.Type == "" || req.PriceType == "" {
		respondWithError(w, http.StatusBadRequest, "validation_error", "Type и PriceType обязательны")
		return
	}

	event, err := h.eventService.CreateEvent(r.Context(), &req)
	if err != nil {
		logger.Log.Error("Ошибка при создании события", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "internal_error", "Ошибка при создании события")
		return
	}

	logger.Log.Info("Событие создано", zap.String("id", event.ID))
	respondWithJSON(w, http.StatusCreated, event)
}

// DeleteEvent удаляет событие
// DELETE /v1/api/events/{id}
// @Summary Удалить событие
// @Description Удаляет событие по ID
// @Tags events
// @Produce json
// @Param id path string true "ID события"
// @Success 200 {object} map[string]string "message: Событие удалено"
// @Failure 404 {object} models.ErrorResponse "Событие не найдено"
// @Failure 500 {object} models.ErrorResponse "Внутренняя ошибка сервера"
// @Router /v1/api/events/{id} [delete].
func (h *EventHandler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondWithError(w, http.StatusBadRequest, "bad_request", "ID события обязателен")
		return
	}

	err := h.eventService.DeleteEvent(r.Context(), id)
	if err != nil {
		logger.Log.Error("Ошибка при удалении события", zap.String("id", id), zap.Error(err))
		respondWithError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}

	logger.Log.Info("Событие удалено", zap.String("id", id))
	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Событие удалено",
	})
}
