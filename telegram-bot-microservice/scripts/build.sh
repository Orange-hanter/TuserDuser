#!/bin/bash

# Build the Go microservice
echo "Building the Telegram bot microservice..."

# Generate gRPC code from the proto file
echo "Generating gRPC code from proto file..."
protoc --go_out=. --go-grpc_out=. api/proto/bot.proto

# Build the Go application
echo "Compiling the Go application..."
go build -o telegram-bot-microservice cmd/server/main.go

echo "Build completed successfully!"
