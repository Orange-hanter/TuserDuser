package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"event-api/internal/models"

	"go.uber.org/zap"
)

// CreatorService управляет событиями автора.
type CreatorService struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewCreatorService создаёт новый сервис.
func NewCreatorService(db *sql.DB, logger *zap.Logger) *CreatorService {
	return &CreatorService{db: db, logger: logger}
}

// GetMyEvents возвращает события автора по категориям.
func (s *CreatorService) GetMyEvents(ctx context.Context, creatorID string) (*models.CreatorEventsResponse, error) {
	resp := &models.CreatorEventsResponse{
		Pending:  []models.CreatorEvent{},
		Active:   []models.CreatorEvent{},
		Rejected: []models.CreatorEvent{},
	}

	// Pending events (pending + needs_revision)
	pending, err := s.queryPendingEvents(ctx, creatorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending events: %w", err)
	}
	resp.Pending = pending

	// Active events (approved, not yet ended)
	active, err := s.queryActiveEvents(ctx, creatorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active events: %w", err)
	}
	resp.Active = active

	// Rejected events
	rejected, err := s.queryRejectedEvents(ctx, creatorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rejected events: %w", err)
	}
	resp.Rejected = rejected

	return resp, nil
}

// GetBlockedEvents возвращает заблокированные события автора.
func (s *CreatorService) GetBlockedEvents(ctx context.Context, creatorID string) ([]models.BlockedEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, type, start_time, end_time, duration, place, price_type, 
		       need_registration, details, block_reason, blocked_at, created_at, updated_at
		FROM events_blocked
		WHERE creator_id = $1
		ORDER BY blocked_at DESC
	`, creatorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.BlockedEvent
	for rows.Next() {
		var e models.BlockedEvent
		var details []byte
		var place, blockReason sql.NullString
		var createdAt, updatedAt sql.NullTime

		err := rows.Scan(
			&e.ID, &e.Type, &e.StartTime, &e.EndTime, &e.Duration,
			&place, &e.PriceType, &e.NeedRegistration, &details,
			&blockReason, &e.BlockedAt, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}

		if place.Valid {
			e.Place = place.String
		}
		if blockReason.Valid {
			e.BlockReason = blockReason.String
		}
		if createdAt.Valid {
			e.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			e.UpdatedAt = updatedAt.Time
		}
		e.Status = models.EventStatusBlocked
		json.Unmarshal(details, &e.Details)

		events = append(events, e)
	}

	if events == nil {
		events = []models.BlockedEvent{}
	}
	return events, nil
}

// GetEventComments возвращает комментарии модерации для события.
func (s *CreatorService) GetEventComments(ctx context.Context, eventID, creatorID string) ([]models.ReviewComment, error) {
	// Проверяем что событие принадлежит автору
	var exists bool
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM events_pending WHERE id = $1 AND creator_id = $2
			UNION
			SELECT 1 FROM events WHERE id = $1 AND creator_id = $2
			UNION
			SELECT 1 FROM events_rejected WHERE id = $1 AND creator_id = $2
			UNION
			SELECT 1 FROM events_blocked WHERE id = $1 AND creator_id = $2
		)
	`, eventID, creatorID).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("event not found or access denied")
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, event_id, author_id, author_role, comment, created_at
		FROM event_review_comments
		WHERE event_id = $1
		ORDER BY created_at ASC
	`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.ReviewComment
	for rows.Next() {
		var c models.ReviewComment
		err := rows.Scan(&c.ID, &c.EventID, &c.AuthorID, &c.AuthorRole, &c.Comment, &c.CreatedAt)
		if err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}

	if comments == nil {
		comments = []models.ReviewComment{}
	}
	return comments, nil
}

// GetEventCommentsForAdmin возвращает комментарии модерации для события без проверки владельца.
// Используется админами для просмотра комментариев к любому событию.
func (s *CreatorService) GetEventCommentsForAdmin(ctx context.Context, eventID string) ([]models.ReviewComment, error) {
	// Проверяем что событие существует в одной из таблиц
	var exists bool
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM events_pending WHERE id = $1
			UNION
			SELECT 1 FROM events WHERE id = $1
			UNION
			SELECT 1 FROM events_rejected WHERE id = $1
			UNION
			SELECT 1 FROM events_blocked WHERE id = $1
		)
	`, eventID).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("event not found")
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, event_id, author_id, author_role, comment, created_at
		FROM event_review_comments
		WHERE event_id = $1
		ORDER BY created_at ASC
	`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.ReviewComment
	for rows.Next() {
		var c models.ReviewComment
		err := rows.Scan(&c.ID, &c.EventID, &c.AuthorID, &c.AuthorRole, &c.Comment, &c.CreatedAt)
		if err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}

	if comments == nil {
		comments = []models.ReviewComment{}
	}
	return comments, nil
}

// AddComment добавляет комментарий к событию (для админа или автора).
func (s *CreatorService) AddComment(ctx context.Context, eventID, authorID, authorRole, comment string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO event_review_comments (event_id, author_id, author_role, comment)
		VALUES ($1, $2, $3, $4)
	`, eventID, authorID, authorRole, comment)
	return err
}

// RequestRevision запрашивает доработку события (для админа).
func (s *CreatorService) RequestRevision(ctx context.Context, eventID, adminID, comment string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Обновляем статус
	result, err := tx.ExecContext(ctx, `
		UPDATE events_pending 
		SET status = 'needs_revision', review_comment = $2, updated_at = NOW()
		WHERE id = $1 AND status = 'pending'
	`, eventID, comment)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("event not found or not in pending status")
	}

	// Добавляем комментарий
	_, err = tx.ExecContext(ctx, `
		INSERT INTO event_review_comments (event_id, author_id, author_role, comment)
		VALUES ($1, $2, 'admin', $3)
	`, eventID, adminID, comment)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// BlockEvent блокирует событие (для админа).
func (s *CreatorService) BlockEvent(ctx context.Context, eventID, adminID, reason string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Пробуем найти в events (одобренные)
	var found bool
	err = tx.QueryRowContext(ctx, `
		INSERT INTO events_blocked (id, type, start_time, end_time, duration, place, price_type, 
		                            need_registration, details, creator_id, block_reason, created_at, updated_at)
		SELECT id, type, start_time, end_time, duration, place, price_type,
		       need_registration, details, creator_id, $2, created_at, updated_at
		FROM events WHERE id = $1
		RETURNING true
	`, eventID, reason).Scan(&found)

	if err == sql.ErrNoRows {
		// Пробуем найти в events_pending
		err = tx.QueryRowContext(ctx, `
			INSERT INTO events_blocked (id, type, start_time, end_time, duration, place, price_type,
			                            need_registration, details, creator_id, block_reason, created_at, updated_at)
			SELECT id, type, start_time, end_time, duration, place, price_type,
			       need_registration, details, creator_id, $2, created_at, updated_at
			FROM events_pending WHERE id = $1
			RETURNING true
		`, eventID, reason).Scan(&found)
	}

	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if !found {
		return fmt.Errorf("event not found")
	}

	// Удаляем из исходных таблиц
	tx.ExecContext(ctx, "DELETE FROM events WHERE id = $1", eventID)
	tx.ExecContext(ctx, "DELETE FROM events_pending WHERE id = $1", eventID)

	// Добавляем комментарий
	_, err = tx.ExecContext(ctx, `
		INSERT INTO event_review_comments (event_id, author_id, author_role, comment)
		VALUES ($1, $2, 'admin', $3)
	`, eventID, adminID, "Событие заблокировано: "+reason)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *CreatorService) queryPendingEvents(ctx context.Context, creatorID string) ([]models.CreatorEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, type, start_time, end_time, duration, place, price_type,
		       need_registration, details, status, review_comment, created_at, updated_at
		FROM events_pending
		WHERE creator_id = $1 AND status IN ('pending', 'needs_revision')
		ORDER BY created_at DESC
	`, creatorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanCreatorEvents(rows)
}

func (s *CreatorService) queryActiveEvents(ctx context.Context, creatorID string) ([]models.CreatorEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, type, start_time, end_time, duration, place, price_type,
		       need_registration, details, 'approved' as status, '' as review_comment, created_at, updated_at
		FROM events
		WHERE creator_id = $1 AND end_time > $2
		ORDER BY start_time ASC
	`, creatorID, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanCreatorEvents(rows)
}

func (s *CreatorService) queryRejectedEvents(ctx context.Context, creatorID string) ([]models.CreatorEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, type, start_time, end_time, duration, place, price_type,
		       need_registration, details, 'rejected' as status, review_comment, created_at, updated_at
		FROM events_rejected
		WHERE creator_id = $1
		ORDER BY updated_at DESC
	`, creatorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanCreatorEvents(rows)
}

func (s *CreatorService) scanCreatorEvents(rows *sql.Rows) ([]models.CreatorEvent, error) {
	var events []models.CreatorEvent
	for rows.Next() {
		var e models.CreatorEvent
		var details []byte
		var place, reviewComment sql.NullString

		err := rows.Scan(
			&e.ID, &e.Type, &e.StartTime, &e.EndTime, &e.Duration,
			&place, &e.PriceType, &e.NeedRegistration, &details,
			&e.Status, &reviewComment, &e.CreatedAt, &e.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if place.Valid {
			e.Place = place.String
		}
		if reviewComment.Valid {
			e.ReviewComment = reviewComment.String
		}
		json.Unmarshal(details, &e.Details)
		events = append(events, e)
	}

	if events == nil {
		events = []models.CreatorEvent{}
	}
	return events, nil
}
