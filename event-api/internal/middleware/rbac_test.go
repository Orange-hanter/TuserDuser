package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"event-api/internal/logger"
	"event-api/internal/models"

	"go.uber.org/zap"
)

func TestMain(m *testing.M) {
	logger.Log = zap.NewNop()
	os.Exit(m.Run())
}

func TestRequireRole(t *testing.T) {
	handlerCalled := false
	protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	tests := []struct {
		name           string
		userRole       string
		requiredRoles  []string
		expectedStatus int
		expectedBody   string
		callsNext      bool
	}{
		{
			name:           "admin can access admin endpoint",
			userRole:       models.RoleAdmin,
			requiredRoles:  []string{models.RoleAdmin},
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
			callsNext:      true,
		},
		{
			name:           "user cannot access admin endpoint",
			userRole:       models.RoleUser,
			requiredRoles:  []string{models.RoleAdmin},
			expectedStatus: http.StatusForbidden,
			expectedBody:   `{"error":"forbidden","message":"Недостаточно прав для этого действия","code":403}`,
			callsNext:      false,
		},
		{
			name:           "missing role header returns forbidden",
			userRole:       "",
			requiredRoles:  []string{models.RoleUser},
			expectedStatus: http.StatusForbidden,
			expectedBody:   `{"error":"forbidden","message":"Недостаточно прав","code":403}`,
			callsNext:      false,
		},
	}

	for _, tt := range tests {
		// reset flag for each test
		handlerCalled = false

		t.Run(tt.name, func(t *testing.T) {
			wrapped := RequireRole(tt.requiredRoles...)(protectedHandler)
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.userRole != "" {
				req.Header.Set("X-User-Role", tt.userRole)
			}
			rec := httptest.NewRecorder()

			wrapped.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if body := rec.Body.String(); body != tt.expectedBody {
				t.Fatalf("unexpected body: %s", body)
			}

			if handlerCalled != tt.callsNext {
				t.Fatalf("next handler call mismatch: expected %v got %v", tt.callsNext, handlerCalled)
			}
		})
	}
}

func TestRequireAdmin(t *testing.T) {
	wrapped := RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("X-User-Role", models.RoleAdmin)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected admin request to pass, got %d", rec.Code)
	}
}

func TestRequireCreatorOrAdmin(t *testing.T) {
	tests := []struct {
		name     string
		userRole string
		status   int
	}{
		{name: "creator passes", userRole: models.RoleCreator, status: http.StatusOK},
		{name: "admin passes", userRole: models.RoleAdmin, status: http.StatusOK},
		{name: "user denied", userRole: models.RoleUser, status: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/events", nil)
			req.Header.Set("X-User-Role", tt.userRole)
			RequireCreatorOrAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})).ServeHTTP(rec, req)

			if rec.Code != tt.status {
				t.Fatalf("expected %d got %d", tt.status, rec.Code)
			}
		})
	}
}

func TestHasPermission(t *testing.T) {
	tests := []struct {
		name       string
		role       string
		permission string
		expected   bool
	}{
		{name: "admin has any permission", role: models.RoleAdmin, permission: "random", expected: true},
		{name: "creator can create", role: models.RoleCreator, permission: "events.create", expected: true},
		{name: "creator lacks unrelated", role: models.RoleCreator, permission: "users.read", expected: false},
		{name: "support reads users", role: models.RoleSupport, permission: "users.read", expected: true},
		{name: "user only read", role: models.RoleUser, permission: "events.create", expected: false},
		{name: "unknown role", role: "guest", permission: "events.read", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasPermission(tt.role, tt.permission); got != tt.expected {
				t.Fatalf("HasPermission(%s,%s)=%v want %v", tt.role, tt.permission, got, tt.expected)
			}
		})
	}
}

func TestRespondWithError(t *testing.T) {
	rec := httptest.NewRecorder()
	respondWithError(rec, http.StatusForbidden, "forbidden", "Недостаточно прав")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 got %d", rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected json content type got %s", ct)
	}

	expected := map[string]any{
		"error":   "forbidden",
		"message": "Недостаточно прав",
		"code":    float64(http.StatusForbidden),
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal body: %v", err)
	}

	if len(body) != len(expected) {
		t.Fatalf("unexpected body size: %#v", body)
	}

	for key, value := range expected {
		if body[key] != value {
			t.Fatalf("expected %s to be %v got %v", key, value, body[key])
		}
	}
}
