// Package polling provides long polling support for Telegram Bot API.
package polling

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Update represents a Telegram update received via long polling.
type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message,omitempty"`
}

// Message represents a Telegram message.
type Message struct {
	MessageID int64  `json:"message_id"`
	From      *User  `json:"from,omitempty"`
	Chat      *Chat  `json:"chat"`
	Date      int64  `json:"date"`
	Text      string `json:"text,omitempty"`
}

// User represents a Telegram user.
type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

// Chat represents a Telegram chat.
type Chat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title,omitempty"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

// UpdateHandler is a function that processes incoming updates.
type UpdateHandler func(ctx context.Context, update *Update) error

// Config holds polling configuration.
type Config struct {
	BotToken       string
	BaseURL        string
	PollTimeout    int
	RetryDelay     time.Duration
	MaxRetries     int
	AllowedUpdates []string
}

// Poller implements long polling for Telegram Bot API.
type Poller struct {
	cfg     Config
	client  *http.Client
	handler UpdateHandler
	logger  *zap.Logger

	mu       sync.Mutex
	running  bool
	stopCh   chan struct{}
	offset   int64
	lastErr  error
	errCount int
}

// NewPoller creates a new long polling client.
func NewPoller(cfg Config, handler UpdateHandler, logger *zap.Logger) *Poller {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.telegram.org"
	}
	if cfg.PollTimeout == 0 {
		cfg.PollTimeout = 30
	}
	if cfg.RetryDelay == 0 {
		cfg.RetryDelay = 3 * time.Second
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 10
	}

	return &Poller{
		cfg:     cfg,
		handler: handler,
		logger:  logger,
		client: &http.Client{
			Timeout: time.Duration(cfg.PollTimeout+10) * time.Second,
		},
	}
}

// Start begins the polling loop.
func (p *Poller) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return fmt.Errorf("poller already running")
	}
	p.running = true
	p.stopCh = make(chan struct{})
	p.mu.Unlock()

	p.logger.Info("starting long polling",
		zap.Int("timeout", p.cfg.PollTimeout),
		zap.Strings("allowed_updates", p.cfg.AllowedUpdates),
	)

	if err := p.deleteWebhook(ctx); err != nil {
		p.logger.Warn("failed to delete webhook", zap.Error(err))
	}

	go p.pollLoop(ctx)
	return nil
}

// Stop gracefully stops the polling loop.
func (p *Poller) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return
	}

	p.running = false
	close(p.stopCh)
	p.logger.Info("stopping long polling")
}

// IsRunning returns true if poller is currently running.
func (p *Poller) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

func (p *Poller) pollLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("polling stopped: context cancelled")
			return
		case <-p.stopCh:
			p.logger.Info("polling stopped: stop signal received")
			return
		default:
			updates, err := p.getUpdates(ctx)
			if err != nil {
				p.handleError(err)
				continue
			}

			p.mu.Lock()
			p.errCount = 0
			p.lastErr = nil
			p.mu.Unlock()

			for _, update := range updates {
				if update.UpdateID >= p.offset {
					p.offset = update.UpdateID + 1
				}

				if err := p.handler(ctx, &update); err != nil {
					p.logger.Error("failed to handle update",
						zap.Int64("update_id", update.UpdateID),
						zap.Error(err),
					)
				}
			}
		}
	}
}

func (p *Poller) getUpdates(ctx context.Context) ([]Update, error) {
	params := url.Values{}
	params.Set("timeout", strconv.Itoa(p.cfg.PollTimeout))

	if p.offset > 0 {
		params.Set("offset", strconv.FormatInt(p.offset, 10))
	}

	if len(p.cfg.AllowedUpdates) > 0 {
		allowedJSON, _ := json.Marshal(p.cfg.AllowedUpdates)
		params.Set("allowed_updates", string(allowedJSON))
	}

	apiURL := fmt.Sprintf("%s/bot%s/getUpdates?%s", p.cfg.BaseURL, p.cfg.BotToken, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var apiResp struct {
		OK          bool     `json:"ok"`
		Description string   `json:"description,omitempty"`
		Result      []Update `json:"result,omitempty"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if !apiResp.OK {
		return nil, fmt.Errorf("telegram api error: %s", apiResp.Description)
	}

	if len(apiResp.Result) > 0 {
		p.logger.Debug("received updates", zap.Int("count", len(apiResp.Result)))
	}

	return apiResp.Result, nil
}

func (p *Poller) deleteWebhook(ctx context.Context) error {
	apiURL := fmt.Sprintf("%s/bot%s/deleteWebhook", p.cfg.BaseURL, p.cfg.BotToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, nil)
	if err != nil {
		return err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var apiResp struct {
		OK          bool   `json:"ok"`
		Description string `json:"description,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return err
	}

	if !apiResp.OK {
		return fmt.Errorf("telegram: %s", apiResp.Description)
	}

	p.logger.Info("webhook deleted, switched to polling mode")
	return nil
}

func (p *Poller) handleError(err error) {
	p.mu.Lock()
	p.lastErr = err
	p.errCount++
	errCount := p.errCount
	p.mu.Unlock()

	delay := p.cfg.RetryDelay * time.Duration(minInt(errCount, p.cfg.MaxRetries))

	p.logger.Error("polling error, will retry",
		zap.Error(err),
		zap.Int("consecutive_errors", errCount),
		zap.Duration("retry_delay", delay),
	)

	time.Sleep(delay)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
