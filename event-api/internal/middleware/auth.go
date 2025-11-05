package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"event-api/internal/models"
	"event-api/internal/service"
)

// AuthMiddleware проверяет JWT токен.
func AuthMiddleware(authService *service.AuthService) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(models.ErrorResponse{
					Error:   "unauthorized",
					Message: "Missing authorization header",
					Code:    http.StatusUnauthorized,
				})
				return
			}

			// Парсим Bearer токен
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(models.ErrorResponse{
					Error:   "unauthorized",
					Message: "Invalid authorization header format",
					Code:    http.StatusUnauthorized,
				})
				return
			}

			tokenString := parts[1]

			// Валидируем токен
			claims, err := authService.ValidateJWT(tokenString)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(models.ErrorResponse{
					Error:   "unauthorized",
					Message: err.Error(),
					Code:    http.StatusUnauthorized,
				})
				return
			}

			// Сохраняем claims в контексте запроса
			r.Header.Set("X-User-ID", claims["user_id"].(string))
			r.Header.Set("X-User-Email", claims["email"].(string))

			next.ServeHTTP(w, r)
		})
	}
}
