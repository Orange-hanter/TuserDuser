package main

import (
	"encoding/json"
	"net/http"

	"event-api/internal/config"
	"event-api/internal/handlers"
	"event-api/internal/logger"
	"event-api/internal/middleware"
	"event-api/internal/service"
	"event-api/internal/telemetry"

	"github.com/go-chi/chi/v5"
	"github.com/rs/cors"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
)

// buildHTTPHandler constructs the complete HTTP router with all routes.
func buildHTTPHandler(
	cfg *config.Config,
	authHandler *handlers.AuthHandler,
	eventHandler *handlers.EventHandler,
	discoveryHandler *handlers.DiscoveryHandler,
	userHandler *handlers.UserHandler,
	authService *service.AuthService,
	telegramHandler *handlers.TelegramHandler,
	creatorHandler *handlers.CreatorHandler,
	adminRoleRequestHandler *handlers.AdminRoleRequestHandler,
	versionInfo VersionInfo,
) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.SecurityHeaders)
	r.Use(telemetry.HTTPMiddleware) // OpenTelemetry tracing

	// Health & diagnostics
	r.Get("/health", handlers.HealthCheck)
	r.Get("/version", versionHandler(versionInfo))
	r.Handle("/metrics", http.HandlerFunc(handlers.MetricsEndpoint))

	// Telegram webhooks
	if telegramHandler != nil {
		r.Route("/webhooks/telegram", func(r chi.Router) {
			r.Post("/{botAlias}", telegramHandler.Webhook)
		})
	}

	// API v1 routes
	r.Route("/v1", func(r chi.Router) {
		// Public auth endpoints
		registerPublicAuthRoutes(r, authHandler)

		// Public event endpoints
		registerPublicEventRoutes(r, eventHandler, userHandler)

		// Authenticated user endpoints
		authenticated := r.With(middleware.AuthMiddleware(authService))

		registerAuthenticatedUserRoutes(authenticated, authHandler, userHandler)
		registerDiscoveryRoutes(authenticated, discoveryHandler, telegramHandler)

		// Creator/Admin endpoints
		creatorOrAdmin := authenticated.With(middleware.RequireCreatorOrAdmin)
		registerCreatorRoutes(creatorOrAdmin, eventHandler, creatorHandler)

		// Admin-only endpoints
		adminOnly := authenticated.With(middleware.RequireAdmin)
		registerAdminRoutes(adminOnly, eventHandler, creatorHandler, authHandler, adminRoleRequestHandler)
	})

	// Swagger documentation
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// CORS wrapper
	return wrapWithCORS(r, cfg)
}

// registerPublicAuthRoutes registers auth endpoints that don't require authentication.
func registerPublicAuthRoutes(r chi.Router, authHandler *handlers.AuthHandler) {
	r.Post("/api/auth/check-user", authHandler.CheckUserExists)
	r.Post("/api/auth/register", authHandler.Register)
	r.Post("/api/auth/verify", authHandler.Verify)
	r.Post("/api/auth/resend-code", authHandler.ResendCode)
	r.Post("/api/auth/login", authHandler.Login)
	r.Post("/api/auth/logout", authHandler.Logout)
}

// registerPublicEventRoutes registers event endpoints that are publicly available.
func registerPublicEventRoutes(
	r chi.Router,
	eventHandler *handlers.EventHandler,
	userHandler *handlers.UserHandler,
) {
	r.Get("/api/events", eventHandler.GetApprovedEvents)
	r.Get("/api/events/approved", eventHandler.GetApprovedEvents)
	r.Get("/api/events/{id}", eventHandler.GetEventByID)
	r.Get("/api/events/{event_id}/participants", userHandler.GetEventParticipants)

	// Public user profile endpoint - no authentication required
	r.Get("/api/users/public/{userId}", userHandler.GetPublicProfile)
}

// registerAuthenticatedUserRoutes registers endpoints for authenticated users.
func registerAuthenticatedUserRoutes(
	r chi.Router,
	authHandler *handlers.AuthHandler,
	userHandler *handlers.UserHandler,
) {
	r.Get("/api/auth/me", authHandler.GetMe)

	// User profile and events
	r.Get("/api/users/me", userHandler.GetMe)
	r.Get("/api/users/me/events/upcoming", userHandler.GetUpcomingEvents)
	r.Get("/api/users/me/events/history", userHandler.GetEventHistory)
	r.Post("/api/users/me/events/{event_id}/subscribe", userHandler.Subscribe)

	// Role request endpoints
	r.Post("/api/users/request-role", userHandler.RequestRole)
	r.Get("/api/users/request-role/status", userHandler.GetRoleRequestStatus)
	r.Get("/api/users/request-role/all", userHandler.GetAllRoleRequests)
}

// registerDiscoveryRoutes registers discovery-related endpoints.
func registerDiscoveryRoutes(
	r chi.Router,
	discoveryHandler *handlers.DiscoveryHandler,
	telegramHandler *handlers.TelegramHandler,
) {
	// Telegram notifications (if enabled)
	if telegramHandler != nil {
		r.Route("/api/notifications/telegram", func(r chi.Router) {
			r.Post("/link", telegramHandler.IssueLink)
			r.Get("/status", telegramHandler.BindingStatus)
		})
	}

	// Discovery endpoints
	r.Route("/api/discovery", func(r chi.Router) {
		r.Get("/next", discoveryHandler.Next)
		r.Post("/action", discoveryHandler.Action)
		r.Post("/book", discoveryHandler.Book)
		r.Get("/history", discoveryHandler.History)
	})
}

// registerCreatorRoutes registers endpoints for creators/admins to manage events.
func registerCreatorRoutes(
	r chi.Router,
	eventHandler *handlers.EventHandler,
	creatorHandler *handlers.CreatorHandler,
) {
	// Event creation and deletion
	r.Post("/api/events", eventHandler.CreateEvent)
	r.Delete("/api/events/{id}", eventHandler.DeleteEvent)

	// Creator's own events
	r.Get("/api/creator/events", creatorHandler.GetMyEvents)
	r.Get("/api/creator/events/blocked", creatorHandler.GetBlockedEvents)
	r.Get("/api/creator/events/{eventId}/comments", creatorHandler.GetEventComments)
	r.Post("/api/creator/events/{eventId}/comments", creatorHandler.AddComment)
}

// registerAdminRoutes registers admin-only endpoints for event moderation and user management.
func registerAdminRoutes(
	r chi.Router,
	eventHandler *handlers.EventHandler,
	creatorHandler *handlers.CreatorHandler,
	authHandler *handlers.AuthHandler,
	adminRoleRequestHandler *handlers.AdminRoleRequestHandler,
) {
	// Event moderation
	r.Get("/api/events/pending", eventHandler.GetPendingEvents)
	r.Post("/api/events/{id}/review", eventHandler.ReviewPendingEvent)

	// Comments and revision requests
	r.Get("/api/admin/events/{eventId}/comments", creatorHandler.GetEventCommentsAsAdmin)
	r.Post("/api/admin/events/{eventId}/request-revision", creatorHandler.RequestRevision)
	r.Post("/api/admin/events/{eventId}/block", creatorHandler.BlockEvent)

	// User management
	r.Get("/api/admin/users", authHandler.GetAllUsers)
	r.Put("/api/admin/users/role", authHandler.UpdateUserRole)

	// Role request management (admin only)
	r.Get("/api/admin/role-requests/pending", adminRoleRequestHandler.GetPendingRoleRequests)
	r.Post("/api/admin/role-requests/{requestId}/approve", adminRoleRequestHandler.ApproveRoleRequest)
	r.Post("/api/admin/role-requests/{requestId}/reject", adminRoleRequestHandler.RejectRoleRequest)
}

// wrapWithCORS applies CORS configuration to the router.
func wrapWithCORS(r chi.Router, cfg *config.Config) http.Handler {
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

// versionHandler returns an HTTP handler that serves version information.
func versionHandler(info VersionInfo) http.HandlerFunc {
	// @Summary Service version
	// @Description Returns build and runtime version information for the service
	// @Tags monitoring
	// @Produce json
	// @Success 200 {object} VersionInfo
	// @Router /version [get]
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(info); err != nil {
			logger.Log.Error("failed to write version response", zap.Error(err))
			http.Error(w, "failed to render version info", http.StatusInternalServerError)
			return
		}
	}
}
