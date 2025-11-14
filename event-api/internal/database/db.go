// Package database предоставляет функциональность для взаимодействия с базой данных PostgreSQL.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // register postgres driver
	"go.uber.org/zap"
)

// Database представляет подключение к БД.
type Database struct {
	DB     *sql.DB
	Logger *zap.Logger
}

// Config содержит параметры подключения к БД.
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
	MaxConn  int
	MinConn  int
}

// NewDatabase создает новое подключение к БД.
func NewDatabase(cfg *Config, logger *zap.Logger) (*Database, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.DBName,
		cfg.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		logger.Error("Ошибка при открытии подключения к БД", zap.Error(err))
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Настройка пула подключений
	db.SetMaxOpenConns(cfg.MaxConn)
	db.SetMaxIdleConns(cfg.MinConn)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Проверка подключения с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		logger.Error("Ошибка при проверке подключения", zap.Error(err))
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("✅ Успешно подключились к БД",
		zap.String("host", cfg.Host),
		zap.String("port", cfg.Port),
		zap.String("dbname", cfg.DBName),
	)

	return &Database{
		DB:     db,
		Logger: logger,
	}, nil
}

// Close закрывает подключение к БД.
func (d *Database) Close() error {
	if d.DB != nil {
		d.Logger.Info("Закрываем подключение к БД")
		return d.DB.Close()
	}
	return nil
}

// Health проверяет здоровье подключения к БД.
func (d *Database) Health() error {
	return d.DB.PingContext(context.Background())
}

// Query выполняет SELECT запрос.
func (d *Database) Query(query string, args ...interface{}) (*sql.Rows, error) {
	d.Logger.Debug("Выполняем запрос", zap.String("query", query))
	return d.DB.QueryContext(context.Background(), query, args...)
}

// QueryRow выполняет SELECT запрос, возвращающий одну строку.
func (d *Database) QueryRow(query string, args ...interface{}) *sql.Row {
	d.Logger.Debug("Выполняем QueryRow", zap.String("query", query))
	return d.DB.QueryRowContext(context.Background(), query, args...)
}

// Exec выполняет INSERT/UPDATE/DELETE запрос.
func (d *Database) Exec(query string, args ...interface{}) (sql.Result, error) {
	d.Logger.Debug("Выполняем Exec", zap.String("query", query))
	return d.DB.ExecContext(context.Background(), query, args...)
}

// BeginTx начинает новую транзакцию.
func (d *Database) BeginTx() (*sql.Tx, error) {
	d.Logger.Debug("Начинаем транзакцию")
	return d.DB.BeginTx(context.Background(), nil)
}
