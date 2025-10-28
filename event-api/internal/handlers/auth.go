package handlers

import (
	"encoding/json"
	"net/http"

	"event-api/internal/logger"
	"event-api/internal/models"
	"event-api/internal/service"

	"go.uber.org/zap"
)

// AuthHandler управляет всеми auth endpoints
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler создает новый auth handler
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Register обрабатывает регистрацию нового пользователя
// POST /v1/api/auth/register
// @Summary Регистрация нового пользователя
// @Description Создает нового пользователя и отправляет код верификации
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.RegisterRequest true "Данные для регистрации"
// @Success 201 {object} map[string]interface{} "user:models.User, verify_code:string"
// @Failure 400 {object} models.ErrorResponse "Неверный формат запроса"
// @Failure 409 {object} models.ErrorResponse "Пользователь уже существует"
// @Router /v1/api/auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Error("Ошибка при парсинге RegisterRequest", zap.Error(err))
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
	if req.Email == "" || req.Password == "" || req.Phone == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse{
			Error:   "validation_error",
			Message: "Email, password и phone обязательны",
			Code:    http.StatusBadRequest,
		})
		return
	}

	if len(req.Password) < 8 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse{
			Error:   "validation_error",
			Message: "Пароль должен быть минимум 8 символов",
			Code:    http.StatusBadRequest,
		})
		return
	}

	user, verifyCode, err := h.authService.Register(&req)
	if err != nil {
		logger.Log.Error("Ошибка при регистрации", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(models.ErrorResponse{
			Error:   "conflict",
			Message: err.Error(),
			Code:    http.StatusConflict,
		})
		return
	}

	logger.Log.Info("Новый пользователь зарегистрирован",
		zap.String("email", user.Email),
		zap.String("user_id", user.ID),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user":        user,
		"verify_code": verifyCode, // В production отправляем через email/SMS
	})
}

// Verify проверяет код верификации
// POST /v1/api/auth/verify
// @Summary Верификация email
// @Description Проверяет код верификации для подтверждения email
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.VerifyRequest true "Код верификации"
// @Success 200 {object} models.VerifyResponse "Email успешно верифицирован"
// @Failure 400 {object} models.ErrorResponse "Неверный код или формат запроса"
// @Router /v1/api/auth/verify [post]
func (h *AuthHandler) Verify(w http.ResponseWriter, r *http.Request) {
	var req models.VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Error("Ошибка при парсинге VerifyRequest", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse{
			Error:   "bad_request",
			Message: "Неверный формат запроса",
			Code:    http.StatusBadRequest,
		})
		return
	}

	err := h.authService.VerifyCode(req.Email, req.Code)
	if err != nil {
		logger.Log.Warn("Ошибка при верификации", zap.String("email", req.Email), zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse{
			Error:   "verification_failed",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	logger.Log.Info("Email верифицирован", zap.String("email", req.Email))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(models.VerifyResponse{
		Message:  "Email успешно верифицирован",
		Verified: true,
	})
}

// Login аутентифицирует пользователя и выдает JWT
// POST /v1/api/auth/login
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
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Error("Ошибка при парсинге LoginRequest", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse{
			Error:   "bad_request",
			Message: "Неверный формат запроса",
			Code:    http.StatusBadRequest,
		})
		return
	}

	authResponse, err := h.authService.Login(&req)
	if err != nil {
		logger.Log.Warn("Ошибка при входе", zap.String("email", req.Email), zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(models.ErrorResponse{
			Error:   "unauthorized",
			Message: err.Error(),
			Code:    http.StatusUnauthorized,
		})
		return
	}

	logger.Log.Info("Пользователь успешно вошел", zap.String("email", req.Email))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(authResponse)
}

// Logout отзывает JWT токен
// POST /v1/api/auth/logout
// @Summary Выход из системы
// @Description Отзывает JWT токен пользователя
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer токен"
// @Param request body models.LogoutRequest false "Токен в теле запроса (опционально)"
// @Success 200 {object} map[string]interface{} "message:string"
// @Failure 400 {object} models.ErrorResponse "Token не найден"
// @Failure 500 {object} models.ErrorResponse "Внутренняя ошибка сервера"
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

	if token == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse{
			Error:   "bad_request",
			Message: "Token не найден",
			Code:    http.StatusBadRequest,
		})
		return
	}

	err := h.authService.Logout(token)
	if err != nil {
		logger.Log.Error("Ошибка при выходе", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse{
			Error:   "internal_error",
			Message: "Ошибка при выходе",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	logger.Log.Info("Пользователь успешно вышел")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Успешно вышли из системы",
	})
}

// GetMe возвращает текущего пользователя
// GET /v1/api/auth/me
// @Summary Получить текущего пользователя
// @Description Возвращает информацию о текущем аутентифицированном пользователе
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.User "Информация о пользователе"
// @Failure 401 {object} models.ErrorResponse "Пользователь не аутентифицирован"
// @Failure 404 {object} models.ErrorResponse "Пользователь не найден"
// @Router /v1/api/auth/me [get]
func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	// UserID поставляется middleware AuthMiddleware
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not found in context",
			Code:    http.StatusUnauthorized,
		})
		return
	}

	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		logger.Log.Error("Ошибка при получении пользователя", zap.String("user_id", userID), zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.ErrorResponse{
			Error:   "not_found",
			Message: "Пользователь не найден",
			Code:    http.StatusNotFound,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}
