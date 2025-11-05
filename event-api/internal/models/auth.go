package models

import "time"

// User представляет пользователя в системе.
type User struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Password  string    `json:"-"`
	Verified  bool      `json:"verified"`
}

// RegisterRequest - запрос для регистрации.
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
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
	Verified bool   `json:"verified"`
	Exp      int64  `json:"exp"`
	Iat      int64  `json:"iat"`
}
