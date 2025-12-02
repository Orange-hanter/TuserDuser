package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"event-api/internal/discovery"
	"event-api/internal/models"

	"go.uber.org/zap"
)

// UserService provides user-related functionality.
type UserService struct {
	db        *sql.DB
	logger    *zap.Logger
	discovery *discovery.Service
}

// NewUserService creates a new UserService.
func NewUserService(db *sql.DB, logger *zap.Logger, discovery *discovery.Service) *UserService {
	return &UserService{
		db:        db,
		logger:    logger,
		discovery: discovery,
	}
}

// GetUserProfile retrieves the full profile for a user, including Telegram binding info.
func (s *UserService) GetUserProfile(ctx context.Context, userID string) (*models.UserProfile, error) {
	// Fetch user details
	var user models.UserProfile
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, created_at, phone
		FROM users 
		WHERE id = $1
	`, userID).Scan(&user.ID, &user.Email, &user.CreatedAt, &user.CellPhone)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	// Fetch telegram info
	var tgInfo models.TelegramInfo
	var tgFirstName, tgLastName sql.NullString
	err = s.db.QueryRowContext(ctx, `
		SELECT telegram_username, chat_id, status, telegram_first_name, telegram_last_name
		FROM telegram_bindings 
		WHERE user_id = $1
	`, userID).Scan(&tgInfo.Username, &tgInfo.ChatID, &tgInfo.Status, &tgFirstName, &tgLastName)

	if err == nil {
		user.TelegramRegistered = true
		user.TelegramInfo = &tgInfo

		// Construct name from Telegram info if available
		if tgFirstName.Valid {
			user.Name = tgFirstName.String
			if tgLastName.Valid {
				user.Name += " " + tgLastName.String
			}
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		s.logger.Warn("failed to fetch telegram info", zap.Error(err))
	}

	if user.Name == "" {
		user.Name = user.Email // Fallback
	}

	return &user, nil
}

// GetUpcomingEvents retrieves events the user is subscribed to that haven't started yet.
func (s *UserService) GetUpcomingEvents(ctx context.Context, userID string) ([]models.EventWithSubscription, error) {
	query := `
		SELECT e.id, e.type, e.start_time, e.end_time, e.place, e.price_type, e.duration, e.need_registration, e.details,
		       es.status, es.subscribed_at
		FROM event_subscriptions es
		JOIN events e ON es.event_id = e.id
		WHERE es.user_id = $1 AND e.start_time > NOW() AND es.status = 'confirmed'
		ORDER BY e.start_time ASC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch upcoming events: %w", err)
	}
	defer rows.Close()

	var events []models.EventWithSubscription
	for rows.Next() {
		var evt models.EventWithSubscription
		var detailsJSON []byte

		err := rows.Scan(
			&evt.ID, &evt.Type, &evt.StartTime, &evt.EndTime, &evt.Place, &evt.PriceType, &evt.Duration, &evt.NeedRegistration, &detailsJSON,
			&evt.SubscriptionStatus, &evt.SubscribedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if len(detailsJSON) > 0 {
			if err := json.Unmarshal(detailsJSON, &evt.Details); err != nil {
				s.logger.Warn("failed to unmarshal event details", zap.Error(err))
			}
		}

		events = append(events, evt)
	}

	if events == nil {
		events = []models.EventWithSubscription{}
	}

	return events, nil
}

// GetEventHistory retrieves past events for the user with pagination.
func (s *UserService) GetEventHistory(ctx context.Context, userID string, limit, offset int) ([]models.EventWithSubscription, error) {
	query := `
		SELECT e.id, e.type, e.start_time, e.end_time, e.place, e.price_type, e.duration, e.need_registration, e.details,
		       es.status, es.subscribed_at
		FROM event_subscriptions es
		JOIN events e ON es.event_id = e.id
		WHERE es.user_id = $1 AND e.start_time < NOW()
		ORDER BY e.start_time DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch event history: %w", err)
	}
	defer rows.Close()

	var events []models.EventWithSubscription
	for rows.Next() {
		var evt models.EventWithSubscription
		var detailsJSON []byte

		err := rows.Scan(
			&evt.ID, &evt.Type, &evt.StartTime, &evt.EndTime, &evt.Place, &evt.PriceType, &evt.Duration, &evt.NeedRegistration, &detailsJSON,
			&evt.SubscriptionStatus, &evt.SubscribedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if len(detailsJSON) > 0 {
			if err := json.Unmarshal(detailsJSON, &evt.Details); err != nil {
				s.logger.Warn("failed to unmarshal event details", zap.Error(err))
			}
		}

		if evt.SubscriptionStatus == models.SubscriptionStatusCancelled {
			evt.AttendanceStatus = "cancelled"
		} else {
			evt.AttendanceStatus = "attended"
		}

		events = append(events, evt)
	}

	if events == nil {
		events = []models.EventWithSubscription{}
	}

	return events, nil
}

// ErrEventFull indicates that the event has reached its capacity.
var ErrEventFull = errors.New("event_full")

// SubscribeToEvent handles user subscription to an event, including capacity checks and conflict resolution.
func (s *UserService) SubscribeToEvent(ctx context.Context, userID, eventID string, metadata map[string]interface{}) (*models.EventSubscription, error) {
	// Transaction for safety
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			s.logger.Error("failed to rollback transaction", zap.Error(err))
		}
	}()

	// Check event details
	var detailsJSON []byte
	var needReg bool
	err = tx.QueryRowContext(ctx, `
		SELECT details, need_registration 
		FROM events 
		WHERE id = $1 AND start_time > NOW()
	`, eventID).Scan(&detailsJSON, &needReg)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("event not found or already started")
		}
		return nil, fmt.Errorf("failed to fetch event: %w", err)
	}

	var details map[string]interface{}
	if len(detailsJSON) > 0 {
		if err := json.Unmarshal(detailsJSON, &details); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event details: %w", err)
		}
	}

	// Check if already subscribed
	var currentStatus string
	var subscribedAt time.Time
	err = tx.QueryRowContext(ctx, `
		SELECT status, subscribed_at FROM event_subscriptions WHERE user_id = $1 AND event_id = $2
	`, userID, eventID).Scan(&currentStatus, &subscribedAt)

	if err == nil {
		// Already subscribed
		return &models.EventSubscription{
			EventID:      eventID,
			Status:       models.SubscriptionStatus(currentStatus),
			SubscribedAt: subscribedAt,
		}, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to check subscription: %w", err)
	}

	// Check capacity
	if val, ok := details["capacity"]; ok {
		capacity := int(val.(float64))

		var currentParticipants int
		err := tx.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM event_subscriptions WHERE event_id = $1 AND status = 'confirmed'
		`, eventID).Scan(&currentParticipants)
		if err != nil {
			return nil, fmt.Errorf("failed to count participants: %w", err)
		}

		if currentParticipants >= capacity {
			// Full - return error for now as we don't have waitlist logic fully defined
			return nil, ErrEventFull
		}
	}

	// Subscribe
	status := models.SubscriptionStatusConfirmed
	metaJSON, _ := json.Marshal(metadata)
	now := time.Now()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO event_subscriptions (user_id, event_id, status, metadata, subscribed_at)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, eventID, status, metaJSON, now)
	if err != nil {
		return nil, fmt.Errorf("failed to insert subscription: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Trigger side effects
	if s.discovery != nil {
		// We do this after commit to ensure consistency, but if it fails, we just log it.
		// Ideally this should be robust (e.g. via outbox pattern or queue), but for now direct call.
		_, err := s.discovery.RegisterBooking(ctx, userID, eventID)
		if err != nil {
			s.logger.Error("failed to register booking in discovery engine", zap.Error(err))
		}
	}

	return &models.EventSubscription{
		EventID:      eventID,
		Status:       status,
		SubscribedAt: now,
	}, nil
}

// GetEventParticipants returns a list of participants for a specific event.
// Returns participants sorted by registration time.
//
// @Summary Get event participants
// @Description Retrieves the list of confirmed participants for an event
// @Tags events
// @Accept json
// @Produce json
// @Param eventID path string true "Event ID"
// @Success 200 {array} models.Participant
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
func (s *UserService) GetEventParticipants(ctx context.Context, eventID string) ([]models.Participant, error) {
	// Consolidated query: read from event_subscriptions (single source of truth)
	// and JOIN telegram_bindings/users for public_name display.
	query := `
		SELECT 
			es.user_id,
			COALESCE(
				NULLIF(TRIM(CONCAT(tb.telegram_first_name, ' ', tb.telegram_last_name)), ''),
				tb.telegram_username,
				u.email,
				'Anonymous'
			) AS public_name,
			NULL::text AS avatar_url,
			es.status
		FROM event_subscriptions es
		LEFT JOIN telegram_bindings tb ON tb.user_id = es.user_id::text
		LEFT JOIN users u ON u.id = es.user_id
		WHERE es.event_id::text = $1::text AND es.status = 'confirmed'
		ORDER BY es.subscribed_at ASC
	`

	s.logger.Debug("executing participants query", zap.String("query", query), zap.String("event_id", eventID))
	rows, err := s.db.QueryContext(ctx, query, eventID)
	if err != nil {
		s.logger.Error("failed to query participants", zap.Error(err), zap.String("event_id", eventID))
		return nil, fmt.Errorf("failed to fetch participants: %w", err)
	}
	defer rows.Close()

	var participants []models.Participant
	for rows.Next() {
		var p models.Participant
		if err := rows.Scan(&p.UserID, &p.PublicName, &p.AvatarURL, &p.Status); err != nil {
			s.logger.Error("failed to scan participant", zap.Error(err))
			continue
		}
		participants = append(participants, p)
	}

	if err := rows.Err(); err != nil {
		s.logger.Error("error iterating participant rows", zap.Error(err), zap.String("event_id", eventID))
		return nil, fmt.Errorf("error iterating participants: %w", err)
	}

	return participants, nil
}

// RequestRole handles a user's request to upgrade their role.
func (s *UserService) RequestRole(ctx context.Context, userID, role, reason string) (*models.RoleRequestResponse, error) {
	// Check if user already has the requested role
	var currentRole string
	err := s.db.QueryRowContext(ctx, "SELECT role FROM users WHERE id = $1", userID).Scan(&currentRole)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	// Check if user already has the requested role
	if currentRole == models.RoleCreator && role == models.RoleCreator {
		return &models.RoleRequestResponse{
			Message: "You already have the creator role",
			Status:  "already_granted",
		}, nil
	}

	// Save the role request to database
	query := `
		INSERT INTO role_requests (user_id, requested_role, reason, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'pending', NOW(), NOW())
		ON CONFLICT (user_id, requested_role) DO UPDATE 
		SET reason = $3, status = 'pending', updated_at = NOW()
	`

	_, err = s.db.ExecContext(ctx, query, userID, role, reason)
	if err != nil {
		return nil, fmt.Errorf("failed to save role request: %w", err)
	}

	s.logger.Info("Role request created", zap.String("user_id", userID), zap.String("role", role))

	return &models.RoleRequestResponse{
		Message: "Role request submitted successfully. Admins will review your request.",
		Status:  "pending",
	}, nil
}

// GetRoleRequests retrieves all role requests for a user.
func (s *UserService) GetRoleRequests(ctx context.Context, userID string) ([]models.RoleRequestStatus, error) {
	query := `
		SELECT id, requested_role, status, reason, review_notes, created_at, updated_at, reviewed_at, reviewed_by
		FROM role_requests
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch role requests: %w", err)
	}
	defer rows.Close()

	var requests []models.RoleRequestStatus
	for rows.Next() {
		var req models.RoleRequestStatus
		if err := rows.Scan(
			&req.ID,
			&req.RequestedRole,
			&req.CurrentStatus,
			&req.Reason,
			&req.ReviewNotes,
			&req.CreatedAt,
			&req.UpdatedAt,
			&req.ReviewedAt,
			&req.ReviewedBy,
		); err != nil {
			s.logger.Error("failed to scan role request", zap.Error(err))
			continue
		}
		requests = append(requests, req)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating role requests: %w", err)
	}

	return requests, nil
}

// GetRoleRequestStatus retrieves a specific role request status.
func (s *UserService) GetRoleRequestStatus(ctx context.Context, userID, requestedRole string) (*models.RoleRequestStatus, error) {
	query := `
		SELECT id, requested_role, status, reason, review_notes, created_at, updated_at, reviewed_at, reviewed_by
		FROM role_requests
		WHERE user_id = $1 AND requested_role = $2
		ORDER BY created_at DESC
		LIMIT 1
	`

	var req models.RoleRequestStatus
	err := s.db.QueryRowContext(ctx, query, userID, requestedRole).Scan(
		&req.ID,
		&req.RequestedRole,
		&req.CurrentStatus,
		&req.Reason,
		&req.ReviewNotes,
		&req.CreatedAt,
		&req.UpdatedAt,
		&req.ReviewedAt,
		&req.ReviewedBy,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("role request not found")
		}
		return nil, fmt.Errorf("failed to fetch role request: %w", err)
	}

	return &req, nil
}

// GetPendingRoleRequests retrieves all pending role requests for admin review.
func (s *UserService) GetPendingRoleRequests(ctx context.Context, limit, offset int) ([]models.RoleRequestStatus, int, error) {
	// Get total count
	var total int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM role_requests WHERE status = 'pending'").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count role requests: %w", err)
	}

	// Get paginated results
	query := `
		SELECT id, requested_role, status, reason, review_notes, created_at, updated_at, reviewed_at, reviewed_by
		FROM role_requests
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT $1 OFFSET $2
	`

	rows, err := s.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch pending role requests: %w", err)
	}
	defer rows.Close()

	var requests []models.RoleRequestStatus
	for rows.Next() {
		var req models.RoleRequestStatus
		if err := rows.Scan(
			&req.ID,
			&req.RequestedRole,
			&req.CurrentStatus,
			&req.Reason,
			&req.ReviewNotes,
			&req.CreatedAt,
			&req.UpdatedAt,
			&req.ReviewedAt,
			&req.ReviewedBy,
		); err != nil {
			s.logger.Error("failed to scan role request", zap.Error(err))
			continue
		}
		requests = append(requests, req)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating role requests: %w", err)
	}

	return requests, total, nil
}

// ApproveRoleRequest approves a pending role request and updates user role.
func (s *UserService) ApproveRoleRequest(ctx context.Context, requestID, adminID, notes string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get the role request
	var userID, requestedRole string
	err = tx.QueryRowContext(ctx, `
		SELECT user_id, requested_role FROM role_requests WHERE id = $1 AND status = 'pending'
	`, requestID).Scan(&userID, &requestedRole)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("role request not found")
		}
		return fmt.Errorf("failed to fetch role request: %w", err)
	}

	// Update role request status
	_, err = tx.ExecContext(ctx, `
		UPDATE role_requests
		SET status = 'approved', reviewed_by = $1, reviewed_at = NOW(), review_notes = $2, updated_at = NOW()
		WHERE id = $3
	`, adminID, notes, requestID)
	if err != nil {
		return fmt.Errorf("failed to update role request: %w", err)
	}

	// Update user role
	_, err = tx.ExecContext(ctx, `
		UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2
	`, requestedRole, userID)
	if err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}

	s.logger.Info("Role request approved",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("role", requestedRole),
		zap.String("admin_id", adminID),
	)

	return tx.Commit()
}

// RejectRoleRequest rejects a pending role request.
func (s *UserService) RejectRoleRequest(ctx context.Context, requestID, adminID, reason string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE role_requests
		SET status = 'rejected', reviewed_by = $1, reviewed_at = NOW(), review_notes = $2, updated_at = NOW()
		WHERE id = $3 AND status = 'pending'
	`, adminID, reason, requestID)
	if err != nil {
		return fmt.Errorf("failed to update role request: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil || rows == 0 {
		return errors.New("role request not found")
	}

	s.logger.Info("Role request rejected",
		zap.String("request_id", requestID),
		zap.String("admin_id", adminID),
	)

	return nil
}
