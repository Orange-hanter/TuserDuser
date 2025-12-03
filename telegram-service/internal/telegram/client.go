// Package telegram provides Telegram Bot API client functionality.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client defines the interface for Telegram Bot API operations.
type Client interface {
	SendMessage(ctx context.Context, msg OutboundMessage) (*SendResult, error)
}

// OutboundMessage represents a message to be sent via Telegram.
type OutboundMessage struct {
	ChatID      int64
	Text        string
	ParseMode   string          // "MarkdownV2", "HTML", or empty
	Silent      bool            // disable_notification
	ReplyMarkup json.RawMessage // inline keyboard JSON
}

// SendResult contains the result of a send operation.
type SendResult struct {
	MessageID int64
	SentAt    time.Time
}

// APIError represents a Telegram Bot API error.
type APIError struct {
	Code        int
	Description string
	RetryAfter  time.Duration
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return fmt.Sprintf("telegram api error %d: %s", e.Code, e.Description)
}

// IsBlocked returns true if the error indicates user blocked the bot.
func (e *APIError) IsBlocked() bool {
	return e.Code == 403
}

// IsRateLimited returns true if the error indicates rate limiting.
func (e *APIError) IsRateLimited() bool {
	return e.Code == 429
}

// sendMessageResponse is the Telegram API response structure.
type sendMessageResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
	ErrorCode   int    `json:"error_code,omitempty"`
	Result      *struct {
		MessageID int64 `json:"message_id"`
		Date      int64 `json:"date"`
	} `json:"result,omitempty"`
	Parameters *struct {
		RetryAfter int `json:"retry_after,omitempty"`
	} `json:"parameters,omitempty"`
}

// HTTPClient implements the Client interface using HTTP.
type HTTPClient struct {
	botToken string
	baseURL  string
	client   *http.Client
}

// NewHTTPClient creates a new Telegram Bot API client.
func NewHTTPClient(botToken, baseURL string) *HTTPClient {
	if baseURL == "" {
		baseURL = "https://api.telegram.org"
	}
	return &HTTPClient{
		botToken: botToken,
		baseURL:  baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendMessage sends a message to a Telegram chat.
func (c *HTTPClient) SendMessage(ctx context.Context, msg OutboundMessage) (*SendResult, error) {
	payload := map[string]any{
		"chat_id": msg.ChatID,
		"text":    msg.Text,
	}
	if msg.ParseMode != "" {
		payload["parse_mode"] = msg.ParseMode
	}
	if msg.Silent {
		payload["disable_notification"] = true
	}
	if len(msg.ReplyMarkup) > 0 {
		payload["reply_markup"] = json.RawMessage(msg.ReplyMarkup)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", c.baseURL, c.botToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	var apiResp sendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if !apiResp.OK {
		apiErr := &APIError{
			Code:        apiResp.ErrorCode,
			Description: apiResp.Description,
		}
		if apiResp.Parameters != nil && apiResp.Parameters.RetryAfter > 0 {
			apiErr.RetryAfter = time.Duration(apiResp.Parameters.RetryAfter) * time.Second
		}
		return nil, apiErr
	}

	result := &SendResult{
		SentAt: time.Now(),
	}
	if apiResp.Result != nil {
		result.MessageID = apiResp.Result.MessageID
		result.SentAt = time.Unix(apiResp.Result.Date, 0)
	}

	return result, nil
}
