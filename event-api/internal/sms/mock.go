// Package sms предоставляет функциональность для отправки SMS сообщений.
package sms

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// MockProvider имитирует отправку SMS (для разработки и тестирования).
type MockProvider struct {
	logger *zap.Logger
}

// NewMockProvider создает новый Mock провайдер.
func NewMockProvider(logger *zap.Logger) *MockProvider {
	return &MockProvider{
		logger: logger,
	}
}

// SendSMS имитирует отправку SMS.
func (p *MockProvider) SendSMS(_ context.Context, phone, message string) error {
	p.logger.Info("📱 [MOCK SMS] Отправка сообщения",
		zap.String("phone", phone),
		zap.String("message", message),
	)

	// Имитация задержки сети
	time.Sleep(100 * time.Millisecond)

	// Для тестирования можно имитировать ошибки
	// Например, если номер начинается с +7000, возвращаем ошибку
	if len(phone) > 5 && phone[:5] == "+7000" {
		return fmt.Errorf("тестовая ошибка: недействительный номер телефона")
	}

	p.logger.Info("✅ [MOCK SMS] Сообщение успешно отправлено",
		zap.String("phone", phone),
	)

	return nil
}

// GetName возвращает название провайдера.
func (p *MockProvider) GetName() string {
	return "Mock SMS Provider"
}
