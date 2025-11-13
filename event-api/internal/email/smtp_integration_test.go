package email_test

import (
	"context"
	"os"
	"strconv"
	"testing"

	"event-api/internal/email"

	"go.uber.org/zap"
)

// TestSMTPIntegrationSendEmail sends a real email using SMTP provider.
// It is skipped unless the following env vars are set:
//
//	SMTP_HOST, SMTP_PORT, SMTP_USERNAME, SMTP_PASSWORD, EMAIL_FROM, EMAIL_TO
//
// Optional:
//
//	EMAIL_FROM_NAME
func TestSMTPIntegrationSendEmail(t *testing.T) {
	host := os.Getenv("SMTP_HOST")
	portStr := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USERNAME")
	pass := os.Getenv("SMTP_PASSWORD")
	from := os.Getenv("EMAIL_FROM")
	fromName := os.Getenv("EMAIL_FROM_NAME")
	to := os.Getenv("EMAIL_TO")

	if host == "" || portStr == "" || from == "" || to == "" {
		t.Skip("Skipping SMTP integration test: set SMTP_HOST, SMTP_PORT, EMAIL_FROM, EMAIL_TO (and credentials if required)")
		return
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("invalid SMTP_PORT: %v", err)
	}

	logger, _ := zap.NewDevelopment()

	cfg := &email.Config{
		Provider:     "smtp",
		SMTPHost:     host,
		SMTPPort:     port,
		SMTPUsername: user,
		SMTPPassword: pass,
		From:         from,
		FromName:     fromName,
	}

	svc, err := email.NewService(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create SMTP email service: %v", err)
	}

	ctx := context.Background()
	subject := "Event API SMTP integration test"
	body := "This is a test message sent by the SMTP integration test."

	if err := svc.SendEmail(ctx, to, subject, body); err != nil {
		t.Fatalf("SendEmail failed: %v", err)
	}

	t.Logf("SMTP email sent to %s via %s:%d", to, host, port)
}
