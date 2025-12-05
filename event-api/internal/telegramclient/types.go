// Package telegramclient provides gRPC types for telegram-service communication.
// These types mirror the proto definitions in telegram-service.
package telegramclient

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc"
)

// Request/Response types matching telegram-service proto

type GenerateBindingLinkRequest struct {
	UserId string `json:"user_id"`
}

type GenerateBindingLinkResponse struct {
	Success       bool   `json:"success"`
	ErrorCode     string `json:"error_code"`
	ErrorMessage  string `json:"error_message"`
	Deeplink      string `json:"deeplink"`
	Token         string `json:"token"`
	Code          string `json:"code"` // Short 6-character code for manual entry
	ExpiresAtUnix int64  `json:"expires_at_unix"`
}

type SendVerificationCodeRequest struct {
	UserId           string `json:"user_id"`
	Code             string `json:"code"`
	ExpiresInMinutes int32  `json:"expires_in_minutes"`
}

type SendVerificationCodeResponse struct {
	Success      bool   `json:"success"`
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	SentAtUnix   int64  `json:"sent_at_unix"`
	MessageId    string `json:"message_id"`
}

type SendEventReminderRequest struct {
	UserId           string `json:"user_id"`
	EventId          string `json:"event_id"`
	EventTitle       string `json:"event_title"`
	EventDescription string `json:"event_description"`
	EventStartUnix   int64  `json:"event_start_unix"`
	EventLocation    string `json:"event_location"`
	DeeplinkUrl      string `json:"deeplink_url"`
}

type SendEventReminderResponse struct {
	Success      bool   `json:"success"`
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	SentAtUnix   int64  `json:"sent_at_unix"`
	MessageId    string `json:"message_id"`
}

type SendMessageRequest struct {
	UserId          string `json:"user_id"`
	Text            string `json:"text"`
	ParseMode       string `json:"parse_mode"`
	Silent          bool   `json:"silent"`
	ReplyMarkupJson string `json:"reply_markup_json"`
}

type SendMessageResponse struct {
	Success      bool   `json:"success"`
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	SentAtUnix   int64  `json:"sent_at_unix"`
	MessageId    string `json:"message_id"`
}

type IsUserBoundRequest struct {
	UserId string `json:"user_id"`
}

type IsUserBoundResponse struct {
	Success      bool   `json:"success"`
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	IsBound      bool   `json:"is_bound"`
	Status       string `json:"status"`
}

type GetBindingStatusRequest struct {
	UserId string `json:"user_id"`
}

type GetBindingStatusResponse struct {
	Success           bool   `json:"success"`
	ErrorCode         string `json:"error_code"`
	ErrorMessage      string `json:"error_message"`
	Status            string `json:"status"`
	TelegramUsername  string `json:"telegram_username"`
	TelegramFirstName string `json:"telegram_first_name"`
	TelegramLastName  string `json:"telegram_last_name"`
	ChatId            int64  `json:"chat_id"`
	CreatedAtUnix     int64  `json:"created_at_unix"`
	UpdatedAtUnix     int64  `json:"updated_at_unix"`
	BlockedReason     string `json:"blocked_reason"`
}

type UnbindUserRequest struct {
	UserId string `json:"user_id"`
	Reason string `json:"reason"`
}

type UnbindUserResponse struct {
	Success      bool   `json:"success"`
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

// RegisterPendingVerificationRequest - запрос для регистрации отложенной верификации.
type RegisterPendingVerificationRequest struct {
	UserId           string `json:"user_id"`
	VerificationCode string `json:"verification_code"`
	TTLMinutes       int32  `json:"ttl_minutes,omitempty"`
}

// RegisterPendingVerificationResponse - ответ на регистрацию отложенной верификации.
type RegisterPendingVerificationResponse struct {
	Success       bool   `json:"success"`
	ErrorCode     string `json:"error_code"`
	ErrorMessage  string `json:"error_message"`
	Deeplink      string `json:"deeplink"`
	Token         string `json:"token"`
	Code          string `json:"code"`
	ExpiresAtUnix int64  `json:"expires_at_unix"`
}

// GetPendingVerificationStatusRequest - запрос статуса pending verification.
type GetPendingVerificationStatusRequest struct {
	UserId string `json:"user_id"`
}

// GetPendingVerificationStatusResponse - ответ статуса pending verification.
type GetPendingVerificationStatusResponse struct {
	Success       bool   `json:"success"`
	ErrorCode     string `json:"error_code"`
	ErrorMessage  string `json:"error_message"`
	HasPending    bool   `json:"has_pending"`
	ExpiresAtUnix int64  `json:"expires_at_unix"`
}

// gRPC method implementations using the connection
// These use manual invocation since we don't have generated stubs

func (c *Client) generateBindingLink(ctx context.Context, req *GenerateBindingLinkRequest) (*GenerateBindingLinkResponse, error) {
	resp := &GenerateBindingLinkResponse{}
	err := c.conn.Invoke(ctx, "/telegram.v1.TelegramService/GenerateBindingLink", req, resp)
	return resp, err
}

func (c *Client) sendVerificationCode(ctx context.Context, req *SendVerificationCodeRequest) (*SendVerificationCodeResponse, error) {
	resp := &SendVerificationCodeResponse{}
	err := c.conn.Invoke(ctx, "/telegram.v1.TelegramService/SendVerificationCode", req, resp)
	return resp, err
}

func (c *Client) sendEventReminder(ctx context.Context, req *SendEventReminderRequest) (*SendEventReminderResponse, error) {
	resp := &SendEventReminderResponse{}
	err := c.conn.Invoke(ctx, "/telegram.v1.TelegramService/SendEventReminder", req, resp)
	return resp, err
}

func (c *Client) sendMessage(ctx context.Context, req *SendMessageRequest) (*SendMessageResponse, error) {
	resp := &SendMessageResponse{}
	err := c.conn.Invoke(ctx, "/telegram.v1.TelegramService/SendMessage", req, resp)
	return resp, err
}

func (c *Client) isUserBound(ctx context.Context, req *IsUserBoundRequest) (*IsUserBoundResponse, error) {
	resp := &IsUserBoundResponse{}
	err := c.conn.Invoke(ctx, "/telegram.v1.TelegramService/IsUserBound", req, resp)
	return resp, err
}

func (c *Client) getBindingStatus(ctx context.Context, req *GetBindingStatusRequest) (*GetBindingStatusResponse, error) {
	resp := &GetBindingStatusResponse{}
	err := c.conn.Invoke(ctx, "/telegram.v1.TelegramService/GetBindingStatus", req, resp)
	return resp, err
}

func (c *Client) unbindUser(ctx context.Context, req *UnbindUserRequest) (*UnbindUserResponse, error) {
	resp := &UnbindUserResponse{}
	err := c.conn.Invoke(ctx, "/telegram.v1.TelegramService/UnbindUser", req, resp)
	return resp, err
}

func (c *Client) registerPendingVerification(ctx context.Context, req *RegisterPendingVerificationRequest) (*RegisterPendingVerificationResponse, error) {
	resp := &RegisterPendingVerificationResponse{}
	err := c.conn.Invoke(ctx, "/telegram.v1.TelegramService/RegisterPendingVerification", req, resp)
	return resp, err
}

func (c *Client) getPendingVerificationStatus(ctx context.Context, req *GetPendingVerificationStatusRequest) (*GetPendingVerificationStatusResponse, error) {
	resp := &GetPendingVerificationStatusResponse{}
	err := c.conn.Invoke(ctx, "/telegram.v1.TelegramService/GetPendingVerificationStatus", req, resp)
	return resp, err
}

// Codec implementation for JSON encoding (alternative to protobuf)
// This allows the client to work without generated protobuf code

type jsonCodec struct{}

func (jsonCodec) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (jsonCodec) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (jsonCodec) Name() string {
	return "json"
}

// WithJSONCodec returns a dial option that uses JSON instead of protobuf.
// Use this if you don't have protobuf stubs generated.
func WithJSONCodec() grpc.DialOption {
	return grpc.WithDefaultCallOptions(grpc.ForceCodec(jsonCodec{}))
}
