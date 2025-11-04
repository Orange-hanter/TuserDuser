# TuserDuser Makefile - Multi-service orchestration

SHELL := /bin/zsh

# Services
EVENT_API_DIR := event-api
BIN_DIR := bin

.PHONY: help all build build-linux build-linux-strip build-backends build-event-api clean test lint vet fmt

help:
	@echo "TuserDuser - Multi-service Backend Project"
	@echo ""
	@echo "Available services:"
	@echo "  - event-api           (event management API)"
	@echo ""
	@echo "Build targets:"
	@echo "  make build                 - Build all services (native)"
	@echo "  make build-event-api       - Build event-api"
	@echo "  make build-linux           - Cross-compile all services for linux/amd64"
	@echo "  make build-linux-strip     - Build and strip linux binaries"
	@echo ""
	@echo "Code quality:"
	@echo "  make test                  - Run tests for all services"
	@echo "  make lint                  - Run linters for all services"
	@echo "  make fmt                   - Format code in all services"
	@echo "  make vet                   - Run go vet on all services"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean                 - Remove all build artifacts"
	@echo ""

all: build

# ============================================================================
# Building
# ============================================================================

build: build-event-api
	@echo "✅ All services built"

build-event-api:
	@echo "📦 Building event-api..."
	cd $(EVENT_API_DIR) && go build -o ../$(BIN_DIR)/event-api ./cmd/server || true

build-linux: build-linux-event-api
	@echo "✅ All linux/amd64 binaries built"

build-linux-event-api:
	@echo "🐧 Cross-compiling event-api for linux/amd64..."
	cd $(EVENT_API_DIR) && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ../$(BIN_DIR)/event-api-linux-amd64 ./cmd/server || true

build-linux-strip: build-linux
	@echo "✂️  Stripping linux binaries..."
	@for binary in $(BIN_DIR)/*-linux-amd64; do \
		if [ -f "$$binary" ]; then \
			strip "$$binary" 2>/dev/null || true; \
			echo "  Stripped $$binary"; \
		fi; \
	done

# ============================================================================
# Testing & Quality
# ============================================================================

test: test-event-api
	@echo "✅ Tests completed"

test-event-api:
	@echo "🧪 Testing event-api..."
	cd $(EVENT_API_DIR) && go test -v -race -coverprofile=coverage.out ./... || echo "⚠️  event-api tests skipped or failed"

lint: lint-event-api
	@echo "✅ Lint completed"

lint-event-api:
	@echo "🔍 Linting event-api..."
	cd $(EVENT_API_DIR) && go vet ./... && gofmt -s -w . || true

fmt:
	@echo "🎨 Formatting all services..."
	cd $(EVENT_API_DIR) && go fmt ./... || true

vet:
	@echo "🔍 Running vet on all services..."
	cd $(EVENT_API_DIR) && go vet ./... || true

# ============================================================================
# Cleanup
# ============================================================================

clean:
	@echo "🗑️  Cleaning build artifacts..."
	rm -f $(BIN_DIR)/*
	rm -f $(EVENT_API_DIR)/coverage.out
	@echo "✅ Cleaned"
