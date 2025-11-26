package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"event-api/internal/models"
	"event-api/internal/service"

	"github.com/go-chi/chi/v5"
)

// UserHandler handles user-related HTTP requests.
type UserHandler struct {
	userService *service.UserService
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// GetMe godoc
// @Summary Get current user profile
// @Description Returns full user profile and status flags.
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.UserProfile
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/users/me [get]
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	profile, err := h.userService.GetUserProfile(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

// GetUpcomingEvents godoc
// @Summary Get upcoming events
// @Description Returns events the user is currently subscribed to (confirmed participation).
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.EventWithSubscription
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/users/me/events/upcoming [get]
func (h *UserHandler) GetUpcomingEvents(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	events, err := h.userService.GetUpcomingEvents(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(events); err != nil {
		http.Error(w, "failed to write response", http.StatusInternalServerError)
		return
	}
}

// GetEventHistory godoc
// @Summary Get event history
// @Description Returns attended or missed past events.
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} models.EventWithSubscription
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/users/me/events/history [get]
func (h *UserHandler) GetEventHistory(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	limit := 20
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil {
			limit = val
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil {
			offset = val
		}
	}

	events, err := h.userService.GetEventHistory(r.Context(), userID, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

// Subscribe godoc
// @Summary Subscribe to an event
// @Description Idempotent subscription endpoint. Checks capacity and handles waitlisting.
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param event_id path string true "Event ID"
// @Param request body models.SubscribeRequest false "Subscription metadata"
// @Success 200 {object} models.EventSubscription "Already subscribed"
// @Success 201 {object} models.EventSubscription "Created"
// @Success 202 {object} models.EventSubscription "Waitlisted"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse "Event full"
// @Failure 500 {object} models.ErrorResponse
// @Router /api/users/me/events/{event_id}/subscribe [post]
func (h *UserHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	eventID := chi.URLParam(r, "event_id")

	var req models.SubscribeRequest
	// Body is optional
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	sub, err := h.userService.SubscribeToEvent(r.Context(), userID, eventID, req.Metadata)
	if err != nil {
		if errors.Is(err, service.ErrEventFull) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			if err := json.NewEncoder(w).Encode(map[string]string{"error": "event_full", "message": "Event is full"}); err != nil {
				http.Error(w, "failed to write response", http.StatusInternalServerError)
			}
			return
		}
		if err.Error() == "event not found or already started" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if sub.Status == models.SubscriptionStatusWaitlisted {
		w.WriteHeader(http.StatusAccepted)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	if err := json.NewEncoder(w).Encode(sub); err != nil {
		http.Error(w, "failed to write response", http.StatusInternalServerError)
	}
}
