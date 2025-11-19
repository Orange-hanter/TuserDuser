package models

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
