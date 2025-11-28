package models

import "time"

// Справочник ролей пользователей в системе и контракт для обновления роли.
//
// Допустимые роли (значения строк):
// - `user` — Обычный пользователь: может просматривать и взаимодействовать с публичными ресурсами.
// - `creator` — Создатель: может создавать события и управлять собственным контентом.
// - `support` — Поддержка: служебная роль для обработки обращений и модерации (ограниченная).
// - `admin` — Администратор: полные права, управление пользователями и системой.
//
// Эти значения также представлены в `internal/models/auth.go` как константы:
// `RoleUser`, `RoleCreator`, `RoleSupport`, `RoleAdmin`.

// UpdateRoleRequest - запрос для обновления роли пользователя.
//
// Swagger/validation примечание: поле `role` должно содержать одно из перечисленных
// строковых значений (user, creator, support, admin). Пример JSON:
//
//	{
//	  "user_id": "<id>",
//	  "role": "creator"
//	}
type UpdateRoleRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Role   string `json:"role" binding:"required"`
}

// RoleRequest - запрос для申请 роли пользователем.
//
// Позволяет пользователю запросить повышение роли с указанием причины.
// Пример JSON:
//
//	{
//	  "role": "creator",
//	  "reason": "Хочу создавать события"
//	}
type RoleRequest struct {
	Role   string `json:"role" binding:"required,oneof=creator support"`
	Reason string `json:"reason" binding:"required,min=10,max=500"`
}

// RoleRequestResponse - ответ на запрос роли.
type RoleRequestResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"` // "pending", "approved", "rejected"
}

// RoleRequestStatus - статус запроса на повышение роли
type RoleRequestStatus struct {
	ID            string     `json:"id" example:"req_123"`
	RequestedRole string     `json:"requested_role" example:"creator"`
	CurrentStatus string     `json:"status" example:"pending"` // "pending", "approved", "rejected"
	Reason        string     `json:"reason" example:"Хочу создавать события"`
	ReviewNotes   *string    `json:"review_notes,omitempty" example:"Approved"`
	CreatedAt     time.Time  `json:"created_at" example:"2025-11-20T15:30:00Z"`
	UpdatedAt     time.Time  `json:"updated_at" example:"2025-11-21T10:30:00Z"`
	ReviewedAt    *time.Time `json:"reviewed_at,omitempty" example:"2025-11-21T10:30:00Z"`
	ReviewedBy    *string    `json:"reviewed_by,omitempty" example:"admin_user_id"`
}

// RoleRequestsListResponse - список запросов на повышение ролей
type RoleRequestsListResponse struct {
	Requests []RoleRequestStatus `json:"requests"`
	Total    int                 `json:"total"`
}

// RoleRequestApprovalRequest - запрос на одобрение роли
type RoleRequestApprovalRequest struct {
	RequestID string `json:"request_id" binding:"required"`
	Notes     string `json:"notes" binding:"max=500"`
}

// RoleRequestRejectionRequest - запрос на отклонение роли
type RoleRequestRejectionRequest struct {
	RequestID string `json:"request_id" binding:"required"`
	Reason    string `json:"reason" binding:"required,min=10,max=500"`
}
