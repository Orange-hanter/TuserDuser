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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"event-api/internal/discovery"
	"event-api/internal/logger"
	"event-api/internal/models"
	"event-api/internal/service"

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
	service     *discovery.Service
	userService *service.UserService
}

// NewDiscoveryHandler создает инстанс `DiscoveryHandler`.
//
// Параметры:
//   - `service` — реализация discovery-логики, отвечающая за получение событий,
//     применение действий пользователя и бронирования.
//   - `userService` — сервис для управления подписками на события (опционально).
func NewDiscoveryHandler(service *discovery.Service, userService *service.UserService) *DiscoveryHandler {
	return &DiscoveryHandler{service: service, userService: userService}
}

// Next возвращает следующее событие в очереди для текущего пользователя.
//
// Описание поведения:
//   - Проверяет наличие `X-User-ID` (middleware должен установить его).
//   - Опционально принимает фильтры через query-параметры:
//   - types: типы событий (через запятую)
//   - priceTypes: типы цен (через запятую)
//   - places: места проведения (через запятую, поиск по подстроке)
//   - dateFrom: начало диапазона дат (RFC3339)
//   - dateTo: конец диапазона дат (RFC3339)
//   - Вызывает `service.NextEventFiltered`, который применяет правила очереди,
//     пропуская конфликтующие события и возвращая первое подходящее.
//   - Возможные ответы: 200 с данными события, 404 если очередь пуста,
//     409 при конфликте действий, 500 при внутренних ошибках.
//
// @Summary Получить следующее событие окна
// @Description Возвращает следующее событие в очереди с учетом бронирований, конфликтов и фильтров
// @Tags discovery
// @Produce json
// @Security BearerAuth
// @Param types query string false "Типы событий через запятую" example("Конференция,Мастер-класс")
// @Param priceTypes query string false "Типы цен через запятую" example("free,paid")
// @Param places query string false "Места проведения (поиск по подстроке)" example("Коворкинг")
// @Param dateFrom query string false "Начало диапазона дат (RFC3339)" example("2025-01-01T00:00:00Z")
// @Param dateTo query string false "Конец диапазона дат (RFC3339)" example("2025-12-31T23:59:59Z")
// @Success 200 {object} discovery.NextEventWithAuthor
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

	// Parse filter from query parameters
	filter, err := parseFilterFromQuery(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	result, err := h.service.NextEventFiltered(r.Context(), userID, filter)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	// Enrich response with event author public profile when available
	type nextResponse struct {
		discovery.NextEvent
		Author *models.PublicUserProfile `json:"author,omitempty"`
	}

	author := h.lookupEventAuthor(r.Context(), result.Event.Metadata)
	respondWithJSON(w, http.StatusOK, nextResponse{NextEvent: result, Author: author})
}

// lookupEventAuthor retrieves the public profile of the event creator if available.
func (h *DiscoveryHandler) lookupEventAuthor(ctx context.Context, metadata map[string]any) *models.PublicUserProfile {
	if h.userService == nil || metadata == nil {
		return nil
	}

	creatorID := extractCreatorID(metadata)
	if creatorID == "" {
		return nil
	}

	profile, _, err := h.userService.GetPublicProfile(ctx, creatorID)
	if err != nil || profile == nil {
		return nil
	}
	return profile
}

// extractCreatorID extracts creator ID from event metadata, checking both naming conventions.
func extractCreatorID(metadata map[string]any) string {
	if v, ok := metadata["creator_id"].(string); ok && v != "" {
		return v
	}
	if v, ok := metadata["creatorId"].(string); ok && v != "" {
		return v
	}
	return ""
}

// parseFilterFromQuery extracts discovery filter from query parameters.
func parseFilterFromQuery(r *http.Request) (discovery.Filter, error) {
	q := r.URL.Query()
	filter := discovery.Filter{}

	// Parse comma-separated string values
	if types := q.Get("types"); types != "" {
		filter.Types = splitAndTrim(types)
	}
	if priceTypes := q.Get("priceTypes"); priceTypes != "" {
		filter.PriceTypes = splitAndTrim(priceTypes)
	}
	if places := q.Get("places"); places != "" {
		filter.Places = splitAndTrim(places)
	}

	if dateFrom := q.Get("dateFrom"); dateFrom != "" {
		t, err := time.Parse(time.RFC3339, dateFrom)
		if err != nil {
			return filter, fmt.Errorf("parse dateFrom: %w", err)
		}
		filter.DateFrom = &t
	}
	if dateTo := q.Get("dateTo"); dateTo != "" {
		t, err := time.Parse(time.RFC3339, dateTo)
		if err != nil {
			return filter, fmt.Errorf("parse dateTo: %w", err)
		}
		filter.DateTo = &t
	}

	return filter, nil
}

// splitAndTrim splits comma-separated string and trims whitespace.
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
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

	// Создаём подписку на событие в event_subscriptions
	if h.userService != nil {
		_, subErr := h.userService.SubscribeToEvent(r.Context(), userID, req.EventID, nil)
		if subErr != nil {
			logger.Log.Warn("failed to create subscription after booking",
				zap.String("user_id", userID),
				zap.String("event_id", req.EventID),
				zap.Error(subErr))
			// Не возвращаем ошибку — бронирование уже успешно
		}
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
	switch {
	case errors.Is(err, discovery.ErrQueueEmpty):
		respondWithError(w, http.StatusNotFound, "queue_empty", "События для окна не найдены")
	case errors.Is(err, discovery.ErrInvalidAction):
		respondWithError(w, http.StatusBadRequest, "invalid_action", err.Error())
	case errors.Is(err, discovery.ErrOutOfOrderAction), errors.Is(err, discovery.ErrNoActiveEvent):
		respondWithError(w, http.StatusConflict, "queue_conflict", err.Error())
	case errors.Is(err, discovery.ErrEventNotFound):
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
