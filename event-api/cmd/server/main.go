package main

import (
	"net/http"
	"os"

	"event-api/internal/config"
	"event-api/internal/handlers"
	"event-api/internal/logger"
	"event-api/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/rs/cors"
)

func main() {
	logger.Init()
	defer logger.Sync()

	cfg := config.Load()

	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.SecurityHeaders)

	// Routes
	r.Get("/health", handlers.HealthCheck)

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
