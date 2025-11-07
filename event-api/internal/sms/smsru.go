package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
)

// SMSRuProvider провайдер для SMS.RU API.
type SMSRuProvider struct {
	apiID  string
	from   string
	client *http.Client
	logger *zap.Logger
}

// SMSRuResponse ответ от SMS.RU API.
type SMSRuResponse struct {
	Status     string `json:"status"`
	StatusCode int    `json:"status_code"`
	SMS        map[string]struct {
		Status     string `json:"status"`
		StatusCode int    `json:"status_code"`
		SMSID      string `json:"sms_id"`
	} `json:"sms"`
	Balance float64 `json:"balance"`
}

// NewSMSRuProvider создает новый провайдер SMS.RU.
func NewSMSRuProvider(cfg *Config, logger *zap.Logger) (*SMSRuProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API ключ для SMS.RU не указан")
	}

	return &SMSRuProvider{
		apiID: cfg.APIKey,
		from:  cfg.From,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}, nil
}

// SendSMS отправляет SMS через SMS.RU.
func (p *SMSRuProvider) SendSMS(ctx context.Context, phone, message string) error {
	// SMS.RU API endpoint
	apiURL := "https://sms.ru/sms/send"

	// Подготовка параметров
	params := url.Values{}
	params.Set("api_id", p.apiID)
	params.Set("to", phone)
	params.Set("msg", message)
	params.Set("json", "1") // JSON ответ

	if p.from != "" {
		params.Set("from", p.from)
	}

	// Создание запроса
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(params.Encode()))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Отправка запроса
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка отправки запроса: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			p.logger.Warn("Не удалось закрыть тело ответа SMS.RU", zap.Error(err))
		}
	}()

	// Чтение ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	// Парсинг JSON ответа
	var smsResp SMSRuResponse
	if err := json.Unmarshal(body, &smsResp); err != nil {
		return fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	// Проверка статуса
	if smsResp.StatusCode != 100 {
		return fmt.Errorf("ошибка SMS.RU API: код %d, статус %s", smsResp.StatusCode, smsResp.Status)
	}

	p.logger.Info("SMS отправлена через SMS.RU",
		zap.String("phone", phone),
		zap.Float64("balance", smsResp.Balance),
	)

	return nil
}

// GetName возвращает название провайдера.
func (p *SMSRuProvider) GetName() string {
	return "SMS.RU"
}
