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

// SMSCProvider провайдер для SMSC.RU API
type SMSCProvider struct {
	login    string
	password string
	from     string
	client   *http.Client
	logger   *zap.Logger
}

// SMSCResponse ответ от SMSC.RU API
type SMSCResponse struct {
	ID      int     `json:"id"`
	Cnt     int     `json:"cnt"`
	Balance float64 `json:"balance"`
	Error   string  `json:"error"`
	ErrCode int     `json:"error_code"`
}

// NewSMSCProvider создает новый провайдер SMSC.RU
func NewSMSCProvider(cfg *Config, logger *zap.Logger) (*SMSCProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("логин для SMSC.RU не указан")
	}
	if cfg.APIToken == "" {
		return nil, fmt.Errorf("пароль для SMSC.RU не указан")
	}

	return &SMSCProvider{
		login:    cfg.APIKey,
		password: cfg.APIToken,
		from:     cfg.From,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}, nil
}

// SendSMS отправляет SMS через SMSC.RU
func (p *SMSCProvider) SendSMS(ctx context.Context, phone, message string) error {
	// SMSC.RU API endpoint
	apiURL := "https://smsc.ru/sys/send.php"

	// Подготовка параметров
	params := url.Values{}
	params.Set("login", p.login)
	params.Set("psw", p.password)
	params.Set("phones", phone)
	params.Set("mes", message)
	params.Set("fmt", "3") // JSON ответ
	params.Set("charset", "utf-8")

	if p.from != "" {
		params.Set("sender", p.from)
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
	defer resp.Body.Close()

	// Чтение ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	// Парсинг JSON ответа
	var smscResp SMSCResponse
	if err := json.Unmarshal(body, &smscResp); err != nil {
		return fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	// Проверка на ошибки
	if smscResp.Error != "" || smscResp.ErrCode != 0 {
		return fmt.Errorf("ошибка SMSC.RU API: код %d, %s", smscResp.ErrCode, smscResp.Error)
	}

	p.logger.Info("SMS отправлена через SMSC.RU",
		zap.String("phone", phone),
		zap.Int("id", smscResp.ID),
		zap.Float64("balance", smscResp.Balance),
	)

	return nil
}

// GetName возвращает название провайдера
func (p *SMSCProvider) GetName() string {
	return "SMSC.RU"
}
