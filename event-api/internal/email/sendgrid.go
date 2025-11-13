package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

// SendGridProvider провайдер для отправки email через SendGrid API.
type SendGridProvider struct {
	config *Config
	logger *zap.Logger
	client *http.Client
}

// NewSendGridProvider создает новый SendGrid провайдер.
func NewSendGridProvider(cfg *Config, logger *zap.Logger) (*SendGridProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("SendGrid API key не указан")
	}
	if cfg.From == "" {
		return nil, fmt.Errorf("email отправителя не указан")
	}

	return &SendGridProvider{
		config: cfg,
		logger: logger,
		client: &http.Client{},
	}, nil
}

// sendGridRequest структура запроса для SendGrid API.
type sendGridRequest struct {
	Personalizations []sendGridPersonalization `json:"personalizations"`
	From             sendGridEmail             `json:"from"`
	Subject          string                    `json:"subject"`
	Content          []sendGridContent         `json:"content"`
}

type sendGridPersonalization struct {
	To []sendGridEmail `json:"to"`
}

type sendGridEmail struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type sendGridContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// SendEmail отправляет текстовый email через SendGrid.
func (p *SendGridProvider) SendEmail(ctx context.Context, to, subject, body string) error {
	return p.send(ctx, to, subject, "text/plain", body)
}

// SendHTMLEmail отправляет HTML email через SendGrid.
func (p *SendGridProvider) SendHTMLEmail(ctx context.Context, to, subject, htmlBody string) error {
	return p.send(ctx, to, subject, "text/html", htmlBody)
}

// send отправляет email через SendGrid API.
func (p *SendGridProvider) send(ctx context.Context, to, subject, contentType, body string) error {
	reqBody := sendGridRequest{
		Personalizations: []sendGridPersonalization{
			{
				To: []sendGridEmail{
					{Email: to},
				},
			},
		},
		From: sendGridEmail{
			Email: p.config.From,
			Name:  p.config.FromName,
		},
		Subject: subject,
		Content: []sendGridContent{
			{
				Type:  contentType,
				Value: body,
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("ошибка сериализации запроса: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.sendgrid.com/v3/mail/send", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка отправки запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil {
			return fmt.Errorf("SendGrid API error (status %d): %v", resp.StatusCode, errorResp)
		}
		return fmt.Errorf("SendGrid API error: status %d", resp.StatusCode)
	}

	return nil
}

// GetName возвращает имя провайдера.
func (p *SendGridProvider) GetName() string {
	return "sendgrid"
}
