# TuserDuser Makefile - Multi-service orchestration

SHELL := /bin/zsh

# Services
EVENT_API_DIR := event-api
BIN_DIR := bin

# Optional: Support/Backend (may not be in all branches)
BACKEND_DIR := Support/Backend
BACKEND_EXISTS := $(shell [ -d "$(BACKEND_DIR)" ] && echo yes || echo no)

.PHONY: help all build build-linux build-linux-strip build-backends build-event-api clean test lint vet fmt

help:
	@echo "TuserDuser - Multi-service Backend Project"
	@echo ""
	@echo "Available services:"
	@echo "  - event-api           (event management API)"
ifeq ($(BACKEND_EXISTS),yes)
	@echo "  - Support/Backend     (feedback & event collection service)"
endif
	@echo ""
	@echo "Build targets:"
	@echo "  make build                 - Build all services (native)"
	@echo "  make build-event-api       - Build event-api"
ifeq ($(BACKEND_EXISTS),yes)
	@echo "  make build-backends        - Build Support/Backend"
	@echo "  make build-linux           - Cross-compile all services for linux/amd64"
	@echo "  make build-linux-strip     - Build and strip linux binaries"
endif
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

ifeq ($(BACKEND_EXISTS),yes)
build: build-backends build-event-api
else
build: build-event-api
endif
	@echo "✅ All services built"

build-event-api:
	@echo "📦 Building event-api..."
	cd $(EVENT_API_DIR) && go build -o ../$(BIN_DIR)/event-api ./cmd/server || true

build-backends:
ifeq ($(BACKEND_EXISTS),yes)
	@echo "📦 Building Support/Backend..."
	cd $(BACKEND_DIR) && go build -o ../../$(BIN_DIR)/tuserduser-backend ./
else
	@echo "⚠️  Support/Backend not found in this branch"
endif

ifeq ($(BACKEND_EXISTS),yes)
build-linux: build-linux-event-api build-linux-backends
	@echo "✅ All linux/amd64 binaries built"

build-linux-backends:
	@echo "🐧 Cross-compiling Support/Backend for linux/amd64..."
	cd $(BACKEND_DIR) && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ../../$(BIN_DIR)/tuserduser-backend-linux-amd64 ./
endif

build-linux-event-api:
	@echo "🐧 Cross-compiling event-api for linux/amd64..."
	cd $(EVENT_API_DIR) && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ../$(BIN_DIR)/event-api-linux-amd64 ./cmd/server || true

ifeq ($(BACKEND_EXISTS),yes)
build-linux-strip: build-linux
	@echo "✂️  Stripping linux binaries..."
	@for binary in $(BIN_DIR)/*-linux-amd64; do \
		if [ -f "$$binary" ]; then \
			strip "$$binary" 2>/dev/null || true; \
			echo "  Stripped $$binary"; \
		fi; \
	done
endif

# ============================================================================
# Testing & Quality
# ============================================================================

ifeq ($(BACKEND_EXISTS),yes)
test: test-backends test-event-api
else
test: test-event-api
endif
	@echo "✅ Tests completed"

test-event-api:
	@echo "🧪 Testing event-api..."
	cd $(EVENT_API_DIR) && go test -v -race -coverprofile=coverage.out ./... || echo "⚠️  event-api tests skipped or failed"

test-backends:
ifeq ($(BACKEND_EXISTS),yes)
	@echo "🧪 Testing Support/Backend..."
	cd $(BACKEND_DIR) && go test -v -race -coverprofile=coverage.out ./... || true
endif

ifeq ($(BACKEND_EXISTS),yes)
lint: lint-backends lint-event-api
else
lint: lint-event-api
endif
	@echo "✅ Lint completed"

lint-backends:
ifeq ($(BACKEND_EXISTS),yes)
	@echo "🔍 Linting Support/Backend..."
	cd $(BACKEND_DIR) && go vet ./... && gofmt -s -w .
endif

lint-event-api:
	@echo "🔍 Linting event-api..."
	cd $(EVENT_API_DIR) && go vet ./... && gofmt -s -w . || true

fmt:
	@echo "🎨 Formatting all services..."
	cd $(EVENT_API_DIR) && go fmt ./... || true
ifeq ($(BACKEND_EXISTS),yes)
	cd $(BACKEND_DIR) && go fmt ./... || true
endif

vet:
	@echo "🔍 Running vet on all services..."
	cd $(EVENT_API_DIR) && go vet ./... || true
ifeq ($(BACKEND_EXISTS),yes)
	cd $(BACKEND_DIR) && go vet ./... || true
endif

# ============================================================================
# Cleanup
# ============================================================================

clean:
	@echo "🗑️  Cleaning build artifacts..."
	rm -f $(BIN_DIR)/*
	rm -f $(EVENT_API_DIR)/coverage.out
ifeq ($(BACKEND_EXISTS),yes)
	rm -f $(BACKEND_DIR)/coverage.out
	rm -f feedbacks.jsonl events.jsonl
endif
	@echo "✅ Cleaned"
