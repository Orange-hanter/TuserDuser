// cmd/server/main.go
package main

import (
	"context"
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

func main() {
	logger.Init()
	defer func() {
		if err := logger.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to sync logger: %v\n", err)
		}
	}()

	cfg := config.Load()

	// Инициализируем подключение к БД
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
		fmt.Println(logger.FormatError(
			"Failed to Connect to Database",
			err,
			"Host: "+cfg.DBHost,
			"Port: "+cfg.DBPort,
			"Database: "+cfg.DBName,
		))
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to close database: %v\n", err)
		}
	}()

	// Запускаем миграции
	migrator := migrations.NewMigrator(db.DB, logger.Log)
	if err := migrator.RunMigrations(); err != nil {
		fmt.Println(logger.FormatError(
			"Migration Execution Failed",
			err,
			"Check your database connection",
			"Ensure all migration files are valid",
		))
		os.Exit(1)
	}

	fmt.Println(logger.FormatSuccess(
		"Database Initialized Successfully",
		"Host: "+cfg.DBHost,
		"Database: "+cfg.DBName,
		"Migrations: Applied",
	))

	// Инициализируем подключение к Redis
	redisConfig := &redisClient.Config{
		Host:     cfg.RedisHost,
		Port:     cfg.RedisPort,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}

	redis, err := redisClient.NewClient(redisConfig, logger.Log)
	if err != nil {
		fmt.Println(logger.FormatError(
			"Failed to Connect to Redis",
			err,
			"Host: "+cfg.RedisHost,
			"Port: "+cfg.RedisPort,
		))
		os.Exit(1)
	}
	defer func() {
		if err := redis.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to close redis: %v\n", err)
		}
	}()

	// Инициализируем SMS сервис
	smsConfig := &sms.Config{
		Provider: cfg.SMSProvider,
		APIKey:   cfg.SMSAPIKey,
		APIToken: cfg.SMSAPIToken,
		From:     cfg.SMSFrom,
	}

	smsService, err := sms.NewService(smsConfig, logger.Log)
	if err != nil {
		fmt.Println(logger.FormatError(
			"Failed to Initialize SMS Service",
			err,
			"Provider: "+cfg.SMSProvider,
		))
		os.Exit(1)
	}

	// Инициализируем Email сервис
	emailConfig := &email.Config{
		Provider:     cfg.EmailProvider,
		APIKey:       cfg.EmailAPIKey,
		SMTPHost:     cfg.SMTPHost,
		SMTPPort:     cfg.SMTPPort,
		SMTPUsername: cfg.SMTPUsername,
		SMTPPassword: cfg.SMTPPassword,
		From:         cfg.EmailFrom,
		FromName:     cfg.EmailFromName,
	}

	emailService, err := email.NewService(emailConfig, logger.Log)
	if err != nil {
		fmt.Println(logger.FormatError(
			"Failed to Initialize Email Service",
			err,
			"Provider: "+cfg.EmailProvider,
		))
		os.Exit(1)
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

	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.SecurityHeaders)

	// Health check (без версии)
	r.Get("/health", handlers.HealthCheck)

	// API v1 routes
	r.Route("/v1", func(r chi.Router) {
		// Auth routes (public)
		r.Post("/api/auth/register", authHandler.Register)
		r.Post("/api/auth/verify", authHandler.Verify)
		r.Post("/api/auth/login", authHandler.Login)

		// Auth routes (protected)
		r.Post("/api/auth/logout", authHandler.Logout)
		r.With(middleware.AuthMiddleware(authService)).Get("/api/auth/me", authHandler.GetMe)

		// Events routes (public)
		r.Get("/api/events", eventHandler.GetAllEvents)
		r.Get("/api/events/{id}", eventHandler.GetEventByID)

		// Events routes (protected)
		r.With(middleware.AuthMiddleware(authService)).Post("/api/events", eventHandler.CreateEvent)
		r.With(middleware.AuthMiddleware(authService)).Delete("/api/events/{id}", eventHandler.DeleteEvent)
	})

	// Swagger routes
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   cfg.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Requested-With"},
		ExposedHeaders:   []string{"Content-Length", "X-JSON-Response"},
		AllowCredentials: true,
		MaxAge:           3600, // 1 час
	})

	handler := c.Handler(r)

	// Создаем HTTP сервер с явными настройками
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: handler,
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
			fmt.Println(logger.FormatError(
				"Server Launch Failed",
				err,
				"Port: "+cfg.Port,
				"Environment: "+cfg.Env,
			))
			os.Exit(1)
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
		fmt.Println(logger.FormatError(
			"Server Shutdown Failed",
			err,
			"Forced shutdown may cause data loss",
		))
		os.Exit(1)
	}

	fmt.Println(logger.FormatSuccess(
		"Server Shutdown Complete",
		"All connections closed gracefully",
		"Resources cleaned up",
	))

	// Синхронизация логгера перед выходом
	if err := logger.Sync(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to sync logger: %v\n", err)
	}
}
