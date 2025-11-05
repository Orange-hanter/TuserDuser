package sms

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
)

// TwilioProvider провайдер для Twilio API.
type TwilioProvider struct {
	client     *http.Client
	logger     *zap.Logger
	accountSID string
	authToken  string
	from       string
}

// TwilioResponse ответ от Twilio API.
type TwilioResponse struct {
	SID         string `json:"sid"`
	Status      string `json:"status"`
	To          string `json:"to"`
	From        string `json:"from"`
	Body        string `json:"body"`
	ErrorMsg    string `json:"error_message,omitempty"`
	DateCreated string `json:"date_created"`
	ErrorCode   int    `json:"error_code,omitempty"`
}

// NewTwilioProvider создает новый провайдер Twilio.
func NewTwilioProvider(cfg *Config, logger *zap.Logger) (*TwilioProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("Account SID для Twilio не указан")
	}
	if cfg.APIToken == "" {
		return nil, fmt.Errorf("Auth Token для Twilio не указан")
	}
	if cfg.From == "" {
		return nil, fmt.Errorf("номер отправителя для Twilio не указан")
	}

	return &TwilioProvider{
		accountSID: cfg.APIKey,
		authToken:  cfg.APIToken,
		from:       cfg.From,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}, nil
}

// SendSMS отправляет SMS через Twilio.
func (p *TwilioProvider) SendSMS(ctx context.Context, phone, message string) error {
	// Twilio API endpoint
	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", p.accountSID)

	// Подготовка параметров
	params := url.Values{}
	params.Set("To", phone)
	params.Set("From", p.from)
	params.Set("Body", message)

	// Создание запроса
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(params.Encode()))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}

	// Basic Auth
	auth := base64.StdEncoding.EncodeToString([]byte(p.accountSID + ":" + p.authToken))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Отправка запроса
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка отправки запроса: %w", err)
	}
	defer resp.Body.Close()

	// Чтение ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	// Парсинг JSON ответа
	var twilioResp TwilioResponse
	if err := json.Unmarshal(body, &twilioResp); err != nil {
		return fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	// Проверка на ошибки
	if twilioResp.ErrorCode != 0 {
		return fmt.Errorf("ошибка Twilio API: код %d, %s", twilioResp.ErrorCode, twilioResp.ErrorMsg)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("ошибка HTTP: код %d, %s", resp.StatusCode, twilioResp.ErrorMsg)
	}

	p.logger.Info("SMS отправлена через Twilio",
		zap.String("phone", phone),
		zap.String("sid", twilioResp.SID),
		zap.String("status", twilioResp.Status),
	)

	return nil
}

// GetName возвращает название провайдера.
func (p *TwilioProvider) GetName() string {
	return "Twilio"
}
