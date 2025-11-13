package email

import (
	"context"

	"go.uber.org/zap"
)

// MockProvider мок-провайдер для тестирования (только логирование).
type MockProvider struct {
	logger *zap.Logger
}

// NewMockProvider создает новый мок-провайдер.
func NewMockProvider(logger *zap.Logger) *MockProvider {
	return &MockProvider{
		logger: logger,
	}
}

// SendEmail отправляет email (в моке только логирует).
func (p *MockProvider) SendEmail(ctx context.Context, to, subject, body string) error {
	p.logger.Info("📧 [MOCK] Email отправлен",
		zap.String("to", to),
		zap.String("subject", subject),
		zap.String("body", body),
	)
	return nil
}

// SendHTMLEmail отправляет HTML email (в моке только логирует).
func (p *MockProvider) SendHTMLEmail(ctx context.Context, to, subject, htmlBody string) error {
	p.logger.Info("📧 [MOCK] HTML Email отправлен",
		zap.String("to", to),
		zap.String("subject", subject),
		zap.Int("html_length", len(htmlBody)),
	)
	return nil
}

// GetName возвращает имя провайдера.
func (p *MockProvider) GetName() string {
	return "mock"
}
