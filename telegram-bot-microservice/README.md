# Telegram Bot Microservice

This project is a microservice that functions as a Telegram bot, processing messages from Telegram and sending them to users. The existing Telegram logic has been migrated into this separate service, which communicates with a PostgreSQL database via gRPC requests.

## Project Structure

```
telegram-bot-microservice
├── cmd
│   └── server
│       └── main.go          # Entry point of the microservice
├── api
│   └── proto
│       └── bot.proto        # gRPC service and message types
├── internal
│   ├── server
│   │   └── grpc_server.go   # gRPC server implementation
│   ├── adapters
│   │   └── telegram_adapter.go # Logic for interacting with the Telegram API
│   ├── service
│   │   └── bot_service.go    # Core business logic for the bot
│   ├── repository
│   │   └── postgres.go       # PostgreSQL database interactions
│   └── config
│       └── config.go        # Configuration settings
├── migrations
│   └── 0001_init.sql        # SQL script for initializing the database schema
├── configs
│   └── config.yaml          # Configuration settings in YAML format
├── scripts
│   └── build.sh             # Shell script for building the microservice
├── docker
│   └── Dockerfile           # Instructions for building a Docker image
├── tests
│   ├── unit
│   │   └── service_test.go   # Unit tests for the bot service
│   └── integration
│       └── grpc_integration_test.go # Integration tests for the gRPC server
├── Makefile                  # Build instructions and commands
├── go.mod                    # Module dependencies
├── .golangci.yml             # Configuration for GolangCI-Lint
└── README.md                 # Project documentation
```

## Setup Instructions

1. **Clone the repository:**

   ```
   git clone <repository-url>
   cd telegram-bot-microservice
   ```

2. **Install dependencies:**

   ```
   go mod tidy
   ```

3. **Build the project:**

   ```
   ./scripts/build.sh
   ```

4. **Run the microservice:**

   ```
   go run cmd/server/main.go
   ```

5. **Database Migration:**
   Ensure that the PostgreSQL database is set up and run the migration script:
   ```
   psql -U <username> -d <database> -f migrations/0001_init.sql
   ```

## Usage

Once the microservice is running, it will listen for incoming messages from Telegram. The bot processes these messages and interacts with users based on the defined logic in the `bot_service.go` file.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request for any improvements or bug fixes.

## License

This project is licensed under the MIT License. See the LICENSE file for details.
