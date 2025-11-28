// Package main contains the entry point for the event-api server.
// cmd/server/main.go
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"

	"event-api/internal/config"
	"event-api/internal/logger"

	"go.uber.org/zap"

	_ "event-api/docs" // This is required for Swagger
)

// TODO: create cron (or smth like tasks manager) to manage events notification
// TODO: configure NGINX to navigate telegramm webhooks

// @title Event API
// @version 1.0
// @description API для управления событиями и аутентификации пользователей
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host api.tuserduser.online
// @schemes https http
// @host localhost:8080
// @schemes http
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func run(versionInfo VersionInfo) error {
	logger.Init()
	defer syncLogger()

	logVersionInfo(versionInfo)

	// Create app context for background workers
	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	// Load configuration
	cfg := config.Load()
	logConfig(cfg)

	// Initialize application container with all services and handlers
	container, err := NewAppContainer(appCtx, cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize application container: %w", err)
	}
	defer func() {
		if err := container.Close(); err != nil {
			logger.Log.Error("failed to close container", zap.Error(err))
		}
	}()

	// Start discovery-related background workers
	container.StartDiscoveryWorkers(appCtx)

	// Build HTTP router
	httpRouter := container.BuildHTTPRouter(versionInfo)

	// Create HTTP server
	container.CreateHTTPServer(httpRouter)

	// Setup graceful shutdown
	shutdownSignal := WaitForShutdownSignal()
	PrintStartupMessage(cfg.Port, cfg.Env, len(cfg.CORSAllowedOrigins), cfg.ShutdownTimeout)

	// Start server in goroutine
	go func() {
		if err := container.HTTPServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Error("server failed", zap.Error(err))
		}
	}()

	// Wait for shutdown signal
	<-shutdownSignal

	// Execute graceful shutdown
	shutdownConfig := GracefulShutdownConfig{TimeoutSeconds: cfg.ShutdownTimeout}
	return ExecuteGracefulShutdown(appCtx, appCancel, container.HTTPServer, container, shutdownConfig)
}

func main() {
	showVersion := parseVersionFlag()
	versionInfo := newVersionInfo()

	if showVersion {
		fmt.Println(versionInfo.String())
		return
	}

	if err := run(versionInfo); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseVersionFlag() bool {
	showVersion := flag.Bool("version", false, "Print version information and exit")
	flag.BoolVar(showVersion, "v", false, "Print version information and exit")
	flag.Parse()
	return *showVersion
}

func logVersionInfo(info VersionInfo) {
	logger.Log.Info("backend version",
		zap.String("version", info.Version),
		zap.String("commit", info.Commit),
		zap.String("build_time", info.BuildTime),
		zap.String("go_version", info.GoVersion),
	)
}

func logConfig(cfg *config.Config) {
	logger.Log.Info("configuration loaded",
		zap.String("env", cfg.Env),
		zap.String("port", cfg.Port),
		zap.Int("shutdown_timeout", cfg.ShutdownTimeout),
		zap.Int("cors_origins", len(cfg.CORSAllowedOrigins)),
		zap.String("db_host", cfg.DBHost),
		zap.String("db_name", cfg.DBName),
		zap.Int("db_max_conn", cfg.DBMaxConn),
		zap.Int("db_min_conn", cfg.DBMinConn),
		zap.String("redis_host", cfg.RedisHost),
		zap.Int("redis_db", cfg.RedisDB),
		zap.String("sms_provider", cfg.SMSProvider),
		zap.String("sms_from", cfg.SMSFrom),
		zap.String("email_provider", cfg.EmailProvider),
		zap.String("email_from", cfg.EmailFrom),
		zap.String("smtp_host", cfg.SMTPHost),
		zap.Int("smtp_port", cfg.SMTPPort),
		zap.Bool("telegram_enabled", cfg.TelegramEnabled),
		zap.String("telegram_webhook_alias", cfg.TelegramWebhookAlias),
		zap.Int("telegram_rate_limit_per_sec", cfg.TelegramRateLimitPerSec),
		zap.Int("telegram_max_attempts", cfg.TelegramMaxAttempts),
	)
}

func syncLogger() {
	if err := logger.Sync(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to sync logger: %v\n", err)
	}
}
