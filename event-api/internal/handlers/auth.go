// Package handlers содержит HTTP обработчики для аутентификации и управления пользователями.
//
// Этот пакет реализует HTTP-ориентированный слой (handlers) поверх бизнес-логики
// (AuthService) и вспомогательных хранилищ (например, Telegram bindings storage).
//
// Документирование ожиданий и поведения:
//   - Каждый метод-обработчик ожидает JSON в теле запроса (если иное не указано) и
//     возвращает JSON-ответы с соответствующими HTTP-кодами.
//   - Валидация входных данных выполняется на уровне handler'ов с явными сообщениями
//     об ошибке и кодами ошибок в структуре `models.ErrorResponse`.
//   - Логирование делится на уровни: Info для успешных операций, Warn для
//     ожидаемых проблем (например, неверные креды), Error для непредвиденных ошибок.
//   - Некоторые эндпоинты требуют аутентификации (см. комментарии `@Security BearerAuth`).
//
// Примеры использования (curl):
//  1. Регистрация:
//     curl -X POST -H "Content-Type: application/json" -d '{"email":"a@b.ru","password":"12345678","phone":"+79001234567"}' http://.../api/auth/register
//  2. Вход:
//     curl -X POST -H "Content-Type: application/json" -d '{"email":"a@b.ru","password":"12345678"}' http://.../api/auth/login
//
// Замечания по безопасности:
//   - Handlers не хранят и не валидируют токены напрямую — это задача сервисного слоя.
//   - Для операций, связанных с ролью/администрированием, предполагается наличие
//     middleware, проверяющего права пользователя (RBAC).
//   - В development режиме некоторые данные (verify_code) могут возвращаться в ответе
//     для удобства тестирования; в production это следует отключать.
package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"event-api/internal/logger"
	"event-api/internal/models"
	"event-api/internal/telegramclient"

	"go.uber.org/zap"
)

// AuthService описывает необходимые методы для обслуживания auth endpoints.
//
// Описание контрактов:
//   - Register: принимает `models.RegisterRequest`, создает запись пользователя в БД,
//     генерирует код верификации (или отправляет его в сторонние сервисы) и
//     возвращает созданного пользователя и код (код может возвращаться в ответе
//     только в development/testing средах).
//   - VerifyAndIssueToken: проверяет код верификации для указанного email и, при успехе,
//     возвращает структуру `models.AuthResponse`, содержащую access и refresh токены.
//   - Login: валидирует учетные данные, возвращает `models.AuthResponse` при успехе.
//   - Logout: инвалидирует refresh токен (или помечает сессию как неактивную).
//   - GetUserByID/GetAllUsers/UpdateUserRole: CRUD-операции для пользователей,
//     используемые административными эндпоинтами. UpdateUserRole должен проверять
//     корректность роли на уровне сервисного слоя.
//
// Все методы должны возвращать понятные ошибки, которые handlers переводят в
// корректные HTTP-коды и JSON-ошибки.
type AuthService interface {
	Register(*models.RegisterRequest) (*models.User, string, error)
	VerifyAndIssueToken(email, code string) (*models.AuthResponse, error)
	Login(*models.LoginRequest) (*models.AuthResponse, error)
	Logout(token string) error
	GetUserByID(userID string) (*models.User, error)
	UpdateUserRole(userID, role string) error
	GetAllUsers() ([]*models.User, error)
}

// AuthHandler управляет всеми auth endpoints.
//
// Полезная информация о поле `telegramClient`:
//   - `telegramClient` используется для проверки привязки Telegram к пользователю
//     (например, в `GetMe`) через gRPC запрос к telegram-service.
//     Может быть nil — handlers должны корректно обрабатывать этот случай.
//   - `authService` содержит бизнес-логику и абстрагирует работу с БД и токенами.
type AuthHandler struct {
	authService    AuthService
	telegramClient *telegramclient.Client
}

// NewAuthHandler создает новый auth handler.
//
// Параметры:
// - `authService` — реализация интерфейса `AuthService`, отвечающая за бизнес-логику.
// - `telegramClient` — (опционально) gRPC клиент для telegram-service; может быть nil.
//
// Возвращает готовую структуру `AuthHandler`, которую можно использовать при
// регистрации маршрутов HTTP сервера.
func NewAuthHandler(authService AuthService, telegramClient *telegramclient.Client) *AuthHandler {
	return &AuthHandler{
		authService:    authService,
		telegramClient: telegramClient,
	}
}

// respondWithError отправляет JSON ответ с ошибкой.
//
// Формат ответа соответствует `models.ErrorResponse` и содержит поля:
// - error: краткий тип ошибки (например, "bad_request", "unauthorized")
// - message: подробное сообщение об ошибке, пригодное для отображения клиенту
// - code: HTTP статус код
//
// Функция логирует внутренние ошибки кодирования JSON, но не раскрывает
// внутренние детали клиенту.
func respondWithError(w http.ResponseWriter, statusCode int, errorType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(models.ErrorResponse{
		Error:   errorType,
		Message: message,
		Code:    statusCode,
	}); err != nil {
		logger.Log.Error("Ошибка при отправке ответа с ошибкой",
			zap.String("error_type", errorType),
			zap.Error(err))
	}
}

// respondWithJSON отправляет JSON ответ с данными.
//
// Преобразует `data` в JSON и отправляет с указанным `statusCode`.
// Логирует ошибку кодирования, но не изменяет уже отправленный статус код.
func respondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Log.Error("Ошибка при отправке JSON ответа", zap.Error(err))
	}
}

// Register обрабатывает регистрацию нового пользователя.
//
// Ожидаемый JSON в теле запроса (models.RegisterRequest):
//
//	{
//	  "email": "user@example.com",
//	  "password": "securepassword",
//	  "phone": "+79001234567",
//	  "verification_type": "email|sms|both" (опционально)
//	}
//
// Поведение:
//   - Выполняет базовую валидацию (наличие email/password/phone, длина пароля).
//   - Делегирует создание пользователя в `authService.Register`.
//   - В случае успеха возвращает HTTP 201 и объект с полем `user` и (в dev)
//     `verify_code` для тестирования.
//   - В случае конфликта (пользователь уже существует) возвращает 409.
//
// Безопасность:
// - Никогда не логировать пароли в явном виде.
// - В production не возвращать `verify_code` в ответе.
//
// @Summary Регистрация нового пользователя
// @Description Создает нового пользователя и отправляет код верификации через email и/или SMS (в зависимости от verification_type: "email", "sms", "both")
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.RegisterRequest true "Данные для регистрации (verification_type опционален: email/sms/both, по умолчанию both)"
// @Success 201 {object} map[string]interface{} "user:models.User, verify_code:string"
// @Failure 400 {object} models.ErrorResponse "Неверный формат запроса"
// @Failure 409 {object} models.ErrorResponse "Пользователь уже существует"
// @Router /v1/api/auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Error("Ошибка при парсинге RegisterRequest", zap.Error(err))
		respondWithError(w, http.StatusBadRequest, "bad_request", "Неверный формат запроса")
		return
	}

	// Валидация
	if req.Email == "" || req.Password == "" || req.Phone == "" {
		respondWithError(w, http.StatusBadRequest, "validation_error", "Email, password и phone обязательны")
		return
	}

	if len(req.Password) < 8 {
		respondWithError(w, http.StatusBadRequest, "validation_error", "Пароль должен быть минимум 8 символов")
		return
	}

	user, verifyCode, err := h.authService.Register(&req)
	if err != nil {
		logger.Log.Error("Ошибка при регистрации", zap.Error(err))
		respondWithError(w, http.StatusConflict, "conflict", err.Error())
		return
	}

	logger.Log.Info("Новый пользователь зарегистрирован",
		zap.String("email", user.Email),
		zap.String("user_id", user.ID),
	)

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"user":        user,
		"verify_code": verifyCode, // В development режиме возвращаем код для тестирования
	})
}

// Verify проверяет код верификации
//
// @Summary Верификация email
// @Description Проверяет код верификации для подтверждения email и возвращает JWT токен
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.VerifyRequest true "Код верификации"
// @Success 200 {object} models.AuthResponse "Email верифицирован, токен выдан"
// @Failure 400 {object} models.ErrorResponse "Неверный код или формат запроса"
// @Router /v1/api/auth/verify [post]
func (h *AuthHandler) Verify(w http.ResponseWriter, r *http.Request) {
	// Ожидаемый JSON в теле запроса (models.VerifyRequest): {"email":"...","code":"1234"}
	var req models.VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Error("Ошибка при парсинге VerifyRequest", zap.Error(err))
		respondWithError(w, http.StatusBadRequest, "bad_request", "Неверный формат запроса")
		return
	}

	// Делегируем проверку и выдачу токена в сервисный слой.
	authResponse, err := h.authService.VerifyAndIssueToken(req.Email, req.Code)
	if err != nil {
		// Валидационные/пользовательские ошибки (не найден/неверный код) обычно
		// трактуются как Bad Request и логируются как Warn — это ожидаемое поведение.
		logger.Log.Warn("Ошибка при верификации", zap.String("email", req.Email), zap.Error(err))
		respondWithError(w, http.StatusBadRequest, "verification_failed", err.Error())
		return
	}

	logger.Log.Info("Email верифицирован и токен выдан", zap.String("email", req.Email))
	respondWithJSON(w, http.StatusOK, authResponse)
}

// Login аутентифицирует пользователя и выдает JWT
//
// @Summary Вход в систему
// @Description Аутентифицирует пользователя и возвращает JWT токены
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "Данные для входа"
// @Success 200 {object} models.AuthResponse "Успешная аутентификация"
// @Failure 400 {object} models.ErrorResponse "Неверный формат запроса"
// @Failure 401 {object} models.ErrorResponse "Неверные учетные данные"
// @Router /v1/api/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	// Ожидаемый JSON (models.LoginRequest): {"email":"...","password":"..."}
	// Поведение:
	// - Делегирует проверку учетных данных в `authService.Login`.
	// - При неуспехе возвращает 401 Unauthorized с объясняющим сообщением.
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Error("Ошибка при парсинге LoginRequest", zap.Error(err))
		respondWithError(w, http.StatusBadRequest, "bad_request", "Неверный формат запроса")
		return
	}

	authResponse, err := h.authService.Login(&req)
	if err != nil {
		// Ошибки аутентификации считаются ожидаемыми — логируем как Warn.
		logger.Log.Warn("Ошибка при входе", zap.String("email", req.Email))
		respondWithError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}

	// Устанавливаем cookie для удобства работы фронтенда
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    authResponse.AccessToken,
		Expires:  authResponse.ExpiresAt,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		// Secure: true, // TODO: включить в продакшене
	})

	logger.Log.Info("Пользователь успешно вошел", zap.String("email", req.Email))
	respondWithJSON(w, http.StatusOK, authResponse)
}

// Logout выходит из системы (инвалидирует токены)
//
// @Summary Выход из системы
// @Description Инвалидирует refresh токен пользователя
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.LogoutRequest true "Refresh токен для инвалидации"
// @Success 200 {object} map[string]string "Успешный выход"
// @Failure 400 {object} models.ErrorResponse "Неверный формат запроса"
// @Router /v1/api/auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")

	// Парсим Bearer токен из заголовка
	var token string
	if authHeader != "" && len(authHeader) > 7 {
		if authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}
	}

	// Если в заголовке нет, пробуем из тела запроса
	if token == "" {
		var req models.LogoutRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			token = req.Token
		}
	}

	// Если все еще нет, пробуем из cookie
	if token == "" {
		if cookie, err := r.Cookie("access_token"); err == nil {
			token = cookie.Value
		}
	}

	// Очищаем cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	if token == "" {
		respondWithError(w, http.StatusBadRequest, "bad_request", "Token не найден")
		return
	}

	// Попытка залогアウトить токен — сервисный слой может пометить сессию
	// как неактивную или удалить refresh токен из хранилища.
	err := h.authService.Logout(token)
	if err != nil {
		// Ошибки при выходе считаются серверными проблемами, логируются и
		// приводят к 500 для клиента с общим сообщением.
		logger.Log.Error("Ошибка при выходе", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "internal_error", "Ошибка при выходе")
		return
	}

	logger.Log.Info("Пользователь успешно вышел")
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Успешно вышли из системы",
	})
}

// GetMe возвращает информацию о текущем пользователе
//
// @Summary Получить информацию о текущем пользователе
// @Description Возвращает данные текущего авторизованного пользователя с информацией о статусе регистрации в Telegram и ролью пользователя (role: user|creator|support|admin)
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Информация о пользователе с полями: user (models.User), role (string), telegram_registered (bool), telegram_info (object, опционально)"
// @Failure 401 {object} models.ErrorResponse "Не авторизован"
// @Router /v1/api/auth/me [get]
func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	// UserID поставляется middleware AuthMiddleware
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		logger.Log.Error("Ошибка при получении пользователя", zap.String("user_id", userID), zap.Error(err))
		respondWithError(w, http.StatusNotFound, "not_found", "Пользователь не найден")
		return
	}

	// Build response with Telegram binding status (if telegramStore provided).
	response := map[string]interface{}{
		"user":                user,
		"telegram_registered": false,
	}

	// Добавляем явное поле роли пользователя для удобства клиентов.
	// Ожидается, что модель `models.User` содержит поле `Role string`.
	response["role"] = user.Role

	if h.telegramClient != nil {
		// Если telegramClient доступен, пробуем получить привязку через gRPC.
		// Ошибки при получении привязки не приводят к фейлу эндпоинта —
		// gRPC запрос non-critical, здесь мы ведем себя терпимо.
		status, err := h.telegramClient.GetBindingStatus(r.Context(), userID)
		if err == nil && status != nil && status.Status == "active" {
			response["telegram_registered"] = true
			response["telegram_info"] = map[string]interface{}{
				"username":   status.Username,
				"chat_id":    status.ChatID,
				"status":     status.Status,
				"updated_at": status.UpdatedAt,
			}
		}
	}

	// Возвращаем информацию о пользователе и (опционально) данные Telegram.
	respondWithJSON(w, http.StatusOK, response)
}

// UpdateUserRole обновляет роль пользователя (только для администраторов).
//
// Описание:
//   - Позволяет администратору назначать пользователю одну из предопределённых ролей.
//   - Допустимые значения поля `role`: `user`, `creator`, `support`, `admin`.
//   - При назначении роли `reject`/`block` не используется — используйте только
//     указанные допустимые строки. Если передано недопустимое значение — возвращается 400.
//
// Пример запроса:
//
//	{
//	  "user_id": "<user-id>",
//	  "role": "creator"
//	}
//
// @Summary Обновить роль пользователя
// @Description Назначает пользователю одну из ролей: `user`, `creator`, `support`, `admin`. Только для администраторов.
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.UpdateRoleRequest true "Данные для обновления роли (role: user|creator|support|admin)"
// @Success 200 {object} models.User "Обновленный пользователь"
// @Failure 400 {object} models.ErrorResponse "Неверный формат запроса или недопустимая роль"
// @Failure 403 {object} models.ErrorResponse "Недостаточно прав"
// @Failure 404 {object} models.ErrorResponse "Пользователь не найден"
// @Router /v1/api/admin/users/role [put]
func (h *AuthHandler) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	var req models.UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Error("Ошибка при парсинге UpdateRoleRequest", zap.Error(err))
		respondWithError(w, http.StatusBadRequest, "bad_request", "Неверный формат запроса")
		return
	}

	// Валидация роли
	validRoles := map[string]bool{
		models.RoleUser:    true,
		models.RoleCreator: true,
		models.RoleSupport: true,
		models.RoleAdmin:   true,
	}

	if !validRoles[req.Role] {
		respondWithError(w, http.StatusBadRequest, "validation_error", "Недопустимая роль")
		return
	}

	if err := h.authService.UpdateUserRole(req.UserID, req.Role); err != nil {
		logger.Log.Error("Ошибка при обновлении роли", zap.Error(err))
		respondWithError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}

	user, err := h.authService.GetUserByID(req.UserID)
	if err != nil {
		logger.Log.Error("Ошибка при получении обновленного пользователя", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "internal_error", "Ошибка при получении пользователя")
		return
	}

	logger.Log.Info("Роль пользователя обновлена",
		zap.String("user_id", req.UserID),
		zap.String("new_role", req.Role),
	)

	respondWithJSON(w, http.StatusOK, user)
}

// GetAllUsers возвращает список всех пользователей (только для администраторов)
//
// @Summary Получить список всех пользователей
// @Description Возвращает список всех пользователей системы
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.User "Список пользователей"
// @Failure 403 {object} models.ErrorResponse "Недостаточно прав"
// @Failure 500 {object} models.ErrorResponse "Внутренняя ошибка сервера"
// @Router /v1/api/admin/users [get]
func (h *AuthHandler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.authService.GetAllUsers()
	if err != nil {
		// В большинстве случаев ошибки получения списка пользователей — это
		// проблемы с БД или репозиторием. Логируем и возвращаем 500.
		logger.Log.Error("Ошибка при получении списка пользователей", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "internal_error", "Ошибка при получении пользователей")
		return
	}

	respondWithJSON(w, http.StatusOK, users)
}
