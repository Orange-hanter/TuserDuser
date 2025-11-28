package main

import (
	"fmt"

	"event-api/internal/config"
	"event-api/internal/database"
	"event-api/internal/email"
	"event-api/internal/logger"
	"event-api/internal/migrations"
	redisClient "event-api/internal/redis"
	"event-api/internal/sms"
)

// initDatabase initializes database connection and runs migrations.
func initDatabase(cfg *config.Config) (*database.Database, error) {
	dbConfig := &database.Config{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		DBName:   cfg.DBName,
		SSLMode:  cfg.DBSSLMode,
		MaxConn:  cfg.DBMaxConn,
		MinConn:  cfg.DBMinConn,
	}

	db, err := database.NewDatabase(dbConfig, logger.Log)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database (host=%s, port=%s, db=%s): %w",
			cfg.DBHost, cfg.DBPort, cfg.DBName, err)
	}

	// Run migrations
	migrator := migrations.NewMigrator(db.DB, logger.Log)
	if err := migrator.RunMigrations(); err != nil {
		return nil, fmt.Errorf("migration execution failed: %w", err)
	}

	fmt.Println(logger.FormatSuccess(
		"Database Initialized Successfully",
		"Host: "+cfg.DBHost,
		"Database: "+cfg.DBName,
		"Migrations: Applied",
	))

	return db, nil
}

// initRedis initializes Redis connection.
// Returns nil client if Redis is not available (non-critical failure).
func initRedis(cfg *config.Config) (*redisClient.Client, error) {
	redisConfig := &redisClient.Config{
		Host:     cfg.RedisHost,
		Port:     cfg.RedisPort,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}

	client, err := redisClient.NewClient(redisConfig, logger.Log)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis (host=%s, port=%s): %w",
			cfg.RedisHost, cfg.RedisPort, err)
	}

	return client, nil
}

// initSMSService initializes SMS service.
func initSMSService(cfg *config.Config) (*sms.Service, error) {
	smsConfig := &sms.Config{
		Provider: cfg.SMSProvider,
		APIKey:   cfg.SMSAPIKey,
		APIToken: cfg.SMSAPIToken,
		From:     cfg.SMSFrom,
	}

	service, err := sms.NewService(smsConfig, logger.Log)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SMS service (provider=%s): %w", cfg.SMSProvider, err)
	}

	return service, nil
}

// initEmailService initializes email service.
func initEmailService(cfg *config.Config) (*email.Service, error) {
	emailConfig := &email.Config{
		Provider:     cfg.EmailProvider,
		APIKey:       cfg.EmailAPIKey,
		SMTPHost:     cfg.SMTPHost,
		SMTPPort:     cfg.SMTPPort,
		SMTPUsername: cfg.SMTPUsername,
		SMTPPassword: cfg.SMTPPassword,
		UseSSL:       cfg.SMTPUseSSL,
		From:         cfg.EmailFrom,
		FromName:     cfg.EmailFromName,
	}

	service, err := email.NewService(emailConfig, logger.Log)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize email service (provider=%s): %w", cfg.EmailProvider, err)
	}

	return service, nil
}
