package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"event-api/internal/models"

	"go.uber.org/zap"
)

// EventService управляет событиями.
type EventService struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewEventService создает новый сервис событий.
func NewEventService(db *sql.DB, logger *zap.Logger) *EventService {
	return &EventService{
		db:     db,
		logger: logger,
	}
}

// GetApprovedEvents получает все одобренные события.
func (s *EventService) GetApprovedEvents(ctx context.Context) ([]*models.Event, error) {
	query := `
		SELECT id, type, start_time, end_time, duration, place, 
		       price_type, need_registration, details, created_at, updated_at, creator_id
		FROM events
		ORDER BY start_time DESC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		s.logger.Error("Failed to query events", zap.Error(err))
		return nil, fmt.Errorf("ошибка при получении событий: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Error("Failed to close rows after querying events", zap.Error(err))
		}
	}()

	var events []*models.Event
	for rows.Next() {
		var event models.Event
		var detailsJSON []byte

		err := rows.Scan(
			&event.ID,
			&event.Type,
			&event.StartTime,
			&event.EndTime,
			&event.Duration,
			&event.Place,
			&event.PriceType,
			&event.NeedRegistration,
			&detailsJSON,
			&event.CreatedAt,
			&event.UpdatedAt,
			&event.CreatorID,
		)
		if err != nil {
			s.logger.Error("Failed to scan event", zap.Error(err))
			continue
		}

		// Парсим JSONB поле details
		if len(detailsJSON) > 0 {
			if err := json.Unmarshal(detailsJSON, &event.Details); err != nil {
				s.logger.Warn("Failed to unmarshal event details", zap.Error(err))
				event.Details = make(map[string]interface{})
			}
		} else {
			event.Details = make(map[string]interface{})
		}

		events = append(events, &event)
	}

	if err := rows.Err(); err != nil {
		s.logger.Error("Error iterating events", zap.Error(err))
		return nil, fmt.Errorf("ошибка при обработке событий: %w", err)
	}

	return events, nil
}

// GetAllEvents оставлен для обратной совместимости.
// Deprecated: используйте GetApprovedEvents вместо него.
func (s *EventService) GetAllEvents(ctx context.Context) ([]*models.Event, error) {
	return s.GetApprovedEvents(ctx)
}

// GetEventByID получает событие по ID.
func (s *EventService) GetEventByID(ctx context.Context, id string) (*models.Event, error) {
	query := `
		SELECT id, type, start_time, end_time, duration, place, 
		       price_type, need_registration, details, created_at, updated_at
		FROM events
		WHERE id = $1
	`

	var event models.Event
	var detailsJSON []byte

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&event.ID,
		&event.Type,
		&event.StartTime,
		&event.EndTime,
		&event.Duration,
		&event.Place,
		&event.PriceType,
		&event.NeedRegistration,
		&detailsJSON,
		&event.CreatedAt,
		&event.UpdatedAt,
	)

	if err != nil {
		s.logger.Error("Failed to get event by ID", zap.String("id", id), zap.Error(err))
		return nil, fmt.Errorf("событие не найдено: %w", err)
	}

	// Парсим JSONB поле details
	if len(detailsJSON) > 0 {
		if err := json.Unmarshal(detailsJSON, &event.Details); err != nil {
			s.logger.Warn("Failed to unmarshal event details", zap.Error(err))
			event.Details = make(map[string]interface{})
		}
	} else {
		event.Details = make(map[string]interface{})
	}

	return &event, nil
}

// CreateEvent создает новое событие в таблице ожидания модерации.
func (s *EventService) CreateEvent(ctx context.Context, req *models.CreateEventRequest) (*models.PendingEvent, error) {
	detailsJSON, err := json.Marshal(req.Details)
	if err != nil {
		return nil, fmt.Errorf("ошибка при сериализации details: %w", err)
	}

	// Подготавливаем creator_id (может быть пустым)
	var creatorID interface{}
	if req.CreatorID != "" {
		creatorID = req.CreatorID
	}

	query := `
		INSERT INTO events_pending (type, start_time, end_time, duration, place, price_type, need_registration, details, creator_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, type, start_time, end_time, duration, place, price_type, need_registration, details, status, review_comment, created_at, updated_at, reviewed_at
	`

	var event models.PendingEvent
	var returnedDetailsJSON []byte
	var reviewedAt sql.NullTime
	var reviewComment sql.NullString

	err = s.db.QueryRowContext(ctx, query,
		req.Type,
		req.StartTime,
		req.EndTime,
		req.Duration,
		req.Place,
		req.PriceType,
		req.NeedRegistration,
		detailsJSON,
		creatorID,
	).Scan(
		&event.ID,
		&event.Type,
		&event.StartTime,
		&event.EndTime,
		&event.Duration,
		&event.Place,
		&event.PriceType,
		&event.NeedRegistration,
		&returnedDetailsJSON,
		&event.Status,
		&reviewComment,
		&event.CreatedAt,
		&event.UpdatedAt,
		&reviewedAt,
	)

	if err != nil {
		s.logger.Error("Failed to create event", zap.Error(err))
		return nil, fmt.Errorf("ошибка при создании события: %w", err)
	}

	event.ReviewComment = reviewComment.String

	// Парсим JSONB поле details
	if len(returnedDetailsJSON) > 0 {
		if reviewedAt.Valid {
			event.ReviewedAt = &reviewedAt.Time
		}
		if err := json.Unmarshal(returnedDetailsJSON, &event.Details); err != nil {
			s.logger.Warn("Failed to unmarshal event details", zap.Error(err))
			event.Details = make(map[string]interface{})
		}
	} else {
		event.Details = make(map[string]interface{})
	}

	s.logger.Info("Event submitted for review", zap.String("id", event.ID), zap.String("type", event.Type))

	return &event, nil
}

// DeleteEvent удаляет событие.
func (s *EventService) DeleteEvent(ctx context.Context, id string) error {
	query := `DELETE FROM events WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		s.logger.Error("Failed to delete event", zap.String("id", id), zap.Error(err))
		return fmt.Errorf("ошибка при удалении события: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка при проверке удаления: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("событие не найдено")
	}

	s.logger.Info("Event deleted", zap.String("id", id))
	return nil
}

// GetPendingEvents возвращает события, ожидающие модерации.
func (s *EventService) GetPendingEvents(ctx context.Context) ([]*models.PendingEvent, error) {
	query := `
		SELECT id, type, start_time, end_time, duration, place,
		       price_type, need_registration, details, status, review_comment,
		       created_at, updated_at, reviewed_at
		FROM events_pending
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		s.logger.Error("Failed to query pending events", zap.Error(err))
		return nil, fmt.Errorf("ошибка при получении событий на модерации: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Error("Failed to close rows after querying pending events", zap.Error(err))
		}
	}()

	var events []*models.PendingEvent
	for rows.Next() {
		event := &models.PendingEvent{}
		var detailsJSON []byte
		var reviewedAt sql.NullTime
		var reviewComment sql.NullString

		if err := rows.Scan(
			&event.ID,
			&event.Type,
			&event.StartTime,
			&event.EndTime,
			&event.Duration,
			&event.Place,
			&event.PriceType,
			&event.NeedRegistration,
			&detailsJSON,
			&event.Status,
			&reviewComment,
			&event.CreatedAt,
			&event.UpdatedAt,
			&reviewedAt,
		); err != nil {
			s.logger.Error("Failed to scan pending event", zap.Error(err))
			continue
		}

		event.ReviewComment = reviewComment.String

		if len(detailsJSON) > 0 {
			if reviewedAt.Valid {
				event.ReviewedAt = &reviewedAt.Time
			}
			if err := json.Unmarshal(detailsJSON, &event.Details); err != nil {
				s.logger.Warn("Failed to unmarshal pending event details", zap.Error(err))
				event.Details = make(map[string]interface{})
			}
		} else {
			event.Details = make(map[string]interface{})
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		s.logger.Error("Error iterating pending events", zap.Error(err))
		return nil, fmt.Errorf("ошибка при обработке событий на модерации: %w", err)
	}

	return events, nil
}

// ReviewPendingEvent изменяет статус ожидаемого события.
func (s *EventService) ReviewPendingEvent(ctx context.Context, id, action, comment string) error {
	status := strings.ToLower(action)
	switch status {
	case models.EventStatusApproved, "approve":
		status = models.EventStatusApproved
	case models.EventStatusRejected, "reject", "block":
		status = models.EventStatusRejected
	default:
		return fmt.Errorf("неподдерживаемое действие: %s", action)
	}

	query := `
		UPDATE events_pending
		SET status = $2,
		    review_comment = NULLIF($3, ''),
		    reviewed_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1 AND status = 'pending'
		RETURNING id
	`

	var updatedID string
	if err := s.db.QueryRowContext(ctx, query, id, status, comment).Scan(&updatedID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("событие не найдено или уже обработано")
		}
		s.logger.Error("Failed to review pending event", zap.String("id", id), zap.Error(err))
		return fmt.Errorf("ошибка при обновлении статуса события: %w", err)
	}

	s.logger.Info("Pending event reviewed", zap.String("id", updatedID), zap.String("status", status))
	return nil
}
