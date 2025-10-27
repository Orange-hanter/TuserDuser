.PHONY: build run test test-verbose test-coverage docker docker-build docker-run

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

docker-build:
	docker build -t event-api .

docker-run: docker-build
	docker run -p 8080:8080 --env-file .env event-api
