package handlers

import (
	"net/http"

	"event-api/internal/logger"

	"go.uber.org/zap"
)

// HealthCheck проверяет состояние сервиса
// GET /health
// @Summary Проверка здоровья сервиса
// @Description Возвращает статус OK если сервис работает
// @Tags health
// @Produce plain
// @Success 200 {string} string "OK"
// @Router /health [get].
func HealthCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("OK")); err != nil {
		logger.Log.Error("failed to write health check response", zap.Error(err))
	}
}
