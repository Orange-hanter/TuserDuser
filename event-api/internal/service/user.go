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
		SELECT id, email, created_at 
		FROM users 
		WHERE id = $1
	`, userID).Scan(&user.ID, &user.Email, &user.CreatedAt)
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
