// Package models содержит модели данных для публичного профиля пользователя.
package models

import (
	"encoding/json"
	"time"
)

// PublicUserProfile представляет публичную информацию о пользователе.
// Не содержит приватных данных (email, phone, birthdate и т.д.).
type PublicUserProfile struct {
	ID                string            `json:"id" example:"941b955e-ea57-dee3-565f-5684f81c4f14"`
	DisplayName       string            `json:"displayName" example:"Иван Иванов"`
	Username          *string           `json:"username,omitempty" example:"ivanivanov"`
	AvatarURL         *string           `json:"avatarUrl,omitempty" example:"https://cdn.example.com/avatars/941b...jpg"`
	Bio               *string           `json:"bio,omitempty" example:"Организатор мероприятий, спикер"`
	City              *string           `json:"city,omitempty" example:"Москва"`
	Country           *string           `json:"country,omitempty" example:"RU"`
	PublicEventsCount int               `json:"publicEventsCount" example:"12"`
	IsVerified        bool              `json:"isVerified" example:"true"`
	SocialLinks       map[string]string `json:"socialLinks,omitempty" swaggertype:"object,string" example:"twitter:https://twitter.com/ivan,telegram:https://t.me/ivan"`
	CreatedAt         time.Time         `json:"createdAt" example:"2024-05-02T15:23:45Z"`
	UpdatedAt         time.Time         `json:"updatedAt" example:"2025-11-01T12:10:00Z"`
}

// SocialLinks представляет публичные социальные ссылки пользователя.
type SocialLinks struct {
	Twitter  *string `json:"twitter,omitempty" example:"https://twitter.com/ivan"`
	Telegram *string `json:"telegram,omitempty" example:"https://t.me/ivan"`
	VK       *string `json:"vk,omitempty" example:"https://vk.com/ivan"`
	GitHub   *string `json:"github,omitempty" example:"https://github.com/ivan"`
	LinkedIn *string `json:"linkedin,omitempty" example:"https://linkedin.com/in/ivan"`
	Website  *string `json:"website,omitempty" example:"https://ivan.dev"`
}

// ParseSocialLinks парсит JSONB из БД в map.
func ParseSocialLinks(data []byte) map[string]string {
	if len(data) == 0 {
		return nil
	}
	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	// Возвращаем nil вместо пустой map для красивого JSON
	if len(result) == 0 {
		return nil
	}
	return result
}

// PublicProfileNotFoundError представляет ошибку "пользователь не найден".
type PublicProfileNotFoundError struct {
	UserID string
}

func (e *PublicProfileNotFoundError) Error() string {
	return "user not found"
}

// InvalidUUIDError представляет ошибку невалидного UUID.
type InvalidUUIDError struct {
	Value string
}

func (e *InvalidUUIDError) Error() string {
	return "userId must be a valid UUID"
}
