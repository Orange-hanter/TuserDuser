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
	id TEXT PRIMARY KEY,
	email VARCHAR(255) UNIQUE NOT NULL,
	phone VARCHAR(20) NOT NULL,
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
	user_id TEXT NOT NULL,
	expires_at TIMESTAMPTZ NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_telegram_binding_tokens_expires_at
ON telegram_binding_tokens (expires_at);

CREATE TABLE IF NOT EXISTS telegram_bindings (
	user_id TEXT PRIMARY KEY,
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
	user_id TEXT NOT NULL,
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
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (event_id) REFERENCES events(id)
);

CREATE INDEX IF NOT EXISTS idx_event_subscriptions_user_id ON event_subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_event_subscriptions_event_id ON event_subscriptions(event_id);
`

// dropEventSubscriptions - SQL для удаления таблицы подписок на события.
const dropEventSubscriptions = `
DROP TABLE IF EXISTS event_subscriptions;
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

	// Generate textual ID
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return fmt.Errorf("failed to generate admin id: %w", err)
	}
	adminID := hex.EncodeToString(idBytes)

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
