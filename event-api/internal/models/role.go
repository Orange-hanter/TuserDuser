package models

// UpdateRoleRequest - запрос для обновления роли пользователя
type UpdateRoleRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Role   string `json:"role" binding:"required"`
}
