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

.PHONY: all build build-linux build-linux-strip rebuild run run-dev clean tidy vet fmt test lint help

all: build

help:
	@echo "Common targets:"
	@echo "  make build   - Build the backend binary into $(BIN)"
	@echo "  make build-linux - Cross-compile linux/amd64 binary into $(BIN)-linux-amd64"
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

# Cross-compile for linux/amd64 (static)
LINUX_BIN := $(BIN_DIR)/$(APP_NAME)-linux-amd64

.PHONY: build-linux build-linux-strip
build-linux:
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) -C $(BACKEND_DIR) build -o $(abspath $(LINUX_BIN)) ./
	@echo "Built $(LINUX_BIN)"

build-linux-strip: build-linux
	@if command -v strip >/dev/null 2>&1; then \
		strip $(LINUX_BIN) || true; \
		echo "Stripped $(LINUX_BIN)"; \
	else \
		echo "strip not found; skipping"; \
	fi

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
