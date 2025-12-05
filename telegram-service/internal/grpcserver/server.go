// Package grpcserver implements the gRPC server for telegram-service.
package grpcserver

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"telegram-service/internal/metrics"
	"telegram-service/internal/service"
)

// TelegramServiceServer implements the gRPC TelegramService.
type TelegramServiceServer struct {
	UnimplementedTelegramServiceServer
	service *service.TelegramService
	logger  *zap.Logger
}

// NewTelegramServiceServer creates a new gRPC server handler.
func NewTelegramServiceServer(svc *service.TelegramService, logger *zap.Logger) *TelegramServiceServer {
	return &TelegramServiceServer{
		service: svc,
		logger:  logger,
	}
}

// GenerateBindingLink creates a deep link for user to bind their Telegram account.
func (s *TelegramServiceServer) GenerateBindingLink(ctx context.Context, req *GenerateBindingLinkRequest) (*GenerateBindingLinkResponse, error) {
	if req.UserId == "" {
		return &GenerateBindingLinkResponse{
			Success:      false,
			ErrorCode:    "invalid_user_id",
			ErrorMessage: "user_id is required",
		}, nil
	}

	result, err := s.service.GenerateBindingLink(ctx, req.UserId)
	if err != nil {
		s.logger.Error("failed to generate binding link",
			zap.String("user_id", req.UserId),
			zap.Error(err),
		)
		return &GenerateBindingLinkResponse{
			Success:      false,
			ErrorCode:    "service_unavailable",
			ErrorMessage: "failed to generate binding link",
		}, nil
	}

	return &GenerateBindingLinkResponse{
		Success:       true,
		Deeplink:      result.DeepLink,
		Token:         result.Token,
		Code:          result.Code,
		ExpiresAtUnix: result.ExpiresAt.Unix(),
	}, nil
}

// SendVerificationCode sends a verification code to a bound user.
func (s *TelegramServiceServer) SendVerificationCode(ctx context.Context, req *SendVerificationCodeRequest) (*SendVerificationCodeResponse, error) {
	if req.UserId == "" {
		return &SendVerificationCodeResponse{
			Success:      false,
			ErrorCode:    "invalid_user_id",
			ErrorMessage: "user_id is required",
		}, nil
	}

	if req.Code == "" {
		return &SendVerificationCodeResponse{
			Success:      false,
			ErrorCode:    "invalid_code",
			ErrorMessage: "code is required",
		}, nil
	}

	result, err := s.service.SendVerificationCode(ctx, req.UserId, req.Code, req.ExpiresInMinutes)
	if err != nil {
		return s.handleSendError(err, req.UserId, "send verification code")
	}

	return &SendVerificationCodeResponse{
		Success:    true,
		SentAtUnix: result.SentAt.Unix(),
		MessageId:  formatMessageID(result.MessageID),
	}, nil
}

// SendEventReminder sends an event reminder to a bound user.
func (s *TelegramServiceServer) SendEventReminder(ctx context.Context, req *SendEventReminderRequest) (*SendEventReminderResponse, error) {
	if req.UserId == "" {
		return &SendEventReminderResponse{
			Success:      false,
			ErrorCode:    "invalid_user_id",
			ErrorMessage: "user_id is required",
		}, nil
	}

	startTime := time.Unix(req.EventStartUnix, 0)
	result, err := s.service.SendEventReminder(
		ctx,
		req.UserId,
		req.EventId,
		req.EventTitle,
		req.EventDescription,
		startTime,
		req.EventLocation,
		req.DeeplinkUrl,
	)
	if err != nil {
		resp := &SendEventReminderResponse{}
		s.populateSendError(resp, err, req.UserId, "send event reminder")
		return resp, nil
	}

	return &SendEventReminderResponse{
		Success:    true,
		SentAtUnix: result.SentAt.Unix(),
		MessageId:  formatMessageID(result.MessageID),
	}, nil
}

// SendMessage sends a custom message to a bound user.
func (s *TelegramServiceServer) SendMessage(ctx context.Context, req *SendMessageRequest) (*SendMessageResponse, error) {
	if req.UserId == "" {
		return &SendMessageResponse{
			Success:      false,
			ErrorCode:    "invalid_user_id",
			ErrorMessage: "user_id is required",
		}, nil
	}

	if req.Text == "" {
		return &SendMessageResponse{
			Success:      false,
			ErrorCode:    "invalid_text",
			ErrorMessage: "text is required",
		}, nil
	}

	result, err := s.service.SendMessage(ctx, req.UserId, req.Text, req.ParseMode, req.Silent, req.ReplyMarkupJson)
	if err != nil {
		resp := &SendMessageResponse{}
		s.populateSendMessageError(resp, err, req.UserId, "send message")
		return resp, nil
	}

	return &SendMessageResponse{
		Success:    true,
		SentAtUnix: result.SentAt.Unix(),
		MessageId:  formatMessageID(result.MessageID),
	}, nil
}

// IsUserBound checks if a user has an active Telegram binding.
func (s *TelegramServiceServer) IsUserBound(ctx context.Context, req *IsUserBoundRequest) (*IsUserBoundResponse, error) {
	if req.UserId == "" {
		return &IsUserBoundResponse{
			Success:      false,
			ErrorCode:    "invalid_user_id",
			ErrorMessage: "user_id is required",
		}, nil
	}

	isBound, bindingStatus, err := s.service.IsUserBound(ctx, req.UserId)
	if err != nil {
		s.logger.Error("failed to check binding status",
			zap.String("user_id", req.UserId),
			zap.Error(err),
		)
		return &IsUserBoundResponse{
			Success:      false,
			ErrorCode:    "service_unavailable",
			ErrorMessage: "failed to check binding status",
		}, nil
	}

	return &IsUserBoundResponse{
		Success: true,
		IsBound: isBound,
		Status:  bindingStatus,
	}, nil
}

// GetBindingStatus returns detailed binding information.
func (s *TelegramServiceServer) GetBindingStatus(ctx context.Context, req *GetBindingStatusRequest) (*GetBindingStatusResponse, error) {
	if req.UserId == "" {
		return &GetBindingStatusResponse{
			Success:      false,
			ErrorCode:    "invalid_user_id",
			ErrorMessage: "user_id is required",
		}, nil
	}

	binding, err := s.service.GetBindingStatus(ctx, req.UserId)
	if err != nil {
		if err == service.ErrUserNotBound {
			return &GetBindingStatusResponse{
				Success:      false,
				ErrorCode:    "user_not_bound",
				ErrorMessage: "user does not have a Telegram binding",
			}, nil
		}
		s.logger.Error("failed to get binding status",
			zap.String("user_id", req.UserId),
			zap.Error(err),
		)
		return &GetBindingStatusResponse{
			Success:      false,
			ErrorCode:    "service_unavailable",
			ErrorMessage: "failed to get binding status",
		}, nil
	}

	resp := &GetBindingStatusResponse{
		Success:           true,
		Status:            string(binding.Status),
		TelegramUsername:  binding.Username,
		TelegramFirstName: binding.FirstName,
		TelegramLastName:  binding.LastName,
		ChatId:            binding.ChatID,
		CreatedAtUnix:     binding.CreatedAt.Unix(),
		UpdatedAtUnix:     binding.UpdatedAt.Unix(),
	}
	if binding.BlockedReason != nil {
		resp.BlockedReason = *binding.BlockedReason
	}

	return resp, nil
}

// UnbindUser removes the Telegram binding for a user.
func (s *TelegramServiceServer) UnbindUser(ctx context.Context, req *UnbindUserRequest) (*UnbindUserResponse, error) {
	if req.UserId == "" {
		return &UnbindUserResponse{
			Success:      false,
			ErrorCode:    "invalid_user_id",
			ErrorMessage: "user_id is required",
		}, nil
	}

	err := s.service.UnbindUser(ctx, req.UserId, req.Reason)
	if err != nil {
		if err == service.ErrUserNotBound {
			return &UnbindUserResponse{
				Success:      false,
				ErrorCode:    "user_not_bound",
				ErrorMessage: "user does not have a Telegram binding",
			}, nil
		}
		s.logger.Error("failed to unbind user",
			zap.String("user_id", req.UserId),
			zap.Error(err),
		)
		return &UnbindUserResponse{
			Success:      false,
			ErrorCode:    "service_unavailable",
			ErrorMessage: "failed to unbind user",
		}, nil
	}

	return &UnbindUserResponse{Success: true}, nil
}

// RegisterPendingVerification registers a verification code to be sent after user binds Telegram.
// This is the main entry point for the deferred telegram verification flow:
// 1. event-api calls this when user registers with verification_type=telegram
// 2. Returns binding link (deeplink + 6-char code) for user to bind Telegram
// 3. When user binds, the verification code is automatically sent to their Telegram
func (s *TelegramServiceServer) RegisterPendingVerification(ctx context.Context, req *RegisterPendingVerificationRequest) (*RegisterPendingVerificationResponse, error) {
	if req.UserId == "" {
		return &RegisterPendingVerificationResponse{
			Success:      false,
			ErrorCode:    "invalid_user_id",
			ErrorMessage: "user_id is required",
		}, nil
	}

	if req.VerificationCode == "" {
		return &RegisterPendingVerificationResponse{
			Success:      false,
			ErrorCode:    "invalid_code",
			ErrorMessage: "verification_code is required",
		}, nil
	}

	result, err := s.service.RegisterPendingVerification(ctx, req.UserId, req.VerificationCode, req.TtlMinutes)
	if err != nil {
		s.logger.Error("failed to register pending verification",
			zap.String("user_id", req.UserId),
			zap.Error(err),
		)
		return &RegisterPendingVerificationResponse{
			Success:      false,
			ErrorCode:    "service_unavailable",
			ErrorMessage: "failed to register pending verification",
		}, nil
	}

	return &RegisterPendingVerificationResponse{
		Success:       true,
		Deeplink:      result.DeepLink,
		Token:         result.Token,
		Code:          result.Code,
		ExpiresAtUnix: result.ExpiresAt.Unix(),
	}, nil
}

// GetPendingVerificationStatus checks if a user has a pending verification code waiting.
func (s *TelegramServiceServer) GetPendingVerificationStatus(ctx context.Context, req *GetPendingVerificationStatusRequest) (*GetPendingVerificationStatusResponse, error) {
	if req.UserId == "" {
		return &GetPendingVerificationStatusResponse{
			Success:      false,
			ErrorCode:    "invalid_user_id",
			ErrorMessage: "user_id is required",
		}, nil
	}

	hasPending, expiresAt, err := s.service.GetPendingVerificationStatus(ctx, req.UserId)
	if err != nil {
		s.logger.Error("failed to get pending verification status",
			zap.String("user_id", req.UserId),
			zap.Error(err),
		)
		return &GetPendingVerificationStatusResponse{
			Success:      false,
			ErrorCode:    "service_unavailable",
			ErrorMessage: "failed to get pending verification status",
		}, nil
	}

	resp := &GetPendingVerificationStatusResponse{
		Success:    true,
		HasPending: hasPending,
	}
	if hasPending {
		resp.ExpiresAtUnix = expiresAt.Unix()
	}

	return resp, nil
}

// handleSendError converts service errors to gRPC response.
func (s *TelegramServiceServer) handleSendError(err error, userID, operation string) (*SendVerificationCodeResponse, error) {
	s.logger.Error("failed to "+operation,
		zap.String("user_id", userID),
		zap.Error(err),
	)

	resp := &SendVerificationCodeResponse{Success: false}

	switch err {
	case service.ErrUserNotBound:
		resp.ErrorCode = "user_not_bound"
		resp.ErrorMessage = "user does not have an active Telegram binding"
	case service.ErrBlocked:
		resp.ErrorCode = "blocked"
		resp.ErrorMessage = "user has blocked the bot"
	case service.ErrRateLimited:
		resp.ErrorCode = "rate_limited"
		resp.ErrorMessage = "rate limited by Telegram"
	default:
		resp.ErrorCode = "send_failed"
		resp.ErrorMessage = "failed to send message"
	}

	return resp, nil
}

// populateSendError populates error response for SendEventReminderResponse.
func (s *TelegramServiceServer) populateSendError(resp *SendEventReminderResponse, err error, userID, operation string) {
	s.logger.Error("failed to "+operation,
		zap.String("user_id", userID),
		zap.Error(err),
	)

	resp.Success = false
	switch err {
	case service.ErrUserNotBound:
		resp.ErrorCode = "user_not_bound"
		resp.ErrorMessage = "user does not have an active Telegram binding"
	case service.ErrBlocked:
		resp.ErrorCode = "blocked"
		resp.ErrorMessage = "user has blocked the bot"
	case service.ErrRateLimited:
		resp.ErrorCode = "rate_limited"
		resp.ErrorMessage = "rate limited by Telegram"
	default:
		resp.ErrorCode = "send_failed"
		resp.ErrorMessage = "failed to send message"
	}
}

// populateSendMessageError populates error response for SendMessageResponse.
func (s *TelegramServiceServer) populateSendMessageError(resp *SendMessageResponse, err error, userID, operation string) {
	s.logger.Error("failed to "+operation,
		zap.String("user_id", userID),
		zap.Error(err),
	)

	resp.Success = false
	switch err {
	case service.ErrUserNotBound:
		resp.ErrorCode = "user_not_bound"
		resp.ErrorMessage = "user does not have an active Telegram binding"
	case service.ErrBlocked:
		resp.ErrorCode = "blocked"
		resp.ErrorMessage = "user has blocked the bot"
	case service.ErrRateLimited:
		resp.ErrorCode = "rate_limited"
		resp.ErrorMessage = "rate limited by Telegram"
	default:
		resp.ErrorCode = "send_failed"
		resp.ErrorMessage = "failed to send message"
	}
}

func formatMessageID(id int64) string {
	if id == 0 {
		return ""
	}
	return string(rune(id))
}

// UnaryServerInterceptor creates a gRPC interceptor for logging and metrics.
func UnaryServerInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		statusCode := codes.OK
		if err != nil {
			statusCode = status.Code(err)
		}

		metrics.GRPCRequestDuration.WithLabelValues(
			info.FullMethod,
			statusCode.String(),
		).Observe(duration.Seconds())

		if err != nil {
			logger.Error("gRPC request failed",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
				zap.Error(err),
			)
		} else {
			logger.Debug("gRPC request completed",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
			)
		}

		return resp, err
	}
}
