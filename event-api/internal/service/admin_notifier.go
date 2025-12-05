package service

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"event-api/internal/models"
	"event-api/internal/telegramclient"

	"go.uber.org/zap"
)

// AdminNotifier handles sending notifications to admin users via Telegram.
type AdminNotifier struct {
	db             *sql.DB
	telegramClient *telegramclient.Client
	logger         *zap.Logger
}

// NewAdminNotifier creates a new AdminNotifier instance.
func NewAdminNotifier(db *sql.DB, telegramClient *telegramclient.Client, logger *zap.Logger) *AdminNotifier {
	return &AdminNotifier{
		db:             db,
		telegramClient: telegramClient,
		logger:         logger,
	}
}

// GetAdminUserIDs returns a list of user IDs that have admin role.
func (n *AdminNotifier) GetAdminUserIDs(ctx context.Context) ([]string, error) {
	query := `SELECT id FROM users WHERE role = $1`

	rows, err := n.db.QueryContext(ctx, query, models.RoleAdmin)
	if err != nil {
		return nil, fmt.Errorf("failed to query admin users: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			n.logger.Error("failed to close rows", zap.Error(err))
		}
	}()

	var adminIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			n.logger.Error("failed to scan admin user id", zap.Error(err))
			continue
		}
		adminIDs = append(adminIDs, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating admin users: %w", err)
	}

	return adminIDs, nil
}

// NotifyAdminsNewEvent sends a notification to all admins about a new event pending moderation.
func (n *AdminNotifier) NotifyAdminsNewEvent(ctx context.Context, event *models.PendingEvent, creatorEmail string) {
	if n.telegramClient == nil {
		n.logger.Debug("telegram client not configured, skipping admin notification")
		return
	}

	adminIDs, err := n.GetAdminUserIDs(ctx)
	if err != nil {
		n.logger.Error("failed to get admin user IDs", zap.Error(err))
		return
	}

	if len(adminIDs) == 0 {
		n.logger.Debug("no admin users found, skipping notification")
		return
	}

	// Build notification message
	eventType := event.Type
	if eventType == "" {
		eventType = "Не указан"
	}

	place := event.Place
	if place == "" {
		place = "Не указано"
	}

	startTime := event.StartTime.Format("02.01.2006 15:04")

	message := fmt.Sprintf(
		"🆕 *Новый запрос на создание события*\n\n"+
			"📋 *Тип:* %s\n"+
			"📍 *Место:* %s\n"+
			"🕐 *Начало:* %s\n"+
			"👤 *Создатель:* %s\n"+
			"🆔 *ID события:* `%s`\n\n"+
			"⏳ Ожидает модерации",
		escapeMarkdownV2(eventType),
		escapeMarkdownV2(place),
		escapeMarkdownV2(startTime),
		escapeMarkdownV2(creatorEmail),
		event.ID,
	)

	n.sendToAdmins(ctx, adminIDs, message)
}

// NotifyAdminsRoleRequest sends a notification to all admins about a new role upgrade request.
func (n *AdminNotifier) NotifyAdminsRoleRequest(ctx context.Context, userID, userEmail, requestedRole, reason string) {
	if n.telegramClient == nil {
		n.logger.Debug("telegram client not configured, skipping admin notification")
		return
	}

	adminIDs, err := n.GetAdminUserIDs(ctx)
	if err != nil {
		n.logger.Error("failed to get admin user IDs", zap.Error(err))
		return
	}

	if len(adminIDs) == 0 {
		n.logger.Debug("no admin users found, skipping notification")
		return
	}

	// Build notification message
	roleDisplay := requestedRole
	switch requestedRole {
	case models.RoleCreator:
		roleDisplay = "Создатель событий"
	case models.RoleSupport:
		roleDisplay = "Поддержка"
	case models.RoleAdmin:
		roleDisplay = "Администратор"
	}

	reasonText := reason
	if reasonText == "" {
		reasonText = "Не указана"
	}

	message := fmt.Sprintf(
		"🔑 *Запрос на повышение уровня доступа*\n\n"+
			"👤 *Пользователь:* %s\n"+
			"🆔 *ID:* `%s`\n"+
			"📊 *Запрошенная роль:* %s\n"+
			"📝 *Причина:* %s\n\n"+
			"⏳ Ожидает рассмотрения",
		escapeMarkdownV2(userEmail),
		userID,
		escapeMarkdownV2(roleDisplay),
		escapeMarkdownV2(reasonText),
	)

	n.sendToAdmins(ctx, adminIDs, message)
}

// sendToAdmins sends a message to all admin users in parallel.
func (n *AdminNotifier) sendToAdmins(ctx context.Context, adminIDs []string, message string) {
	var wg sync.WaitGroup
	successCount := 0
	failCount := 0
	var mu sync.Mutex

	for _, adminID := range adminIDs {
		wg.Add(1)
		go func(userID string) {
			defer wg.Done()

			_, err := n.telegramClient.SendMessage(ctx, userID, message, "MarkdownV2", false, "")
			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				failCount++
				n.logger.Warn("failed to send admin notification",
					zap.String("admin_id", userID),
					zap.Error(err),
				)
			} else {
				successCount++
				n.logger.Debug("admin notification sent",
					zap.String("admin_id", userID),
				)
			}
		}(adminID)
	}

	wg.Wait()

	n.logger.Info("admin notifications completed",
		zap.Int("success", successCount),
		zap.Int("failed", failCount),
		zap.Int("total", len(adminIDs)),
	)
}

// escapeMarkdownV2 escapes special characters for Telegram MarkdownV2.
func escapeMarkdownV2(text string) string {
	special := []byte{'_', '*', '[', ']', '(', ')', '~', '`', '>', '#', '+', '-', '=', '|', '{', '}', '.', '!'}
	result := make([]byte, 0, len(text)*2)
	for i := 0; i < len(text); i++ {
		for _, s := range special {
			if text[i] == s {
				result = append(result, '\\')
				break
			}
		}
		result = append(result, text[i])
	}
	return string(result)
}
