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
	"event-api/internal/handlers"
	"event-api/internal/logger"
	"event-api/internal/middleware"
	"event-api/internal/migrations"
	"event-api/internal/service"
	"event-api/internal/worker"

	"github.com/go-chi/chi/v5"
	"github.com/rs/cors"
	httpSwagger "github.com/swaggo/http-swagger"

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

// @host localhost:8080
// @BasePath /v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	logger.Init()
	defer logger.Sync()

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
	defer db.Close()

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

	// Инициализируем worker pool
	workerPool := worker.NewPool(5, 100, logger.Log)
	workerPool.Start()

	// Инициализируем сервисы
	authService := service.NewAuthService(cfg, workerPool, logger.Log)

	// Инициализируем handlers
	authHandler := handlers.NewAuthHandler(authService)

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
	})

	// Swagger routes
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:"+cfg.Port+"/swagger/doc.json"),
	))

	// CORS
	c := cors.New(cors.Options{
		AllowedOrigins: cfg.CORSAllowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
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
	logger.Sync()
}
