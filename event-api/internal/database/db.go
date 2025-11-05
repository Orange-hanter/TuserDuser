package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
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

	// Проверка подключения
	ctx := make(chan error, 1)
	go func() {
		if err := db.Ping(); err != nil {
			ctx <- fmt.Errorf("failed to ping database: %w", err)
		}
		ctx <- nil
	}()

	// Ждем максимум 5 секунд
	select {
	case err := <-ctx:
		if err != nil {
			logger.Error("Ошибка при проверке подключения", zap.Error(err))
			return nil, err
		}
	case <-time.After(5 * time.Second):
		logger.Error("Timeout при подключении к БД")
		return nil, fmt.Errorf("database connection timeout")
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
	return d.DB.Ping()
}

// Query выполняет SELECT запрос.
func (d *Database) Query(query string, args ...interface{}) (*sql.Rows, error) {
	d.Logger.Debug("Выполняем запрос", zap.String("query", query))
	return d.DB.Query(query, args...)
}

// QueryRow выполняет SELECT запрос, возвращающий одну строку.
func (d *Database) QueryRow(query string, args ...interface{}) *sql.Row {
	d.Logger.Debug("Выполняем QueryRow", zap.String("query", query))
	return d.DB.QueryRow(query, args...)
}

// Exec выполняет INSERT/UPDATE/DELETE запрос.
func (d *Database) Exec(query string, args ...interface{}) (sql.Result, error) {
	d.Logger.Debug("Выполняем Exec", zap.String("query", query))
	return d.DB.Exec(query, args...)
}

// BeginTx начинает новую транзакцию.
func (d *Database) BeginTx() (*sql.Tx, error) {
	d.Logger.Debug("Начинаем транзакцию")
	return d.DB.Begin()
}
