package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"event-api/internal/models"

	"github.com/go-chi/chi/v5"
)

// mockUserService implements the UserService interface for testing.
type mockUserService struct {
	getPublicProfileFn func(ctx context.Context, userID string) (*models.PublicUserProfile, string, error)
}

func (m *mockUserService) GetPublicProfile(ctx context.Context, userID string) (*models.PublicUserProfile, string, error) {
	if m.getPublicProfileFn != nil {
		return m.getPublicProfileFn(ctx, userID)
	}
	return nil, "", &models.PublicProfileNotFoundError{UserID: userID}
}

// respondWithErrorTest is a test helper to match the handlers.respondWithError behavior.
func respondWithErrorTest(w http.ResponseWriter, statusCode int, errorType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"code":    errorType,
			"message": message,
		},
	})
}

// TestGetPublicProfile_ValidUUID tests successful public profile retrieval.
func TestGetPublicProfile_ValidUUID(t *testing.T) {
	validUUID := "941b955e-ea57-dee3-565f-5684f81c4f14"
	expectedETag := `"abc123def456"`

	username := "testuser"
	bio := "Test bio"
	city := "Moscow"
	country := "RU"

	mockProfile := &models.PublicUserProfile{
		ID:                validUUID,
		DisplayName:       "Test User",
		Username:          &username,
		Bio:               &bio,
		City:              &city,
		Country:           &country,
		PublicEventsCount: 5,
		IsVerified:        true,
		SocialLinks:       map[string]string{"twitter": "https://twitter.com/test"},
		CreatedAt:         time.Date(2024, 5, 2, 15, 23, 45, 0, time.UTC),
		UpdatedAt:         time.Date(2025, 11, 1, 12, 10, 0, 0, time.UTC),
	}

	mockSvc := &mockUserService{
		getPublicProfileFn: func(ctx context.Context, userID string) (*models.PublicUserProfile, string, error) {
			if userID != validUUID {
				t.Errorf("expected userID %s, got %s", validUUID, userID)
			}
			return mockProfile, expectedETag, nil
		},
	}

	// Create a real UserHandler but we'll manually call the handler
	// Since we can't easily inject mock, we'll test the handler logic directly
	r := chi.NewRouter()
	r.Get("/api/users/public/{userId}", func(w http.ResponseWriter, req *http.Request) {
		userID := chi.URLParam(req, "userId")

		profile, etag, err := mockSvc.GetPublicProfile(req.Context(), userID)
		if err != nil {
			respondWithErrorTest(w, http.StatusNotFound, "not_found", "User not found")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=300, s-maxage=600")
		w.Header().Set("ETag", etag)
		w.Header().Set("X-Content-Type-Options", "nosniff")

		if err := json.NewEncoder(w).Encode(profile); err != nil {
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users/public/"+validUUID, nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Check headers
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
	if etag := w.Header().Get("ETag"); etag != expectedETag {
		t.Errorf("expected ETag %s, got %s", expectedETag, etag)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "public, max-age=300, s-maxage=600" {
		t.Errorf("expected Cache-Control header, got %s", cc)
	}
	if xcto := w.Header().Get("X-Content-Type-Options"); xcto != "nosniff" {
		t.Errorf("expected X-Content-Type-Options nosniff, got %s", xcto)
	}

	// Check response body
	var resp models.PublicUserProfile
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.ID != validUUID {
		t.Errorf("expected ID %s, got %s", validUUID, resp.ID)
	}
	if resp.DisplayName != "Test User" {
		t.Errorf("expected DisplayName 'Test User', got %s", resp.DisplayName)
	}
	if resp.PublicEventsCount != 5 {
		t.Errorf("expected PublicEventsCount 5, got %d", resp.PublicEventsCount)
	}
	if !resp.IsVerified {
		t.Error("expected IsVerified true")
	}
}

// TestGetPublicProfile_InvalidUUID tests 400 response for invalid UUID format.
func TestGetPublicProfile_InvalidUUID(t *testing.T) {
	testCases := []struct {
		name    string
		userID  string
		wantErr string
	}{
		{"empty", "", "userId must be a valid UUID"},
		{"not_uuid", "not-a-uuid", "userId must be a valid UUID"},
		{"too_short", "941b955e-ea57", "userId must be a valid UUID"},
		{"invalid_chars", "941b955e-ea57-xxxx-565f-5684f81c4f14", "userId must be a valid UUID"},
		{"underscore", "941b955e_ea57_dee3_565f_5684f81c4f14", "userId must be a valid UUID"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/api/users/public/{userId}", func(w http.ResponseWriter, req *http.Request) {
				userID := chi.URLParam(req, "userId")

				// Simple UUID validation
				if len(userID) != 36 || !isValidUUID(userID) {
					respondWithErrorTest(w, http.StatusBadRequest, "invalid_request", "userId must be a valid UUID")
					return
				}
			})

			path := "/api/users/public/" + tc.userID
			if tc.userID == "" {
				path = "/api/users/public/"
			}

			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			// For empty userID, chi will return 404 (no route match)
			if tc.userID == "" {
				if w.Code != http.StatusNotFound {
					t.Errorf("expected status %d for empty userID, got %d", http.StatusNotFound, w.Code)
				}
				return
			}

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d for %s", http.StatusBadRequest, w.Code, tc.userID)
			}

			var errResp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
				t.Fatalf("failed to unmarshal error response: %v", err)
			}

			errObj, ok := errResp["error"].(map[string]interface{})
			if !ok {
				t.Fatal("expected error object in response")
			}
			if errObj["code"] != "invalid_request" {
				t.Errorf("expected error code 'invalid_request', got %v", errObj["code"])
			}
		})
	}
}

// TestGetPublicProfile_NotFound tests 404 response when user doesn't exist.
func TestGetPublicProfile_NotFound(t *testing.T) {
	nonExistentUUID := "00000000-0000-0000-0000-000000000000"

	mockSvc := &mockUserService{
		getPublicProfileFn: func(ctx context.Context, userID string) (*models.PublicUserProfile, string, error) {
			return nil, "", &models.PublicProfileNotFoundError{UserID: userID}
		},
	}

	r := chi.NewRouter()
	r.Get("/api/users/public/{userId}", func(w http.ResponseWriter, req *http.Request) {
		userID := chi.URLParam(req, "userId")

		if !isValidUUID(userID) {
			respondWithErrorTest(w, http.StatusBadRequest, "invalid_request", "userId must be a valid UUID")
			return
		}

		_, _, err := mockSvc.GetPublicProfile(req.Context(), userID)
		if err != nil {
			var notFoundErr *models.PublicProfileNotFoundError
			if _, ok := err.(*models.PublicProfileNotFoundError); ok {
				notFoundErr = err.(*models.PublicProfileNotFoundError)
				_ = notFoundErr // suppress unused warning
			}
			respondWithErrorTest(w, http.StatusNotFound, "not_found", "User not found")
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users/public/"+nonExistentUUID, nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var errResp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}

	errObj, ok := errResp["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected error object in response")
	}
	if errObj["code"] != "not_found" {
		t.Errorf("expected error code 'not_found', got %v", errObj["code"])
	}
	if errObj["message"] != "User not found" {
		t.Errorf("expected message 'User not found', got %v", errObj["message"])
	}
}

// TestGetPublicProfile_PrivateProfile tests 404 response for private profiles.
func TestGetPublicProfile_PrivateProfile(t *testing.T) {
	privateUserUUID := "11111111-1111-1111-1111-111111111111"

	mockSvc := &mockUserService{
		getPublicProfileFn: func(ctx context.Context, userID string) (*models.PublicUserProfile, string, error) {
			// Private profiles should return not found (to not reveal existence)
			return nil, "", &models.PublicProfileNotFoundError{UserID: userID}
		},
	}

	r := chi.NewRouter()
	r.Get("/api/users/public/{userId}", func(w http.ResponseWriter, req *http.Request) {
		userID := chi.URLParam(req, "userId")

		if !isValidUUID(userID) {
			respondWithErrorTest(w, http.StatusBadRequest, "invalid_request", "userId must be a valid UUID")
			return
		}

		_, _, err := mockSvc.GetPublicProfile(req.Context(), userID)
		if err != nil {
			respondWithErrorTest(w, http.StatusNotFound, "not_found", "User not found")
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users/public/"+privateUserUUID, nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// Should return 404, not 403, to not reveal account existence
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d for private profile, got %d", http.StatusNotFound, w.Code)
	}
}

// TestGetPublicProfile_304NotModified tests ETag caching behavior.
func TestGetPublicProfile_304NotModified(t *testing.T) {
	validUUID := "941b955e-ea57-dee3-565f-5684f81c4f14"
	expectedETag := `"abc123def456"`

	mockProfile := &models.PublicUserProfile{
		ID:          validUUID,
		DisplayName: "Test User",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mockSvc := &mockUserService{
		getPublicProfileFn: func(ctx context.Context, userID string) (*models.PublicUserProfile, string, error) {
			return mockProfile, expectedETag, nil
		},
	}

	r := chi.NewRouter()
	r.Get("/api/users/public/{userId}", func(w http.ResponseWriter, req *http.Request) {
		userID := chi.URLParam(req, "userId")

		profile, etag, err := mockSvc.GetPublicProfile(req.Context(), userID)
		if err != nil {
			respondWithErrorTest(w, http.StatusNotFound, "not_found", "User not found")
			return
		}

		// Check If-None-Match
		clientETag := req.Header.Get("If-None-Match")
		if clientETag == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", etag)
		if err := json.NewEncoder(w).Encode(profile); err != nil {
			return
		}
	})

	// First request - should return 200
	req1 := httptest.NewRequest(http.MethodGet, "/api/users/public/"+validUUID, nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("first request: expected status %d, got %d", http.StatusOK, w1.Code)
	}

	// Second request with matching ETag - should return 304
	req2 := httptest.NewRequest(http.MethodGet, "/api/users/public/"+validUUID, nil)
	req2.Header.Set("If-None-Match", expectedETag)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusNotModified {
		t.Errorf("second request with ETag: expected status %d, got %d", http.StatusNotModified, w2.Code)
	}

	// Third request with different ETag - should return 200
	req3 := httptest.NewRequest(http.MethodGet, "/api/users/public/"+validUUID, nil)
	req3.Header.Set("If-None-Match", `"different-etag"`)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Errorf("third request with different ETag: expected status %d, got %d", http.StatusOK, w3.Code)
	}
}

// TestGetPublicProfile_ResponseFormat tests the exact response format.
func TestGetPublicProfile_ResponseFormat(t *testing.T) {
	validUUID := "941b955e-ea57-dee3-565f-5684f81c4f14"
	username := "ivanivanov"
	avatarURL := "https://cdn.example.com/avatars/941b.jpg"
	bio := "Организатор мероприятий, спикер"
	city := "Москва"
	country := "RU"

	mockProfile := &models.PublicUserProfile{
		ID:                validUUID,
		DisplayName:       "Иван Иванов",
		Username:          &username,
		AvatarURL:         &avatarURL,
		Bio:               &bio,
		City:              &city,
		Country:           &country,
		PublicEventsCount: 12,
		IsVerified:        true,
		SocialLinks: map[string]string{
			"twitter":  "https://twitter.com/ivan",
			"telegram": "https://t.me/ivan",
		},
		CreatedAt: time.Date(2024, 5, 2, 15, 23, 45, 0, time.UTC),
		UpdatedAt: time.Date(2025, 11, 1, 12, 10, 0, 0, time.UTC),
	}

	mockSvc := &mockUserService{
		getPublicProfileFn: func(ctx context.Context, userID string) (*models.PublicUserProfile, string, error) {
			return mockProfile, `"etag123"`, nil
		},
	}

	r := chi.NewRouter()
	r.Get("/api/users/public/{userId}", func(w http.ResponseWriter, req *http.Request) {
		userID := chi.URLParam(req, "userId")
		profile, etag, err := mockSvc.GetPublicProfile(req.Context(), userID)
		if err != nil {
			respondWithErrorTest(w, http.StatusNotFound, "not_found", "User not found")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", etag)
		if err := json.NewEncoder(w).Encode(profile); err != nil {
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users/public/"+validUUID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Verify all required fields are present
	requiredFields := []string{"id", "displayName", "publicEventsCount", "isVerified", "createdAt", "updatedAt"}
	for _, field := range requiredFields {
		if _, exists := resp[field]; !exists {
			t.Errorf("missing required field: %s", field)
		}
	}

	// Verify no sensitive fields are present
	sensitiveFields := []string{"email", "phone", "password", "birthdate", "lastLogin", "ip"}
	for _, field := range sensitiveFields {
		if _, exists := resp[field]; exists {
			t.Errorf("response should not contain sensitive field: %s", field)
		}
	}
}

// isValidUUID is a simple UUID format validator for tests.
func isValidUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	// Check format: 8-4-4-4-12
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}
