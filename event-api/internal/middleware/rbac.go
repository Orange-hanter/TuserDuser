package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"event-api/internal/logger"
	"event-api/internal/models"

	"go.uber.org/zap"
)

// RequireRole проверяет, что пользователь имеет одну из требуемых ролей.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRole := r.Header.Get("X-User-Role")

			if userRole == "" {
				logger.Log.Warn("Missing user role in request")
				respondWithError(w, http.StatusForbidden, "forbidden", "Недостаточно прав")
				return
			}

			// Проверяем, есть ли роль пользователя в списке разрешенных
			hasRole := false
			for _, role := range roles {
				if userRole == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				logger.Log.Warn("User lacks required role",
					zap.String("user_role", userRole),
					zap.Strings("required_roles", roles),
				)
				respondWithError(w, http.StatusForbidden, "forbidden", "Недостаточно прав для этого действия")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin проверяет, что пользователь является администратором.
func RequireAdmin(next http.Handler) http.Handler {
	return RequireRole(models.RoleAdmin)(next)
}

// RequireCreatorOrAdmin проверяет, что пользователь является создателем или администратором.
func RequireCreatorOrAdmin(next http.Handler) http.Handler {
	return RequireRole(models.RoleCreator, models.RoleAdmin)(next)
}

// HasPermission проверяет, имеет ли роль определенное разрешение.
func HasPermission(role string, permission string) bool {
	permissions := getRolePermissions(role)
	for _, p := range permissions {
		if p == permission || p == "*" {
			return true
		}
	}
	return false
}

// getRolePermissions возвращает список разрешений для роли.
func getRolePermissions(role string) []string {
	switch strings.ToLower(role) {
	case models.RoleAdmin:
		return []string{"*"} // Администратор имеет все права
	case models.RoleCreator:
		return []string{
			"events.create",
			"events.read",
			"events.update_own",
			"events.delete_own",
		}
	case models.RoleSupport:
		return []string{
			"events.read",
			"users.read",
		}
	case models.RoleUser:
		return []string{
			"events.read",
		}
	default:
		return []string{}
	}
}

func respondWithError(w http.ResponseWriter, statusCode int, errorType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	response := fmt.Sprintf(`{"error":"%s","message":"%s","code":%d}`, errorType, message, statusCode)
	w.Write([]byte(response))
}
