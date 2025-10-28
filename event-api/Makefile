.PHONY: build run test test-verbose test-coverage docker docker-build docker-run logger-demo docker-up docker-down docker-logs

build:
	go build -o bin/server ./cmd/server

run: build
	./bin/server

test:
	go test ./...

test-verbose:
	go test -v ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-coverage-report:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

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
	docker run -p 8080:8080 --env-file .env event-api
