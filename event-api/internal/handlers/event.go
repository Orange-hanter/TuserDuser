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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse{
			Error:   "internal_error",
			Message: "Ошибка при получении событий",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(events)
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse{
			Error:   "bad_request",
			Message: "ID события обязателен",
			Code:    http.StatusBadRequest,
		})
		return
	}

	event, err := h.eventService.GetEventByID(r.Context(), id)
	if err != nil {
		logger.Log.Error("Ошибка при получении события", zap.String("id", id), zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.ErrorResponse{
			Error:   "not_found",
			Message: "Событие не найдено",
			Code:    http.StatusNotFound,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(event)
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse{
			Error:   "bad_request",
			Message: "Неверный формат запроса",
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Валидация
	if req.Type == "" || req.PriceType == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse{
			Error:   "validation_error",
			Message: "Type и PriceType обязательны",
			Code:    http.StatusBadRequest,
		})
		return
	}

	event, err := h.eventService.CreateEvent(r.Context(), &req)
	if err != nil {
		logger.Log.Error("Ошибка при создании события", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse{
			Error:   "internal_error",
			Message: "Ошибка при создании события",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	logger.Log.Info("Событие создано", zap.String("id", event.ID))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(event)
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse{
			Error:   "bad_request",
			Message: "ID события обязателен",
			Code:    http.StatusBadRequest,
		})
		return
	}

	err := h.eventService.DeleteEvent(r.Context(), id)
	if err != nil {
		logger.Log.Error("Ошибка при удалении события", zap.String("id", id), zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.ErrorResponse{
			Error:   "not_found",
			Message: err.Error(),
			Code:    http.StatusNotFound,
		})
		return
	}

	logger.Log.Info("Событие удалено", zap.String("id", id))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Событие удалено",
	})
}
