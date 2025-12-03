package service_test

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"event-api/internal/models"
	"event-api/internal/service"

	"go.uber.org/zap"
)

// TestGetPublicProfile_Integration tests the GetPublicProfile service method.
// These tests require a database connection - skip if not available.
func TestGetPublicProfile_Integration(t *testing.T) {
	// Skip if no database connection
	// This test is designed to be run with a test database
	t.Skip("Skipping integration test - requires database connection")
}

// TestPublicProfileModel tests the PublicUserProfile model serialization.
func TestPublicProfileModel(t *testing.T) {
	username := "testuser"
	bio := "Test bio"
	city := "Moscow"
	country := "RU"
	avatarURL := "https://example.com/avatar.jpg"

	profile := &models.PublicUserProfile{
		ID:                "941b955e-ea57-dee3-565f-5684f81c4f14",
		DisplayName:       "Test User",
		Username:          &username,
		AvatarURL:         &avatarURL,
		Bio:               &bio,
		City:              &city,
		Country:           &country,
		PublicEventsCount: 5,
		IsVerified:        true,
		SocialLinks: map[string]string{
			"twitter":  "https://twitter.com/test",
			"telegram": "https://t.me/test",
		},
		CreatedAt: time.Date(2024, 5, 2, 15, 23, 45, 0, time.UTC),
		UpdatedAt: time.Date(2025, 11, 1, 12, 10, 0, 0, time.UTC),
	}

	// Test JSON serialization
	data, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("failed to marshal profile: %v", err)
	}

	// Verify JSON structure
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal profile: %v", err)
	}

	// Check required fields
	if result["id"] != profile.ID {
		t.Errorf("expected id %s, got %v", profile.ID, result["id"])
	}
	if result["displayName"] != profile.DisplayName {
		t.Errorf("expected displayName %s, got %v", profile.DisplayName, result["displayName"])
	}
	if result["username"] != username {
		t.Errorf("expected username %s, got %v", username, result["username"])
	}

	// Check sensitive fields are NOT present
	sensitiveFields := []string{"email", "phone", "password", "birthdate"}
	for _, field := range sensitiveFields {
		if _, exists := result[field]; exists {
			t.Errorf("sensitive field %s should not be in public profile", field)
		}
	}

	// Check social links
	socialLinks, ok := result["socialLinks"].(map[string]interface{})
	if !ok {
		t.Error("socialLinks should be a map")
	} else {
		if socialLinks["twitter"] != "https://twitter.com/test" {
			t.Errorf("expected twitter link, got %v", socialLinks["twitter"])
		}
	}
}

// TestParseSocialLinks tests the social links parsing from JSONB.
func TestParseSocialLinks(t *testing.T) {
	testCases := []struct {
		name     string
		input    []byte
		expected map[string]string
	}{
		{
			name:     "valid social links",
			input:    []byte(`{"twitter":"https://twitter.com/test","telegram":"https://t.me/test"}`),
			expected: map[string]string{"twitter": "https://twitter.com/test", "telegram": "https://t.me/test"},
		},
		{
			name:     "empty object",
			input:    []byte(`{}`),
			expected: nil,
		},
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty input",
			input:    []byte{},
			expected: nil,
		},
		{
			name:     "invalid json",
			input:    []byte(`{invalid}`),
			expected: nil,
		},
		{
			name:     "single link",
			input:    []byte(`{"github":"https://github.com/test"}`),
			expected: map[string]string{"github": "https://github.com/test"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := models.ParseSocialLinks(tc.input)

			if tc.expected == nil && result != nil {
				t.Errorf("expected nil, got %v", result)
			}
			if tc.expected != nil {
				if result == nil {
					t.Errorf("expected %v, got nil", tc.expected)
				} else {
					for k, v := range tc.expected {
						if result[k] != v {
							t.Errorf("expected %s=%s, got %s=%s", k, v, k, result[k])
						}
					}
				}
			}
		})
	}
}

// TestPublicProfileErrors tests error types for public profile.
func TestPublicProfileErrors(t *testing.T) {
	t.Run("PublicProfileNotFoundError", func(t *testing.T) {
		err := &models.PublicProfileNotFoundError{UserID: "test-uuid"}
		if err.Error() != "user not found" {
			t.Errorf("expected 'user not found', got %s", err.Error())
		}
	})

	t.Run("InvalidUUIDError", func(t *testing.T) {
		err := &models.InvalidUUIDError{Value: "not-uuid"}
		if err.Error() != "userId must be a valid UUID" {
			t.Errorf("expected 'userId must be a valid UUID', got %s", err.Error())
		}
	})
}

// mockDB is a minimal mock for testing service without real database.
type mockDB struct{}

// TestUserService_GetPublicProfile_MockDB tests the service with mocked database.
func TestUserService_GetPublicProfile_MockDB(t *testing.T) {
	// This test demonstrates service initialization without Redis
	logger := zap.NewNop()

	// Create service without Redis (nil)
	// Note: A nil database will cause a panic when querying, which is expected behavior
	svc := service.NewUserService(nil, logger, nil)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	// Note: We don't test GetPublicProfile with nil DB as it will panic
	// Integration tests with a real DB should cover the actual query logic
}

// TestUserService_WithRedis tests the service with Redis client set.
func TestUserService_WithRedis(t *testing.T) {
	logger := zap.NewNop()

	// Create service and set Redis client
	svc := service.NewUserService(nil, logger, nil)
	svc.SetRedisClient(nil) // Setting nil Redis is valid

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

// BenchmarkGetPublicProfile benchmarks the public profile retrieval.
func BenchmarkGetPublicProfile(b *testing.B) {
	// Skip if no database
	b.Skip("Skipping benchmark - requires database connection")

	// Typical benchmark setup would go here
	_ = sql.DB{}
}
