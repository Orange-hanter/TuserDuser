# Telegram Service

Standalone gRPC microservice for Telegram Bot API integration. This service handles all Telegram-related functionality, isolating the core application from Telegram specifics.

## Architecture

```
┌─────────────────┐         gRPC          ┌───────────────────┐
│    event-api    │◄─────────────────────►│  telegram-service │
│  (core service) │                       │                   │
└─────────────────┘                       └─────────┬─────────┘
                                                    │
                                          ┌─────────▼─────────┐
                                          │  Telegram Bot API │
                                          └───────────────────┘
```

## Features

- **User Binding**: Generate deep links for users to connect their Telegram accounts
- **Message Sending**: Send verification codes, event reminders, and custom messages
- **Webhook Processing**: Handle incoming Telegram updates (commands, etc.)
- **Long Polling Mode**: Works without static IP using polling instead of webhooks
- **Failure Handling**: Automatic user blocking detection (403), rate limiting (429)
- **Metrics**: Prometheus metrics for monitoring

## Update Modes

The service supports two modes for receiving Telegram updates:

### Webhook Mode (default)

Best for production with static IP/domain. Telegram pushes updates to your server.

```bash
TELEGRAM_UPDATE_MODE=webhook
TELEGRAM_WEBHOOK_SECRET=your-secret
```

### Polling Mode

Best for development or when you don't have a static IP. The service polls Telegram for updates.

```bash
TELEGRAM_UPDATE_MODE=polling
TELEGRAM_POLLING_TIMEOUT=30        # Long polling timeout in seconds
TELEGRAM_POLLING_RETRY_DELAY=3     # Retry delay on error
```

**Note**: When switching to polling mode, the service automatically deletes any existing webhook.

## gRPC API

### Service: `telegram.v1.TelegramService`

| Method                 | Description                                  |
| ---------------------- | -------------------------------------------- |
| `GenerateBindingLink`  | Create a deep link for user Telegram binding |
| `SendVerificationCode` | Send a verification code to a bound user     |
| `SendEventReminder`    | Send an event reminder notification          |
| `SendMessage`          | Send a custom message                        |
| `IsUserBound`          | Check if user has active Telegram binding    |
| `GetBindingStatus`     | Get detailed binding information             |
| `UnbindUser`           | Remove user's Telegram binding               |

## Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL 15+
- Docker & Docker Compose (optional)

### Local Development

1. Copy environment file:

```bash
cp .env.example .env
```

2. Configure your `.env` with Telegram bot credentials.

3. Start the database:

```bash
docker-compose up -d telegram-db
```

4. Run the service:

```bash
make run
```

### Docker Deployment

```bash
# Build and start all services
docker-compose up -d

# View logs
docker-compose logs -f telegram-service
```

## Configuration

| Variable                       | Description                          | Default   |
| ------------------------------ | ------------------------------------ | --------- |
| `GRPC_PORT`                    | gRPC server port                     | `50051`   |
| `HTTP_PORT`                    | HTTP server port (webhooks, metrics) | `8081`    |
| `DATABASE_URL`                 | PostgreSQL connection string         | -         |
| `TELEGRAM_BOT_TOKEN`           | Telegram Bot API token               | -         |
| `TELEGRAM_BOT_USERNAME`        | Bot username (without @)             | -         |
| `TELEGRAM_UPDATE_MODE`         | `webhook` or `polling`               | `webhook` |
| `TELEGRAM_WEBHOOK_SECRET`      | Webhook secret for validation        | -         |
| `TELEGRAM_WEBHOOK_ALIAS`       | URL path segment for webhook         | `default` |
| `TELEGRAM_POLLING_TIMEOUT`     | Long polling timeout (seconds)       | `30`      |
| `TELEGRAM_POLLING_RETRY_DELAY` | Retry delay on error (seconds)       | `3`       |
| `TELEGRAM_BINDING_SECRET`      | HMAC secret for token signing        | -         |
| `TELEGRAM_BINDING_TTL`         | Token validity in seconds            | `3600`    |

## Database Schema

The service maintains its own database with the following tables:

- `telegram_bindings` - User to Telegram chat mappings
- `telegram_binding_tokens` - Single-use binding tokens
- `telegram_webhook_events` - Audit log of incoming webhooks
- `telegram_delivery` - Message delivery tracking

## Endpoints

### HTTP

| Endpoint                             | Description               |
| ------------------------------------ | ------------------------- |
| `POST /webhooks/telegram/{botAlias}` | Telegram webhook receiver |
| `GET /health`                        | Health check              |
| `GET /metrics`                       | Prometheus metrics        |

### Metrics

- `telegram_messages_total{status,reason}` - Message send counter
- `telegram_bindings_total{status}` - Binding state changes
- `telegram_binding_links_generated_total` - Link generation counter
- `telegram_webhook_requests_total{status}` - Webhook request counter
- `telegram_grpc_request_duration_seconds{method,status}` - gRPC latency

## Integration with event-api

Add to `event-api` configuration:

```env
TELEGRAM_SERVICE_ENABLED=true
TELEGRAM_SERVICE_ADDRESS=localhost:50051
TELEGRAM_SERVICE_TIMEOUT=1s
```

Use the gRPC client:

```go
import "event-api/internal/telegramclient"

client, err := telegramclient.NewClient(telegramclient.Config{
    Address: "localhost:50051",
    Timeout: time.Second,
}, logger)

// Generate binding link
result, err := client.GenerateBindingLink(ctx, userID)

// Send verification code
result, err := client.SendVerificationCode(ctx, userID, "123456", 5)

// Check if user is bound
isBound, status, err := client.IsUserBound(ctx, userID)
```

## Webhook Setup

1. Set up a public URL for your webhook (e.g., via ngrok for development)
2. Register webhook with Telegram:

```bash
curl -X POST "https://api.telegram.org/bot<TOKEN>/setWebhook" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://your-domain.com/webhooks/telegram/default",
    "secret_token": "your_webhook_secret"
  }'
```

## Development

```bash
# Install tools
make tools

# Run tests
make test

# Run linter
make lint

# Generate protobuf code (requires protoc)
make proto

# Test gRPC endpoint
make grpc-test
```

## Error Handling

The service uses machine-readable error codes:

| Error Code            | Description                               |
| --------------------- | ----------------------------------------- |
| `invalid_user_id`     | User ID is missing or invalid             |
| `user_not_bound`      | User doesn't have active Telegram binding |
| `blocked`             | User has blocked the bot                  |
| `rate_limited`        | Rate limited by Telegram                  |
| `send_failed`         | Failed to send message                    |
| `service_unavailable` | Service is unavailable                    |

## License

Internal use only.
