package sms

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// Provider интерфейс для SMS провайдеров
type Provider interface {
	SendSMS(ctx context.Context, phone, message string) error
	GetName() string
}

// Config конфигурация для SMS сервиса
type Config struct {
	Provider string // "mock", "twilio", "smsru", "smsc"
	APIKey   string
	APIToken string
	From     string // Номер отправителя
}

// Service сервис для отправки SMS
type Service struct {
	provider Provider
	logger   *zap.Logger
}

// NewService создает новый SMS сервис
func NewService(cfg *Config, logger *zap.Logger) (*Service, error) {
	var provider Provider
	var err error

	switch cfg.Provider {
	case "mock", "":
		provider = NewMockProvider(logger)
	case "twilio":
		provider, err = NewTwilioProvider(cfg, logger)
	case "smsru":
		provider, err = NewSMSRuProvider(cfg, logger)
	case "smsc":
		provider, err = NewSMSCProvider(cfg, logger)
	default:
		return nil, fmt.Errorf("неизвестный SMS провайдер: %s", cfg.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации SMS провайдера: %w", err)
	}

	logger.Info("SMS сервис инициализирован",
		zap.String("provider", provider.GetName()),
	)

	return &Service{
		provider: provider,
		logger:   logger,
	}, nil
}

// SendSMS отправляет SMS
func (s *Service) SendSMS(ctx context.Context, phone, message string) error {
	s.logger.Info("Отправка SMS",
		zap.String("phone", phone),
		zap.String("provider", s.provider.GetName()),
	)

	if err := s.provider.SendSMS(ctx, phone, message); err != nil {
		s.logger.Error("Ошибка отправки SMS",
			zap.String("phone", phone),
			zap.String("provider", s.provider.GetName()),
			zap.Error(err),
		)
		return fmt.Errorf("ошибка отправки SMS: %w", err)
	}

	s.logger.Info("SMS успешно отправлена",
		zap.String("phone", phone),
		zap.String("provider", s.provider.GetName()),
	)

	return nil
}

// SendVerificationCode отправляет код верификации
func (s *Service) SendVerificationCode(ctx context.Context, phone, code string) error {
	message := fmt.Sprintf("Ваш код верификации: %s\nКод действителен 10 минут.", code)
	return s.SendSMS(ctx, phone, message)
}

// SendPasswordReset отправляет код сброса пароля
func (s *Service) SendPasswordReset(ctx context.Context, phone, code string) error {
	message := fmt.Sprintf("Код сброса пароля: %s\nНе сообщайте никому этот код.", code)
	return s.SendSMS(ctx, phone, message)
}

// SendNotification отправляет уведомление
func (s *Service) SendNotification(ctx context.Context, phone, text string) error {
	return s.SendSMS(ctx, phone, text)
}
