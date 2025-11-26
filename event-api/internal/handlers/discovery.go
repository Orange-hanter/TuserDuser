// Package handlers содержит HTTP-обработчики, которые выступают тонким
// транспортным слоем поверх бизнес-логики. Handlers принимают HTTP-запросы,
// выполняют базовую валидацию/парсинг, переводят ошибки бизнес-логики в
// читабельные JSON-ответы и управляют HTTP статус-кодами.
//
// Этот файл содержит обработчики для discovery-движка — механизма, который
// предоставляет пользователю последовательность событий (time-slot discovery),
// позволяет отдавать реакции (like/dislike), а также бронировать события.
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

// DiscoveryHandler предоставляет HTTP-обёртку для discovery.Service.
//
// Поведение:
//   - Делегирует основную логику и валидацию в `discovery.Service`.
//   - Преобразует ошибки домена в JSON-ответы с понятными кодами и типами ошибок.
//   - Ожидает, что middleware аутентификации поставит `X-User-ID` в заголовок
//     запроса (см. `extractUserID`).
type DiscoveryHandler struct {
	service *discovery.Service
}

// NewDiscoveryHandler создает инстанс `DiscoveryHandler`.
//
// Параметры:
//   - `service` — реализация discovery-логики, отвечающая за получение событий,
//     применение действий пользователя и бронирования.
func NewDiscoveryHandler(service *discovery.Service) *DiscoveryHandler {
	return &DiscoveryHandler{service: service}
}

// Next возвращает следующее событие в очереди для текущего пользователя.
//
// Описание поведения:
//   - Проверяет наличие `X-User-ID` (middleware должен установить его).
//   - Вызывает `service.NextEvent`, который применяет правила очереди,
//     пропуская конфликтующие события и возвращая первое подходящее.
//   - Возможные ответы: 200 с данными события, 404 если очередь пуста,
//     409 при конфликте действий, 500 при внутренних ошибках.
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

// Action обрабатывает реакцию пользователя на текущее событие очереди.
//
// Ожидаемый JSON (models.DiscoveryActionRequest):
//
//	{
//	  "eventId": "<id>",
//	  "action": "like|dislike|neutral|book"
//	}
//
// Поведение:
//   - Валидирует вход (eventId обязателен, action — корректен).
//   - Если action == "book" — клиент должен использовать отдельный эндпоинт
//     `/book` (чтобы отделить семантику брони от простых реакций).
//   - Делегирует выполнение `service.ApplyAction`, возвращает запись истории при успехе.
//
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
		respondWithError(w, http.StatusBadRequest, "validation_error", "Используйте /book или другое действие")
		return
	}
	entry, err := h.service.ApplyAction(r.Context(), userID, req.EventID, action)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}
	respondWithJSON(w, http.StatusOK, entry)
}

// Book подтверждает участие в событии (бронь) и переносит конфликтующие
// события в конец очереди.
//
// Ожидаемый JSON (models.DiscoveryBookRequest): {"eventId":"<id>"}
//
// Поведение:
//   - Валидирует наличие `eventId`.
//   - Вызывает `service.BookEvent`, который выполняет операцию бронирования и
//     разрешает конфликты в соответствии с бизнес-правилами.
//   - Возвращает результат бронирования или соответствующую ошибку.
//
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

// History возвращает хронологическую историю действий пользователя в discovery.
//
// Поведение:
// - Читает `X-User-ID` и запрашивает историю у сервиса.
// - Возвращает массив записей истории или 500 при ошибке получения.
//
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
