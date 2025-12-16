package main

import (
	"context"
	"fmt"
	"net/http"

	"event-api/internal/config"
	"event-api/internal/database"
	"event-api/internal/discovery"
	"event-api/internal/email"
	"event-api/internal/handlers"
	"event-api/internal/logger"
	redisClient "event-api/internal/redis"
	"event-api/internal/service"
	"event-api/internal/sms"
	"event-api/internal/telegramclient"
	"event-api/internal/telemetry"
	"event-api/internal/worker"
	"time"

	"go.uber.org/zap"
)

// AppContainer holds all application services, handlers, and dependencies.
// This eliminates parameter passing clutter and provides single source of truth
// for all initialized components.
type AppContainer struct {
	// Configuration
	Config *config.Config

	// Infrastructure
	DB    *database.Database
	Redis *redisClient.Client

	// Worker pool
	WorkerPool *worker.Pool

	// External services
	SMSService   *sms.Service
	EmailService *email.Service

	// Business services
	AuthService            *service.AuthService
	EventService           *service.EventService
	UserService            *service.UserService
	CreatorService         *service.CreatorService
	DiscoveryService       *discovery.Service
	AdminNotifier          *service.AdminNotifier
	EventReminderScheduler *service.EventReminderScheduler
	FeedbackService        *service.FeedbackService

	// Handlers
	AuthHandler             *handlers.AuthHandler
	EventHandler            *handlers.EventHandler
	DiscoveryHandler        *handlers.DiscoveryHandler
	UserHandler             *handlers.UserHandler
	CreatorHandler          *handlers.CreatorHandler
	TelegramHandler         *handlers.TelegramGRPCHandler
	AdminRoleRequestHandler *handlers.AdminRoleRequestHandler
	FeedbackHandler         *handlers.FeedbackHandler

	// HTTP components
	HTTPServer *http.Server
	HTTPRouter http.Handler

	// Telemetry
	Telemetry *telemetry.Provider
}

// NewAppContainer initializes and returns a fully configured AppContainer.
func NewAppContainer(ctx context.Context, cfg *config.Config) (*AppContainer, error) {
	container := &AppContainer{
		Config: cfg,
	}

	// Initialize telemetry (OpenTelemetry)
	if err := container.initTelemetry(ctx); err != nil {
		logger.Log.Warn("Telemetry initialization failed, continuing without it", zap.Error(err))
	}

	// Initialize infrastructure
	if err := container.initInfrastructure(ctx); err != nil {
		return nil, err
	}

	// Initialize services
	if err := container.initServices(ctx); err != nil {
		return nil, err
	}

	// Initialize handlers
	container.initHandlers()

	return container, nil
}

// initInfrastructure initializes database, Redis, and worker pool.
func (c *AppContainer) initInfrastructure(ctx context.Context) error {
	var err error

	// Initialize database
	c.DB, err = initDatabase(c.Config)
	if err != nil {
		return err
	}

	// Initialize Redis (optional)
	c.Redis, err = initRedis(c.Config)
	if err != nil {
		logger.Log.Warn("Redis initialization failed, continuing without it", zap.Error(err))
		c.Redis = nil
	}

	// Initialize worker pool
	c.WorkerPool = worker.NewPool(5, 100, logger.Log)
	c.WorkerPool.Start()

	// Initialize external services
	c.SMSService, err = initSMSService(c.Config)
	if err != nil {
		return err
	}

	c.EmailService, err = initEmailService(c.Config)
	if err != nil {
		return err
	}

	logger.Log.Info("✅ Email service initialized",
		zap.String("provider", c.Config.EmailProvider),
		zap.String("from", c.Config.EmailFrom),
	)

	return nil
}

// initServices initializes all business logic services.
func (c *AppContainer) initServices(ctx context.Context) error {
	logger.Log.Info("🔄 Starting services initialization")

	// Auth service
	c.AuthService = service.NewAuthService(
		c.Config,
		c.DB.DB,
		c.Redis,
		c.SMSService,
		c.EmailService,
		c.WorkerPool,
		logger.Log,
	)

	// Event service
	c.EventService = service.NewEventService(c.DB.DB, logger.Log)

	// Discovery service (with optional Redis caching)
	discoveryService, err := initDiscoveryService(ctx, c.Config, c.DB, c.Redis)
	if err != nil {
		return err
	}
	c.DiscoveryService = discoveryService

	// Bootstrap discovery with approved events.
	// Run asynchronously to avoid blocking server startup on potentially
	// long-running Redis SCAN/refresh operations.
	go func() {
		seedCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := refreshDiscoveryState(seedCtx, c.EventService, c.DiscoveryService); err != nil {
			logger.Log.Warn("failed to seed discovery engine (async)", zap.Error(err))
		} else {
			logger.Log.Info("discovery engine seeded (async)")
		}
	}()

	// User service
	c.UserService = service.NewUserService(c.DB.DB, logger.Log, c.DiscoveryService)

	// Creator service
	c.CreatorService = service.NewCreatorService(c.DB.DB, logger.Log)

	return nil
}

// initHandlers initializes all HTTP handlers.
func (c *AppContainer) initHandlers() {
	logger.Log.Info("🔄 Starting handlers initialization")

	// Initialize Telegram gRPC client (if enabled)
	var telegramClient *telegramclient.Client
	if c.Config.TelegramServiceEnabled || c.Config.TelegramEnabled {
		var err error
		telegramCfg := telegramclient.Config{
			Address: c.Config.TelegramServiceAddress,
			Timeout: time.Duration(c.Config.TelegramServiceTimeout) * time.Millisecond,
		}
		if telegramCfg.Address == "" {
			telegramCfg.Address = "localhost:50051" // default address
		}
		telegramClient, err = telegramclient.NewClient(telegramCfg, logger.Log)
		if err != nil {
			logger.Log.Warn("failed to connect to telegram-service, continuing without it",
				zap.String("address", telegramCfg.Address),
				zap.Error(err))
			telegramClient = nil
		} else {
			logger.Log.Info("connected to telegram-service via gRPC",
				zap.String("address", telegramCfg.Address))
		}
	}

	// Initialize AdminNotifier for sending notifications to admins
	if telegramClient != nil {
		c.AdminNotifier = service.NewAdminNotifier(c.DB.DB, telegramClient, logger.Log)
		c.EventService.SetAdminNotifier(c.AdminNotifier)
		c.UserService.SetAdminNotifier(c.AdminNotifier)
		logger.Log.Info("admin notifier enabled")

		// Initialize EventReminderScheduler for sending event reminders
		c.EventReminderScheduler = service.NewEventReminderScheduler(
			c.DB.DB,
			telegramClient,
			logger.Log,
			service.DefaultReminderConfig(),
		)
		if err := c.EventReminderScheduler.Start(); err != nil {
			logger.Log.Error("failed to start event reminder scheduler", zap.Error(err))
		} else {
			logger.Log.Info("event reminder scheduler started")
		}
	}

	// Auth handler (with optional telegram client for binding status)
	c.AuthHandler = handlers.NewAuthHandler(c.AuthService, telegramClient)

	// Set telegram client on auth service for resend functionality
	if telegramClient != nil {
		c.AuthService.SetTelegramClient(telegramClient)
	}

	// Discovery notifier
	var discoveryNotifier func(context.Context, string)
	if c.Config.DiscoveryUpdatesEnabled && c.Redis != nil {
		channel := c.Config.DiscoveryUpdatesChannel
		discoveryNotifier = func(_ context.Context, eventID string) {
			notifyCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			publishDiscoveryUpdate(notifyCtx, c.Redis, channel, eventID)
		}
	}

	// Event handler
	c.EventHandler = handlers.NewEventHandler(c.EventService, discoveryNotifier)

	// Discovery handler
	c.DiscoveryHandler = handlers.NewDiscoveryHandler(c.DiscoveryService, c.UserService)

	// User handler
	c.UserHandler = handlers.NewUserHandler(c.UserService)

	// Creator handler
	c.CreatorHandler = handlers.NewCreatorHandler(c.CreatorService, logger.Log)

	// Admin role request handler
	c.AdminRoleRequestHandler = handlers.NewAdminRoleRequestHandler(c.UserService, logger.Log)

	// Feedback service and handler
	c.FeedbackService = service.NewFeedbackService(c.DB.DB, logger.Log)
	if c.AdminNotifier != nil {
		c.FeedbackService.SetAdminNotifier(c.AdminNotifier)
	}
	c.FeedbackHandler = handlers.NewFeedbackHandler(c.FeedbackService)

	// Telegram handler (if enabled and gRPC client connected).
	// Handler should be enabled when either legacy Telegram integration is enabled
	// (`TelegramEnabled`) or when the external telegram-service gRPC integration is enabled
	// (`TelegramServiceEnabled`). Previously the code only checked `TelegramEnabled`,
	// which caused the handler to be disabled when only `TelegramServiceEnabled` was set.
	if (c.Config.TelegramEnabled || c.Config.TelegramServiceEnabled) && telegramClient != nil {
		c.TelegramHandler = handlers.NewTelegramGRPCHandler(telegramClient, logger.Log)
		logger.Log.Info("telegram handler enabled via gRPC",
			zap.Bool("telegram_enabled", c.Config.TelegramEnabled),
			zap.Bool("telegram_service_enabled", c.Config.TelegramServiceEnabled),
		)
	} else {
		logger.Log.Info("telegram handler disabled",
			zap.Bool("telegram_enabled", c.Config.TelegramEnabled),
			zap.Bool("telegram_service_enabled", c.Config.TelegramServiceEnabled),
		)
	}

	logger.Log.Info("✅ All handlers initialized")
}

// BuildHTTPRouter builds and returns the HTTP router.
func (c *AppContainer) BuildHTTPRouter(versionInfo VersionInfo) http.Handler {
	return buildHTTPHandler(
		c.Config,
		c.AuthHandler,
		c.EventHandler,
		c.DiscoveryHandler,
		c.UserHandler,
		c.AuthService,
		c.TelegramHandler,
		c.CreatorHandler,
		c.AdminRoleRequestHandler,
		c.FeedbackHandler,
		versionInfo,
	)
}

// CreateHTTPServer creates and configures the HTTP server.
func (c *AppContainer) CreateHTTPServer(handler http.Handler) {
	c.HTTPServer = &http.Server{
		Addr:              ":" + c.Config.Port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

// Close gracefully closes all resources in the container.
func (c *AppContainer) Close() error {
	// Shutdown telemetry first
	if c.Telemetry != nil {
		if err := c.Telemetry.Shutdown(context.Background()); err != nil {
			logger.Log.Error("failed to shutdown telemetry", zap.Error(err))
		}
	}

	// Stop event reminder scheduler
	if c.EventReminderScheduler != nil {
		c.EventReminderScheduler.Stop()
	}

	if c.WorkerPool != nil {
		c.WorkerPool.Shutdown()
	}

	if c.Redis != nil {
		if err := c.Redis.Close(); err != nil {
			logger.Log.Error("failed to close redis", zap.Error(err))
		}
	}

	if c.DB != nil {
		if err := c.DB.Close(); err != nil {
			logger.Log.Error("failed to close database", zap.Error(err))
		}
	}

	return nil
}

// initTelemetry initializes OpenTelemetry tracing.
func (c *AppContainer) initTelemetry(ctx context.Context) error {
	cfg := telemetry.Config{
		ServiceName:    c.Config.OTelServiceName,
		ServiceVersion: "1.0.0",
		Environment:    c.Config.Env,
		OTLPEndpoint:   c.Config.OTelEndpoint,
		Enabled:        c.Config.OTelEnabled,
	}

	provider, err := telemetry.Init(ctx, cfg)
	if err != nil {
		return err
	}

	c.Telemetry = provider
	logger.Log.Info("✅ OpenTelemetry initialized",
		zap.Bool("enabled", cfg.Enabled),
		zap.String("endpoint", cfg.OTLPEndpoint),
		zap.String("service", cfg.ServiceName),
	)

	return nil
}

// initDiscoveryService initializes discovery service with optional Redis support.
func initDiscoveryService(
	ctx context.Context,
	cfg *config.Config,
	db *database.Database,
	redis *redisClient.Client,
) (*discovery.Service, error) {
	var discoveryHistoryRepo discovery.HistoryRepository
	discoveryHistoryRepo = discovery.NewPostgresHistoryRepository(db.DB)

	var discoveryQueueRepo discovery.QueueRepository

	// Use Redis for queues if available
	if redis != nil {
		queueTTL := time.Duration(cfg.DiscoveryQueueTTL) * time.Second
		discoveryQueueRepo = discovery.NewRedisQueueRepository(redis.GetClient(), queueTTL)
		logger.Log.Info("✅ Redis queue repository initialized", zap.Int("ttl_seconds", cfg.DiscoveryQueueTTL))

		// Use Redis for hot history data
		historyTTL := time.Duration(cfg.DiscoveryHistoryTTL) * time.Second
		redisHistoryRepo := discovery.NewRedisHistoryRepository(redis.GetClient(), historyTTL, 100)
		if cfg.DiscoveryHistoryPersistEnabled {
			redisStream := cfg.DiscoveryHistoryPersistStream
			streamGroup := cfg.DiscoveryHistoryPersistStreamGroup
			redisHistoryRepo.EnablePersistenceStream(redisStream, streamGroup)

			// Start consumer to persist Redis Stream history to Postgres.
			consumerName := fmt.Sprintf("event-api-%d", time.Now().UnixNano())
			consumer := discovery.NewHistoryStreamConsumer(redis.GetClient(), db.DB, logger.Log, redisStream, streamGroup, consumerName)
			go func() {
				if err := consumer.Run(ctx); err != nil {
					logger.Log.Warn("discovery history stream consumer stopped", zap.Error(err))
				}
			}()
			logger.Log.Info("✅ Discovery history persistence enabled",
				zap.String("stream", redisStream),
				zap.String("group", streamGroup),
				zap.String("consumer", consumerName),
			)
		}

		discoveryHistoryRepo = redisHistoryRepo
		logger.Log.Info("✅ Redis history repository initialized", zap.Int("ttl_seconds", cfg.DiscoveryHistoryTTL))
	} else {
		// Fallback to in-memory
		discoveryQueueRepo = discovery.NewInMemoryQueueRepository()
		logger.Log.Warn("⚠️  Redis not available, using in-memory queue repository")
	}

	discoveryEngine := discovery.NewEngine(
		discovery.NewInMemoryEventRepository(nil),
		discoveryQueueRepo,
		discoveryHistoryRepo,
		discovery.EngineConfig{},
	)
	discoveryService := discovery.NewService(discoveryEngine)

	// Bootstrap discovery with approved events (if eventService is available)
	// This will be handled by the caller after initialization

	return discoveryService, nil
}
