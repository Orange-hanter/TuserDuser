.PHONY: build run test docker

build:
	go build -o bin/server ./cmd/server

run: build
	./bin/server

test:
	go test ./...

docker-build:
	docker build -t event-api .

docker-run: docker-build
	docker run -p 8080:8080 --env-file .env event-api
