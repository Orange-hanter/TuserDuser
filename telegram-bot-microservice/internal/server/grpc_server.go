package server

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"telegram-bot-microservice/api/proto"
	"telegram-bot-microservice/internal/service"
)

type GRPCServer struct {
	server     *grpc.Server
	botService *service.BotService
}

func NewGRPCServer(botService *service.BotService) *GRPCServer {
	return &GRPCServer{
		server:     grpc.NewServer(),
		botService: botService,
	}
}

func (s *GRPCServer) RegisterServices() {
	proto.RegisterBotServiceServer(s.server, s.botService)
}

func (s *GRPCServer) Start(address string) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Printf("gRPC server listening on %s", address)
	if err := s.server.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
