.PHONY: build run test test-verbose test-coverage test-race lint fmt vet docker docker-build docker-run logger-demo docker-up docker-down docker-logs ci-test deploy backup restore deploy-nginx deploy-nginx-interactive deploy-env deploy-env-check deploy-env-no-restart logs-access-staging logs-error-staging logs-access-api logs-error-api logs-nginx-access logs-nginx-error logs-all-errors logs-follow-api logs-follow-staging test-email test-email-smtp

build:
	go build -o bin/server ./cmd/server

run: build
	./bin/server

test:
	go test ./...

test-verbose:
	go test -v ./...

test-race:
	go test -v -race ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-coverage-report:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# CI тестирование (как в GitHub Actions)
ci-test:
	@echo "🧪 Running CI tests..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out

# Линтинг и форматирование
lint:
	@echo "🔍 Running golangci-lint..."
	golangci-lint run --timeout=5m

fmt:
	@echo "✨ Formatting code..."
	gofmt -s -w .

vet:
	@echo "🔍 Running go vet..."
	go vet ./...

check: fmt vet lint
	@echo "✅ All checks passed!"

swagger:
	go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/server/main.go -o docs

logger-demo:
	go run ./cmd/logger-demo

# Docker Compose команды
docker-up:
	docker-compose up -d
	@echo "✅ PostgreSQL запущен на localhost:5432"
	@echo "✅ pgAdmin доступен на http://localhost:5050"

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f postgres

docker-restart:
	docker-compose restart

docker-clean:
	docker-compose down -v
	@echo "⚠️  Все данные БД удалены"

# Запуск приложения с Docker БД
dev: docker-up
	@echo "⏳ Ожидание запуска PostgreSQL..."
	@sleep 3
	@echo "🚀 Запуск приложения..."
	@make run

docker-build:
	docker build -t event-api .

docker-run: docker-build
	docker run -p 8080:8080 --env-file .env event-api

# Production Docker Compose
prod-up:
	docker-compose -f docker-compose.prod.yml up -d
	@echo "✅ Production stack запущен"

prod-down:
	docker-compose -f docker-compose.prod.yml down

prod-logs:
	docker-compose -f docker-compose.prod.yml logs -f app

prod-restart:
	docker-compose -f docker-compose.prod.yml restart app

# Деплой, бэкапы и восстановление
deploy:
	@echo "🚀 Deploying to production..."
	./scripts/deploy.sh production

deploy-staging:
	@echo "🚀 Deploying to staging..."
	./scripts/deploy.sh staging

backup:
	@echo "📦 Creating database backup..."
	./scripts/backup.sh

restore:
	@echo "🔄 Restoring database..."
	@echo "Usage: make restore FILE=backups/your_backup.sql.gz"
	@if [ -z "$(FILE)" ]; then \
		echo "❌ Error: FILE parameter is required"; \
		echo "Example: make restore FILE=backups/event_api_backup_20250104_120000.sql.gz"; \
		exit 1; \
	fi
	./scripts/restore.sh $(FILE)

# Nginx configuration deployment
# Деплой конфигурации nginx
REMOTE_HOST ?= 5.35.127.138
REMOTE_USER ?= root
SSH_KEY ?= ~/.ssh/deployer_eventapi_root
NGINX_SOURCE ?= nginx/nginx-staging.conf
NGINX_DEST ?= /etc/nginx/sites-available/staging
SITE_NAME ?= staging

deploy-nginx:
	@echo "🚀 Deploying nginx configuration..."
	@./scripts/deploy-nginx.sh --host $(REMOTE_HOST) --user $(REMOTE_USER) --key $(SSH_KEY) --source $(NGINX_SOURCE) --dest $(NGINX_DEST) --site $(SITE_NAME) --auto

deploy-nginx-interactive:
	@echo "🚀 Deploying nginx configuration (interactive mode)..."
	@./scripts/deploy-nginx.sh --host $(REMOTE_HOST) --user $(REMOTE_USER) --key $(SSH_KEY) --source $(NGINX_SOURCE) --dest $(NGINX_DEST) --site $(SITE_NAME)

# Просмотр логов nginx на удаленном сервере
# Remote nginx logs viewing
LOG_LINES ?= 50

logs-access-staging:
	@./scripts/view-nginx-logs.sh --host $(REMOTE_HOST) --user $(REMOTE_USER) --key $(SSH_KEY) --type access-staging --lines $(LOG_LINES)

logs-error-staging:
	@./scripts/view-nginx-logs.sh --host $(REMOTE_HOST) --user $(REMOTE_USER) --key $(SSH_KEY) --type error-staging --lines $(LOG_LINES)

logs-access-api:
	@./scripts/view-nginx-logs.sh --host $(REMOTE_HOST) --user $(REMOTE_USER) --key $(SSH_KEY) --type access-api --lines $(LOG_LINES)

logs-error-api:
	@./scripts/view-nginx-logs.sh --host $(REMOTE_HOST) --user $(REMOTE_USER) --key $(SSH_KEY) --type error-api --lines $(LOG_LINES)

logs-nginx-access:
	@./scripts/view-nginx-logs.sh --host $(REMOTE_HOST) --user $(REMOTE_USER) --key $(SSH_KEY) --type nginx-access --lines $(LOG_LINES)

logs-nginx-error:
	@./scripts/view-nginx-logs.sh --host $(REMOTE_HOST) --user $(REMOTE_USER) --key $(SSH_KEY) --type nginx-error --lines $(LOG_LINES)

logs-all-errors:
	@./scripts/view-nginx-logs.sh --host $(REMOTE_HOST) --user $(REMOTE_USER) --key $(SSH_KEY) --type all-errors --lines $(LOG_LINES)

logs-follow-api:
	@./scripts/view-nginx-logs.sh --host $(REMOTE_HOST) --user $(REMOTE_USER) --key $(SSH_KEY) --type access-api --follow

logs-follow-staging:
	@./scripts/view-nginx-logs.sh --host $(REMOTE_HOST) --user $(REMOTE_USER) --key $(SSH_KEY) --type access-staging --follow

# Configuration Deployment
deploy-env:
	@./scripts/deploy-env.sh --host $(REMOTE_HOST) --user $(REMOTE_USER) --key $(SSH_KEY)

# Configuration Deployment
deploy-env-staging:
	@./scripts/deploy-env.sh --host $(REMOTE_HOST) --user $(REMOTE_USER) --key $(SSH_KEY) -s ./.env.staging

deploy-env-check:
	@./scripts/deploy-env.sh --host $(REMOTE_HOST) --user $(REMOTE_USER) --key $(SSH_KEY) --dry-run

deploy-env-no-restart:
	@./scripts/deploy-env.sh --host $(REMOTE_HOST) --user $(REMOTE_USER) --key $(SSH_KEY) --no-restart

# Email testing helpers
test-email:
	@./scripts/test-email.sh

test-email-smtp:
	@echo "🧪 Running SMTP integration test (requires SMTP_HOST, SMTP_PORT, EMAIL_FROM, EMAIL_TO)"
	go test -v ./internal/email -run TestSMTPIntegrationSendEmail

# Полная очистка
clean:
	@echo "🧹 Cleaning up..."
	rm -rf bin/
	rm -rf coverage.out coverage.html
	go clean -cache -testcache
	@echo "✅ Clean complete"

# Помощь
help:
	@echo "📋 Available targets:"
	@echo ""
	@echo "  Build & Run:"
	@echo "    make build              - Build the application"
	@echo "    make run                - Build and run the application"
	@echo "    make dev                - Start Docker services and run app"
	@echo ""
	@echo "  Testing:"
	@echo "    make test               - Run tests"
	@echo "    make test-verbose       - Run tests with verbose output"
	@echo "    make test-race          - Run tests with race detector"
	@echo "    make test-coverage      - Generate coverage report (HTML)"
	@echo "    make ci-test            - Run tests as in CI pipeline"
	@echo "    make test-email-smtp    - Run SMTP integration email test (requires env vars)"
	@echo "    make test-email         - Trigger mock email via register endpoint"
	@echo ""
	@echo "  Code Quality:"
	@echo "    make lint               - Run golangci-lint"
	@echo "    make fmt                - Format code with gofmt"
	@echo "    make vet                - Run go vet"
	@echo "    make check              - Run all checks (fmt, vet, lint)"
	@echo ""
	@echo "  Docker Development:"
	@echo "    make docker-up          - Start PostgreSQL & Redis"
	@echo "    make docker-down        - Stop Docker services"
	@echo "    make docker-logs        - View PostgreSQL logs"
	@echo "    make docker-build       - Build Docker image"
	@echo "    make docker-run         - Build and run in Docker"
	@echo ""
	@echo "  Production:"
	@echo "    make prod-up            - Start production stack"
	@echo "    make prod-down          - Stop production stack"
	@echo "    make prod-logs          - View production logs"
	@echo "    make deploy             - Deploy to production"
	@echo "    make deploy-staging     - Deploy to staging"
	@echo ""
	@echo "  Database:"
	@echo "    make backup             - Create database backup"
	@echo "    make restore FILE=...   - Restore from backup"
	@echo ""
	@echo "  Nginx Deployment:"
	@echo "    make deploy-nginx              - Deploy nginx config (auto)"
	@echo "    make deploy-nginx-interactive  - Deploy nginx config (with confirmation)"
	@echo ""
	@echo "  Remote Nginx Logs:"
	@echo "    make logs-access-staging    - View staging access logs"
	@echo "    make logs-error-staging     - View staging error logs"
	@echo "    make logs-access-api        - View API access logs"
	@echo "    make logs-error-api         - View API error logs"
	@echo "    make logs-nginx-access      - View main nginx access logs"
	@echo "    make logs-nginx-error       - View main nginx error logs"
	@echo "    make logs-all-errors        - View all error logs"
	@echo "    make logs-follow-api        - Follow API logs in real-time"
	@echo "    make logs-follow-staging    - Follow staging logs in real-time"
	@echo ""
	@echo "  Configuration Deployment:"
	@echo "    make deploy-env             - Deploy .env to server (with backup & restart)"
	@echo "    make deploy-env-check       - Check what would be deployed (dry-run)"
	@echo "    make deploy-env-no-restart  - Deploy .env without restarting service"
	@echo ""
	@echo "  Logs Options (use with LOG_LINES=N):"
	@echo "    make logs-access-api LOG_LINES=100"
	@echo ""
	@echo "  Other:"
	@echo "    make clean              - Clean build artifacts"
	@echo "    make swagger            - Generate Swagger docs"
	@echo "    make help               - Show this help"

