package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"event-api/internal/logger"
	"event-api/internal/models"
	"event-api/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// FeedbackHandler manages feedback HTTP endpoints.
type FeedbackHandler struct {
	feedbackService *service.FeedbackService
}

// NewFeedbackHandler creates a new FeedbackHandler.
func NewFeedbackHandler(feedbackService *service.FeedbackService) *FeedbackHandler {
	return &FeedbackHandler{
		feedbackService: feedbackService,
	}
}

// CreateFeedback creates a new feedback entry.
// @Summary Create feedback
// @Description Creates a new feedback entry from user. Authentication is optional.
// @Tags feedback
// @Accept json
// @Produce json
// @Param request body models.CreateFeedbackRequest true "Feedback data"
// @Success 201 {object} models.Feedback "Created feedback"
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /v1/api/feedback [post]
func (h *FeedbackHandler) CreateFeedback(w http.ResponseWriter, r *http.Request) {
	var req models.CreateFeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Error("Failed to decode feedback request", zap.Error(err))
		respondWithError(w, http.StatusBadRequest, "bad_request", "Invalid request format")
		return
	}

	// Validate required fields
	if req.Message == "" {
		respondWithError(w, http.StatusBadRequest, "validation_error", "Message is required")
		return
	}

	if req.Category == "" {
		req.Category = models.FeedbackCategoryOther
	}

	// Get authenticated user ID from header (set by middleware, may be empty)
	authenticatedUserID := r.Header.Get("X-User-ID")

	feedback, err := h.feedbackService.CreateFeedback(r.Context(), &req, authenticatedUserID)
	if err != nil {
		logger.Log.Error("Failed to create feedback", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "internal_error", "Failed to create feedback")
		return
	}

	logger.Log.Info("Feedback created",
		zap.String("id", feedback.ID),
		zap.String("category", string(feedback.Category)),
	)

	respondWithJSON(w, http.StatusCreated, feedback)
}

// GetFeedbackList returns a paginated list of feedback (admin only).
// @Summary List feedback
// @Description Returns a paginated list of feedback, sorted by newest first
// @Tags feedback
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(20)
// @Param unreadOnly query bool false "Filter unread only" default(false)
// @Success 200 {object} models.FeedbackListResponse "Feedback list"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /v1/api/admin/feedback [get]
func (h *FeedbackHandler) GetFeedbackList(w http.ResponseWriter, r *http.Request) {
	page := 1
	pageSize := 20
	unreadOnly := false

	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if ps := r.URL.Query().Get("pageSize"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 {
			pageSize = parsed
		}
	}

	if uo := r.URL.Query().Get("unreadOnly"); uo == "true" || uo == "1" {
		unreadOnly = true
	}

	response, err := h.feedbackService.GetFeedbackList(r.Context(), page, pageSize, unreadOnly)
	if err != nil {
		logger.Log.Error("Failed to get feedback list", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "internal_error", "Failed to get feedback list")
		return
	}

	respondWithJSON(w, http.StatusOK, response)
}

// GetFeedbackByID returns a single feedback by ID (admin only).
// @Summary Get feedback by ID
// @Description Returns a single feedback entry by ID
// @Tags feedback
// @Security BearerAuth
// @Produce json
// @Param id path string true "Feedback ID"
// @Success 200 {object} models.Feedback "Feedback"
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 404 {object} models.ErrorResponse "Feedback not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /v1/api/admin/feedback/{id} [get]
func (h *FeedbackHandler) GetFeedbackByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondWithError(w, http.StatusBadRequest, "bad_request", "Feedback ID is required")
		return
	}

	if _, err := uuid.Parse(id); err != nil {
		respondWithError(w, http.StatusBadRequest, "validation_error", "Invalid feedback ID format")
		return
	}

	feedback, err := h.feedbackService.GetFeedbackByID(r.Context(), id)
	if err != nil {
		logger.Log.Error("Failed to get feedback", zap.String("id", id), zap.Error(err))
		respondWithError(w, http.StatusNotFound, "not_found", "Feedback not found")
		return
	}

	respondWithJSON(w, http.StatusOK, feedback)
}

// MarkFeedbackRead marks a feedback as read or unread (admin only).
// @Summary Mark feedback as read/unread
// @Description Marks a feedback entry as read or unread
// @Tags feedback
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Feedback ID"
// @Param request body models.MarkFeedbackReadRequest true "Read status"
// @Success 200 {object} map[string]string "Status updated"
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 404 {object} models.ErrorResponse "Feedback not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /v1/api/admin/feedback/{id}/read [put]
func (h *FeedbackHandler) MarkFeedbackRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondWithError(w, http.StatusBadRequest, "bad_request", "Feedback ID is required")
		return
	}

	if _, err := uuid.Parse(id); err != nil {
		respondWithError(w, http.StatusBadRequest, "validation_error", "Invalid feedback ID format")
		return
	}

	var req models.MarkFeedbackReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "bad_request", "Invalid request format")
		return
	}

	if err := h.feedbackService.MarkFeedbackRead(r.Context(), id, req.IsRead); err != nil {
		logger.Log.Error("Failed to mark feedback", zap.String("id", id), zap.Error(err))
		respondWithError(w, http.StatusNotFound, "not_found", "Feedback not found")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Feedback status updated",
	})
}

// DeleteFeedback deletes a feedback entry (admin only).
// @Summary Delete feedback
// @Description Deletes a feedback entry by ID
// @Tags feedback
// @Security BearerAuth
// @Produce json
// @Param id path string true "Feedback ID"
// @Success 200 {object} map[string]string "Deleted"
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 404 {object} models.ErrorResponse "Feedback not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /v1/api/admin/feedback/{id} [delete]
func (h *FeedbackHandler) DeleteFeedback(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondWithError(w, http.StatusBadRequest, "bad_request", "Feedback ID is required")
		return
	}

	if _, err := uuid.Parse(id); err != nil {
		respondWithError(w, http.StatusBadRequest, "validation_error", "Invalid feedback ID format")
		return
	}

	if err := h.feedbackService.DeleteFeedback(r.Context(), id); err != nil {
		logger.Log.Error("Failed to delete feedback", zap.String("id", id), zap.Error(err))
		respondWithError(w, http.StatusNotFound, "not_found", "Feedback not found")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Feedback deleted",
	})
}

// GetUnreadCount returns the count of unread feedback (admin only).
// @Summary Get unread feedback count
// @Description Returns the count of unread feedback entries
// @Tags feedback
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]int "Unread count"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /v1/api/admin/feedback/unread/count [get]
func (h *FeedbackHandler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	count, err := h.feedbackService.GetUnreadCount(r.Context())
	if err != nil {
		logger.Log.Error("Failed to get unread count", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "internal_error", "Failed to get unread count")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]int{
		"unreadCount": count,
	})
}
