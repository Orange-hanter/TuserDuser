package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"event-api/internal/models"
	"event-api/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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

// GetPublicProfile godoc
// @Summary Get public user profile
// @Description Returns public information about a user by their UUID. Does not require authentication.
// @Tags users
// @Accept json
// @Produce json
// @Param userId path string true "User ID (UUID format)" format(uuid)
// @Success 200 {object} models.PublicUserProfile "Public user profile"
// @Success 304 "Not Modified - resource unchanged since last request"
// @Failure 400 {object} models.ErrorResponse "Invalid UUID format"
// @Failure 404 {object} models.ErrorResponse "User not found or profile is private"
// @Failure 429 {object} models.ErrorResponse "Rate limit exceeded"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/users/public/{userId} [get]
func (h *UserHandler) GetPublicProfile(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")

	// Validate UUID format
	if _, err := uuid.Parse(userID); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "userId must be a valid UUID")
		return
	}

	// Get public profile from service
	profile, etag, err := h.userService.GetPublicProfile(r.Context(), userID)
	if err != nil {
		// Check if it's a not found error
		var notFoundErr *models.PublicProfileNotFoundError
		if errors.As(err, &notFoundErr) {
			respondWithError(w, http.StatusNotFound, "not_found", "User not found")
			return
		}
		// Log internal error but don't expose details
		respondWithError(w, http.StatusInternalServerError, "server_error", "Internal server error")
		return
	}

	// Check If-None-Match for caching
	clientETag := r.Header.Get("If-None-Match")
	if clientETag != "" && strings.TrimSpace(clientETag) == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// Set caching headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300, s-maxage=600")
	w.Header().Set("ETag", etag)
	w.Header().Set("Last-Modified", profile.UpdatedAt.UTC().Format(http.TimeFormat))
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if err := json.NewEncoder(w).Encode(profile); err != nil {
		// Can't send error response as headers are already sent
		return
	}
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

// GetEventParticipants godoc
// @Summary Get event participants
// @Description Returns a list of confirmed participants for a specific event.
// @Tags events
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param event_id path string true "Event ID"
// @Success 200 {array} models.Participant
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/events/{event_id}/participants [get]
func (h *UserHandler) GetEventParticipants(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "event_id")
	if eventID == "" {
		respondWithError(w, http.StatusBadRequest, "bad_request", "Event ID is required")
		return
	}

	participants, err := h.userService.GetEventParticipants(r.Context(), eventID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch participants")
		return
	}

	if participants == nil {
		participants = []models.Participant{}
	}

	respondWithJSON(w, http.StatusOK, participants)
}

// RequestRole godoc
// @Summary Request role upgrade
// @Description User requests a role upgrade (creator, support, etc.) with a reason.
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.RoleRequest true "Role request"
// @Success 200 {object} models.RoleRequestResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/users/request-role [post]
func (h *UserHandler) RequestRole(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	var req models.RoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	// Validate request
	if req.Role == "" || req.Reason == "" {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "Role and reason are required")
		return
	}

	if len(req.Reason) < 10 || len(req.Reason) > 500 {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "Reason must be between 10 and 500 characters")
		return
	}

	// Only allow requesting creator or support roles
	if req.Role != models.RoleCreator && req.Role != models.RoleSupport {
		respondWithError(w, http.StatusBadRequest, "invalid_role", "Can only request creator or support role")
		return
	}

	resp, err := h.userService.RequestRole(r.Context(), userID, req.Role, req.Reason)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, resp)
}

// GetRoleRequestStatus godoc
// @Summary Get role request status
// @Description Get the status of a specific role request.
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param role query string true "Role"
// @Success 200 {object} models.RoleRequestStatus
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/users/request-role/status [get]
func (h *UserHandler) GetRoleRequestStatus(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	role := r.URL.Query().Get("role")
	if role == "" {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "Role parameter is required")
		return
	}

	status, err := h.userService.GetRoleRequestStatus(r.Context(), userID, role)
	if err != nil {
		if err.Error() == "role request not found" {
			respondWithError(w, http.StatusNotFound, "not_found", "Role request not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, status)
}

// GetAllRoleRequests godoc
// @Summary Get all role requests
// @Description Get all role requests for the current user.
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.RoleRequestStatus
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/users/request-role/all [get]
func (h *UserHandler) GetAllRoleRequests(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "User ID not found")
		return
	}

	requests, err := h.userService.GetRoleRequests(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	if requests == nil {
		requests = []models.RoleRequestStatus{}
	}

	respondWithJSON(w, http.StatusOK, requests)
}
