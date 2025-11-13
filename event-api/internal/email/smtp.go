package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"

	"go.uber.org/zap"
)

// SMTPProvider провайдер для отправки email через SMTP.
type SMTPProvider struct {
	config *Config
	logger *zap.Logger
	auth   smtp.Auth
}

// NewSMTPProvider создает новый SMTP провайдер.
func NewSMTPProvider(cfg *Config, logger *zap.Logger) (*SMTPProvider, error) {
	if cfg.SMTPHost == "" {
		return nil, fmt.Errorf("SMTP host не указан")
	}
	if cfg.SMTPPort == 0 {
		return nil, fmt.Errorf("SMTP port не указан")
	}
	if cfg.From == "" {
		return nil, fmt.Errorf("email отправителя не указан")
	}

	var auth smtp.Auth
	if cfg.SMTPUsername != "" && cfg.SMTPPassword != "" {
		auth = smtp.PlainAuth("", cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPHost)
	}

	return &SMTPProvider{
		config: cfg,
		logger: logger,
		auth:   auth,
	}, nil
}

// SendEmail отправляет текстовый email через SMTP.
func (p *SMTPProvider) SendEmail(ctx context.Context, to, subject, body string) error {
	from := p.config.From
	fromName := p.config.FromName
	if fromName != "" {
		from = fmt.Sprintf("%s <%s>", fromName, from)
	}

	msg := []byte(fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"Content-Type: text/plain; charset=UTF-8\r\n"+
			"\r\n"+
			"%s",
		from, to, subject, body,
	))

	addr := fmt.Sprintf("%s:%d", p.config.SMTPHost, p.config.SMTPPort)

	// Для портов 465 (SSL) и 587 (TLS)
	if p.config.SMTPPort == 465 {
		return p.sendWithSSL(addr, to, msg)
	}

	// Для порта 587 или других с STARTTLS
	return smtp.SendMail(addr, p.auth, p.config.From, []string{to}, msg)
}

// SendHTMLEmail отправляет HTML email через SMTP.
func (p *SMTPProvider) SendHTMLEmail(ctx context.Context, to, subject, htmlBody string) error {
	from := p.config.From
	fromName := p.config.FromName
	if fromName != "" {
		from = fmt.Sprintf("%s <%s>", fromName, from)
	}

	msg := []byte(fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=UTF-8\r\n"+
			"\r\n"+
			"%s",
		from, to, subject, htmlBody,
	))

	addr := fmt.Sprintf("%s:%d", p.config.SMTPHost, p.config.SMTPPort)

	if p.config.SMTPPort == 465 {
		return p.sendWithSSL(addr, to, msg)
	}

	return smtp.SendMail(addr, p.auth, p.config.From, []string{to}, msg)
}

// sendWithSSL отправляет email через SSL соединение (порт 465).
func (p *SMTPProvider) sendWithSSL(addr, to string, msg []byte) error {
	tlsConfig := &tls.Config{
		ServerName: p.config.SMTPHost,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("ошибка подключения к SMTP серверу: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, p.config.SMTPHost)
	if err != nil {
		return fmt.Errorf("ошибка создания SMTP клиента: %w", err)
	}
	defer client.Close()

	if p.auth != nil {
		if err = client.Auth(p.auth); err != nil {
			return fmt.Errorf("ошибка аутентификации: %w", err)
		}
	}

	if err = client.Mail(p.config.From); err != nil {
		return fmt.Errorf("ошибка установки отправителя: %w", err)
	}

	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("ошибка установки получателя: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("ошибка открытия данных: %w", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("ошибка записи сообщения: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("ошибка закрытия данных: %w", err)
	}

	return client.Quit()
}

// GetName возвращает имя провайдера.
func (p *SMTPProvider) GetName() string {
	return "smtp"
}
