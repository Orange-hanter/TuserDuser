package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mocking the repository interface
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) SaveMessage(userID string, message string) error {
	args := m.Called(userID, message)
	return args.Error(0)
}

func (m *MockRepository) GetUserMessages(userID string) ([]string, error) {
	args := m.Called(userID)
	return args.Get(0).([]string), args.Error(1)
}

// Test for processing a message
func TestProcessMessage(t *testing.T) {
	mockRepo := new(MockRepository)
	botService := NewBotService(mockRepo)

	userID := "12345"
	message := "Hello, Bot!"

	mockRepo.On("SaveMessage", userID, message).Return(nil)

	err := botService.ProcessMessage(userID, message)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// Test for retrieving user messages
func TestGetUserMessages(t *testing.T) {
	mockRepo := new(MockRepository)
	botService := NewBotService(mockRepo)

	userID := "12345"
	expectedMessages := []string{"Hello, Bot!", "How are you?"}

	mockRepo.On("GetUserMessages", userID).Return(expectedMessages, nil)

	messages, err := botService.GetUserMessages(userID)

	assert.NoError(t, err)
	assert.Equal(t, expectedMessages, messages)
	mockRepo.AssertExpectations(t)
}
