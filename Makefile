# Makefile for Go backend service

SHELL := /bin/zsh

# Paths
BACKEND_DIR := Support/Backend
BIN_DIR := bin
APP_NAME := tuserduser-backend
BIN := $(BIN_DIR)/$(APP_NAME)
ABS_BIN := $(abspath $(BIN))

# Go settings
GO ?= go
PORT ?= 8080

.PHONY: all build rebuild run run-dev clean tidy vet fmt test lint help

all: build

help:
	@echo "Common targets:"
	@echo "  make build   - Build the backend binary into $(BIN)"
	@echo "  make run     - Run the backend from source"
	@echo "  make tidy    - Sync and tidy Go modules"
	@echo "  make vet     - Run go vet"
	@echo "  make fmt     - Format code (go fmt)"
	@echo "  make test    - Run unit tests"
	@echo "  make lint    - Run basic linters (fmt + vet)"
	@echo "  make clean   - Remove build artifacts"

build:
	@mkdir -p $(BIN_DIR)
	$(GO) -C $(BACKEND_DIR) build -o $(ABS_BIN) ./
	@echo "Built $(BIN)"

rebuild: clean build

run: build
	PORT=$(PORT) $(BIN)

run-dev:
	cd $(BACKEND_DIR) && $(GO) run .

tidy:
	cd $(BACKEND_DIR) && $(GO) mod tidy

vet:
	cd $(BACKEND_DIR) && $(GO) vet ./...

fmt:
	cd $(BACKEND_DIR) && $(GO) fmt ./...

test:
	cd $(BACKEND_DIR) && $(GO) test ./...

lint: fmt vet

clean:
	rm -f $(BIN)
	rm -f Support/bin/$(APP_NAME)

install-service:
	@echo "Build binary and run contrib/install_backend.sh as root on the server."
	@echo "Example (on server): sudo ./contrib/install_backend.sh"

uninstall-service:
	@echo "To uninstall run on server as root: sudo ./contrib/uninstall_backend.sh"
