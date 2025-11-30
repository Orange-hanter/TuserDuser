package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"telegram-bot-microservice/internal/config"
	"telegram-bot-microservice/internal/server"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	lis, err := net.Listen("tcp", cfg.Server.Address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	server.RegisterBotService(grpcServer)

	log.Printf("Starting gRPC server on %s", cfg.Server.Address)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
