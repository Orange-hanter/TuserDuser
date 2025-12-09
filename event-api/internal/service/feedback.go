package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"event-api/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// FeedbackService manages feedback operations.
type FeedbackService struct {
	db            *sql.DB
	logger        *zap.Logger
	adminNotifier *AdminNotifier
}

// NewFeedbackService creates a new FeedbackService.
func NewFeedbackService(db *sql.DB, logger *zap.Logger) *FeedbackService {
	return &FeedbackService{
		db:     db,
		logger: logger,
	}
}

// SetAdminNotifier sets the admin notifier for sending notifications about new feedback.
func (s *FeedbackService) SetAdminNotifier(notifier *AdminNotifier) {
	s.adminNotifier = notifier
}

// CreateFeedback creates a new feedback entry.
func (s *FeedbackService) CreateFeedback(ctx context.Context, req *models.CreateFeedbackRequest, authenticatedUserID string) (*models.Feedback, error) {
	id := uuid.New().String()

	// Serialize user info and environment to JSON
	userInfoJSON, err := json.Marshal(req.UserInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize user info: %w", err)
	}

	environmentJSON, err := json.Marshal(req.Environment)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize environment: %w", err)
	}

	// Determine user_id: prefer authenticated user, fallback to userInfo.userId
	var userID *string
	if authenticatedUserID != "" {
		userID = &authenticatedUserID
	} else if req.UserInfo.UserID != "" && req.UserInfo.UserID != "temp" {
		userID = &req.UserInfo.UserID
	}

	query := `
		INSERT INTO feedback (id, category, message, user_id, user_info, environment)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, category, message, user_id, user_info, environment, is_read, created_at, updated_at
	`

	var feedback models.Feedback
	err = s.db.QueryRowContext(ctx, query,
		id,
		req.Category,
		req.Message,
		userID,
		userInfoJSON,
		environmentJSON,
	).Scan(
		&feedback.ID,
		&feedback.Category,
		&feedback.Message,
		&feedback.UserID,
		&feedback.UserInfoJSON,
		&feedback.EnvironmentJSON,
		&feedback.IsRead,
		&feedback.CreatedAt,
		&feedback.UpdatedAt,
	)

	if err != nil {
		s.logger.Error("Failed to create feedback", zap.Error(err))
		return nil, fmt.Errorf("failed to create feedback: %w", err)
	}

	// Deserialize JSON fields
	if len(feedback.UserInfoJSON) > 0 {
		if err := json.Unmarshal(feedback.UserInfoJSON, &feedback.UserInfo); err != nil {
			s.logger.Warn("Failed to unmarshal user info", zap.Error(err))
		}
	}
	if len(feedback.EnvironmentJSON) > 0 {
		if err := json.Unmarshal(feedback.EnvironmentJSON, &feedback.Environment); err != nil {
			s.logger.Warn("Failed to unmarshal environment", zap.Error(err))
		}
	}

	s.logger.Info("Feedback created",
		zap.String("id", feedback.ID),
		zap.String("category", string(feedback.Category)),
	)

	// Notify admins asynchronously with context decoupled from HTTP request
	if s.adminNotifier != nil {
		go func(fb models.Feedback) {
			notifyCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			s.adminNotifier.NotifyAdminsFeedback(notifyCtx, &fb)
		}(feedback)
	}

	return &feedback, nil
}

// GetFeedbackList returns a paginated list of feedback, sorted by newest first.
func (s *FeedbackService) GetFeedbackList(ctx context.Context, page, pageSize int, unreadOnly bool) (*models.FeedbackListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// Count total
	var countQuery string
	var totalCount int
	if unreadOnly {
		countQuery = `SELECT COUNT(*) FROM feedback WHERE is_read = false`
	} else {
		countQuery = `SELECT COUNT(*) FROM feedback`
	}
	if err := s.db.QueryRowContext(ctx, countQuery).Scan(&totalCount); err != nil {
		s.logger.Error("Failed to count feedback", zap.Error(err))
		return nil, fmt.Errorf("failed to count feedback: %w", err)
	}

	// Query feedback
	var query string
	if unreadOnly {
		query = `
			SELECT id, category, message, user_id, user_info, environment, is_read, created_at, updated_at
			FROM feedback
			WHERE is_read = false
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2
		`
	} else {
		query = `
			SELECT id, category, message, user_id, user_info, environment, is_read, created_at, updated_at
			FROM feedback
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2
		`
	}

	rows, err := s.db.QueryContext(ctx, query, pageSize, offset)
	if err != nil {
		s.logger.Error("Failed to query feedback", zap.Error(err))
		return nil, fmt.Errorf("failed to query feedback: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Error("Failed to close rows", zap.Error(err))
		}
	}()

	var feedbacks []*models.Feedback
	for rows.Next() {
		var fb models.Feedback
		err := rows.Scan(
			&fb.ID,
			&fb.Category,
			&fb.Message,
			&fb.UserID,
			&fb.UserInfoJSON,
			&fb.EnvironmentJSON,
			&fb.IsRead,
			&fb.CreatedAt,
			&fb.UpdatedAt,
		)
		if err != nil {
			s.logger.Error("Failed to scan feedback", zap.Error(err))
			continue
		}

		// Deserialize JSON fields
		if len(fb.UserInfoJSON) > 0 {
			if err := json.Unmarshal(fb.UserInfoJSON, &fb.UserInfo); err != nil {
				s.logger.Warn("Failed to unmarshal user info", zap.Error(err))
			}
		}
		if len(fb.EnvironmentJSON) > 0 {
			if err := json.Unmarshal(fb.EnvironmentJSON, &fb.Environment); err != nil {
				s.logger.Warn("Failed to unmarshal environment", zap.Error(err))
			}
		}

		feedbacks = append(feedbacks, &fb)
	}

	if err := rows.Err(); err != nil {
		s.logger.Error("Error iterating feedback", zap.Error(err))
		return nil, fmt.Errorf("error iterating feedback: %w", err)
	}

	return &models.FeedbackListResponse{
		Feedbacks:  feedbacks,
		Total:      totalCount,
		Page:       page,
		PageSize:   pageSize,
		UnreadOnly: unreadOnly,
	}, nil
}

// GetFeedbackByID returns a single feedback by ID.
func (s *FeedbackService) GetFeedbackByID(ctx context.Context, id string) (*models.Feedback, error) {
	query := `
		SELECT id, category, message, user_id, user_info, environment, is_read, created_at, updated_at
		FROM feedback
		WHERE id = $1
	`

	var fb models.Feedback
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&fb.ID,
		&fb.Category,
		&fb.Message,
		&fb.UserID,
		&fb.UserInfoJSON,
		&fb.EnvironmentJSON,
		&fb.IsRead,
		&fb.CreatedAt,
		&fb.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("feedback not found")
	}
	if err != nil {
		s.logger.Error("Failed to get feedback", zap.String("id", id), zap.Error(err))
		return nil, fmt.Errorf("failed to get feedback: %w", err)
	}

	// Deserialize JSON fields
	if len(fb.UserInfoJSON) > 0 {
		if err := json.Unmarshal(fb.UserInfoJSON, &fb.UserInfo); err != nil {
			s.logger.Warn("Failed to unmarshal user info", zap.Error(err))
		}
	}
	if len(fb.EnvironmentJSON) > 0 {
		if err := json.Unmarshal(fb.EnvironmentJSON, &fb.Environment); err != nil {
			s.logger.Warn("Failed to unmarshal environment", zap.Error(err))
		}
	}

	return &fb, nil
}

// MarkFeedbackRead marks a feedback as read or unread.
func (s *FeedbackService) MarkFeedbackRead(ctx context.Context, id string, isRead bool) error {
	query := `
		UPDATE feedback
		SET is_read = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`

	var updatedID string
	err := s.db.QueryRowContext(ctx, query, id, isRead).Scan(&updatedID)

	if err == sql.ErrNoRows {
		return fmt.Errorf("feedback not found")
	}
	if err != nil {
		s.logger.Error("Failed to mark feedback", zap.String("id", id), zap.Error(err))
		return fmt.Errorf("failed to mark feedback: %w", err)
	}

	s.logger.Info("Feedback marked",
		zap.String("id", id),
		zap.Bool("is_read", isRead),
	)

	return nil
}

// DeleteFeedback deletes a feedback by ID.
func (s *FeedbackService) DeleteFeedback(ctx context.Context, id string) error {
	query := `DELETE FROM feedback WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		s.logger.Error("Failed to delete feedback", zap.String("id", id), zap.Error(err))
		return fmt.Errorf("failed to delete feedback: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check deletion: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("feedback not found")
	}

	s.logger.Info("Feedback deleted", zap.String("id", id))
	return nil
}

// GetUnreadCount returns the count of unread feedback.
func (s *FeedbackService) GetUnreadCount(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM feedback WHERE is_read = false`

	var count int
	if err := s.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		s.logger.Error("Failed to count unread feedback", zap.Error(err))
		return 0, fmt.Errorf("failed to count unread feedback: %w", err)
	}

	return count, nil
}
