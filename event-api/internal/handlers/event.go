// Package handlers содержит HTTP-обработчики, выполняющие роль тонкой
// транспортной прослойки поверх сервисного слоя. Handlers принимают и парсят
// HTTP-запросы, выполняют базовую валидацию, переводят доменные ошибки в
// читабельные JSON-ответы и управляют HTTP-статусами.
//
// Этот файл реализует обработчики для работы с событиями (events):
// публичные события, события на модерации, создание, ревью и удаление.
package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"event-api/internal/logger"
	"event-api/internal/models"
	"event-api/internal/service"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// EventHandler управляет всеми endpoints, связанными с событиями (events).
//
// Responsibilities:
// - Принимать HTTP-запросы, выполнять базовую валидацию/парсинг.
// - Делегировать бизнес-логику в `service.EventService`.
// - Обрабатывать ошибки сервисного слоя и возвращать понятные клиенту JSON-ответы.
type EventHandler struct {
	eventService *service.EventService
}

// NewEventHandler создает новый экземпляр `EventHandler`.
//
// Параметры:
// - `eventService` — реализация бизнес-логики работы с событиями.
func NewEventHandler(eventService *service.EventService) *EventHandler {
	return &EventHandler{
		eventService: eventService,
	}
}

// GetApprovedEvents возвращает все публично доступные одобренные события.
//
// Поведение:
// - Запрашивает у сервисного слоя список всех событий со статусом "approved".
// - В случае ошибки сервисного слоя возвращает 500 с общей информацией.
//
// @Summary Получить одобренные события
// @Description Возвращает список всех событий, доступных публично
// @Tags events
// @Produce json
// @Success 200 {array} models.Event "Список событий"
// @Failure 500 {object} models.ErrorResponse "Внутренняя ошибка сервера"
// @Router /v1/api/events/approved [get]
func (h *EventHandler) GetApprovedEvents(w http.ResponseWriter, r *http.Request) {
	events, err := h.eventService.GetApprovedEvents(r.Context())
	if err != nil {
		logger.Log.Error("Ошибка при получении событий", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "internal_error", "Ошибка при получении событий")
		return
	}

	respondWithJSON(w, http.StatusOK, events)
}

// GetPendingEvents возвращает события, ожидающие модерации.
//
// Поведение:
// - Возвращает события в статусе `pending` — используется админами/модераторами.
// - Предполагает, что caller авторизован (проверка через middleware).
// - В случае проблем с сервисом возвращает 500.
//
// @Summary Получить события на модерации
// @Description Возвращает список событий в статусе pending (требуется авторизация)
// @Tags events
// @Produce json
// @Success 200 {array} models.PendingEvent
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /v1/api/events/pending [get]
func (h *EventHandler) GetPendingEvents(w http.ResponseWriter, r *http.Request) {
	events, err := h.eventService.GetPendingEvents(r.Context())
	if err != nil {
		logger.Log.Error("Ошибка при получении событий на модерации", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "internal_error", "Ошибка при получении событий на модерации")
		return
	}

	respondWithJSON(w, http.StatusOK, events)
}

// GetEventByID возвращает событие по его идентификатору.
//
// Поведение и валидация:
// - Читает `id` из path-параметра; если пустой — возвращает 400.
// - Делегирует поиск событию в `eventService.GetEventByID`.
// - Возвращает 200 и объект события или 404/500 в зависимости от ошибки.
//
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

// CreateEvent создаёт новое событие и отправляет его на модерацию.
//
// Ожидаемый JSON (models.CreateEventRequest) — см. модель:
// - Поля `Type` и `PriceType` обязательны; при их отсутствии возвращается 400.
// - При успешном создании событие возвращается в статусе pending с кодом 201.
//
// @Summary Создать событие
// @Description Создает новое событие
// @Tags events
// @Accept json
// @Produce json
// @Param request body models.CreateEventRequest true "Данные события"
// @Success 201 {object} models.PendingEvent "Событие, отправленное на модерацию"
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

// ReviewPendingEvent переводит событие из очереди модерации в одобренные или
// отклонённые.
//
// Поведение:
// - Читает `id` из path; валидирует тело запроса (`action`, комментарий при reject).
// - Допускаются action: `approve`, `reject`, `block`.
// - При отклонении комментарий обязателен.
// - Делегирует операцию в `eventService.ReviewPendingEvent`.
//
// @Summary Одобрить или отклонить событие
// @Description Переводит событие из очереди модерации в основной список или архив
// @Tags events
// @Accept json
// @Produce json
// @Param id path string true "ID события"
// @Param request body models.ReviewEventRequest true "Данные решения"
// @Success 200 {object} map[string]string
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /v1/api/events/{id}/review [post]
func (h *EventHandler) ReviewPendingEvent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondWithError(w, http.StatusBadRequest, "bad_request", "ID события обязателен")
		return
	}

	var req models.ReviewEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "bad_request", "Неверный формат запроса")
		return
	}

	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action == "" {
		respondWithError(w, http.StatusBadRequest, "validation_error", "Action обязателен")
		return
	}

	rejectAction := action == models.EventStatusRejected || action == "reject" || action == "block"
	if rejectAction && strings.TrimSpace(req.Comment) == "" {
		respondWithError(w, http.StatusBadRequest, "validation_error", "Комментарий обязателен при отклонении")
		return
	}

	switch action {
	case "approve":
		action = models.EventStatusApproved
	case "reject", "block":
		action = models.EventStatusRejected
	}

	if err := h.eventService.ReviewPendingEvent(r.Context(), id, action, req.Comment); err != nil {
		logger.Log.Error("Ошибка при изменении статуса события", zap.String("id", id), zap.Error(err))
		respondWithError(w, http.StatusBadRequest, "review_failed", err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"status": action,
	})
}

// DeleteEvent удаляет событие по указанному ID.
//
// Поведение:
// - Читает `id` из path и вызывает `eventService.DeleteEvent`.
// - В случае отсутствия события возвращает 404; при ошибках сервиса — 500.
//
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
