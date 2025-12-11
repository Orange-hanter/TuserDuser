package service

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"event-api/internal/discovery"
	"event-api/internal/models"
	"event-api/internal/redis"

	"go.uber.org/zap"
)

// UserService provides user-related functionality.
type UserService struct {
	db            *sql.DB
	logger        *zap.Logger
	discovery     *discovery.Service
	redis         *redis.Client
	adminNotifier *AdminNotifier
}

// NewUserService creates a new UserService.
func NewUserService(db *sql.DB, logger *zap.Logger, discovery *discovery.Service) *UserService {
	return &UserService{
		db:        db,
		logger:    logger,
		discovery: discovery,
	}
}

// NewUserServiceWithRedis creates a new UserService with Redis caching support.
func NewUserServiceWithRedis(db *sql.DB, logger *zap.Logger, disc *discovery.Service, redisClient *redis.Client) *UserService {
	return &UserService{
		db:        db,
		logger:    logger,
		discovery: disc,
		redis:     redisClient,
	}
}

// SetRedisClient sets the Redis client for caching.
func (s *UserService) SetRedisClient(redisClient *redis.Client) {
	s.redis = redisClient
}

// SetAdminNotifier sets the admin notifier for sending notifications about role requests.
func (s *UserService) SetAdminNotifier(notifier *AdminNotifier) {
	s.adminNotifier = notifier
}

// Constants for public profile caching.
const (
	publicProfileCacheTTL    = 5 * time.Minute
	publicProfileCachePrefix = "public_profile:"
)

// GetPublicProfile retrieves the public profile for a user by ID.
// Returns only publicly visible information. Supports Redis caching.
// Returns PublicProfileNotFoundError if user not found or profile is private.
func (s *UserService) GetPublicProfile(ctx context.Context, userID string) (*models.PublicUserProfile, string, error) {
	cacheKey := publicProfileCachePrefix + userID

	// Try to get from cache first
	if s.redis != nil {
		cached, err := s.redis.Get(ctx, cacheKey)
		if err == nil && cached != "" {
			var profile models.PublicUserProfile
			if err := json.Unmarshal([]byte(cached), &profile); err == nil {
				// Generate ETag from cached data
				etag := generateETag([]byte(cached))
				s.logger.Debug("cache hit for public profile", zap.String("user_id", userID))
				return &profile, etag, nil
			}
		}
	}

	// Query database for public profile data
	var profile models.PublicUserProfile
	var username, displayName, avatarURL, bio, city, country sql.NullString
	var publicSocialJSON []byte
	var isProfilePublic bool

	query := `
		SELECT
			u.id,
			u.username,
			u.display_name,
			u.avatar_url,
			u.bio,
			u.city,
			u.country,
			COALESCE(u.is_verified, false) as is_verified,
			COALESCE(u.public_social, '{}'::jsonb) as public_social,
			COALESCE(u.is_profile_public, true) as is_profile_public,
			u.created_at,
			u.updated_at,
			(SELECT COUNT(*) FROM events WHERE creator_id = u.id) as public_events_count
		FROM users u
		WHERE u.id = $1
	`

	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&profile.ID,
		&username,
		&displayName,
		&avatarURL,
		&bio,
		&city,
		&country,
		&profile.IsVerified,
		&publicSocialJSON,
		&isProfilePublic,
		&profile.CreatedAt,
		&profile.UpdatedAt,
		&profile.PublicEventsCount,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", &models.PublicProfileNotFoundError{UserID: userID}
		}
		return nil, "", fmt.Errorf("failed to fetch public profile: %w", err)
	}

	// Check if profile is public - return 404 to not reveal private account existence
	if !isProfilePublic {
		return nil, "", &models.PublicProfileNotFoundError{UserID: userID}
	}

	// Map nullable fields
	if username.Valid {
		profile.Username = &username.String
	}
	if displayName.Valid && displayName.String != "" {
		profile.DisplayName = displayName.String
	} else {
		// Fallback to "User" if no display name set
		profile.DisplayName = "User"
	}
	if avatarURL.Valid {
		profile.AvatarURL = &avatarURL.String
	}
	if bio.Valid {
		profile.Bio = &bio.String
	}
	if city.Valid {
		profile.City = &city.String
	}
	if country.Valid {
		profile.Country = &country.String
	}

	// Parse social links
	profile.SocialLinks = models.ParseSocialLinks(publicSocialJSON)

	// Cache the profile
	if s.redis != nil {
		profileJSON, err := json.Marshal(profile)
		if err == nil {
			if err := s.redis.Set(ctx, cacheKey, string(profileJSON), publicProfileCacheTTL); err != nil {
				s.logger.Warn("failed to cache public profile", zap.Error(err), zap.String("user_id", userID))
			}
		}
	}

	// Generate ETag
	profileJSON, _ := json.Marshal(profile)
	etag := generateETag(profileJSON)

	return &profile, etag, nil
}

// InvalidatePublicProfileCache removes the cached public profile for a user.
func (s *UserService) InvalidatePublicProfileCache(ctx context.Context, userID string) error {
	if s.redis == nil {
		return nil
	}
	cacheKey := publicProfileCachePrefix + userID
	return s.redis.Del(ctx, cacheKey)
}

// generateETag creates an ETag from content using MD5 hash.
func generateETag(content []byte) string {
	hash := md5.Sum(content)
	return `"` + hex.EncodeToString(hash[:]) + `"`
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

// ErrSubscriptionNotFound indicates that no subscription was found for the user and event.
var ErrSubscriptionNotFound = errors.New("subscription_not_found")

// ErrEventAlreadyStarted indicates that the event has already started and cannot be unsubscribed from.
var ErrEventAlreadyStarted = errors.New("event_already_started")

// UnsubscribeFromEvent handles user unsubscription from an event.
// It cancels the subscription by updating the status to 'cancelled'.
func (s *UserService) UnsubscribeFromEvent(ctx context.Context, userID, eventID string) error {
	// Transaction for safety
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			s.logger.Error("failed to rollback transaction", zap.Error(err))
		}
	}()

	// Check if event exists and hasn't started yet
	var startTime time.Time
	err = tx.QueryRowContext(ctx, `
		SELECT start_time 
		FROM events 
		WHERE id = $1
	`, eventID).Scan(&startTime)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("event not found")
		}
		return fmt.Errorf("failed to fetch event: %w", err)
	}

	// Check if event has already started
	if time.Now().After(startTime) {
		return ErrEventAlreadyStarted
	}

	// Check if subscription exists and is not already cancelled
	var currentStatus string
	err = tx.QueryRowContext(ctx, `
		SELECT status FROM event_subscriptions WHERE user_id = $1 AND event_id = $2
	`, userID, eventID).Scan(&currentStatus)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSubscriptionNotFound
		}
		return fmt.Errorf("failed to check subscription: %w", err)
	}

	if currentStatus == string(models.SubscriptionStatusCancelled) {
		// Already cancelled, return success (idempotent)
		return nil
	}

	// Cancel the subscription
	_, err = tx.ExecContext(ctx, `
		UPDATE event_subscriptions 
		SET status = $1, updated_at = $2
		WHERE user_id = $3 AND event_id = $4
	`, models.SubscriptionStatusCancelled, time.Now(), userID, eventID)
	if err != nil {
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// Remove booking from discovery to allow event to reappear in queue
	if s.discovery != nil {
		if err := s.discovery.CancelBooking(ctx, userID, eventID); err != nil {
			// Log error but don't fail - subscription is already cancelled
			s.logger.Error("failed to cancel booking in discovery engine",
				zap.Error(err),
				zap.String("user_id", userID),
				zap.String("event_id", eventID),
			)
		}
	}

	s.logger.Info("user unsubscribed from event",
		zap.String("user_id", userID),
		zap.String("event_id", eventID),
	)

	return nil
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
			es.user_id::uuid,
			COALESCE(
				NULLIF(TRIM(CONCAT(tb.telegram_first_name, ' ', tb.telegram_last_name)), ''),
				tb.telegram_username,
				u.email,
				'Anonymous'
			) AS public_name,
			NULL::text AS avatar_url,
			es.status
		FROM event_subscriptions es
		LEFT JOIN telegram_bindings tb ON tb.user_id::uuid = es.user_id
		LEFT JOIN users u ON u.id = es.user_id
		WHERE es.event_id = $1 AND es.status = 'confirmed'
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
	var currentRole, userEmail string
	err := s.db.QueryRowContext(ctx, "SELECT role, email FROM users WHERE id = $1", userID).Scan(&currentRole, &userEmail)
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

	// Notify admins about the new role request (async, decoupled from HTTP request context)
	if s.adminNotifier != nil {
		go func(uid, email, r, rsn string) {
			notifyCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			s.adminNotifier.NotifyAdminsRoleRequest(notifyCtx, uid, email, r, rsn)
		}(userID, userEmail, role, reason)
	}

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
