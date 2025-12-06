package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"event-api/internal/telegramclient"

	"go.uber.org/zap"
)

// Reminder errors
var (
	ErrSchedulerAlreadyRunning     = errors.New("scheduler already running")
	ErrTelegramClientNotConfigured = errors.New("telegram client not configured")
)

// ReminderConfig holds configuration for the event reminder system.
type ReminderConfig struct {
	// CheckInterval - how often to check for upcoming events (default: 1 minute)
	CheckInterval time.Duration
	// ReminderOffsets - time offsets before event start to send reminders
	// e.g., [24*time.Hour, 1*time.Hour, 15*time.Minute] for 24h, 1h, 15min reminders
	ReminderOffsets []time.Duration
	// SendRateLimit - maximum messages per second (Telegram limit is ~30/sec)
	SendRateLimit int
	// QueueSize - size of the notification queue
	QueueSize int
	// Workers - number of concurrent workers for sending notifications
	Workers int
}

// DefaultReminderConfig returns default configuration.
func DefaultReminderConfig() ReminderConfig {
	return ReminderConfig{
		CheckInterval: 1 * time.Minute,
		ReminderOffsets: []time.Duration{
			24 * time.Hour,   // 24 hours before
			1 * time.Hour,    // 1 hour before
			15 * time.Minute, // 15 minutes before
		},
		SendRateLimit: 25, // slightly below Telegram's limit
		QueueSize:     1000,
		Workers:       3,
	}
}

// EventReminderNotification represents a single reminder notification to send.
type EventReminderNotification struct {
	UserID       string
	EventID      string
	EventTitle   string
	EventPlace   string
	EventStart   time.Time
	ReminderType string // "24h", "1h", "15min"
}

// EventReminderScheduler manages scheduling and sending event reminders.
type EventReminderScheduler struct {
	db             *sql.DB
	telegramClient *telegramclient.Client
	logger         *zap.Logger
	config         ReminderConfig

	// Internal state
	queue       chan EventReminderNotification
	rateLimiter *time.Ticker
	stopCh      chan struct{}
	wg          sync.WaitGroup
	running     bool
	mu          sync.Mutex
}

// NewEventReminderScheduler creates a new event reminder scheduler.
func NewEventReminderScheduler(
	db *sql.DB,
	telegramClient *telegramclient.Client,
	logger *zap.Logger,
	config ReminderConfig,
) *EventReminderScheduler {
	if config.CheckInterval == 0 {
		config = DefaultReminderConfig()
	}
	if config.SendRateLimit <= 0 {
		config.SendRateLimit = 25
	}
	if config.QueueSize <= 0 {
		config.QueueSize = 1000
	}
	if config.Workers <= 0 {
		config.Workers = 3
	}

	return &EventReminderScheduler{
		db:             db,
		telegramClient: telegramClient,
		logger:         logger,
		config:         config,
		queue:          make(chan EventReminderNotification, config.QueueSize),
		stopCh:         make(chan struct{}),
	}
}

// Start begins the reminder scheduler.
func (s *EventReminderScheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return ErrSchedulerAlreadyRunning
	}

	if s.telegramClient == nil {
		s.logger.Warn("telegram client not configured, event reminders disabled")
		return nil
	}

	s.running = true
	s.rateLimiter = time.NewTicker(time.Second / time.Duration(s.config.SendRateLimit))

	// Start workers
	for i := 0; i < s.config.Workers; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}

	// Start scheduler
	s.wg.Add(1)
	go s.scheduleLoop()

	s.logger.Info("event reminder scheduler started",
		zap.Int("workers", s.config.Workers),
		zap.Int("rate_limit", s.config.SendRateLimit),
		zap.Duration("check_interval", s.config.CheckInterval),
	)

	return nil
}

// Stop gracefully stops the reminder scheduler.
func (s *EventReminderScheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	close(s.stopCh)
	if s.rateLimiter != nil {
		s.rateLimiter.Stop()
	}
	s.wg.Wait()
	s.logger.Info("event reminder scheduler stopped")
}

// scheduleLoop periodically checks for upcoming events and queues reminders.
func (s *EventReminderScheduler) scheduleLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.CheckInterval)
	defer ticker.Stop()

	// Initial check
	s.checkAndQueueReminders()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkAndQueueReminders()
		}
	}
}

// checkAndQueueReminders finds upcoming events and queues notifications.
func (s *EventReminderScheduler) checkAndQueueReminders() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, offset := range s.config.ReminderOffsets {
		reminderType := formatReminderType(offset)

		notifications, err := s.getUpcomingEventSubscribers(ctx, offset)
		if err != nil {
			s.logger.Error("failed to get upcoming event subscribers",
				zap.Duration("offset", offset),
				zap.Error(err),
			)
			continue
		}

		for _, notif := range notifications {
			notif.ReminderType = reminderType
			select {
			case s.queue <- notif:
				// queued successfully
			default:
				s.logger.Warn("notification queue full, dropping notification",
					zap.String("user_id", notif.UserID),
					zap.String("event_id", notif.EventID),
				)
			}
		}

		if len(notifications) > 0 {
			s.logger.Info("queued event reminders",
				zap.String("reminder_type", reminderType),
				zap.Int("count", len(notifications)),
			)
		}
	}
}

// getUpcomingEventSubscribers returns users subscribed to events starting within the given offset.
// Only returns users with active Telegram bindings who haven't received this reminder yet.
func (s *EventReminderScheduler) getUpcomingEventSubscribers(ctx context.Context, offset time.Duration) ([]EventReminderNotification, error) {
	// Calculate time window: events starting in [now + offset - check_interval, now + offset + check_interval]
	// This ensures we catch events in the current check window
	now := time.Now()
	windowStart := now.Add(offset).Add(-s.config.CheckInterval)
	windowEnd := now.Add(offset).Add(s.config.CheckInterval)

	// Query for subscribed users with telegram bindings
	// Exclude users who already received this reminder type
	query := `
		SELECT DISTINCT
			es.user_id,
			e.id as event_id,
			e.type as event_title,
			COALESCE(e.place, '') as event_place,
			e.start_time
		FROM event_subscriptions es
		INNER JOIN events e ON e.id = es.event_id
		INNER JOIN telegram_bindings tb ON tb.user_id = es.user_id AND tb.status = 'active'
		LEFT JOIN event_reminder_log erl ON erl.user_id = es.user_id 
			AND erl.event_id = e.id 
			AND erl.reminder_type = $3
		WHERE es.status = 'confirmed'
			AND e.start_time >= $1
			AND e.start_time <= $2
			AND erl.id IS NULL
		ORDER BY e.start_time ASC
	`

	reminderType := formatReminderType(offset)
	rows, err := s.db.QueryContext(ctx, query, windowStart, windowEnd, reminderType)
	if err != nil {
		return nil, fmt.Errorf("failed to query upcoming events: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Error("failed to close rows", zap.Error(err))
		}
	}()

	var notifications []EventReminderNotification
	for rows.Next() {
		var notif EventReminderNotification
		if err := rows.Scan(
			&notif.UserID,
			&notif.EventID,
			&notif.EventTitle,
			&notif.EventPlace,
			&notif.EventStart,
		); err != nil {
			s.logger.Error("failed to scan notification row", zap.Error(err))
			continue
		}
		notifications = append(notifications, notif)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return notifications, nil
}

// worker processes notifications from the queue with rate limiting.
func (s *EventReminderScheduler) worker(id int) {
	defer s.wg.Done()

	for {
		select {
		case <-s.stopCh:
			return
		case notif, ok := <-s.queue:
			if !ok {
				return
			}
			// Wait for rate limiter
			select {
			case <-s.stopCh:
				return
			case <-s.rateLimiter.C:
				// proceed with sending
			}
			s.sendReminder(notif)
		}
	}
}

// sendReminder sends a single reminder notification.
func (s *EventReminderScheduler) sendReminder(notif EventReminderNotification) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Format message based on reminder type
	var timeLabel string
	switch notif.ReminderType {
	case "24h":
		timeLabel = "через 24 часа"
	case "1h":
		timeLabel = "через 1 час"
	case "15min":
		timeLabel = "через 15 минут"
	default:
		timeLabel = "скоро"
	}

	_, err := s.telegramClient.SendEventReminder(
		ctx,
		notif.UserID,
		notif.EventID,
		notif.EventTitle,
		fmt.Sprintf("Начало %s", timeLabel),
		notif.EventStart,
		notif.EventPlace,
		"", // deeplink URL - can be added later
	)

	if err != nil {
		s.logger.Warn("failed to send event reminder",
			zap.String("user_id", notif.UserID),
			zap.String("event_id", notif.EventID),
			zap.String("reminder_type", notif.ReminderType),
			zap.Error(err),
		)
		return
	}

	// Log successful reminder
	s.logReminderSent(ctx, notif)

	s.logger.Debug("event reminder sent",
		zap.String("user_id", notif.UserID),
		zap.String("event_id", notif.EventID),
		zap.String("reminder_type", notif.ReminderType),
	)
}

// logReminderSent records that a reminder was sent to prevent duplicates.
func (s *EventReminderScheduler) logReminderSent(ctx context.Context, notif EventReminderNotification) {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO event_reminder_log (user_id, event_id, reminder_type, sent_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id, event_id, reminder_type) DO NOTHING
	`, notif.UserID, notif.EventID, notif.ReminderType)

	if err != nil {
		s.logger.Error("failed to log reminder sent",
			zap.String("user_id", notif.UserID),
			zap.String("event_id", notif.EventID),
			zap.Error(err),
		)
	}
}

// formatReminderType converts duration offset to a string label.
func formatReminderType(offset time.Duration) string {
	switch {
	case offset >= 24*time.Hour:
		return "24h"
	case offset >= 1*time.Hour:
		return "1h"
	case offset >= 15*time.Minute:
		return "15min"
	default:
		return "immediate"
	}
}

// QueueSize returns the current number of notifications in the queue.
func (s *EventReminderScheduler) QueueSize() int {
	return len(s.queue)
}

// IsRunning returns whether the scheduler is running.
func (s *EventReminderScheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// SendImmediateReminder sends an immediate reminder to all subscribers of an event.
// This is useful for manual/emergency notifications.
// Returns the number of notifications queued.
func (s *EventReminderScheduler) SendImmediateReminder(ctx context.Context, eventID string) (int, error) {
	if s.telegramClient == nil {
		return 0, ErrTelegramClientNotConfigured
	}

	// Get all subscribers with telegram bindings for this event
	query := `
		SELECT DISTINCT
			es.user_id,
			e.id as event_id,
			e.type as event_title,
			COALESCE(e.place, '') as event_place,
			e.start_time
		FROM event_subscriptions es
		INNER JOIN events e ON e.id = es.event_id
		INNER JOIN telegram_bindings tb ON tb.user_id = es.user_id AND tb.status = 'active'
		WHERE es.status = 'confirmed'
			AND e.id = $1
		ORDER BY es.subscribed_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, eventID)
	if err != nil {
		return 0, fmt.Errorf("failed to query event subscribers: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Error("failed to close rows", zap.Error(err))
		}
	}()

	var queued int
	for rows.Next() {
		var notif EventReminderNotification
		if err := rows.Scan(
			&notif.UserID,
			&notif.EventID,
			&notif.EventTitle,
			&notif.EventPlace,
			&notif.EventStart,
		); err != nil {
			s.logger.Error("failed to scan notification row", zap.Error(err))
			continue
		}
		notif.ReminderType = "immediate"

		select {
		case s.queue <- notif:
			queued++
		default:
			s.logger.Warn("notification queue full, stopping immediate send",
				zap.String("event_id", eventID),
				zap.Int("queued", queued),
			)
			return queued, nil
		}
	}

	if err := rows.Err(); err != nil {
		return queued, fmt.Errorf("error iterating rows: %w", err)
	}

	s.logger.Info("queued immediate reminders",
		zap.String("event_id", eventID),
		zap.Int("count", queued),
	)

	return queued, nil
}

// GetStats returns statistics about the reminder scheduler.
func (s *EventReminderScheduler) GetStats() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	return map[string]interface{}{
		"running":        s.running,
		"queue_size":     len(s.queue),
		"queue_capacity": cap(s.queue),
		"workers":        s.config.Workers,
		"rate_limit":     s.config.SendRateLimit,
	}
}
