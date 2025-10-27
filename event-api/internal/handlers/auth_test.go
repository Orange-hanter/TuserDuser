package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"event-api/internal/config"
	"event-api/internal/logger"
	"event-api/internal/models"
	"event-api/internal/service"
	"go.uber.org/zap"
)

func init() {
	// Initialize logger for tests
	logger.Log, _ = zap.NewDevelopment()
}

func TestRegisterHandler(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test", JWTExpiration: 3600}
	authService := service.NewAuthService(cfg)
	handler := NewAuthHandler(authService)

	tests := []struct {
		name           string
		payload        models.RegisterRequest
		expectedStatus int
	}{
		{
			name: "valid registration",
			payload: models.RegisterRequest{
				Email:    "test@example.com",
				Phone:    "+79991234567",
				Password: "password123",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "missing email",
			payload: models.RegisterRequest{
				Phone:    "+79991234567",
				Password: "password123",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handler.Register(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Register() status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestVerifyHandler(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test", JWTExpiration: 3600}
	authService := service.NewAuthService(cfg)
	handler := NewAuthHandler(authService)

	// Регистрируем пользователя для получения кода
	user, code, _ := authService.Register(&models.RegisterRequest{
		Email:    "verify@example.com",
		Phone:    "+79991234567",
		Password: "password123",
	})

	tests := []struct {
		name           string
		payload        models.VerifyRequest
		expectedStatus int
	}{
		{
			name: "valid verification",
			payload: models.VerifyRequest{
				Email: user.Email,
				Code:  code,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "wrong code",
			payload: models.VerifyRequest{
				Email: user.Email,
				Code:  "000000",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "/api/auth/verify", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handler.Verify(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Verify() status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestLoginHandler(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test", JWTExpiration: 3600}
	authService := service.NewAuthService(cfg)
	handler := NewAuthHandler(authService)

	// Регистрируем и верифицируем пользователя
	user, code, _ := authService.Register(&models.RegisterRequest{
		Email:    "login@example.com",
		Phone:    "+79991234567",
		Password: "password123",
	})
	authService.VerifyCode(user.Email, code)

	tests := []struct {
		name           string
		payload        models.LoginRequest
		expectedStatus int
	}{
		{
			name: "valid login",
			payload: models.LoginRequest{
				Email:    "login@example.com",
				Password: "password123",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "wrong password",
			payload: models.LoginRequest{
				Email:    "login@example.com",
				Password: "wrongpassword",
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handler.Login(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Login() status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestLogoutHandler(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test", JWTExpiration: 3600}
	authService := service.NewAuthService(cfg)
	handler := NewAuthHandler(authService)

	// Регистрируем, верифицируем и логируемся
	user, code, _ := authService.Register(&models.RegisterRequest{
		Email:    "logout@example.com",
		Phone:    "+79991234567",
		Password: "password123",
	})
	authService.VerifyCode(user.Email, code)
	response, _ := authService.Login(&models.LoginRequest{
		Email:    user.Email,
		Password: "password123",
	})

	t.Run("valid logout", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{})
		req := httptest.NewRequest("POST", "/api/auth/logout", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+response.AccessToken)
		w := httptest.NewRecorder()

		handler.Logout(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Logout() status = %v, want %v", w.Code, http.StatusOK)
		}
	})
}

func TestGetMeHandler(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test", JWTExpiration: 3600}
	authService := service.NewAuthService(cfg)
	handler := NewAuthHandler(authService)

	// Регистрируем, верифицируем и логируемся
	user, code, _ := authService.Register(&models.RegisterRequest{
		Email:    "getme@example.com",
		Phone:    "+79991234567",
		Password: "password123",
	})
	authService.VerifyCode(user.Email, code)
	authService.Login(&models.LoginRequest{
		Email:    user.Email,
		Password: "password123",
	})

	t.Run("valid get me", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/me", nil)
		req.Header.Set("X-User-ID", user.ID)
		w := httptest.NewRecorder()

		handler.GetMe(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("GetMe() status = %v, want %v", w.Code, http.StatusOK)
		}

		var resp models.User
		json.NewDecoder(w.Body).Decode(&resp)

		if resp.Email != user.Email {
			t.Errorf("GetMe() email mismatch: got %v, want %v", resp.Email, user.Email)
		}
	})

	t.Run("missing user id", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/me", nil)
		w := httptest.NewRecorder()

		handler.GetMe(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("GetMe() status = %v, want %v", w.Code, http.StatusUnauthorized)
		}
	})
}
