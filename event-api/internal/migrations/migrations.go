package migrations

import (
	"database/sql"
	"fmt"

	"go.uber.org/zap"
)

// Migrator управляет миграциями БД
type Migrator struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewMigrator создает новый Migrator
func NewMigrator(db *sql.DB, logger *zap.Logger) *Migrator {
	return &Migrator{
		db:     db,
		logger: logger,
	}
}

// RunMigrations запускает все миграции
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
	return nil
}

// migration представляет одну миграцию
type migration struct {
	name string
	up   string
	down string
}

// createMigrationsTable создает таблицу для отслеживания миграций
func (m *Migrator) createMigrationsTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) UNIQUE NOT NULL,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)
	`
	_, err := m.db.Exec(query)
	return err
}

// runMigration запускает одну миграцию
func (m *Migrator) runMigration(mig migration) error {
	// Проверяем, была ли уже применена эта миграция
	var exists bool
	err := m.db.QueryRow(
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

	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Выполняем SQL
	if _, err := tx.Exec(mig.up); err != nil {
		return fmt.Errorf("failed to apply migration %s: %w", mig.name, err)
	}

	// Записываем в schema_migrations
	if _, err := tx.Exec(
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

// createUsersTable создает таблицу users
const createUsersTable = `
CREATE TABLE IF NOT EXISTS users (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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

// dropUsersTable удаляет таблицу users
const dropUsersTable = `
DROP TABLE IF EXISTS users CASCADE;
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

		tx, err := m.db.Begin()
		if err != nil {
			return err
		}

		if _, err := tx.Exec(mig.down); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to rollback migration %s: %w", mig.name, err)
		}

		if _, err := tx.Exec(
			"DELETE FROM schema_migrations WHERE name = $1",
			mig.name,
		); err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		m.logger.Info("✅ Миграция отката успешна", zap.String("name", mig.name))
	}

	m.logger.Info("✅ Все миграции отката успешно выполнены")
	return nil
}
