package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"

	"go.uber.org/zap"
)

// SMTPProvider провайдер для отправки email через SMTP.
type SMTPProvider struct {
	config *Config
	logger *zap.Logger
	auth   smtp.Auth
	useSSL bool
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

	host, scheme := normalizeSMTPHost(cfg.SMTPHost)
	if host == "" {
		return nil, fmt.Errorf("SMTP host не указан")
	}
	cfg.SMTPHost = host
	useSSL := cfg.UseSSL || scheme == "smtps" || cfg.SMTPPort == 465

	var auth smtp.Auth
	if cfg.SMTPUsername != "" && cfg.SMTPPassword != "" {
		auth = smtp.PlainAuth("", cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPHost)
	}

	return &SMTPProvider{
		config: cfg,
		logger: logger,
		auth:   auth,
		useSSL: useSSL,
	}, nil
}

// SendEmail отправляет текстовый email через SMTP.
func (p *SMTPProvider) SendEmail(ctx context.Context, to, subject, body string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
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

	if p.useSSL {
		return p.sendWithSSL(ctx, addr, to, msg)
	}

	// Для порта 587 или других с STARTTLS
	return p.sendWithStartTLS(ctx, addr, to, msg)
}

// SendHTMLEmail отправляет HTML email через SMTP.
func (p *SMTPProvider) SendHTMLEmail(ctx context.Context, to, subject, htmlBody string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
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

	if p.useSSL {
		return p.sendWithSSL(ctx, addr, to, msg)
	}

	return p.sendWithStartTLS(ctx, addr, to, msg)
}

// sendWithSSL отправляет email через SSL соединение (порт 465).
func (p *SMTPProvider) sendWithSSL(ctx context.Context, addr, to string, msg []byte) error {
	tlsConfig := &tls.Config{
		ServerName: p.config.SMTPHost,
		MinVersion: tls.VersionTLS12,
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	dialer := &tls.Dialer{Config: tlsConfig}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("ошибка подключения к SMTP серверу: %w", err)
	}

	defer func() {
		if cerr := conn.Close(); cerr != nil {
			p.logger.Error("failed to close SMTP TLS connection", zap.Error(cerr))
		}
	}()

	client, err := smtp.NewClient(conn, p.config.SMTPHost)
	if err != nil {
		return fmt.Errorf("ошибка создания SMTP клиента: %w", err)
	}

	defer func() {
		if cerr := client.Close(); cerr != nil {
			p.logger.Error("failed to close SMTP client", zap.Error(cerr))
		}
	}()

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

func (p *SMTPProvider) sendWithStartTLS(ctx context.Context, addr, to string, msg []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return smtp.SendMail(addr, p.auth, p.config.From, []string{to}, msg)
}

// GetName возвращает имя провайдера.
func (p *SMTPProvider) GetName() string {
	return "smtp"
}

func normalizeSMTPHost(host string) (string, string) {
	host = strings.TrimSpace(host)
	scheme := ""
	switch {
	case strings.HasPrefix(host, "smtps://"):
		scheme = "smtps"
		host = strings.TrimPrefix(host, "smtps://")
	case strings.HasPrefix(host, "smtp://"):
		scheme = "smtp"
		host = strings.TrimPrefix(host, "smtp://")
	}
	host = strings.TrimSuffix(host, "/")
	return host, scheme
}
