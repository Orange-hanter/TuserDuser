// Package telegramclient provides a gRPC client for the telegram-service.
// This client is used by event-api to communicate with telegram-service.
package telegramclient

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// Client wraps gRPC calls to telegram-service.
type Client struct {
	conn    *grpc.ClientConn
	timeout time.Duration
	logger  *zap.Logger
}

// Config holds configuration for the telegram service client.
type Config struct {
	Address string        // gRPC server address (e.g., "localhost:50051")
	Timeout time.Duration // Request timeout (default: 1s)
}

// NewClient creates a new gRPC client for telegram-service.
func NewClient(cfg Config, logger *zap.Logger) (*Client, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = time.Second // 1s default as per spec
	}

	conn, err := grpc.Dial(
		cfg.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             3 * time.Second,
			PermitWithoutStream: true,
		}),
		WithJSONCodec(), // Use JSON codec since we don't have generated proto stubs
	)
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:    conn,
		timeout: cfg.Timeout,
		logger:  logger,
	}, nil
}

// Close closes the gRPC connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// BindingLinkResult contains the result of GenerateBindingLink.
type BindingLinkResult struct {
	DeepLink  string
	Token     string
	Code      string
	ExpiresAt time.Time
}

// GenerateBindingLink requests a new binding deep link for a user.
func (c *Client) GenerateBindingLink(ctx context.Context, userID string) (*BindingLinkResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req := &GenerateBindingLinkRequest{UserId: userID}
	resp, err := c.generateBindingLink(ctx, req)
	if err != nil {
		c.logger.Error("gRPC GenerateBindingLink failed",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return nil, &ServiceError{Code: "service_unavailable", Message: "failed to connect to telegram service"}
	}

	if !resp.Success {
		return nil, &ServiceError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return &BindingLinkResult{
		DeepLink:  resp.Deeplink,
		Token:     resp.Token,
		Code:      resp.Code,
		ExpiresAt: time.Unix(resp.ExpiresAtUnix, 0),
	}, nil
}

// SendResult contains the result of a send operation.
type SendResult struct {
	SentAt    time.Time
	MessageID string
}

// SendVerificationCode sends a verification code to a user via Telegram.
func (c *Client) SendVerificationCode(ctx context.Context, userID, code string, expiresInMinutes int32) (*SendResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req := &SendVerificationCodeRequest{
		UserId:           userID,
		Code:             code,
		ExpiresInMinutes: expiresInMinutes,
	}
	resp, err := c.sendVerificationCode(ctx, req)
	if err != nil {
		c.logger.Error("gRPC SendVerificationCode failed",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return nil, &ServiceError{Code: "service_unavailable", Message: "failed to connect to telegram service"}
	}

	if !resp.Success {
		return nil, &ServiceError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return &SendResult{
		SentAt:    time.Unix(resp.SentAtUnix, 0),
		MessageID: resp.MessageId,
	}, nil
}

// SendEventReminder sends an event reminder to a user via Telegram.
func (c *Client) SendEventReminder(ctx context.Context, userID, eventID, title, description string, startTime time.Time, location, deeplinkURL string) (*SendResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req := &SendEventReminderRequest{
		UserId:           userID,
		EventId:          eventID,
		EventTitle:       title,
		EventDescription: description,
		EventStartUnix:   startTime.Unix(),
		EventLocation:    location,
		DeeplinkUrl:      deeplinkURL,
	}
	resp, err := c.sendEventReminder(ctx, req)
	if err != nil {
		c.logger.Error("gRPC SendEventReminder failed",
			zap.String("user_id", userID),
			zap.String("event_id", eventID),
			zap.Error(err),
		)
		return nil, &ServiceError{Code: "service_unavailable", Message: "failed to connect to telegram service"}
	}

	if !resp.Success {
		return nil, &ServiceError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return &SendResult{
		SentAt:    time.Unix(resp.SentAtUnix, 0),
		MessageID: resp.MessageId,
	}, nil
}

// SendMessage sends a custom message to a user via Telegram.
func (c *Client) SendMessage(ctx context.Context, userID, text, parseMode string, silent bool, replyMarkup string) (*SendResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req := &SendMessageRequest{
		UserId:          userID,
		Text:            text,
		ParseMode:       parseMode,
		Silent:          silent,
		ReplyMarkupJson: replyMarkup,
	}
	resp, err := c.sendMessage(ctx, req)
	if err != nil {
		c.logger.Error("gRPC SendMessage failed",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return nil, &ServiceError{Code: "service_unavailable", Message: "failed to connect to telegram service"}
	}

	if !resp.Success {
		return nil, &ServiceError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return &SendResult{
		SentAt:    time.Unix(resp.SentAtUnix, 0),
		MessageID: resp.MessageId,
	}, nil
}

// BindingStatus contains user binding information.
type BindingStatus struct {
	IsBound   bool
	Status    string
	Username  string
	FirstName string
	LastName  string
	ChatID    int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsUserBound checks if a user has an active Telegram binding.
func (c *Client) IsUserBound(ctx context.Context, userID string) (bool, string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req := &IsUserBoundRequest{UserId: userID}
	resp, err := c.isUserBound(ctx, req)
	if err != nil {
		c.logger.Error("gRPC IsUserBound failed",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return false, "", &ServiceError{Code: "service_unavailable", Message: "failed to connect to telegram service"}
	}

	if !resp.Success {
		return false, "", &ServiceError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return resp.IsBound, resp.Status, nil
}

// GetBindingStatus returns detailed binding information for a user.
func (c *Client) GetBindingStatus(ctx context.Context, userID string) (*BindingStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req := &GetBindingStatusRequest{UserId: userID}
	resp, err := c.getBindingStatus(ctx, req)
	if err != nil {
		c.logger.Error("gRPC GetBindingStatus failed",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return nil, &ServiceError{Code: "service_unavailable", Message: "failed to connect to telegram service"}
	}

	if !resp.Success {
		return nil, &ServiceError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return &BindingStatus{
		IsBound:   resp.Status == "active",
		Status:    resp.Status,
		Username:  resp.TelegramUsername,
		FirstName: resp.TelegramFirstName,
		LastName:  resp.TelegramLastName,
		ChatID:    resp.ChatId,
		CreatedAt: time.Unix(resp.CreatedAtUnix, 0),
		UpdatedAt: time.Unix(resp.UpdatedAtUnix, 0),
	}, nil
}

// UnbindUser removes the Telegram binding for a user.
func (c *Client) UnbindUser(ctx context.Context, userID, reason string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req := &UnbindUserRequest{UserId: userID, Reason: reason}
	resp, err := c.unbindUser(ctx, req)
	if err != nil {
		c.logger.Error("gRPC UnbindUser failed",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return &ServiceError{Code: "service_unavailable", Message: "failed to connect to telegram service"}
	}

	if !resp.Success {
		return &ServiceError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return nil
}

// ServiceError represents an error from the telegram service.
type ServiceError struct {
	Code    string
	Message string
}

func (e *ServiceError) Error() string {
	return e.Message
}

// IsUserNotBound returns true if the error indicates user is not bound.
func (e *ServiceError) IsUserNotBound() bool {
	return e.Code == "user_not_bound"
}

// IsBlocked returns true if the error indicates user blocked the bot.
func (e *ServiceError) IsBlocked() bool {
	return e.Code == "blocked"
}

// IsRateLimited returns true if the error indicates rate limiting.
func (e *ServiceError) IsRateLimited() bool {
	return e.Code == "rate_limited"
}
