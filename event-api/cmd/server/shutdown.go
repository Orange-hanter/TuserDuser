package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"event-api/internal/logger"

	"go.uber.org/zap"
)

// GracefulShutdownConfig holds configuration for graceful shutdown.
type GracefulShutdownConfig struct {
	TimeoutSeconds int
}

// WaitForShutdownSignal blocks until a shutdown signal (SIGINT/SIGTERM) is received.
func WaitForShutdownSignal() <-chan os.Signal {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	return quit
}

// ExecuteGracefulShutdown performs graceful shutdown of the application.
// It:
// 1. Cancels the app context to stop background workers.
// 2. Stops the HTTP server with timeout.
// 3. Closes all resources in the container.
func ExecuteGracefulShutdown(
	ctx context.Context,
	appCancel context.CancelFunc,
	srv *http.Server,
	container *AppContainer,
	config GracefulShutdownConfig,
) error {
	fmt.Println(logger.FormatInfo(
		"Server Shutdown Initiated",
		"Graceful shutdown started",
		"Timeout: "+fmt.Sprintf("%d seconds", config.TimeoutSeconds),
	))

	// Cancel app context to stop background workers (discovery updater, etc.)
	appCancel()

	// Create context with timeout for shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(config.TimeoutSeconds)*time.Second)
	defer cancel()

	// Close container resources
	if err := container.Close(); err != nil {
		logger.Log.Error("failed to close container resources", zap.Error(err))
	}

	// Gracefully shutdown HTTP server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Log.Error("server shutdown failed", zap.Error(err))
		return err
	}

	fmt.Println(logger.FormatSuccess(
		"Server Shutdown Complete",
		"All connections closed gracefully",
		"Resources cleaned up",
	))

	return nil
}

// PrintStartupMessage prints server startup information.
func PrintStartupMessage(port, env string, corsOriginCount, shutdownTimeout int) {
	fmt.Println(logger.FormatSuccess(
		"Server Started Successfully",
		"Port: "+port,
		"Environment: "+env,
		"CORS: "+fmt.Sprintf("%d origins", corsOriginCount),
		"Shutdown Timeout: "+fmt.Sprintf("%d seconds", shutdownTimeout),
	))
}
