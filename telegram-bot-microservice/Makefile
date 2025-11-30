.PHONY: build run test clean

build:
	go build -o telegram-bot-microservice ./cmd/server

run: build
	./telegram-bot-microservice

test:
	go test ./...

clean:
	go clean
	rm -f telegram-bot-microservice

proto:
	protoc --go_out=. --go-grpc_out=. ./api/proto/bot.proto

migrate:
	migrate -path ./migrations -database "postgres://user:password@localhost:5432/dbname?sslmode=disable" up

docker:
	docker build -t telegram-bot-microservice .