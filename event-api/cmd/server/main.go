// cmd/server/main.go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"event-api/internal/config"
	"event-api/internal/database"
	"event-api/internal/email"
	"event-api/internal/handlers"
	"event-api/internal/logger"
	"event-api/internal/middleware"
	"event-api/internal/migrations"
	redisClient "event-api/internal/redis"
	"event-api/internal/service"
	"event-api/internal/sms"
	"event-api/internal/worker"

	"github.com/go-chi/chi/v5"
	"github.com/rs/cors"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"

	_ "event-api/docs" // This is required for Swagger
)

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

	cfg := config.Load()
	logConfig(cfg)

	// Инициализируем подключение к БД
	db, err := initDatabase(cfg)
	if err != nil {
		return err
	}
	defer closeDatabase(db)

	// Запускаем миграции
	migrator := migrations.NewMigrator(db.DB, logger.Log)
	if err := migrator.RunMigrations(); err != nil {
		return fmt.Errorf("migration execution failed: %w", err)
	}

	fmt.Println(logger.FormatSuccess(
		"Database Initialized Successfully",
		"Host: "+cfg.DBHost,
		"Database: "+cfg.DBName,
		"Migrations: Applied",
	))

	// Инициализируем подключение к Redis
	redis, err := initRedis(cfg)
	if err != nil {
		return err
	}
	defer closeRedis(redis)

	// Инициализируем SMS сервис
	smsService, err := initSMSService(cfg)
	if err != nil {
		return err
	}

	// Инициализируем Email сервис
	emailService, err := initEmailService(cfg)
	if err != nil {
		return err
	}

	logger.Log.Info("✅ Email service initialized",
		zap.String("provider", cfg.EmailProvider),
		zap.String("from", cfg.EmailFrom),
	)

	// Инициализируем worker pool
	workerPool := worker.NewPool(5, 100, logger.Log)
	workerPool.Start()

	// Инициализируем сервисы
	authService := service.NewAuthService(cfg, db.DB, redis, smsService, emailService, workerPool, logger.Log)
	eventService := service.NewEventService(db.DB, logger.Log)

	// Инициализируем handlers
	authHandler := handlers.NewAuthHandler(authService)
	eventHandler := handlers.NewEventHandler(eventService)

	handler := buildHTTPHandler(cfg, authHandler, eventHandler, authService, versionInfo)

	// Создаем HTTP сервер с явными настройками
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Канал для сигналов graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println(logger.FormatSuccess(
		"Server Started Successfully",
		"Port: "+cfg.Port,
		"Environment: "+cfg.Env,
		"CORS: "+fmt.Sprintf("%d origins", len(cfg.CORSAllowedOrigins)),
		"Shutdown Timeout: "+fmt.Sprintf("%d seconds", cfg.ShutdownTimeout),
	))

	// Запуск сервера в горутине
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Error("server failed", zap.Error(err))
		}
	}()

	// Ожидание сигнала для graceful shutdown
	<-quit
	fmt.Println(logger.FormatInfo(
		"Server Shutdown Initiated",
		"Graceful shutdown started",
		"Timeout: "+fmt.Sprintf("%d seconds", cfg.ShutdownTimeout),
	))

	// Создаем контекст с таймаутом для shutdown
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.ShutdownTimeout)*time.Second)
	defer cancel()

	// Останавливаем worker pool
	workerPool.Shutdown()

	// Graceful shutdown сервера
	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Error("Server Shutdown Failed", zap.Error(err))
		return err
	}

	fmt.Println(logger.FormatSuccess(
		"Server Shutdown Complete",
		"All connections closed gracefully",
		"Resources cleaned up",
	))

	return nil
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

func versionHandler(info VersionInfo) http.HandlerFunc {
	response := info
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.Log.Error("failed to write version response", zap.Error(err))
			http.Error(w, "failed to render version info", http.StatusInternalServerError)
			return
		}
	}
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
	)
}

func syncLogger() {
	if err := logger.Sync(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to sync logger: %v\n", err)
	}
}

func initDatabase(cfg *config.Config) (*database.Database, error) {
	config := &database.Config{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		DBName:   cfg.DBName,
		SSLMode:  cfg.DBSSLMode,
		MaxConn:  cfg.DBMaxConn,
		MinConn:  cfg.DBMinConn,
	}

	db, err := database.NewDatabase(config, logger.Log)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database (host=%s, port=%s, db=%s): %w",
			cfg.DBHost, cfg.DBPort, cfg.DBName, err)
	}
	return db, nil
}

func closeDatabase(db *database.Database) {
	if db == nil {
		return
	}
	if err := db.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to close database: %v\n", err)
	}
}

func initRedis(cfg *config.Config) (*redisClient.Client, error) {
	config := &redisClient.Config{
		Host:     cfg.RedisHost,
		Port:     cfg.RedisPort,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}

	client, err := redisClient.NewClient(config, logger.Log)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis (host=%s, port=%s): %w",
			cfg.RedisHost, cfg.RedisPort, err)
	}
	return client, nil
}

func closeRedis(client *redisClient.Client) {
	if client == nil {
		return
	}
	if err := client.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to close redis: %v\n", err)
	}
}

func initSMSService(cfg *config.Config) (*sms.Service, error) {
	config := &sms.Config{
		Provider: cfg.SMSProvider,
		APIKey:   cfg.SMSAPIKey,
		APIToken: cfg.SMSAPIToken,
		From:     cfg.SMSFrom,
	}

	service, err := sms.NewService(config, logger.Log)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SMS service (provider=%s): %w", cfg.SMSProvider, err)
	}
	return service, nil
}

func initEmailService(cfg *config.Config) (*email.Service, error) {
	config := &email.Config{
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

	service, err := email.NewService(config, logger.Log)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize email service (provider=%s): %w", cfg.EmailProvider, err)
	}
	return service, nil
}

func buildHTTPHandler(cfg *config.Config, authHandler *handlers.AuthHandler, eventHandler *handlers.EventHandler, authService *service.AuthService, versionInfo VersionInfo) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.SecurityHeaders)
	r.Get("/health", handlers.HealthCheck)
	r.Get("/version", versionHandler(versionInfo))

	r.Route("/v1", func(r chi.Router) {
		// Public auth endpoints
		r.Post("/api/auth/register", authHandler.Register)
		r.Post("/api/auth/verify", authHandler.Verify)
		r.Post("/api/auth/login", authHandler.Login)
		r.Post("/api/auth/logout", authHandler.Logout)

		// Public event endpoints (read-only, no auth required)
		r.Get("/api/events", eventHandler.GetApprovedEvents)
		r.Get("/api/events/approved", eventHandler.GetApprovedEvents)
		r.Get("/api/events/{id}", eventHandler.GetEventByID)

		// Authenticated user endpoints
		authenticated := r.With(middleware.AuthMiddleware(authService))
		authenticated.Get("/api/auth/me", authHandler.GetMe)

		// Creator/Admin: Create events (requires creator or admin role)
		creatorOrAdmin := authenticated.With(middleware.RequireCreatorOrAdmin)
		creatorOrAdmin.Post("/api/events", eventHandler.CreateEvent)
		creatorOrAdmin.Delete("/api/events/{id}", eventHandler.DeleteEvent)

		// Admin only: Event moderation endpoints
		adminOnly := authenticated.With(middleware.RequireAdmin)
		adminOnly.Get("/api/events/pending", eventHandler.GetPendingEvents)
		adminOnly.Post("/api/events/{id}/review", eventHandler.ReviewPendingEvent)

		// Admin only: User management
		adminOnly.Get("/api/admin/users", authHandler.GetAllUsers)
		adminOnly.Put("/api/admin/users/role", authHandler.UpdateUserRole)
	})

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	c := cors.New(cors.Options{
		AllowedOrigins:   cfg.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Requested-With"},
		ExposedHeaders:   []string{"Content-Length", "X-JSON-Response"},
		AllowCredentials: true,
		MaxAge:           3600,
	})

	return c.Handler(r)
}
