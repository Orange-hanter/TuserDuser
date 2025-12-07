// Package migrations предоставляет инструменты для управления миграциями базы данных.
package migrations

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// Migrator управляет миграциями БД.
type Migrator struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewMigrator создает новый Migrator.
func NewMigrator(db *sql.DB, logger *zap.Logger) *Migrator {
	return &Migrator{
		db:     db,
		logger: logger,
	}
}

// RunMigrations запускает все миграции.
func (m *Migrator) RunMigrations() error {
	m.logger.Info("🔄 Запускаем миграции БД...")

	// Создаем таблицу для отслеживания миграций
	if err := m.createMigrationsTable(); err != nil {
		m.logger.Error("Ошибка при создании таблицы migrations", zap.Error(err))
		return err
	}

	// Список всех миграций
	migrations := []migration{
		{
			name: "001_create_users_table",
			up:   createUsersTable,
			down: dropUsersTable,
		},
		{
			name: "002_create_events_table",
			up:   createEventsTable,
			down: dropEventsTable,
		},
		{
			name: "003_create_verification_codes_table",
			up:   createVerificationCodesTable,
			down: dropVerificationCodesTable,
		},
		{
			name: "004_create_events_review_pipeline",
			up:   createEventsReviewTables,
			down: dropEventsReviewTables,
		},
		{
			name: "005_add_role_to_users",
			up:   addRoleToUsers,
			down: dropRoleFromUsers,
		},
		{
			name: "006_create_telegram_notifications",
			up:   createTelegramNotifications,
			down: dropTelegramNotifications,
		},
		{
			name: "007_create_event_subscriptions",
			up:   createEventSubscriptions,
			down: dropEventSubscriptions,
		},
		{
			name: "008_create_discovery_actions",
			up:   createDiscoveryActions,
			down: dropDiscoveryActions,
		},
		{
			name: "009_add_creator_id_to_events",
			up:   addCreatorIdToEvents,
			down: dropCreatorIdFromEvents,
		},
		{
			name: "010_update_review_trigger_with_creator_id",
			up:   updateReviewTriggerWithCreatorId,
			down: revertReviewTriggerWithCreatorId,
		},
		{
			name: "011_create_event_registrations",
			up:   createEventRegistrations,
			down: dropEventRegistrations,
		},
		{
			name: "012_create_role_requests_table",
			up:   createRoleRequestsTable,
			down: dropRoleRequestsTable,
		},
		{
			name: "013_migrate_user_id_to_uuid",
			up:   migrateUserIdToUUID,
			down: revertUserIdToText,
		},
		{
			name: "014_drop_event_registrations",
			up:   dropEventRegistrationsConsolidation,
			down: recreateEventRegistrations,
		},
		{
			name: "015_add_public_profile_fields",
			up:   addPublicProfileFields,
			down: dropPublicProfileFields,
		},
		{
			name: "016_create_event_reminder_log",
			up:   createEventReminderLog,
			down: dropEventReminderLog,
		},
		{
			name: "017_create_feedback_table",
			up:   createFeedbackTable,
			down: dropFeedbackTable,
		},
	}

	// Запускаем каждую миграцию
	for _, mig := range migrations {
		if err := m.runMigration(mig); err != nil {
			m.logger.Error("Ошибка при выполнении миграции",
				zap.String("name", mig.name),
				zap.Error(err),
			)
			return err
		}
	}

	m.logger.Info("✅ Все миграции успешно выполнены")

	// After applying schema migrations, ensure default admin exists (idempotent)
	if err := m.seedDefaultAdmin(); err != nil {
		m.logger.Warn("Не удалось выполнить инициализацию администратора", zap.Error(err))
		// Do not fail startup because of seeding; just warn.
	}
	return nil
}

// migration представляет одну миграцию.
type migration struct {
	name string
	up   string
	down string
}

// createMigrationsTable создает таблицу для отслеживания миграций.
func (m *Migrator) createMigrationsTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) UNIQUE NOT NULL,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)
	`
	_, err := m.db.ExecContext(context.Background(), query)
	return err
}

// runMigration запускает одну миграцию.
func (m *Migrator) runMigration(mig migration) error {
	ctx := context.Background()
	// Проверяем, была ли уже применена эта миграция
	var exists bool
	err := m.db.QueryRowContext(
		ctx,
		"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name = $1)",
		mig.name,
	).Scan(&exists)

	if err != nil {
		return err
	}

	if exists {
		m.logger.Info("⏭️  Миграция уже применена", zap.String("name", mig.name))
		return nil
	}

	// Запускаем миграцию
	m.logger.Info("▶️  Выполняем миграцию", zap.String("name", mig.name))

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTx(tx, m.logger)

	// Выполняем SQL
	if _, err := tx.ExecContext(ctx, mig.up); err != nil {
		return fmt.Errorf("failed to apply migration %s: %w", mig.name, err)
	}

	// Записываем в schema_migrations
	if _, err := tx.ExecContext(
		ctx,
		"INSERT INTO schema_migrations (name) VALUES ($1)",
		mig.name,
	); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	m.logger.Info("✅ Миграция успешно применена", zap.String("name", mig.name))
	return nil
}

// SQL миграции

// createUsersTable создает таблицу users.
const createUsersTable = `
CREATE TABLE IF NOT EXISTS users (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	email VARCHAR(255) UNIQUE NOT NULL,
	phone VARCHAR(20),
	password VARCHAR(255) NOT NULL,
	verified BOOLEAN DEFAULT FALSE,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_verified ON users(verified);
`

// dropUsersTable удаляет таблицу users.
const dropUsersTable = `
DROP TABLE IF EXISTS users CASCADE;
`

// createVerificationCodesTable создает таблицу verification_codes.
const createVerificationCodesTable = `
CREATE TABLE IF NOT EXISTS verification_codes (
	id SERIAL PRIMARY KEY,
	email VARCHAR(255) NOT NULL,
	code VARCHAR(10) NOT NULL,
	expires_at TIMESTAMP NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_verification_codes_email ON verification_codes(email);
CREATE INDEX IF NOT EXISTS idx_verification_codes_expires_at ON verification_codes(expires_at);
`

// dropVerificationCodesTable удаляет таблицу verification_codes.
const dropVerificationCodesTable = `
DROP TABLE IF EXISTS verification_codes CASCADE;
`

// createEventsTable - SQL для создания таблицы events.
const createEventsTable = `
CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(100) NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    duration INTEGER NOT NULL,
    place VARCHAR(255),
    price_type VARCHAR(50) NOT NULL DEFAULT 'free',
    need_registration BOOLEAN NOT NULL DEFAULT false,
    details JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_events_start_time ON events(start_time);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);

INSERT INTO events (type, start_time, end_time, duration, place, price_type, need_registration, details)
VALUES 
    ('Встреча', '2025-10-10T18:05:00Z', '2025-10-10T18:35:00Z', 30, 'Офис', 'free', false, '{"description": "Встреча команды"}'),
    ('Конференция', '2025-11-15T09:00:00Z', '2025-11-15T18:00:00Z', 540, 'Конференц-зал', 'paid', true, '{"price": 1500, "capacity": 100}'),
    ('Вебинар', '2025-10-20T14:00:00Z', '2025-10-20T15:30:00Z', 90, 'Online', 'free', true, '{"link": "https://zoom.us/meeting"}')
ON CONFLICT DO NOTHING;
`

// dropEventsTable - SQL для удаления таблицы events.
const dropEventsTable = `
DROP TABLE IF EXISTS events CASCADE;
`

const createEventsReviewTables = `
CREATE TABLE IF NOT EXISTS events_pending (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	type VARCHAR(100) NOT NULL,
	start_time TIMESTAMPTZ NOT NULL,
	end_time TIMESTAMPTZ NOT NULL,
	duration INTEGER NOT NULL,
	place VARCHAR(255),
	price_type VARCHAR(50) NOT NULL DEFAULT 'free',
	need_registration BOOLEAN NOT NULL DEFAULT false,
	details JSONB NOT NULL DEFAULT '{}'::jsonb,
	status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','rejected')),
	review_comment TEXT,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	reviewed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_events_pending_status ON events_pending(status);
CREATE INDEX IF NOT EXISTS idx_events_pending_created_at ON events_pending(created_at);

CREATE TABLE IF NOT EXISTS events_rejected (
	id UUID PRIMARY KEY,
	type VARCHAR(100) NOT NULL,
	start_time TIMESTAMPTZ NOT NULL,
	end_time TIMESTAMPTZ NOT NULL,
	duration INTEGER NOT NULL,
	place VARCHAR(255),
	price_type VARCHAR(50) NOT NULL DEFAULT 'free',
	need_registration BOOLEAN NOT NULL DEFAULT false,
	details JSONB NOT NULL DEFAULT '{}'::jsonb,
	review_comment TEXT,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	rejected_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_events_rejected_rejected_at ON events_rejected(rejected_at);

CREATE OR REPLACE FUNCTION handle_events_pending_status_change()
RETURNS TRIGGER AS $$
BEGIN
	IF NEW.status = 'approved' AND OLD.status <> 'approved' THEN
		INSERT INTO events (id, type, start_time, end_time, duration, place, price_type, need_registration, details, created_at, updated_at)
		VALUES (NEW.id, NEW.type, NEW.start_time, NEW.end_time, NEW.duration, NEW.place, NEW.price_type, NEW.need_registration, NEW.details, NEW.created_at, NOW())
		ON CONFLICT (id) DO UPDATE SET
			type = EXCLUDED.type,
			start_time = EXCLUDED.start_time,
			end_time = EXCLUDED.end_time,
			duration = EXCLUDED.duration,
			place = EXCLUDED.place,
			price_type = EXCLUDED.price_type,
			need_registration = EXCLUDED.need_registration,
			details = EXCLUDED.details,
			updated_at = NOW();

		DELETE FROM events_pending WHERE id = NEW.id;
		RETURN NULL;
	ELSIF NEW.status = 'rejected' AND OLD.status <> 'rejected' THEN
		INSERT INTO events_rejected (id, type, start_time, end_time, duration, place, price_type, need_registration, details, review_comment, created_at, updated_at, rejected_at)
		VALUES (NEW.id, NEW.type, NEW.start_time, NEW.end_time, NEW.duration, NEW.place, NEW.price_type, NEW.need_registration, NEW.details, NEW.review_comment, NEW.created_at, NEW.updated_at, NOW())
		ON CONFLICT (id) DO UPDATE SET
			type = EXCLUDED.type,
			start_time = EXCLUDED.start_time,
			end_time = EXCLUDED.end_time,
			duration = EXCLUDED.duration,
			place = EXCLUDED.place,
			price_type = EXCLUDED.price_type,
			need_registration = EXCLUDED.need_registration,
			details = EXCLUDED.details,
			review_comment = EXCLUDED.review_comment,
			updated_at = NOW(),
			rejected_at = NOW();

		DELETE FROM events_pending WHERE id = NEW.id;
		RETURN NULL;
	END IF;
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_events_pending_status_change ON events_pending;

CREATE TRIGGER trg_events_pending_status_change
AFTER UPDATE OF status ON events_pending
FOR EACH ROW
WHEN (OLD.status IS DISTINCT FROM NEW.status)
EXECUTE FUNCTION handle_events_pending_status_change();
`

const dropEventsReviewTables = `
DROP TRIGGER IF EXISTS trg_events_pending_status_change ON events_pending;
DROP FUNCTION IF EXISTS handle_events_pending_status_change;
DROP TABLE IF EXISTS events_rejected;
DROP TABLE IF EXISTS events_pending;
`

// addRoleToUsers добавляет колонку role в таблицу users.
const addRoleToUsers = `
ALTER TABLE users
ADD COLUMN IF NOT EXISTS role VARCHAR(20) NOT NULL DEFAULT 'user'
CHECK (role IN ('user', 'creator', 'support', 'admin'));

CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- Установим роль 'user' для всех существующих пользователей
UPDATE users SET role = 'user' WHERE role IS NULL OR role = '';
`

// dropRoleFromUsers удаляет колонку role из таблицы users.
const dropRoleFromUsers = `
DROP INDEX IF EXISTS idx_users_role;
ALTER TABLE users DROP COLUMN IF EXISTS role;
`

const createTelegramNotifications = `
CREATE TABLE IF NOT EXISTS telegram_binding_tokens (
	nonce_hash TEXT PRIMARY KEY,
	user_id UUID NOT NULL,
	expires_at TIMESTAMPTZ NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_telegram_binding_tokens_expires_at
ON telegram_binding_tokens (expires_at);

CREATE TABLE IF NOT EXISTS telegram_bindings (
	user_id UUID PRIMARY KEY,
	chat_id BIGINT UNIQUE NOT NULL,
	status TEXT NOT NULL CHECK (status IN ('active','blocked','pending','revoked')),
	blocked_reason TEXT,
	last_error_code INTEGER,
	last_error_at TIMESTAMPTZ,
	telegram_username TEXT,
	telegram_first_name TEXT,
	telegram_last_name TEXT,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_telegram_bindings_status ON telegram_bindings(status);
CREATE INDEX IF NOT EXISTS idx_telegram_bindings_chat_id ON telegram_bindings(chat_id);

CREATE TABLE IF NOT EXISTS telegram_delivery (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID NOT NULL,
	chat_id BIGINT NOT NULL,
	reminder_id TEXT NOT NULL,
	payload JSONB NOT NULL,
	status TEXT NOT NULL CHECK (status IN ('scheduled','sending','sent','failed','blocked','abandoned')),
	attempt_count INTEGER NOT NULL DEFAULT 0,
	last_error_code INTEGER,
	last_error_msg TEXT,
	next_attempt_at TIMESTAMPTZ,
	message_id TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_telegram_delivery_status_next_attempt
ON telegram_delivery (status, next_attempt_at);

CREATE TABLE IF NOT EXISTS telegram_delivery_log (
	id BIGSERIAL PRIMARY KEY,
	delivery_id UUID NOT NULL REFERENCES telegram_delivery(id) ON DELETE CASCADE,
	status TEXT NOT NULL,
	error_code INTEGER,
	error_msg TEXT,
	attempt INTEGER NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_telegram_delivery_log_delivery_id
ON telegram_delivery_log(delivery_id);

CREATE TABLE IF NOT EXISTS telegram_webhook_events (
	id BIGSERIAL PRIMARY KEY,
	bot_alias TEXT NOT NULL,
	update_id BIGINT,
	payload JSONB NOT NULL,
	received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`

const dropTelegramNotifications = `
DROP TABLE IF EXISTS telegram_webhook_events;
DROP TABLE IF EXISTS telegram_delivery_log;
DROP TABLE IF EXISTS telegram_delivery;
DROP TABLE IF EXISTS telegram_bindings;
DROP TABLE IF EXISTS telegram_binding_tokens;
`

// createEventSubscriptions - SQL для создания таблицы подписок на события.
const createEventSubscriptions = `
CREATE TABLE IF NOT EXISTS event_subscriptions (
    user_id UUID NOT NULL,
    event_id UUID NOT NULL,
    status VARCHAR(50) NOT NULL,
    subscribed_at TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, event_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_event_subscriptions_user_id ON event_subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_event_subscriptions_event_id ON event_subscriptions(event_id);
`

// dropEventSubscriptions - SQL для удаления таблицы подписок на события.
const dropEventSubscriptions = `
DROP TABLE IF EXISTS event_subscriptions;
`

// createDiscoveryActions - SQL для создания таблицы discovery_actions (история свайпов).
// Таблица оптимизирована для аналитики и быстрого поиска последнего действия.
// FK на users не создаётся — это позволяет хранить историю для удалённых пользователей
// и снижает связанность между модулями.
const createDiscoveryActions = `
CREATE TABLE IF NOT EXISTS discovery_actions (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    event_id UUID NOT NULL,
    action VARCHAR(20) NOT NULL CHECK (action IN ('like', 'dislike', 'neutral', 'book')),
    context JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for fetching user's history (ordered by time)
CREATE INDEX IF NOT EXISTS idx_discovery_actions_user_time ON discovery_actions(user_id, created_at DESC);

-- Index for finding last action on specific event
CREATE INDEX IF NOT EXISTS idx_discovery_actions_user_event ON discovery_actions(user_id, event_id, created_at DESC);

-- Index for analytics: action distribution
CREATE INDEX IF NOT EXISTS idx_discovery_actions_action ON discovery_actions(action);

-- Index for analytics: event popularity
CREATE INDEX IF NOT EXISTS idx_discovery_actions_event ON discovery_actions(event_id);

-- Partial index for quick "excluded events" lookup (events user already actioned definitively)
CREATE INDEX IF NOT EXISTS idx_discovery_actions_excluded ON discovery_actions(user_id, event_id)
    WHERE action IN ('like', 'dislike', 'book');

COMMENT ON TABLE discovery_actions IS 'Stores all user swipe actions for discovery queue. Used for queue filtering and analytics.';
COMMENT ON COLUMN discovery_actions.action IS 'like=interested, dislike=not interested, neutral=skip for now, book=confirmed participation';
COMMENT ON COLUMN discovery_actions.context IS 'Additional metadata: conflictedEventIds for book, source for external bookings, etc.';
`

// dropDiscoveryActions - SQL для удаления таблицы discovery_actions.
const dropDiscoveryActions = `
DROP TABLE IF EXISTS discovery_actions;
`

// addCreatorIdToEvents добавляет creator_id и расширяет статусы событий.
const addCreatorIdToEvents = `
-- Добавляем creator_id в events_pending
ALTER TABLE events_pending ADD COLUMN IF NOT EXISTS creator_id UUID;
CREATE INDEX IF NOT EXISTS idx_events_pending_creator ON events_pending(creator_id);

-- Добавляем creator_id в events
ALTER TABLE events ADD COLUMN IF NOT EXISTS creator_id UUID;
CREATE INDEX IF NOT EXISTS idx_events_creator ON events(creator_id);

-- Добавляем creator_id в events_rejected  
ALTER TABLE events_rejected ADD COLUMN IF NOT EXISTS creator_id UUID;
CREATE INDEX IF NOT EXISTS idx_events_rejected_creator ON events_rejected(creator_id);

-- Расширяем статусы: pending, needs_revision, approved, rejected, blocked
ALTER TABLE events_pending DROP CONSTRAINT IF EXISTS events_pending_status_check;
ALTER TABLE events_pending ADD CONSTRAINT events_pending_status_check 
    CHECK (status IN ('pending', 'needs_revision', 'approved', 'rejected'));

-- Таблица для заблокированных событий
CREATE TABLE IF NOT EXISTS events_blocked (
    id UUID PRIMARY KEY,
    type VARCHAR(100) NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    duration INTEGER NOT NULL,
    place VARCHAR(255),
    price_type VARCHAR(50) NOT NULL DEFAULT 'free',
    need_registration BOOLEAN NOT NULL DEFAULT false,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    creator_id UUID,
    block_reason TEXT,
    blocked_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_events_blocked_creator ON events_blocked(creator_id);
CREATE INDEX IF NOT EXISTS idx_events_blocked_at ON events_blocked(blocked_at);

-- Таблица комментариев модерации
CREATE TABLE IF NOT EXISTS event_review_comments (
    id SERIAL PRIMARY KEY,
    event_id UUID NOT NULL,
    author_id UUID NOT NULL,
    author_role VARCHAR(20) NOT NULL,
    comment TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_event_review_comments_event ON event_review_comments(event_id);
`

// dropCreatorIdFromEvents откатывает миграцию.
const dropCreatorIdFromEvents = `
DROP TABLE IF EXISTS event_review_comments;
DROP TABLE IF EXISTS events_blocked;
ALTER TABLE events_pending DROP COLUMN IF EXISTS creator_id;
ALTER TABLE events DROP COLUMN IF EXISTS creator_id;
ALTER TABLE events_rejected DROP COLUMN IF EXISTS creator_id;
`

// updateReviewTriggerWithCreatorId обновляет триггер для копирования creator_id.
const updateReviewTriggerWithCreatorId = `
-- Обновляем триггер для включения creator_id
CREATE OR REPLACE FUNCTION handle_events_pending_status_change()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'approved' AND OLD.status <> 'approved' THEN
        INSERT INTO events (id, type, start_time, end_time, duration, place, price_type, need_registration, details, creator_id, created_at, updated_at)
        VALUES (NEW.id, NEW.type, NEW.start_time, NEW.end_time, NEW.duration, NEW.place, NEW.price_type, NEW.need_registration, NEW.details, NEW.creator_id, NEW.created_at, NOW())
        ON CONFLICT (id) DO UPDATE SET
            type = EXCLUDED.type,
            start_time = EXCLUDED.start_time,
            end_time = EXCLUDED.end_time,
            duration = EXCLUDED.duration,
            place = EXCLUDED.place,
            price_type = EXCLUDED.price_type,
            need_registration = EXCLUDED.need_registration,
            details = EXCLUDED.details,
            creator_id = EXCLUDED.creator_id,
            updated_at = NOW();

        DELETE FROM events_pending WHERE id = NEW.id;
        RETURN NULL;
    ELSIF NEW.status = 'rejected' AND OLD.status <> 'rejected' THEN
        INSERT INTO events_rejected (id, type, start_time, end_time, duration, place, price_type, need_registration, details, review_comment, creator_id, created_at, updated_at, rejected_at)
        VALUES (NEW.id, NEW.type, NEW.start_time, NEW.end_time, NEW.duration, NEW.place, NEW.price_type, NEW.need_registration, NEW.details, NEW.review_comment, NEW.creator_id, NEW.created_at, NEW.updated_at, NOW())
        ON CONFLICT (id) DO UPDATE SET
            type = EXCLUDED.type,
            start_time = EXCLUDED.start_time,
            end_time = EXCLUDED.end_time,
            duration = EXCLUDED.duration,
            place = EXCLUDED.place,
            price_type = EXCLUDED.price_type,
            need_registration = EXCLUDED.need_registration,
            details = EXCLUDED.details,
            review_comment = EXCLUDED.review_comment,
            creator_id = EXCLUDED.creator_id,
            updated_at = NOW(),
            rejected_at = NOW();

        DELETE FROM events_pending WHERE id = NEW.id;
        RETURN NULL;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
`

// revertReviewTriggerWithCreatorId откатывает миграцию триггера.
const revertReviewTriggerWithCreatorId = `
-- Возвращаем старую версию триггера без creator_id
CREATE OR REPLACE FUNCTION handle_events_pending_status_change()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'approved' AND OLD.status <> 'approved' THEN
        INSERT INTO events (id, type, start_time, end_time, duration, place, price_type, need_registration, details, created_at, updated_at)
        VALUES (NEW.id, NEW.type, NEW.start_time, NEW.end_time, NEW.duration, NEW.place, NEW.price_type, NEW.need_registration, NEW.details, NEW.created_at, NOW())
        ON CONFLICT (id) DO UPDATE SET
            type = EXCLUDED.type,
            start_time = EXCLUDED.start_time,
            end_time = EXCLUDED.end_time,
            duration = EXCLUDED.duration,
            place = EXCLUDED.place,
            price_type = EXCLUDED.price_type,
            need_registration = EXCLUDED.need_registration,
            details = EXCLUDED.details,
            updated_at = NOW();

        DELETE FROM events_pending WHERE id = NEW.id;
        RETURN NULL;
    ELSIF NEW.status = 'rejected' AND OLD.status <> 'rejected' THEN
        INSERT INTO events_rejected (id, type, start_time, end_time, duration, place, price_type, need_registration, details, review_comment, created_at, updated_at, rejected_at)
        VALUES (NEW.id, NEW.type, NEW.start_time, NEW.end_time, NEW.duration, NEW.place, NEW.price_type, NEW.need_registration, NEW.details, NEW.review_comment, NEW.created_at, NEW.updated_at, NOW())
        ON CONFLICT (id) DO UPDATE SET
            type = EXCLUDED.type,
            start_time = EXCLUDED.start_time,
            end_time = EXCLUDED.end_time,
            duration = EXCLUDED.duration,
            place = EXCLUDED.place,
            price_type = EXCLUDED.price_type,
            need_registration = EXCLUDED.need_registration,
            details = EXCLUDED.details,
            review_comment = EXCLUDED.review_comment,
            updated_at = NOW(),
            rejected_at = NOW();

        DELETE FROM events_pending WHERE id = NEW.id;
        RETURN NULL;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
`

// Rollback откатывает все миграции (только для разработки!)
func (m *Migrator) Rollback() error {
	m.logger.Warn("⚠️  Откатываем все миграции (только для разработки!)")

	migrations := []migration{
		{
			name: "001_create_users_table",
			down: dropUsersTable,
		},
	}

	// Откатываем в обратном порядке
	for i := len(migrations) - 1; i >= 0; i-- {
		mig := migrations[i]

		m.logger.Info("▶️  Откатываем миграцию", zap.String("name", mig.name))

		if err := m.rollbackMigration(mig); err != nil {
			return err
		}

		m.logger.Info("✅ Миграция отката успешна", zap.String("name", mig.name))
	}

	m.logger.Info("✅ Все миграции отката успешно выполнены")
	return nil
}

func (m *Migrator) rollbackMigration(mig migration) error {
	ctx := context.Background()

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTx(tx, m.logger)

	if _, err := tx.ExecContext(ctx, mig.down); err != nil {
		return fmt.Errorf("failed to rollback migration %s: %w", mig.name, err)
	}

	if _, err := tx.ExecContext(
		ctx,
		"DELETE FROM schema_migrations WHERE name = $1",
		mig.name,
	); err != nil {
		return err
	}

	return tx.Commit()
}

func rollbackTx(tx *sql.Tx, logger *zap.Logger) {
	if tx == nil {
		return
	}

	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		logger.Error("transaction rollback failed", zap.Error(err))
	}
}

// seedDefaultAdmin creates a default admin user if none exists yet. Idempotent.
func (m *Migrator) seedDefaultAdmin() error {
	ctx := context.Background()

	var exists bool
	if err := m.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE role = 'admin')").Scan(&exists); err != nil {
		return fmt.Errorf("failed to check existing admin: %w", err)
	}
	if exists {
		m.logger.Info("👮 Администратор уже существует — пропускаем инициализацию")
		return nil
	}

	// Read defaults from environment, with sensible fallbacks
	email := os.Getenv("ADMIN_EMAIL")
	if email == "" {
		email = "admin@example.com"
	}
	phone := os.Getenv("ADMIN_PHONE")
	if phone == "" {
		phone = "+70000000000"
	}
	password := os.Getenv("ADMIN_PASSWORD")
	generatedPassword := false
	if password == "" {
		// Generate a random 16-hex password
		b := make([]byte, 12)
		if _, err := rand.Read(b); err == nil {
			password = "Adm_" + hex.EncodeToString(b)
			generatedPassword = true
		} else {
			// Fallback hardcoded, user should change it ASAP
			password = "Adm_ChangeMe123!"
			generatedPassword = true
		}
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash default admin password: %w", err)
	}

	// Generate UUID
	adminID := uuid.New().String()

	// Try to insert. If email conflicts but no admin exists, we'll attempt a fallback email once.
	inserted, err := m.tryInsertAdmin(ctx, adminID, email, phone, string(hash))
	if err != nil {
		return err
	}
	if !inserted {
		// Re-check if an admin appeared concurrently
		if err := m.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE role = 'admin')").Scan(&exists); err == nil && exists {
			m.logger.Info("👮 Администратор был создан параллельно — пропускаем")
			return nil
		}
		// Fallback email to avoid collision
		fallbackEmail := fmt.Sprintf("admin+%s@local", adminID[:6])
		inserted2, err2 := m.tryInsertAdmin(ctx, adminID, fallbackEmail, phone, string(hash))
		if err2 != nil {
			return err2
		}
		if !inserted2 {
			return fmt.Errorf("could not create default admin: email conflicts; set unique ADMIN_EMAIL env var")
		}
		email = fallbackEmail
	}

	// Log credentials info (password only if it was generated here)
	fields := []zap.Field{zap.String("email", email), zap.String("phone", phone)}
	if generatedPassword {
		fields = append(fields, zap.String("password", password))
		m.logger.Warn("🚨 Создан администратор по умолчанию. ОБЯЗАТЕЛЬНО смените пароль при первом входе!", fields...)
	} else {
		m.logger.Info("✅ Администратор по умолчанию создан (использованы переменные окружения)", fields...)
	}
	return nil
}

func (m *Migrator) tryInsertAdmin(ctx context.Context, id, email, phone, passwordHash string) (bool, error) {
	res, err := m.db.ExecContext(ctx, `
		INSERT INTO users (id, email, phone, password, role, verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'admin', TRUE, NOW(), NOW())
		ON CONFLICT (email) DO NOTHING
	`, id, email, phone, passwordHash)
	if err != nil {
		return false, fmt.Errorf("failed to insert default admin: %w", err)
	}
	rows, _ := res.RowsAffected()
	return rows > 0, nil
}

const createEventRegistrations = `
DROP TABLE IF EXISTS event_registrations CASCADE;

CREATE TABLE event_registrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    public_name VARCHAR(255) NOT NULL,
    avatar_url TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'confirmed',
    registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(event_id, user_id)
);

CREATE INDEX idx_event_registrations_event_id ON event_registrations(event_id);
CREATE INDEX idx_event_registrations_user_id ON event_registrations(user_id);
CREATE INDEX idx_event_registrations_status ON event_registrations(status);
CREATE INDEX idx_event_registrations_event_status ON event_registrations(event_id, status);
`

const dropEventRegistrations = `
DROP TABLE IF EXISTS event_registrations CASCADE;
`

const createRoleRequestsTable = `
CREATE TABLE IF NOT EXISTS role_requests (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	requested_role VARCHAR(50) NOT NULL,
	reason TEXT NOT NULL,
	status VARCHAR(50) DEFAULT 'pending',
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
	reviewed_at TIMESTAMP,
	review_notes TEXT,
	UNIQUE(user_id, requested_role)
);

CREATE INDEX idx_role_requests_user_id ON role_requests(user_id);
CREATE INDEX idx_role_requests_status ON role_requests(status);
CREATE INDEX idx_role_requests_requested_role ON role_requests(requested_role);
CREATE INDEX idx_role_requests_created_at ON role_requests(created_at);
`

const dropRoleRequestsTable = `
DROP TABLE IF EXISTS role_requests CASCADE;
`

// migrateUserIdToUUID конвертирует users.id и все связанные колонки из TEXT в UUID.
// Миграция идемпотентна — безопасна для повторного применения.
const migrateUserIdToUUID = `
DO $$
DECLARE
    users_id_type text;
BEGIN
    -- Проверяем текущий тип колонки users.id
    SELECT data_type INTO users_id_type
    FROM information_schema.columns
    WHERE table_name = 'users' AND column_name = 'id';

    -- Если уже UUID — ничего не делаем
    IF users_id_type = 'uuid' THEN
        RAISE NOTICE 'users.id is already UUID, skipping migration';
        RETURN;
    END IF;

    -- Шаг 1: Удаляем ВСЕ FK перед изменениями
    ALTER TABLE event_registrations DROP CONSTRAINT IF EXISTS event_registrations_user_id_fkey;
    ALTER TABLE event_subscriptions DROP CONSTRAINT IF EXISTS event_subscriptions_user_id_fkey;
    ALTER TABLE role_requests DROP CONSTRAINT IF EXISTS role_requests_user_id_fkey;
    ALTER TABLE role_requests DROP CONSTRAINT IF EXISTS role_requests_reviewed_by_fkey;

    -- Шаг 2: Конвертируем 32-символьные hex ID в формат UUID с дефисами
    UPDATE users SET id =
        SUBSTRING(id::text, 1, 8) || '-' ||
        SUBSTRING(id::text, 9, 4) || '-' ||
        SUBSTRING(id::text, 13, 4) || '-' ||
        SUBSTRING(id::text, 17, 4) || '-' ||
        SUBSTRING(id::text, 21, 12)
    WHERE LENGTH(id::text) = 32 AND id::text !~ '-';

    UPDATE event_registrations SET user_id =
        SUBSTRING(user_id::text, 1, 8) || '-' ||
        SUBSTRING(user_id::text, 9, 4) || '-' ||
        SUBSTRING(user_id::text, 13, 4) || '-' ||
        SUBSTRING(user_id::text, 17, 4) || '-' ||
        SUBSTRING(user_id::text, 21, 12)
    WHERE LENGTH(user_id::text) = 32 AND user_id::text !~ '-';

    UPDATE event_subscriptions SET user_id =
        SUBSTRING(user_id::text, 1, 8) || '-' ||
        SUBSTRING(user_id::text, 9, 4) || '-' ||
        SUBSTRING(user_id::text, 13, 4) || '-' ||
        SUBSTRING(user_id::text, 17, 4) || '-' ||
        SUBSTRING(user_id::text, 21, 12)
    WHERE LENGTH(user_id::text) = 32 AND user_id::text !~ '-';

    UPDATE telegram_bindings SET user_id =
        SUBSTRING(user_id::text, 1, 8) || '-' ||
        SUBSTRING(user_id::text, 9, 4) || '-' ||
        SUBSTRING(user_id::text, 13, 4) || '-' ||
        SUBSTRING(user_id::text, 17, 4) || '-' ||
        SUBSTRING(user_id::text, 21, 12)
    WHERE LENGTH(user_id::text) = 32 AND user_id::text !~ '-';

    UPDATE telegram_binding_tokens SET user_id =
        SUBSTRING(user_id::text, 1, 8) || '-' ||
        SUBSTRING(user_id::text, 9, 4) || '-' ||
        SUBSTRING(user_id::text, 13, 4) || '-' ||
        SUBSTRING(user_id::text, 17, 4) || '-' ||
        SUBSTRING(user_id::text, 21, 12)
    WHERE LENGTH(user_id::text) = 32 AND user_id::text !~ '-';

    UPDATE telegram_delivery SET user_id =
        SUBSTRING(user_id::text, 1, 8) || '-' ||
        SUBSTRING(user_id::text, 9, 4) || '-' ||
        SUBSTRING(user_id::text, 13, 4) || '-' ||
        SUBSTRING(user_id::text, 17, 4) || '-' ||
        SUBSTRING(user_id::text, 21, 12)
    WHERE LENGTH(user_id::text) = 32 AND user_id::text !~ '-';

    UPDATE role_requests SET user_id =
        SUBSTRING(user_id::text, 1, 8) || '-' ||
        SUBSTRING(user_id::text, 9, 4) || '-' ||
        SUBSTRING(user_id::text, 13, 4) || '-' ||
        SUBSTRING(user_id::text, 17, 4) || '-' ||
        SUBSTRING(user_id::text, 21, 12)
    WHERE LENGTH(user_id::text) = 32 AND user_id::text !~ '-';

    UPDATE role_requests SET reviewed_by =
        SUBSTRING(reviewed_by::text, 1, 8) || '-' ||
        SUBSTRING(reviewed_by::text, 9, 4) || '-' ||
        SUBSTRING(reviewed_by::text, 13, 4) || '-' ||
        SUBSTRING(reviewed_by::text, 17, 4) || '-' ||
        SUBSTRING(reviewed_by::text, 21, 12)
    WHERE reviewed_by IS NOT NULL AND LENGTH(reviewed_by::text) = 32 AND reviewed_by::text !~ '-';

    UPDATE discovery_actions SET user_id =
        SUBSTRING(user_id::text, 1, 8) || '-' ||
        SUBSTRING(user_id::text, 9, 4) || '-' ||
        SUBSTRING(user_id::text, 13, 4) || '-' ||
        SUBSTRING(user_id::text, 17, 4) || '-' ||
        SUBSTRING(user_id::text, 21, 12)
    WHERE LENGTH(user_id::text) = 32 AND user_id::text !~ '-';

    -- Шаг 3: Меняем тип колонок на UUID
    ALTER TABLE users ALTER COLUMN id TYPE UUID USING id::uuid;
    ALTER TABLE event_registrations ALTER COLUMN user_id TYPE UUID USING user_id::uuid;
    ALTER TABLE event_subscriptions ALTER COLUMN user_id TYPE UUID USING user_id::uuid;
    ALTER TABLE telegram_bindings ALTER COLUMN user_id TYPE UUID USING user_id::uuid;
    ALTER TABLE telegram_binding_tokens ALTER COLUMN user_id TYPE UUID USING user_id::uuid;
    ALTER TABLE telegram_delivery ALTER COLUMN user_id TYPE UUID USING user_id::uuid;
    ALTER TABLE role_requests ALTER COLUMN user_id TYPE UUID USING user_id::uuid;
    ALTER TABLE role_requests ALTER COLUMN reviewed_by TYPE UUID USING reviewed_by::uuid;
    ALTER TABLE discovery_actions ALTER COLUMN user_id TYPE UUID USING user_id::uuid;

    -- Шаг 4: Восстанавливаем FK
    ALTER TABLE event_registrations
        ADD CONSTRAINT event_registrations_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

    ALTER TABLE event_subscriptions
        ADD CONSTRAINT event_subscriptions_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

    ALTER TABLE role_requests
        ADD CONSTRAINT role_requests_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

    ALTER TABLE role_requests
        ADD CONSTRAINT role_requests_reviewed_by_fkey
        FOREIGN KEY (reviewed_by) REFERENCES users(id) ON DELETE SET NULL;

    RAISE NOTICE 'Migration completed: users.id converted to UUID';
END $$;
`

// revertUserIdToText откатывает миграцию UUID обратно в TEXT (не рекомендуется).
// Note: event_registrations may not exist if migration 014 was applied.
const revertUserIdToText = `
-- Удаляем FK (IF EXISTS for tables that may be dropped)
ALTER TABLE IF EXISTS event_registrations DROP CONSTRAINT IF EXISTS event_registrations_user_id_fkey;
ALTER TABLE event_subscriptions DROP CONSTRAINT IF EXISTS event_subscriptions_user_id_fkey;
ALTER TABLE role_requests DROP CONSTRAINT IF EXISTS role_requests_user_id_fkey;
ALTER TABLE role_requests DROP CONSTRAINT IF EXISTS role_requests_reviewed_by_fkey;

-- Меняем типы обратно на TEXT
ALTER TABLE users ALTER COLUMN id TYPE TEXT USING id::text;
-- event_registrations may not exist after migration 014
DO $$ BEGIN
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'event_registrations') THEN
        ALTER TABLE event_registrations ALTER COLUMN user_id TYPE TEXT USING user_id::text;
    END IF;
END $$;
ALTER TABLE event_subscriptions ALTER COLUMN user_id TYPE TEXT USING user_id::text;
ALTER TABLE telegram_bindings ALTER COLUMN user_id TYPE TEXT USING user_id::text;
ALTER TABLE telegram_binding_tokens ALTER COLUMN user_id TYPE TEXT USING user_id::text;
ALTER TABLE telegram_delivery ALTER COLUMN user_id TYPE TEXT USING user_id::text;
ALTER TABLE role_requests ALTER COLUMN user_id TYPE TEXT USING user_id::text;
ALTER TABLE role_requests ALTER COLUMN reviewed_by TYPE TEXT USING user_id::text;
ALTER TABLE discovery_actions ALTER COLUMN user_id TYPE TEXT USING user_id::text;

-- Восстанавливаем FK (only if table exists)
DO $$ BEGIN
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'event_registrations') THEN
        ALTER TABLE event_registrations 
            ADD CONSTRAINT event_registrations_user_id_fkey 
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
    END IF;
END $$;

ALTER TABLE event_subscriptions 
    ADD CONSTRAINT event_subscriptions_user_id_fkey 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE role_requests 
    ADD CONSTRAINT role_requests_user_id_fkey 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE role_requests 
    ADD CONSTRAINT role_requests_reviewed_by_fkey 
    FOREIGN KEY (reviewed_by) REFERENCES users(id) ON DELETE SET NULL;
`

// Migration 014: Drop event_registrations table (consolidation to event_subscriptions)
// The event_registrations table is no longer used after consolidation.
// All participant data now comes from event_subscriptions + telegram_bindings/users JOINs.

const dropEventRegistrationsConsolidation = `
-- Drop event_registrations table (consolidated to event_subscriptions)
-- Remove FK references from other migrations first
ALTER TABLE event_registrations DROP CONSTRAINT IF EXISTS event_registrations_event_id_fkey;
ALTER TABLE event_registrations DROP CONSTRAINT IF EXISTS event_registrations_user_id_fkey;

DROP INDEX IF EXISTS idx_event_registrations_event_id;
DROP INDEX IF EXISTS idx_event_registrations_user_id;
DROP INDEX IF EXISTS idx_event_registrations_status;
DROP INDEX IF EXISTS idx_event_registrations_event_status;

DROP TABLE IF EXISTS event_registrations CASCADE;
`

const recreateEventRegistrations = `
-- Recreate event_registrations table (rollback migration)
CREATE TABLE IF NOT EXISTS event_registrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    public_name VARCHAR(255) NOT NULL,
    avatar_url TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'confirmed',
    registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(event_id, user_id)
);

CREATE INDEX idx_event_registrations_event_id ON event_registrations(event_id);
CREATE INDEX idx_event_registrations_user_id ON event_registrations(user_id);
CREATE INDEX idx_event_registrations_status ON event_registrations(status);
CREATE INDEX idx_event_registrations_event_status ON event_registrations(event_id, status);
`

// Migration 015: Add public profile fields to users table
// Adds fields for public profile display without exposing sensitive data.

const addPublicProfileFields = `
-- Add public profile fields to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS username VARCHAR(64) UNIQUE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS display_name VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS bio TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS city VARCHAR(100);
ALTER TABLE users ADD COLUMN IF NOT EXISTS country CHAR(2);
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_verified BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS public_social JSONB DEFAULT '{}'::jsonb;
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_profile_public BOOLEAN DEFAULT TRUE;

-- Index for username lookup (already UNIQUE, but add explicit index for clarity)
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username) WHERE username IS NOT NULL;

-- Index for public profile filtering
CREATE INDEX IF NOT EXISTS idx_users_is_profile_public ON users(is_profile_public);

-- Index for verified users (for discovery/search features)
CREATE INDEX IF NOT EXISTS idx_users_is_verified ON users(is_verified);

COMMENT ON COLUMN users.username IS 'Unique public username/handle';
COMMENT ON COLUMN users.display_name IS 'Public display name shown in profile';
COMMENT ON COLUMN users.avatar_url IS 'URL to user avatar image';
COMMENT ON COLUMN users.bio IS 'Short public bio/description';
COMMENT ON COLUMN users.city IS 'City name for public location display';
COMMENT ON COLUMN users.country IS 'ISO 3166-1 alpha-2 country code';
COMMENT ON COLUMN users.is_verified IS 'Whether the user profile is verified';
COMMENT ON COLUMN users.public_social IS 'JSONB map of public social links (e.g. {"twitter": "https://..."})';
COMMENT ON COLUMN users.is_profile_public IS 'Whether the profile is publicly visible';
`

const dropPublicProfileFields = `
-- Remove public profile fields from users table
DROP INDEX IF EXISTS idx_users_username;
DROP INDEX IF EXISTS idx_users_is_profile_public;
DROP INDEX IF EXISTS idx_users_is_verified;

ALTER TABLE users DROP COLUMN IF EXISTS username;
ALTER TABLE users DROP COLUMN IF EXISTS display_name;
ALTER TABLE users DROP COLUMN IF EXISTS avatar_url;
ALTER TABLE users DROP COLUMN IF EXISTS bio;
ALTER TABLE users DROP COLUMN IF EXISTS city;
ALTER TABLE users DROP COLUMN IF EXISTS country;
ALTER TABLE users DROP COLUMN IF EXISTS is_verified;
ALTER TABLE users DROP COLUMN IF EXISTS public_social;
ALTER TABLE users DROP COLUMN IF EXISTS is_profile_public;
`

// Migration 016: Create event_reminder_log table
// Tracks sent reminders to prevent duplicate notifications.

const createEventReminderLog = `
-- Create event_reminder_log table to track sent reminders
CREATE TABLE IF NOT EXISTS event_reminder_log (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    event_id UUID NOT NULL,
    reminder_type VARCHAR(20) NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, event_id, reminder_type)
);

-- Index for efficient lookups when checking if reminder was sent
CREATE INDEX IF NOT EXISTS idx_event_reminder_log_lookup 
    ON event_reminder_log(user_id, event_id, reminder_type);

-- Index for cleanup of old records
CREATE INDEX IF NOT EXISTS idx_event_reminder_log_sent_at 
    ON event_reminder_log(sent_at);

-- Foreign keys (soft - allow orphans for audit purposes)
-- No CASCADE DELETE - we want to keep history even if event/user is deleted

COMMENT ON TABLE event_reminder_log IS 'Tracks sent event reminders to prevent duplicate notifications';
COMMENT ON COLUMN event_reminder_log.reminder_type IS 'Type of reminder: 24h, 1h, 15min, etc.';
`

const dropEventReminderLog = `
DROP INDEX IF EXISTS idx_event_reminder_log_lookup;
DROP INDEX IF EXISTS idx_event_reminder_log_sent_at;
DROP TABLE IF EXISTS event_reminder_log;
`

// Migration 017: Create feedback table for user feedback/bug reports

const createFeedbackTable = `
-- Create feedback table for user feedback and bug reports
CREATE TABLE IF NOT EXISTS feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category VARCHAR(50) NOT NULL DEFAULT 'other',
    message TEXT NOT NULL,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    user_info JSONB NOT NULL DEFAULT '{}'::jsonb,
    environment JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for filtering by read status
CREATE INDEX IF NOT EXISTS idx_feedback_is_read ON feedback(is_read);

-- Index for sorting by creation time (newest first)
CREATE INDEX IF NOT EXISTS idx_feedback_created_at ON feedback(created_at DESC);

-- Index for filtering by category
CREATE INDEX IF NOT EXISTS idx_feedback_category ON feedback(category);

-- Index for user lookup
CREATE INDEX IF NOT EXISTS idx_feedback_user_id ON feedback(user_id) WHERE user_id IS NOT NULL;

COMMENT ON TABLE feedback IS 'User feedback and bug reports from the application';
COMMENT ON COLUMN feedback.category IS 'Type of feedback: bug, feature, inconvenience, other';
COMMENT ON COLUMN feedback.message IS 'The feedback message from user';
COMMENT ON COLUMN feedback.user_info IS 'JSONB with user info from form (userId, email, firstName, lastName)';
COMMENT ON COLUMN feedback.environment IS 'JSONB with environment info (userAgent, screenSize, url, pwa, os)';
COMMENT ON COLUMN feedback.is_read IS 'Whether the feedback has been read by admin';
`

const dropFeedbackTable = `
DROP INDEX IF EXISTS idx_feedback_is_read;
DROP INDEX IF EXISTS idx_feedback_created_at;
DROP INDEX IF EXISTS idx_feedback_category;
DROP INDEX IF EXISTS idx_feedback_user_id;
DROP TABLE IF EXISTS feedback;
`
