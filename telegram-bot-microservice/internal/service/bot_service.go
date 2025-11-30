package service

import (
	"context"
	"errors"
	"log"

	pb "telegram-bot-microservice/api/proto"
	"telegram-bot-microservice/internal/repository"
)

type BotService struct {
	repo repository.Repository
	pb.UnimplementedBotServiceServer
}

func NewBotService(repo repository.Repository) *BotService {
	return &BotService{repo: repo}
}

func (s *BotService) ProcessMessage(ctx context.Context, req *pb.ProcessMessageRequest) (*pb.ProcessMessageResponse, error) {
	if req == nil || req.Message == "" {
		return nil, errors.New("invalid request: message cannot be empty")
	}

	// Process the incoming message
	log.Printf("Processing message: %s", req.Message)

	// Here you can add your business logic for processing the message
	// For example, saving the message to the database
	err := s.repo.SaveMessage(req.UserId, req.Message)
	if err != nil {
		return nil, err
	}

	response := &pb.ProcessMessageResponse{
		Status:  "success",
		Message: "Message processed successfully",
	}

	return response, nil
}

func (s *BotService) GetUserMessages(ctx context.Context, req *pb.GetUserMessagesRequest) (*pb.GetUserMessagesResponse, error) {
	if req == nil || req.UserId == "" {
		return nil, errors.New("invalid request: user ID cannot be empty")
	}

	messages, err := s.repo.GetMessagesByUserId(req.UserId)
	if err != nil {
		return nil, err
	}

	response := &pb.GetUserMessagesResponse{
		Messages: messages,
	}

	return response, nil
}
