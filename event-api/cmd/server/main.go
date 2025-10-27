package main

import (
	"net/http"
	"os"

	"event-api/internal/config"
	"event-api/internal/handlers"
	"event-api/internal/logger"
	"event-api/internal/middleware"
	"event-api/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/rs/cors"
	"go.uber.org/zap"
)

func main() {
	logger.Init()
	defer logger.Sync()

	cfg := config.Load()

	// Инициализируем сервисы
	authService := service.NewAuthService(cfg)

	// Инициализируем handlers
	authHandler := handlers.NewAuthHandler(authService)

	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.SecurityHeaders)

	// Public routes
	r.Get("/health", handlers.HealthCheck)

	// Auth routes (public)
	r.Post("/api/auth/register", authHandler.Register)
	r.Post("/api/auth/verify", authHandler.Verify)
	r.Post("/api/auth/login", authHandler.Login)

	// Auth routes (protected)
	r.Post("/api/auth/logout", authHandler.Logout)
	r.With(middleware.AuthMiddleware(authService)).Get("/api/auth/me", authHandler.GetMe)

	// CORS
	c := cors.New(cors.Options{
		AllowedOrigins: cfg.CORSAllowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(r)

	port := ":" + cfg.Port
	logger.Log.Info("Сервер запущен", zap.String("port", port), zap.String("env", cfg.Env))
	if err := http.ListenAndServe(port, handler); err != nil && err != http.ErrServerClosed {
		logger.Log.Fatal("Не удалось запустить сервер", zap.Error(err))
		os.Exit(1)
	}
}
