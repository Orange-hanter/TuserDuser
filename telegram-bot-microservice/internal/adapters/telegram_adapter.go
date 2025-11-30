package adapters

import (
	"context"
	"log"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"telegram-bot-microservice/internal/service"
)

type TelegramAdapter struct {
	botAPI  *tgbotapi.BotAPI
	service *service.BotService
}

func NewTelegramAdapter(token string, botService *service.BotService) (*TelegramAdapter, error) {
	botAPI, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return &TelegramAdapter{
		botAPI:  botAPI,
		service: botService,
	}, nil
}

func (ta *TelegramAdapter) StartListeningUpdates() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := ta.botAPI.GetUpdatesChan(u)
	if err != nil {
		log.Fatalf("Failed to get updates: %v", err)
	}

	for update := range updates {
		if update.Message == nil {
			continue
		}

		go ta.processMessage(update.Message)
	}
}

func (ta *TelegramAdapter) processMessage(message *tgbotapi.Message) {
	response, err := ta.service.HandleMessage(message.Text, message.Chat.ID)
	if err != nil {
		log.Printf("Error processing message: %v", err)
		return
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	if _, err := ta.botAPI.Send(msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}
