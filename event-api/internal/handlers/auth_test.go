package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"event-api/internal/logger"
	"event-api/internal/models"

	"go.uber.org/zap"
)

type mockAuthService struct {
	registerFn    func(*models.RegisterRequest) (*models.User, string, error)
	verifyFn      func(string, string) (*models.AuthResponse, error)
	loginFn       func(*models.LoginRequest) (*models.AuthResponse, error)
	logoutFn      func(string) error
	getUserFn     func(string) (*models.User, error)
	updateRoleFn  func(string, string) error
	getAllUsersFn func() ([]*models.User, error)
}

func (m *mockAuthService) Register(req *models.RegisterRequest) (*models.User, string, error) {
	if m.registerFn != nil {
		return m.registerFn(req)
	}
	return nil, "", errors.New("register not implemented")
}

func (m *mockAuthService) VerifyAndIssueToken(email, code string) (*models.AuthResponse, error) {
	if m.verifyFn != nil {
		return m.verifyFn(email, code)
	}
	return nil, errors.New("verify not implemented")
}

func (m *mockAuthService) Login(req *models.LoginRequest) (*models.AuthResponse, error) {
	if m.loginFn != nil {
		return m.loginFn(req)
	}
	return nil, errors.New("login not implemented")
}

func (m *mockAuthService) Logout(token string) error {
	if m.logoutFn != nil {
		return m.logoutFn(token)
	}
	return errors.New("logout not implemented")
}

func (m *mockAuthService) GetUserByID(userID string) (*models.User, error) {
	if m.getUserFn != nil {
		return m.getUserFn(userID)
	}
	return nil, errors.New("getUser not implemented")
}

func (m *mockAuthService) UpdateUserRole(userID, role string) error {
	if m.updateRoleFn != nil {
		return m.updateRoleFn(userID, role)
	}
	return errors.New("updateRole not implemented")
}

func (m *mockAuthService) GetAllUsers() ([]*models.User, error) {
	if m.getAllUsersFn != nil {
		return m.getAllUsersFn()
	}
	return nil, errors.New("getAllUsers not implemented")
}

func TestRegisterHandler(t *testing.T) {
	logger.Log = zap.NewNop()

	validUser := &models.User{ID: "u1", Email: "test@example.com"}
	mockSvc := &mockAuthService{
		registerFn: func(req *models.RegisterRequest) (*models.User, string, error) {
			if req.Email != validUser.Email {
				t.Fatalf("unexpected email: %s", req.Email)
			}
			return validUser, "123456", nil
		},
	}
	handler := NewAuthHandler(mockSvc, nil)

	t.Run("valid registration", func(t *testing.T) {
		payload := models.RegisterRequest{
			Email:    validUser.Email,
			Phone:    "+79991234567",
			Password: "password123",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.Register(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d", w.Code)
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp["verify_code"] != "123456" {
			t.Fatalf("expected verify_code 123456, got %v", resp["verify_code"])
		}
	})

	t.Run("validation error", func(t *testing.T) {
		called := false
		reqPayload := models.RegisterRequest{Phone: "+79991234567", Password: "password123"}
		body, _ := json.Marshal(reqPayload)
		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
		w := httptest.NewRecorder()

		mockSvc.registerFn = func(req *models.RegisterRequest) (*models.User, string, error) {
			called = true
			return nil, "", nil
		}

		handler.Register(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
		if called {
			t.Fatalf("service should not be called on validation error")
		}
	})
}

func TestVerifyHandler(t *testing.T) {
	logger.Log = zap.NewNop()
	mockSvc := &mockAuthService{}
	handler := NewAuthHandler(mockSvc, nil)

	t.Run("valid verification", func(t *testing.T) {
		mockSvc.verifyFn = func(email, code string) (*models.AuthResponse, error) {
			return &models.AuthResponse{AccessToken: "token"}, nil
		}

		payload := models.VerifyRequest{Email: "verify@example.com", Code: "123456"}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/auth/verify", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.Verify(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("verification failed", func(t *testing.T) {
		mockSvc.verifyFn = func(email, code string) (*models.AuthResponse, error) {
			return nil, errors.New("invalid code")
		}

		payload := models.VerifyRequest{Email: "verify@example.com", Code: "000"}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/auth/verify", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.Verify(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})
}

func TestLoginHandler(t *testing.T) {
	logger.Log = zap.NewNop()
	mockSvc := &mockAuthService{}
	handler := NewAuthHandler(mockSvc, nil)

	t.Run("login success", func(t *testing.T) {
		mockSvc.loginFn = func(req *models.LoginRequest) (*models.AuthResponse, error) {
			if req.Email != "login@example.com" {
				t.Fatalf("unexpected email %s", req.Email)
			}
			return &models.AuthResponse{AccessToken: "token"}, nil
		}

		payload := models.LoginRequest{Email: "login@example.com", Password: "secret"}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.Login(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("login failure", func(t *testing.T) {
		mockSvc.loginFn = func(req *models.LoginRequest) (*models.AuthResponse, error) {
			return nil, errors.New("bad creds")
		}

		payload := models.LoginRequest{Email: "login@example.com", Password: "wrong"}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.Login(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

func TestLogoutHandler(t *testing.T) {
	logger.Log = zap.NewNop()
	mockSvc := &mockAuthService{}
	handler := NewAuthHandler(mockSvc, nil)

	t.Run("logout success via header", func(t *testing.T) {
		called := false
		mockSvc.logoutFn = func(token string) error {
			called = true
			if token != "token" {
				t.Fatalf("unexpected token %s", token)
			}
			return nil
		}

		req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", bytes.NewReader([]byte(`{}`)))
		req.Header.Set("Authorization", "Bearer token")
		w := httptest.NewRecorder()

		handler.Logout(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if !called {
			t.Fatalf("logout should be called")
		}
	})

	t.Run("missing token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", bytes.NewReader([]byte(`{}`)))
		w := httptest.NewRecorder()

		handler.Logout(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("logout failure", func(t *testing.T) {
		mockSvc.logoutFn = func(token string) error {
			return errors.New("redis down")
		}

		req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", bytes.NewReader([]byte(`{"token":"tk"}`)))
		w := httptest.NewRecorder()

		handler.Logout(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestGetMeHandler(t *testing.T) {
	logger.Log = zap.NewNop()
	mockSvc := &mockAuthService{}
	handler := NewAuthHandler(mockSvc, nil)

	t.Run("user found", func(t *testing.T) {
		mockSvc.getUserFn = func(userID string) (*models.User, error) {
			if userID != "user-1" {
				t.Fatalf("unexpected userID %s", userID)
			}
			return &models.User{ID: userID, Email: "me@example.com", Role: "user"}, nil
		}

		req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
		req.Header.Set("X-User-ID", "user-1")
		w := httptest.NewRecorder()

		handler.GetMe(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var resp struct {
			User               *models.User `json:"user"`
			Role               string       `json:"role"`
			TelegramRegistered bool         `json:"telegram_registered"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.User == nil || resp.User.Email != "me@example.com" {
			t.Fatalf("unexpected user payload: %+v", resp.User)
		}
		if resp.Role == "" {
			t.Fatalf("expected role to be propagated")
		}
	})

	t.Run("missing header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
		w := httptest.NewRecorder()

		handler.GetMe(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("user lookup failure", func(t *testing.T) {
		mockSvc.getUserFn = func(userID string) (*models.User, error) {
			return nil, errors.New("not found")
		}

		req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
		req.Header.Set("X-User-ID", "user-1")
		w := httptest.NewRecorder()

		handler.GetMe(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}
