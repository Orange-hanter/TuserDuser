package handlers

import (
	"net/http"
)

// HealthCheck проверяет состояние сервиса
// GET /health
// @Summary Проверка здоровья сервиса
// @Description Возвращает статус OK если сервис работает
// @Tags health
// @Produce plain
// @Success 200 {string} string "OK"
// @Router /health [get].
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
