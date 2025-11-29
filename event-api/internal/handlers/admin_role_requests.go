package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"event-api/internal/models"
	"event-api/internal/service"

	"go.uber.org/zap"
)

// AdminRoleRequestHandler handles admin operations on role requests.
type AdminRoleRequestHandler struct {
	userService *service.UserService
	logger      *zap.Logger
}

// NewAdminRoleRequestHandler creates a new AdminRoleRequestHandler.
func NewAdminRoleRequestHandler(userService *service.UserService, logger *zap.Logger) *AdminRoleRequestHandler {
	return &AdminRoleRequestHandler{
		userService: userService,
		logger:      logger,
	}
}

// GetPendingRoleRequests godoc
// @Summary Get pending role requests
// @Description Get all pending role requests (admin only).
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} models.RoleRequestsListResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/admin/role-requests/pending [get]
func (h *AdminRoleRequestHandler) GetPendingRoleRequests(w http.ResponseWriter, r *http.Request) {
	limit := 20
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	requests, total, err := h.userService.GetPendingRoleRequests(r.Context(), limit, offset)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	if requests == nil {
		requests = []models.RoleRequestStatus{}
	}

	respondWithJSON(w, http.StatusOK, models.RoleRequestsListResponse{
		Requests: requests,
		Total:    total,
	})
}

// ApproveRoleRequest godoc
// @Summary Approve role request
// @Description Approve a pending role request (admin only).
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.RoleRequestApprovalRequest true "Approval request"
// @Success 200 {object} map[string]string "message and status"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/admin/role-requests/{requestId}/approve [post]
func (h *AdminRoleRequestHandler) ApproveRoleRequest(w http.ResponseWriter, r *http.Request) {
	var req models.RoleRequestApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	adminID := r.Header.Get("X-User-ID")
	if adminID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "Admin ID not found")
		return
	}

	if req.RequestID == "" {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "RequestID is required")
		return
	}

	err := h.userService.ApproveRoleRequest(r.Context(), req.RequestID, adminID, req.Notes)
	if err != nil {
		if err.Error() == "role request not found" {
			respondWithError(w, http.StatusNotFound, "not_found", "Role request not found")
			return
		}
		h.logger.Error("failed to approve role request", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Role request approved successfully",
		"status":  "approved",
	})
}

// RejectRoleRequest godoc
// @Summary Reject role request
// @Description Reject a pending role request (admin only).
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.RoleRequestRejectionRequest true "Rejection request"
// @Success 200 {object} map[string]string "message and status"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/admin/role-requests/{requestId}/reject [post]
func (h *AdminRoleRequestHandler) RejectRoleRequest(w http.ResponseWriter, r *http.Request) {
	var req models.RoleRequestRejectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	adminID := r.Header.Get("X-User-ID")
	if adminID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "Admin ID not found")
		return
	}

	if req.RequestID == "" {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "RequestID is required")
		return
	}

	if req.Reason == "" {
		respondWithError(w, http.StatusBadRequest, "invalid_request", "Reason for rejection is required")
		return
	}

	err := h.userService.RejectRoleRequest(r.Context(), req.RequestID, adminID, req.Reason)
	if err != nil {
		if err.Error() == "role request not found" {
			respondWithError(w, http.StatusNotFound, "not_found", "Role request not found")
			return
		}
		h.logger.Error("failed to reject role request", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Role request rejected successfully",
		"status":  "rejected",
	})
}
