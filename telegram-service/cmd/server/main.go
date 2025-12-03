// Package main is the entry point for telegram-service.
// Telegram-service is a standalone gRPC microservice that handles all
// Telegram Bot API interactions, isolating the core application from Telegram specifics.
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"telegram-service/internal/config"
	"telegram-service/internal/database"
	"telegram-service/internal/grpcserver"
	"telegram-service/internal/metrics"
	"telegram-service/internal/polling"
	"telegram-service/internal/service"
	"telegram-service/internal/telegram"
	"telegram-service/internal/webhook"
)

func main() {
	// Initialize logger
	logger, _ := zap.NewProduction()
	if os.Getenv("ENV") == "development" {
		logger, _ = zap.NewDevelopment()
	}
	defer logger.Sync()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	logger.Info("starting telegram-service",
		zap.String("grpc_port", cfg.GRPCPort),
		zap.String("http_port", cfg.HTTPPort),
		zap.String("env", cfg.Env),
	)

	// Initialize database
	db, err := database.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Run migrations
	if err := database.Migrate(context.Background(), db, logger); err != nil {
		logger.Fatal("failed to run migrations", zap.Error(err))
	}

	// Initialize components
	store := database.NewStore(db)
	telegramClient := telegram.NewHTTPClient(cfg.BotToken, cfg.TelegramAPIBaseURL)
	tokenEncoder := service.NewTokenEncoder(cfg.BindingSecret, cfg.BindingTTLSeconds)

	// Initialize service layer
	telegramService := service.NewTelegramService(
		store,
		telegramClient,
		tokenEncoder,
		cfg.BotUsername,
		logger,
	)

	// Initialize metrics
	metrics.Register()

	// Create gRPC server
	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     15 * time.Second,
			MaxConnectionAge:      30 * time.Second,
			MaxConnectionAgeGrace: 5 * time.Second,
			Time:                  5 * time.Second,
			Timeout:               1 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.UnaryInterceptor(grpcserver.UnaryServerInterceptor(logger)),
	)

	// Register gRPC services
	grpcHandler := grpcserver.NewTelegramServiceServer(telegramService, logger)
	grpcserver.RegisterTelegramServiceServer(grpcServer, grpcHandler)

	// Register health check
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("telegram.v1.TelegramService", grpc_health_v1.HealthCheckResponse_SERVING)

	// Enable reflection for grpcurl/debugging
	if cfg.Env == "development" {
		reflection.Register(grpcServer)
	}

	// Create HTTP server for webhooks and metrics
	httpRouter := chi.NewRouter()
	httpRouter.Use(middleware.RequestID)
	httpRouter.Use(middleware.RealIP)
	httpRouter.Use(middleware.Recoverer)
	httpRouter.Use(middleware.Timeout(30 * time.Second))

	// Poller for long polling mode (will be started if configured)
	var poller *polling.Poller

	// Setup update mode: webhook or polling
	if cfg.UpdateMode == "polling" {
		logger.Info("configuring long polling mode")

		// Create polling update handler
		pollHandler := func(ctx context.Context, update *polling.Update) error {
			if update.Message == nil {
				return nil
			}

			// Convert polling.Update to webhook format and process
			return telegramService.HandleIncomingMessage(
				ctx,
				update.Message.Chat.ID,
				update.Message.From.ID,
				update.Message.From.Username,
				update.Message.From.FirstName,
				update.Message.From.LastName,
				update.Message.Text,
			)
		}

		poller = polling.NewPoller(polling.Config{
			BotToken:       cfg.BotToken,
			BaseURL:        cfg.TelegramAPIBaseURL,
			PollTimeout:    cfg.PollingTimeout,
			RetryDelay:     time.Duration(cfg.PollingRetryDelay) * time.Second,
			MaxRetries:     cfg.MaxRetryAttempts,
			AllowedUpdates: []string{"message"},
		}, pollHandler, logger)
	} else {
		logger.Info("configuring webhook mode")

		// Webhook handler (only in webhook mode)
		webhookHandler := webhook.NewHandler(
			telegramService,
			store,
			cfg.BotToken,
			cfg.WebhookSecret,
			cfg.WebhookAlias,
			logger,
		)
		httpRouter.Post("/webhooks/telegram/{botAlias}", webhookHandler.HandleWebhook)
	}

	// Health and metrics endpoints
	httpRouter.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if poller != nil && poller.IsRunning() {
			w.Write([]byte("OK (polling)"))
		} else {
			w.Write([]byte("OK"))
		}
	})
	httpRouter.Handle("/metrics", promhttp.Handler())

	// Start gRPC server
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		logger.Fatal("failed to listen on gRPC port", zap.Error(err))
	}

	go func() {
		logger.Info("gRPC server starting", zap.String("port", cfg.GRPCPort))
		if err := grpcServer.Serve(grpcListener); err != nil {
			logger.Fatal("gRPC server failed", zap.Error(err))
		}
	}()

	// Start HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler:      httpRouter,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("HTTP server starting", zap.String("port", cfg.HTTPPort))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	// Start poller if in polling mode
	if poller != nil {
		if err := poller.Start(context.Background()); err != nil {
			logger.Fatal("failed to start poller", zap.Error(err))
		}
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down servers...")

	// Stop poller first
	if poller != nil {
		poller.Stop()
	}

	// Set health to not serving
	healthServer.SetServingStatus("telegram.v1.TelegramService", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
	}

	// Gracefully stop gRPC server
	grpcServer.GracefulStop()

	logger.Info("servers stopped")
}
