FROM golang:1.18 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o telegram-bot-microservice ./cmd/server/main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/telegram-bot-microservice .

CMD ["./telegram-bot-microservice"]