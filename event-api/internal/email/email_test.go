package email_test

import (
	"context"
	"testing"

	"event-api/internal/email"

	"go.uber.org/zap"
)

func TestMockProvider(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	config := &email.Config{
		Provider: "mock",
		From:     "test@example.com",
		FromName: "Test Sender",
	}

	service, err := email.NewService(config, logger)
	if err != nil {
		t.Fatalf("Failed to create email service: %v", err)
	}

	ctx := context.Background()

	t.Run("Send text email", func(t *testing.T) {
		err := service.SendEmail(ctx, "user@example.com", "Test Subject", "Test Body")
		if err != nil {
			t.Errorf("SendEmail failed: %v", err)
		}
	})

	t.Run("Send HTML email", func(t *testing.T) {
		err := service.SendHTMLEmail(ctx, "user@example.com", "Test Subject", "<h1>Test HTML</h1>")
		if err != nil {
			t.Errorf("SendHTMLEmail failed: %v", err)
		}
	})

	t.Run("Send verification email", func(t *testing.T) {
		err := service.SendVerificationEmail(ctx, "user@example.com", "123456")
		if err != nil {
			t.Errorf("SendVerificationEmail failed: %v", err)
		}
	})

	t.Run("Send verification HTML email", func(t *testing.T) {
		err := service.SendVerificationHTMLEmail(ctx, "user@example.com", "654321")
		if err != nil {
			t.Errorf("SendVerificationHTMLEmail failed: %v", err)
		}
	})

	t.Run("Send password reset email", func(t *testing.T) {
		err := service.SendPasswordResetEmail(ctx, "user@example.com", "https://example.com/reset?token=abc")
		if err != nil {
			t.Errorf("SendPasswordResetEmail failed: %v", err)
		}
	})
}

func TestSMTPProviderValidation(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	t.Run("Missing SMTP host", func(t *testing.T) {
		config := &email.Config{
			Provider: "smtp",
			From:     "test@example.com",
			SMTPPort: 587,
		}
		_, err := email.NewService(config, logger)
		if err == nil {
			t.Error("Expected error for missing SMTP host")
		}
	})

	t.Run("Missing SMTP port", func(t *testing.T) {
		config := &email.Config{
			Provider: "smtp",
			From:     "test@example.com",
			SMTPHost: "smtp.example.com",
		}
		_, err := email.NewService(config, logger)
		if err == nil {
			t.Error("Expected error for missing SMTP port")
		}
	})

	t.Run("Missing From email", func(t *testing.T) {
		config := &email.Config{
			Provider: "smtp",
			SMTPHost: "smtp.example.com",
			SMTPPort: 587,
		}
		_, err := email.NewService(config, logger)
		if err == nil {
			t.Error("Expected error for missing From email")
		}
	})
}

func TestSendGridProviderValidation(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	t.Run("Missing API key", func(t *testing.T) {
		config := &email.Config{
			Provider: "sendgrid",
			From:     "test@example.com",
		}
		_, err := email.NewService(config, logger)
		if err == nil {
			t.Error("Expected error for missing API key")
		}
	})
}

func TestMailgunProviderValidation(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	t.Run("Missing API key", func(t *testing.T) {
		config := &email.Config{
			Provider: "mailgun",
			From:     "test@example.com",
		}
		_, err := email.NewService(config, logger)
		if err == nil {
			t.Error("Expected error for missing API key")
		}
	})

	t.Run("Invalid From email format", func(t *testing.T) {
		config := &email.Config{
			Provider: "mailgun",
			APIKey:   "test-key",
			From:     "invalid-email",
		}
		_, err := email.NewService(config, logger)
		if err == nil {
			t.Error("Expected error for invalid email format")
		}
	})
}

func TestUnknownProvider(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	config := &email.Config{
		Provider: "unknown-provider",
		From:     "test@example.com",
	}

	_, err := email.NewService(config, logger)
	if err == nil {
		t.Error("Expected error for unknown provider")
	}
}
