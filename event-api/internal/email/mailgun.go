package email

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"go.uber.org/zap"
)

// MailgunProvider провайдер для отправки email через Mailgun API.
type MailgunProvider struct {
	config *Config
	logger *zap.Logger
	client *http.Client
	domain string // Mailgun домен из APIKey или конфига
}

// NewMailgunProvider создает новый Mailgun провайдер.
func NewMailgunProvider(cfg *Config, logger *zap.Logger) (*MailgunProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("mailgun api key не указан")
	}
	if cfg.From == "" {
		return nil, fmt.Errorf("email отправителя не указан")
	}

	// Извлекаем домен из email отправителя
	parts := strings.Split(cfg.From, "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf("неверный формат email отправителя")
	}
	domain := parts[1]

	return &MailgunProvider{
		config: cfg,
		logger: logger,
		client: &http.Client{},
		domain: domain,
	}, nil
}

// SendEmail отправляет текстовый email через Mailgun.
func (p *MailgunProvider) SendEmail(ctx context.Context, to, subject, body string) error {
	data := url.Values{}
	data.Set("from", p.formatFrom())
	data.Set("to", to)
	data.Set("subject", subject)
	data.Set("text", body)

	return p.send(ctx, data)
}

// SendHTMLEmail отправляет HTML email через Mailgun.
func (p *MailgunProvider) SendHTMLEmail(ctx context.Context, to, subject, htmlBody string) error {
	data := url.Values{}
	data.Set("from", p.formatFrom())
	data.Set("to", to)
	data.Set("subject", subject)
	data.Set("html", htmlBody)

	return p.send(ctx, data)
}

// send отправляет email через Mailgun API.
func (p *MailgunProvider) send(ctx context.Context, data url.Values) error {
	apiURL := fmt.Sprintf("https://api.mailgun.net/v3/%s/messages", p.domain)

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.SetBasicAuth("api", p.config.APIKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка отправки запроса: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			p.logger.Error("failed to close Mailgun response body", zap.Error(cerr))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mailgun API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// formatFrom форматирует отправителя с именем.
func (p *MailgunProvider) formatFrom() string {
	if p.config.FromName != "" {
		return fmt.Sprintf("%s <%s>", p.config.FromName, p.config.From)
	}
	return p.config.From
}

// GetName возвращает имя провайдера.
func (p *MailgunProvider) GetName() string {
	return "mailgun"
}
