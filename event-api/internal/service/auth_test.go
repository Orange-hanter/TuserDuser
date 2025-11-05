package service

import (
	"testing"

	"event-api/internal/config"
	"event-api/internal/models"
	"event-api/internal/worker"

	"go.uber.org/zap"
)

func TestRegister(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:     "test-secret",
		JWTExpiration: 3600,
	}
	logger, _ := zap.NewDevelopment()
	workerPool := worker.NewPool(2, 10, logger)
	workerPool.Start()
	defer workerPool.Shutdown()

	authService := NewAuthService(cfg, nil, nil, nil, workerPool, logger)

	tests := []struct {
		name      string
		email     string
		phone     string
		password  string
		wantError bool
	}{
		{
			name:      "valid registration",
			email:     "test@example.com",
			phone:     "+79991234567",
			password:  "password123",
			wantError: false,
		},
		{
			name:      "duplicate email",
			email:     "test@example.com",
			phone:     "+79991234568",
			password:  "password123",
			wantError: true,
		},
		{
			name:      "short password",
			email:     "test2@example.com",
			phone:     "+79991234567",
			password:  "short",
			wantError: false, // Service не проверяет длину
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, code, err := authService.Register(&models.RegisterRequest{
				Email:    tt.email,
				Phone:    tt.phone,
				Password: tt.password,
			})

			if (err != nil) != tt.wantError {
				t.Errorf("Register() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError {
				if user == nil {
					t.Errorf("Register() user is nil")
				}
				if code == "" {
					t.Errorf("Register() code is empty")
				}
				if user.Email != tt.email {
					t.Errorf("Register() email mismatch: got %v, want %v", user.Email, tt.email)
				}
				if user.Verified {
					t.Errorf("Register() user should not be verified")
				}
			}
		})
	}
}

func TestVerifyCode(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:     "test-secret",
		JWTExpiration: 3600,
	}
	logger, _ := zap.NewDevelopment()
	workerPool := worker.NewPool(2, 10, logger)
	workerPool.Start()
	defer workerPool.Shutdown()

	authService := NewAuthService(cfg, nil, nil, nil, workerPool, logger)

	// Регистрируем пользователя
	user, verifyCode, err := authService.Register(&models.RegisterRequest{
		Email:    "verify@example.com",
		Phone:    "+79991234567",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	tests := []struct {
		name      string
		email     string
		code      string
		wantError bool
	}{
		{
			name:      "valid verification",
			email:     user.Email,
			code:      verifyCode,
			wantError: false,
		},
		{
			name:      "wrong code",
			email:     user.Email,
			code:      "000000",
			wantError: true,
		},
		{
			name:      "non-existent email",
			email:     "nonexistent@example.com",
			code:      verifyCode,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := authService.VerifyCode(tt.email, tt.code)
			if (err != nil) != tt.wantError {
				t.Errorf("VerifyCode() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestLogin(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:     "test-secret",
		JWTExpiration: 3600,
	}
	logger, _ := zap.NewDevelopment()
	workerPool := worker.NewPool(2, 10, logger)
	workerPool.Start()
	defer workerPool.Shutdown()

	authService := NewAuthService(cfg, nil, nil, nil, workerPool, logger)

	// Регистрируем и верифицируем пользователя
	user, verifyCode, _ := authService.Register(&models.RegisterRequest{
		Email:    "login@example.com",
		Phone:    "+79991234567",
		Password: "password123",
	})
	authService.VerifyCode(user.Email, verifyCode)

	tests := []struct {
		name      string
		email     string
		password  string
		wantError bool
	}{
		{
			name:      "valid login",
			email:     "login@example.com",
			password:  "password123",
			wantError: false,
		},
		{
			name:      "wrong password",
			email:     "login@example.com",
			password:  "wrongpassword",
			wantError: true,
		},
		{
			name:      "non-existent user",
			email:     "nonexistent@example.com",
			password:  "password123",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := authService.Login(&models.LoginRequest{
				Email:    tt.email,
				Password: tt.password,
			})

			if (err != nil) != tt.wantError {
				t.Errorf("Login() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError {
				if response == nil {
					t.Errorf("Login() response is nil")
				}
				if response.AccessToken == "" {
					t.Errorf("Login() access_token is empty")
				}
				if response.User == nil {
					t.Errorf("Login() user is nil")
				}
			}
		})
	}
}

func TestLogout(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:     "test-secret",
		JWTExpiration: 3600,
	}
	logger, _ := zap.NewDevelopment()
	workerPool := worker.NewPool(2, 10, logger)
	workerPool.Start()
	defer workerPool.Shutdown()

	authService := NewAuthService(cfg, nil, nil, nil, workerPool, logger)

	// Регистрируем и логируемся
	user, verifyCode, _ := authService.Register(&models.RegisterRequest{
		Email:    "logout@example.com",
		Phone:    "+79991234567",
		Password: "password123",
	})
	authService.VerifyCode(user.Email, verifyCode)
	response, _ := authService.Login(&models.LoginRequest{
		Email:    user.Email,
		Password: "password123",
	})

	token := response.AccessToken

	// Тестируем logout
	err := authService.Logout(token)
	if err != nil {
		t.Errorf("Logout() failed: %v", err)
	}

	// Проверяем, что токен в черном списке
	if !authService.IsTokenBlacklisted(token) {
		t.Errorf("Logout() token not in blacklist")
	}
}

func TestGetUserByID(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:     "test-secret",
		JWTExpiration: 3600,
	}
	logger, _ := zap.NewDevelopment()
	workerPool := worker.NewPool(2, 10, logger)
	workerPool.Start()
	defer workerPool.Shutdown()

	authService := NewAuthService(cfg, nil, nil, nil, workerPool, logger)

	// Регистрируем пользователя
	user, _, _ := authService.Register(&models.RegisterRequest{
		Email:    "getuser@example.com",
		Phone:    "+79991234567",
		Password: "password123",
	})

	tests := []struct {
		name      string
		userID    string
		wantError bool
	}{
		{
			name:      "valid user id",
			userID:    user.ID,
			wantError: false,
		},
		{
			name:      "non-existent user id",
			userID:    "nonexistent-id",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrievedUser, err := authService.GetUserByID(tt.userID)
			if (err != nil) != tt.wantError {
				t.Errorf("GetUserByID() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError {
				if retrievedUser == nil {
					t.Errorf("GetUserByID() user is nil")
				}
				if retrievedUser.Email != user.Email {
					t.Errorf("GetUserByID() email mismatch: got %v, want %v", retrievedUser.Email, user.Email)
				}
			}
		})
	}
}

func TestValidateJWT(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:     "test-secret",
		JWTExpiration: 3600,
	}
	logger, _ := zap.NewDevelopment()
	workerPool := worker.NewPool(2, 10, logger)
	workerPool.Start()
	defer workerPool.Shutdown()

	authService := NewAuthService(cfg, nil, nil, nil, workerPool, logger)

	// Регистрируем и логируемся
	user, verifyCode, _ := authService.Register(&models.RegisterRequest{
		Email:    "jwt@example.com",
		Phone:    "+79991234567",
		Password: "password123",
	})
	authService.VerifyCode(user.Email, verifyCode)
	response, _ := authService.Login(&models.LoginRequest{
		Email:    user.Email,
		Password: "password123",
	})

	validToken := response.AccessToken

	tests := []struct {
		name      string
		token     string
		wantError bool
	}{
		{
			name:      "valid token",
			token:     validToken,
			wantError: false,
		},
		{
			name:      "invalid token format",
			token:     "invalid.token.format",
			wantError: true,
		},
		{
			name:      "empty token",
			token:     "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := authService.ValidateJWT(tt.token)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateJWT() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError && claims == nil {
				t.Errorf("ValidateJWT() claims is nil")
			}
		})
	}
}
