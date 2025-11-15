// Package handlers содержит HTTP обработчики для аутентификации и управления пользователями.
package handlers

import (
	"encoding/json"
	"net/http"

	"event-api/internal/logger"
	"event-api/internal/models"

	"go.uber.org/zap"
)

// AuthService описывает необходимые методы для обслуживания auth endpoints.
type AuthService interface {
	Register(*models.RegisterRequest) (*models.User, string, error)
	VerifyAndIssueToken(email, code string) (*models.AuthResponse, error)
	Login(*models.LoginRequest) (*models.AuthResponse, error)
	Logout(token string) error
	GetUserByID(userID string) (*models.User, error)
}

// AuthHandler управляет всеми auth endpoints.
type AuthHandler struct {
	authService AuthService
}

// NewAuthHandler создает новый auth handler.
func NewAuthHandler(authService AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// respondWithError отправляет JSON ответ с ошибкой
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

// respondWithJSON отправляет JSON ответ с данными
func respondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Log.Error("Ошибка при отправке JSON ответа", zap.Error(err))
	}
}

// Register обрабатывает регистрацию нового пользователя
// @Summary Регистрация нового пользователя
// @Description Создает нового пользователя и отправляет код верификации через email и/или SMS (в зависимости от verification_type: "email", "sms", "both")
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.RegisterRequest true "Данные для регистрации (verification_type опционален: email/sms/both, по умолчанию both)"
// @Success 201 {object} map[string]interface{} "user:models.User, verify_code:string"
// @Failure 400 {object} models.ErrorResponse "Неверный формат запроса"
// @Failure 409 {object} models.ErrorResponse "Пользователь уже существует"
// @Router /api/auth/register [post]
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
// @Summary Верификация email
// @Description Проверяет код верификации для подтверждения email и возвращает JWT токен
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.VerifyRequest true "Код верификации"
// @Success 200 {object} models.AuthResponse "Email верифицирован, токен выдан"
// @Failure 400 {object} models.ErrorResponse "Неверный код или формат запроса"
// @Router /api/auth/verify [post]
func (h *AuthHandler) Verify(w http.ResponseWriter, r *http.Request) {
	var req models.VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Error("Ошибка при парсинге VerifyRequest", zap.Error(err))
		respondWithError(w, http.StatusBadRequest, "bad_request", "Неверный формат запроса")
		return
	}

	authResponse, err := h.authService.VerifyAndIssueToken(req.Email, req.Code)
	if err != nil {
		logger.Log.Warn("Ошибка при верификации", zap.String("email", req.Email), zap.Error(err))
		respondWithError(w, http.StatusBadRequest, "verification_failed", err.Error())
		return
	}

	logger.Log.Info("Email верифицирован и токен выдан", zap.String("email", req.Email))
	respondWithJSON(w, http.StatusOK, authResponse)
}

// Login аутентифицирует пользователя и выдает JWT
// @Summary Вход в систему
// @Description Аутентифицирует пользователя и возвращает JWT токены
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "Данные для входа"
// @Success 200 {object} models.AuthResponse "Успешная аутентификация"
// @Failure 400 {object} models.ErrorResponse "Неверный формат запроса"
// @Failure 401 {object} models.ErrorResponse "Неверные учетные данные"
// @Router /api/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Error("Ошибка при парсинге LoginRequest", zap.Error(err))
		respondWithError(w, http.StatusBadRequest, "bad_request", "Неверный формат запроса")
		return
	}

	authResponse, err := h.authService.Login(&req)
	if err != nil {
		logger.Log.Warn("Ошибка при входе", zap.String("email", req.Email))
		respondWithError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}

	logger.Log.Info("Пользователь успешно вошел", zap.String("email", req.Email))
	respondWithJSON(w, http.StatusOK, authResponse)
}

// Logout выходит из системы (инвалидирует токены)
// @Summary Выход из системы
// @Description Инвалидирует refresh токен пользователя
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.LogoutRequest true "Refresh токен для инвалидации"
// @Success 200 {object} map[string]string "Успешный выход"
// @Failure 400 {object} models.ErrorResponse "Неверный формат запроса"
// @Router /api/auth/logout [post]
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

	if token == "" {
		respondWithError(w, http.StatusBadRequest, "bad_request", "Token не найден")
		return
	}

	err := h.authService.Logout(token)
	if err != nil {
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
// @Summary Получить информацию о текущем пользователе
// @Description Возвращает данные текущего авторизованного пользователя
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.User "Информация о пользователе"
// @Failure 401 {object} models.ErrorResponse "Не авторизован"
// @Router /api/auth/me [get]
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

	respondWithJSON(w, http.StatusOK, user)
}
