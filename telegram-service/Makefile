.PHONY: all build run test clean proto docker-build docker-up docker-down lint

# Variables
BINARY_NAME=telegram-service
DOCKER_IMAGE=telegram-service:latest

all: build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	go build -o bin/$(BINARY_NAME) ./cmd/server

# Run locally
run:
	@echo "Running $(BINARY_NAME)..."
	go run ./cmd/server

# Run tests
test:
	@echo "Running tests..."
	go test -v -race ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	protoc --go_out=. --go-grpc_out=. api/proto/telegram_service.proto

# Lint code
lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

# Docker compose up
docker-up:
	@echo "Starting services..."
	docker-compose up -d

# Docker compose down
docker-down:
	@echo "Stopping services..."
	docker-compose down

# Docker compose logs
docker-logs:
	docker-compose logs -f telegram-service

# Run database migrations
migrate:
	@echo "Running migrations..."
	go run ./cmd/migrate

# gRPC health check
health:
	@echo "Checking service health..."
	grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check

# Test gRPC endpoints
grpc-test:
	@echo "Testing gRPC endpoints..."
	grpcurl -plaintext -d '{"user_id": "test-user"}' localhost:50051 telegram.v1.TelegramService/GenerateBindingLink

# Install development tools
tools:
	@echo "Installing development tools..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

help:
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  run           - Run locally"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  clean         - Clean build artifacts"
	@echo "  proto         - Generate protobuf code"
	@echo "  lint          - Run linter"
	@echo "  fmt           - Format code"
	@echo "  deps          - Download dependencies"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-up     - Start Docker services"
	@echo "  docker-down   - Stop Docker services"
	@echo "  docker-logs   - View Docker logs"
	@echo "  health        - Check service health"
	@echo "  grpc-test     - Test gRPC endpoints"
	@echo "  tools         - Install development tools"
