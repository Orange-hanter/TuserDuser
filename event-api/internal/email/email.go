// Package email реализует сервис для отправки email сообщений.
package email

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// Provider интерфейс для email провайдеров.
type Provider interface {
	SendEmail(ctx context.Context, to, subject, body string) error
	SendHTMLEmail(ctx context.Context, to, subject, htmlBody string) error
	GetName() string
}

// Config конфигурация для email сервиса.
type Config struct {
	Provider string // "mock", "smtp", "sendgrid", "mailgun"
	APIKey   string // Для SendGrid, Mailgun и т.д.

	// SMTP настройки
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	UseSSL       bool

	// Общие настройки
	From     string // Email отправителя
	FromName string // Имя отправителя
}

// Service сервис для отправки email.
type Service struct {
	provider Provider
	logger   *zap.Logger
}

// NewService создает новый email сервис.
func NewService(cfg *Config, logger *zap.Logger) (*Service, error) {
	var provider Provider
	var err error

	switch cfg.Provider {
	case "mock", "":
		provider = NewMockProvider(logger)
	case "smtp":
		provider, err = NewSMTPProvider(cfg, logger)
	case "sendgrid":
		provider, err = NewSendGridProvider(cfg, logger)
	case "mailgun":
		provider, err = NewMailgunProvider(cfg, logger)
	default:
		return nil, fmt.Errorf("неизвестный email провайдер: %s", cfg.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации email провайдера: %w", err)
	}

	return &Service{
		provider: provider,
		logger:   logger,
	}, nil
}

// SendEmail отправляет текстовый email.
func (s *Service) SendEmail(ctx context.Context, to, subject, body string) error {
	s.logger.Info("Отправка email",
		zap.String("provider", s.provider.GetName()),
		zap.String("to", to),
		zap.String("subject", subject),
	)

	if err := s.provider.SendEmail(ctx, to, subject, body); err != nil {
		s.logger.Error("Ошибка отправки email",
			zap.String("provider", s.provider.GetName()),
			zap.String("to", to),
			zap.Error(err),
		)
		return fmt.Errorf("не удалось отправить email: %w", err)
	}

	s.logger.Info("Email успешно отправлен",
		zap.String("provider", s.provider.GetName()),
		zap.String("to", to),
	)

	return nil
}

// SendHTMLEmail отправляет HTML email.
func (s *Service) SendHTMLEmail(ctx context.Context, to, subject, htmlBody string) error {
	s.logger.Info("Отправка HTML email",
		zap.String("provider", s.provider.GetName()),
		zap.String("to", to),
		zap.String("subject", subject),
	)

	if err := s.provider.SendHTMLEmail(ctx, to, subject, htmlBody); err != nil {
		s.logger.Error("Ошибка отправки HTML email",
			zap.String("provider", s.provider.GetName()),
			zap.String("to", to),
			zap.Error(err),
		)
		return fmt.Errorf("не удалось отправить HTML email: %w", err)
	}

	s.logger.Info("HTML email успешно отправлен",
		zap.String("provider", s.provider.GetName()),
		zap.String("to", to),
	)

	return nil
}

// SendVerificationEmail отправляет код верификации.
func (s *Service) SendVerificationEmail(ctx context.Context, to, code string) error {
	subject := "Код верификации"
	body := fmt.Sprintf(`
Здравствуйте!

Ваш код верификации: %s

Код действителен в течение 10 минут.

Если вы не регистрировались на нашем сайте, просто проигнорируйте это письмо.

С уважением,
Команда TuserDuser
`, code)

	return s.SendEmail(ctx, to, subject, body)
}

// SendVerificationHTMLEmail отправляет код верификации в HTML формате.
func (s *Service) SendVerificationHTMLEmail(ctx context.Context, to, code string) error {
	subject := "Код верификации"
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #4CAF50; color: white; padding: 20px; text-align: center; }
        .content { background-color: #f9f9f9; padding: 30px; border-radius: 5px; margin-top: 20px; }
        .code { font-size: 32px; font-weight: bold; color: #4CAF50; text-align: center; 
                letter-spacing: 5px; padding: 20px; background: white; border-radius: 5px; 
                margin: 20px 0; }
        .footer { text-align: center; margin-top: 20px; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Код верификации</h1>
        </div>
        <div class="content">
            <p>Здравствуйте!</p>
            <p>Для завершения регистрации введите следующий код:</p>
            <div class="code">%s</div>
            <p>Код действителен в течение <strong>10 минут</strong>.</p>
            <p>Если вы не регистрировались на нашем сайте, просто проигнорируйте это письмо.</p>
        </div>
        <div class="footer">
            <p>С уважением,<br>Команда TuserDuser</p>
        </div>
    </div>
</body>
</html>
`, code)

	return s.SendHTMLEmail(ctx, to, subject, htmlBody)
}

// SendPasswordResetEmail отправляет ссылку для сброса пароля.
func (s *Service) SendPasswordResetEmail(ctx context.Context, to, resetLink string) error {
	subject := "Сброс пароля"
	body := fmt.Sprintf(`
Здравствуйте!

Вы запросили сброс пароля. Перейдите по ссылке ниже для создания нового пароля:

%s

Ссылка действительна в течение 1 часа.

Если вы не запрашивали сброс пароля, просто проигнорируйте это письмо.

С уважением,
Команда TuserDuser
`, resetLink)

	return s.SendEmail(ctx, to, subject, body)
}
