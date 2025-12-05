// Package models содержит модели данных, используемые в системе аутентификации.
package models

import "time"

// Роли пользователей в системе.
const (
	RoleUser    = "user"    // Обычный пользователь - может просматривать события
	RoleCreator = "creator" // Создатель - может создавать события
	RoleSupport = "support" // Поддержка - зарезервировано для будущего функционала
	RoleAdmin   = "admin"   // Администратор - полные права
)

// User представляет пользователя в системе.
type User struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Password  string    `json:"-"`
	Role      string    `json:"role"`
	Verified  bool      `json:"verified"`
}

// RegisterRequest - запрос для регистрации.
type RegisterRequest struct {
	Email            string `json:"email" binding:"required,email"`
	Phone            string `json:"phone"` // Опционально, если не указан - отправляем нуль
	Password         string `json:"password" binding:"required,min=8"`
	VerificationType string `json:"verification_type,omitempty"` // "email", "sms", "both" (default: "both")
}

// VerifyRequest - запрос для верификации кода.
type VerifyRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6"`
}

// LoginRequest - запрос для входа.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// CheckUserExistsRequest - запрос для проверки существования пользователя.
type CheckUserExistsRequest struct {
	Email string `json:"email,omitempty"`
	Phone string `json:"phone,omitempty"`
}

// CheckUserExistsResponse - ответ на проверку существования пользователя.
type CheckUserExistsResponse struct {
	Exists       bool   `json:"exists"`
	ConflictType string `json:"conflict_type,omitempty"` // "email", "phone", "both"
	Message      string `json:"message"`
}

// AuthResponse - ответ с JWT токеном.
type AuthResponse struct {
	ExpiresAt    time.Time `json:"expires_at"`
	User         *User     `json:"user"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresIn    int64     `json:"expires_in"`
}

// VerifyResponse - ответ на верификацию.
type VerifyResponse struct {
	Message  string `json:"message"`
	Verified bool   `json:"verified"`
}

// LogoutRequest - запрос для выхода.
type LogoutRequest struct {
	Token string `json:"token,omitempty"`
}

// ResendCodeRequest - запрос для повторной отправки кода верификации.
type ResendCodeRequest struct {
	Email            string `json:"email" binding:"required,email"`
	VerificationType string `json:"verification_type" binding:"required"` // "email", "telegram", "sms"
}

// ResendCodeResponse - ответ на повторную отправку кода верификации.
type ResendCodeResponse struct {
	Message    string `json:"message"`
	ExpiresIn  int    `json:"expires_in"`            // Время действия кода в секундах
	VerifyCode string `json:"verify_code,omitempty"` // Только в dev режиме
	RetryAfter int    `json:"retry_after,omitempty"` // Для rate limit ошибок
}

// TelegramBindingInfo - информация для привязки Telegram при регистрации.
type TelegramBindingInfo struct {
	Deeplink  string `json:"deeplink"`   // Полная ссылка для открытия в Telegram
	Code      string `json:"code"`       // 6-символьный код для ручного ввода
	ExpiresAt string `json:"expires_at"` // RFC3339 формат времени истечения
}

// RegisterResponse - расширенный ответ на регистрацию.
type RegisterResponse struct {
	User            *User                `json:"user"`
	VerifyCode      string               `json:"verify_code,omitempty"`      // Только в dev режиме
	TelegramBinding *TelegramBindingInfo `json:"telegram_binding,omitempty"` // При verification_type=telegram
}

// ErrorResponse - стандартный ответ об ошибке.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// Claims - пользовательские claims для JWT.
type Claims struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Role     string `json:"role"`
	Verified bool   `json:"verified"`
	Exp      int64  `json:"exp"`
	Iat      int64  `json:"iat"`
}
