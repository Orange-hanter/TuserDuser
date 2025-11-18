package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Client sends outbound messages to Telegram Bot API.
type Client interface {
	SendMessage(ctx context.Context, msg OutboundMessage) (*SendResult, error)
}

// APIError wraps Telegram Bot API error metadata.
type APIError struct {
	Code        int
	Description string
	RetryAfter  time.Duration
}

func (e *APIError) Error() string {
	return fmt.Sprintf("telegram api error %d: %s", e.Code, e.Description)
}

// HTTPClient implements Client over net/http.
type HTTPClient struct {
	botToken string
	baseURL  string
	client   *http.Client
}

// NewHTTPClient builds a client for Telegram API.
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

// SendMessage calls Telegram sendMessage.
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
		return nil, err
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", c.baseURL, c.botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp sendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
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

	if apiResp.Result == nil {
		return nil, fmt.Errorf("telegram api ok without result")
	}
	return &SendResult{MessageID: strconv.FormatInt(apiResp.Result.MessageID, 10)}, nil
}

type sendMessageResponse struct {
	OK          bool                `json:"ok"`
	Result      *telegramMessage    `json:"result"`
	Description string              `json:"description"`
	ErrorCode   int                 `json:"error_code"`
	Parameters  *responseParameters `json:"parameters"`
}

type telegramMessage struct {
	MessageID int64 `json:"message_id"`
}

type responseParameters struct {
	RetryAfter int `json:"retry_after"`
}
