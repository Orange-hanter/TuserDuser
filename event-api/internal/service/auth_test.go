package service

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"testing"
	"time"

	"event-api/internal/config"
	"event-api/internal/email"
	"event-api/internal/models"
	redisClient "event-api/internal/redis"
	"event-api/internal/sms"
	"event-api/internal/worker"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// setupTestAuthService creates an AuthService with mocked dependencies for testing.
func setupTestAuthService(t *testing.T) (*AuthService, sqlmock.Sqlmock, *miniredis.Miniredis, func()) {
	t.Helper()

	logger := zap.NewNop()

	// Setup SQL mock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}

	// Setup Redis mock using miniredis
	mr := miniredis.RunT(t)

	redisClientWrapper, err := redisClient.NewClient(&redisClient.Config{
		Host:     mr.Host(),
		Port:     mr.Port(),
		Password: "",
		DB:       0,
	}, logger)
	if err != nil {
		t.Fatalf("Failed to create Redis client: %v", err)
	}

	smsService, err := sms.NewService(&sms.Config{Provider: "mock"}, logger)
	if err != nil {
		t.Fatalf("Failed to create SMS service: %v", err)
	}

	emailService, err := email.NewService(&email.Config{Provider: "mock"}, logger)
	if err != nil {
		t.Fatalf("Failed to create Email service: %v", err)
	}

	workerPool := worker.NewPool(2, 10, logger)
	workerPool.Start()

	cfg := &config.Config{
		JWTSecret:     "test-secret",
		JWTExpiration: 3600,
	}

	authService := NewAuthService(cfg, db, redisClientWrapper, smsService, emailService, workerPool, logger)

	cleanup := func() {
		db.Close()
		mr.Close()
		redisClientWrapper.Close()
		workerPool.Shutdown()
	}

	return authService, mock, mr, cleanup
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name       string
		req        models.RegisterRequest
		wantError  bool
		setupMock  func(sqlmock.Sqlmock)
		setupRedis func(*miniredis.Miniredis)
		verify     func(t *testing.T, svc *AuthService, email, code string)
	}{
		{
			name: "valid registration stores pending user",
			req: models.RegisterRequest{
				Email:    "test@example.com",
				Phone:    "+79991234567",
				Password: "password123",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM users WHERE email = $1")).
					WithArgs("test@example.com").
					WillReturnError(sql.ErrNoRows)
			},
			verify: func(t *testing.T, svc *AuthService, email, code string) {
				pending, err := svc.loadPendingUser(context.Background(), email)
				if err != nil {
					t.Fatalf("expected pending user in redis: %v", err)
				}
				if pending.VerificationCode != code {
					t.Fatalf("verification code mismatch: got %s want %s", pending.VerificationCode, code)
				}
			},
		},
		{
			name: "duplicate email",
			req: models.RegisterRequest{
				Email:    "duplicate@example.com",
				Phone:    "+79991234568",
				Password: "password123",
			},
			wantError: true,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id"}).AddRow("existing-user-id")
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM users WHERE email = $1")).
					WithArgs("duplicate@example.com").
					WillReturnRows(rows)
			},
		},
		{
			name: "pending registration blocks re-register",
			req: models.RegisterRequest{
				Email:    "pending@example.com",
				Phone:    "+79991234569",
				Password: "password123",
			},
			wantError: true,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM users WHERE email = $1")).
					WithArgs("pending@example.com").
					WillReturnError(sql.ErrNoRows)
			},
			setupRedis: func(mr *miniredis.Miniredis) {
				mr.Set(pendingUserKey("pending@example.com"), "stub")
			},
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			svc, mock, mr, cleanup := setupTestAuthService(t)
			defer cleanup()

			if tc.setupRedis != nil {
				tc.setupRedis(mr)
			}
			if tc.setupMock != nil {
				tc.setupMock(mock)
			}

			user, code, err := svc.Register(&tc.req)

			if tc.wantError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				verifyMockExpectations(t, mock)
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if user == nil || user.Email != normalizeEmail(tc.req.Email) {
				t.Fatalf("unexpected user payload: %+v", user)
			}
			if code == "" {
				t.Fatalf("expected verification code")
			}
			if tc.verify != nil {
				tc.verify(t, svc, normalizeEmail(tc.req.Email), code)
			}

			verifyMockExpectations(t, mock)
		})
	}
}

func TestVerifyCodePendingUser(t *testing.T) {
	tests := []struct {
		name      string
		code      string
		wantError bool
		setupMock func(sqlmock.Sqlmock, *pendingUser)
		validate  func(t *testing.T, mr *miniredis.Miniredis, email string)
	}{
		{
			name: "valid code persists user",
			code: "654321",
			setupMock: func(mock sqlmock.Sqlmock, pending *pendingUser) {
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users (id, email, phone, password, role, verified, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)")).
					WithArgs(pending.ID, pending.Email, pending.Phone, pending.Password, pending.Role, true, sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			validate: func(t *testing.T, mr *miniredis.Miniredis, email string) {
				if mr.Exists(pendingUserKey(email)) {
					t.Fatalf("pending user key should be deleted")
				}
			},
		},
		{
			name:      "wrong code keeps pending user",
			code:      "000000",
			wantError: true,
			validate: func(t *testing.T, mr *miniredis.Miniredis, email string) {
				if !mr.Exists(pendingUserKey(email)) {
					t.Fatalf("pending user key should remain for wrong code")
				}
			},
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			svc, mock, mr, cleanup := setupTestAuthService(t)
			defer cleanup()

			pending := &pendingUser{
				ID:                "pending-id",
				Email:             "verify@example.com",
				Phone:             "+79991230000",
				Password:          mustHashPassword(t, "password123"),
				Role:              models.RoleUser,
				VerificationType:  "both",
				VerificationCode:  "654321",
				CreatedAt:         time.Now(),
				OriginalUpdatedAt: time.Now(),
			}

			seedPendingUser(t, svc, pending)

			if tc.setupMock != nil {
				tc.setupMock(mock, pending)
			}

			err := svc.VerifyCode(pending.Email, tc.code)
			if (err != nil) != tc.wantError {
				t.Fatalf("VerifyCode() error = %v, wantError %v", err, tc.wantError)
			}

			if tc.validate != nil {
				tc.validate(t, mr, pending.Email)
			}

			verifyMockExpectations(t, mock)
		})
	}
}

func TestVerifyCodeExistingUserFallback(t *testing.T) {
	svc, mock, mr, cleanup := setupTestAuthService(t)
	defer cleanup()

	email := "existing@example.com"
	code := "123456"
	verifyKey := fmt.Sprintf("verify:%s", normalizeEmail(email))
	mr.Set(verifyKey, code)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET verified = true, updated_at = $1 WHERE email = $2")).
		WithArgs(sqlmock.AnyArg(), normalizeEmail(email)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := svc.VerifyCode(email, code); err != nil {
		t.Fatalf("VerifyCode fallback failed: %v", err)
	}

	if mr.Exists(verifyKey) {
		t.Fatalf("verify key should be deleted after successful verification")
	}

	verifyMockExpectations(t, mock)
}

func TestVerifyCodeUnknownEmail(t *testing.T) {
	svc, mock, _, cleanup := setupTestAuthService(t)
	defer cleanup()

	if err := svc.VerifyCode("missing@example.com", "111111"); err == nil {
		t.Fatalf("expected error for unknown email")
	}

	verifyMockExpectations(t, mock)
}

func TestLogin(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		password  string
		wantError bool
		setupMock func(sqlmock.Sqlmock)
	}{
		{
			name:     "valid login",
			email:    "login@example.com",
			password: "password123",
			setupMock: func(mock sqlmock.Sqlmock) {
				hash := mustHashPassword(t, "password123")
				rows := sqlmock.NewRows([]string{"id", "email", "phone", "password", "role", "verified", "created_at", "updated_at"}).
					AddRow("user-id-123", "login@example.com", "+79991234567", hash, models.RoleUser, true, time.Now(), time.Now())
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, phone, password, role, verified, created_at, updated_at FROM users WHERE email = $1")).
					WithArgs("login@example.com").
					WillReturnRows(rows)
			},
		},
		{
			name:      "non-existent user",
			email:     "missing@example.com",
			password:  "password123",
			wantError: true,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, phone, password, role, verified, created_at, updated_at FROM users WHERE email = $1")).
					WithArgs("missing@example.com").
					WillReturnError(sql.ErrNoRows)
			},
		},
		{
			name:      "invalid password",
			email:     "login@example.com",
			password:  "wrong",
			wantError: true,
			setupMock: func(mock sqlmock.Sqlmock) {
				hash := mustHashPassword(t, "password123")
				rows := sqlmock.NewRows([]string{"id", "email", "phone", "password", "role", "verified", "created_at", "updated_at"}).
					AddRow("user-id-123", "login@example.com", "+79991234567", hash, models.RoleUser, true, time.Now(), time.Now())
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, phone, password, role, verified, created_at, updated_at FROM users WHERE email = $1")).
					WithArgs("login@example.com").
					WillReturnRows(rows)
			},
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			svc, mock, _, cleanup := setupTestAuthService(t)
			defer cleanup()

			if tc.setupMock != nil {
				tc.setupMock(mock)
			}

			resp, err := svc.Login(&models.LoginRequest{
				Email:    tc.email,
				Password: tc.password,
			})

			if tc.wantError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if resp == nil || resp.User == nil || resp.User.Email != tc.email {
					t.Fatalf("unexpected login response: %+v", resp)
				}
				if resp.AccessToken == "" {
					t.Fatalf("expected access token")
				}
			}

			verifyMockExpectations(t, mock)
		})
	}
}

func TestLogout(t *testing.T) {
	svc, _, mr, cleanup := setupTestAuthService(t)
	defer cleanup()

	testUser := &models.User{ID: "test-user-id", Email: "test@example.com", Role: models.RoleUser, Verified: true}
	token, _, err := svc.GenerateJWT(testUser)
	if err != nil {
		t.Fatalf("GenerateJWT() failed: %v", err)
	}

	if err := svc.Logout(token); err != nil {
		t.Fatalf("Logout() failed: %v", err)
	}

	blacklistKey := fmt.Sprintf("blacklist:%s", token)
	if !mr.Exists(blacklistKey) {
		t.Fatalf("blacklist key not created")
	}
	if ttl := mr.TTL(blacklistKey); ttl <= 0 {
		t.Fatalf("expected ttl to be positive, got %v", ttl)
	}
	if !svc.IsTokenBlacklisted(token) {
		t.Fatalf("token expected to be blacklisted")
	}
}

func TestGetUserByID(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		wantError bool
		setupMock func(sqlmock.Sqlmock)
	}{
		{
			name:   "valid user id",
			userID: "user-id-123",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "phone", "role", "verified", "created_at", "updated_at"}).
					AddRow("user-id-123", "getuser@example.com", "+79991234567", models.RoleUser, true, time.Now(), time.Now())
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, phone, role, verified, created_at, updated_at FROM users WHERE id = $1")).
					WithArgs("user-id-123").
					WillReturnRows(rows)
			},
		},
		{
			name:      "non-existent user",
			userID:    "missing",
			wantError: true,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, phone, role, verified, created_at, updated_at FROM users WHERE id = $1")).
					WithArgs("missing").
					WillReturnError(sql.ErrNoRows)
			},
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			svc, mock, _, cleanup := setupTestAuthService(t)
			defer cleanup()

			if tc.setupMock != nil {
				tc.setupMock(mock)
			}

			user, err := svc.GetUserByID(tc.userID)
			if tc.wantError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if user == nil || user.ID != tc.userID {
					t.Fatalf("unexpected user: %+v", user)
				}
			}

			verifyMockExpectations(t, mock)
		})
	}
}

func TestValidateJWT(t *testing.T) {
	svc, _, _, cleanup := setupTestAuthService(t)
	defer cleanup()

	testUser := &models.User{ID: "test-user-id", Email: "jwt@example.com", Role: models.RoleUser, Verified: true}
	validToken, _, err := svc.GenerateJWT(testUser)
	if err != nil {
		t.Fatalf("GenerateJWT() failed: %v", err)
	}

	tests := []struct {
		name      string
		token     string
		wantError bool
	}{
		{name: "valid token", token: validToken},
		{name: "invalid token format", token: "invalid.token.format", wantError: true},
		{name: "empty token", token: "", wantError: true},
	}

	for _, tt := range tests {
		claims, err := svc.ValidateJWT(tt.token)
		if (err != nil) != tt.wantError {
			t.Fatalf("ValidateJWT() error = %v, wantError %v", err, tt.wantError)
		}
		if !tt.wantError && claims == nil {
			t.Fatalf("expected claims for valid token")
		}
	}
}

func TestVerifyAndIssueToken(t *testing.T) {
	svc, mock, mr, cleanup := setupTestAuthService(t)
	defer cleanup()

	email := "token@example.com"
	code := "222222"
	normEmail := normalizeEmail(email)
	verifyKey := fmt.Sprintf("verify:%s", normEmail)
	mr.Set(verifyKey, code)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET verified = true, updated_at = $1 WHERE email = $2")).
		WithArgs(sqlmock.AnyArg(), normEmail).
		WillReturnResult(sqlmock.NewResult(1, 1))

	rows := sqlmock.NewRows([]string{"id", "email", "phone", "password", "role", "verified", "created_at", "updated_at"}).
		AddRow("user-id-token", normEmail, "+79995554433", mustHashPassword(t, "password123"), models.RoleUser, true, time.Now(), time.Now())
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, phone, password, role, verified, created_at, updated_at FROM users WHERE email = $1")).
		WithArgs(normEmail).
		WillReturnRows(rows)

	resp, err := svc.VerifyAndIssueToken(email, code)
	if err != nil {
		t.Fatalf("VerifyAndIssueToken() failed: %v", err)
	}

	if resp == nil || resp.AccessToken == "" || resp.User == nil || resp.User.Email != normEmail {
		t.Fatalf("unexpected auth response: %+v", resp)
	}
	if mr.Exists(verifyKey) {
		t.Fatalf("verify key should be removed")
	}

	verifyMockExpectations(t, mock)
}

func verifyMockExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %s", err)
	}
}

func mustHashPassword(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	return string(hash)
}

func seedPendingUser(t *testing.T, svc *AuthService, pending *pendingUser) {
	t.Helper()
	if pending.CreatedAt.IsZero() {
		pending.CreatedAt = time.Now()
	}
	if pending.OriginalUpdatedAt.IsZero() {
		pending.OriginalUpdatedAt = pending.CreatedAt
	}
	if err := svc.savePendingUser(context.Background(), pending); err != nil {
		t.Fatalf("failed to seed pending user: %v", err)
	}
}
