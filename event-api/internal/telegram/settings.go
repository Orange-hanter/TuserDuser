package telegram

import (
	"time"

	"event-api/internal/config"
)

// Settings aggregates Telegram integration configuration.
type Settings struct {
	Enabled            bool
	BotToken           string
	BotUsername        string
	WebhookSecret      string
	WebhookAlias       string
	BindingSecret      string
	BindingTTLSeconds  int
	RateLimitPerSecond int
	MaxAttempts        int
	RetryBaseSeconds   int
	APIBaseURL         string
	PollInterval       time.Duration
	BatchSize          int
}

// NewSettingsFrom produces Settings using global config defaults.
func NewSettingsFrom(cfg *config.Config) Settings {
	poll := time.Second
	if cfg.TelegramRateLimitPerSec < 5 {
		poll = 500 * time.Millisecond
	}
	return Settings{
		Enabled:            cfg.TelegramEnabled,
		BotToken:           cfg.TelegramBotToken,
		BotUsername:        cfg.TelegramBotUsername,
		WebhookSecret:      cfg.TelegramWebhookSecret,
		WebhookAlias:       cfg.TelegramWebhookAlias,
		BindingSecret:      cfg.TelegramBindingSecret,
		BindingTTLSeconds:  cfg.TelegramBindingTTL,
		RateLimitPerSecond: cfg.TelegramRateLimitPerSec,
		MaxAttempts:        cfg.TelegramMaxAttempts,
		RetryBaseSeconds:   cfg.TelegramRetryBaseSeconds,
		APIBaseURL:         cfg.TelegramAPIBaseURL,
		PollInterval:       poll,
		BatchSize:          100,
	}
}
