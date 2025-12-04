# Telegram Service

Standalone gRPC microservice for Telegram Bot API integration. This service handles all Telegram-related functionality, isolating the core application from Telegram specifics.

## Architecture

```
┌─────────────────┐         gRPC          ┌───────────────────┐       HTTP        ┌───────────────────┐
│    event-api    │◄─────────────────────►│  telegram-service │◄─────────────────►│  Telegram Bot API │
│  (core service) │       :50051          │                   │                   │                   │
└─────────────────┘                       └─────────┬─────────┘                   └───────────────────┘
                                                    │
                                          ┌─────────▼─────────┐
                                          │   PostgreSQL DB   │
                                          │  (telegram-db)    │
                                          └───────────────────┘
```

### Communication Flow

1. **event-api → telegram-service**: gRPC calls for all Telegram operations
2. **telegram-service → Telegram API**: HTTP requests to Bot API
3. **Telegram → telegram-service**: Webhook or polling for incoming messages
4. **telegram-service → PostgreSQL**: Binding storage, tokens, delivery tracking

## Features

- **User Binding**: Generate deep links and short codes for users to connect their Telegram accounts
- **Short Code System**: 6-character alphanumeric codes as alternative to deep links
- **Message Sending**: Send verification codes, event reminders, and custom messages
- **Webhook Processing**: Handle incoming Telegram updates (commands, etc.)
- **Long Polling Mode**: Works without static IP using polling instead of webhooks
- **Failure Handling**: Automatic user blocking detection (403), rate limiting (429)
- **Metrics**: Prometheus metrics for monitoring
- **gRPC with JSON Codec**: No protobuf compilation required

## User Binding System

The service provides two methods for binding Telegram accounts to users:

### Method 1: Deep Links

Standard Telegram deep links that open the bot directly:

```
https://t.me/YourBotName?start=TOKEN
```

**Limitation**: Deep links don't pass the `start` parameter for users who have previously interacted with the bot.

### Method 2: Short Codes (Recommended)

6-character alphanumeric codes that users can send directly to the bot:

```
Code: A3X9K2
User sends: A3X9K2 to @YourBotName
```

**Advantages**:

- Works for existing bot conversations
- Easy to share via any medium
- Case-insensitive matching

### Binding Flow

```
┌─────────┐     1. POST /link     ┌───────────┐    2. gRPC     ┌──────────────────┐
│  Client │────────────────────►│ event-api │────────────────►│ telegram-service │
│  (App)  │                      │           │                 │                  │
└─────────┘                      └───────────┘                 └─────────┬────────┘
     │                                                                   │
     │   3. Response: {token, deeplink, code, expires_at}               │
     │◄──────────────────────────────────────────────────────────────────┘
     │
     │   4a. User clicks deep link OR
     │   4b. User sends short code to bot
     │
     ▼
┌────────────────┐    5. Update    ┌──────────────────┐
│  Telegram Bot  │───────────────►│ telegram-service │
│                │                 │ (webhook/polling)│
└────────────────┘                 └─────────┬────────┘
                                             │
                                   6. Save binding to DB
                                             │
                                   7. Send confirmation to user
```

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

The service uses a custom JSON codec for gRPC communication, eliminating the need for protobuf compilation.

| Method                 | Description                                                 |
| ---------------------- | ----------------------------------------------------------- |
| `GenerateBindingLink`  | Create a deep link and short code for user Telegram binding |
| `SendVerificationCode` | Send a verification code to a bound user                    |
| `SendEventReminder`    | Send an event reminder notification                         |
| `SendMessage`          | Send a custom message                                       |
| `IsUserBound`          | Check if user has active Telegram binding                   |
| `GetBindingStatus`     | Get detailed binding information                            |
| `UnbindUser`           | Remove user's Telegram binding                              |

### Method Details

#### GenerateBindingLink

Creates a binding token, deep link URL, and 6-character short code.

**Request:**

```json
{
  "user_id": "uuid-string"
}
```

**Response:**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "deep_link": "https://t.me/BotName?start=TOKEN",
  "code": "A3X9K2",
  "expires_at": "2025-01-15T12:00:00Z"
}
```

#### SendVerificationCode

Sends a formatted verification code message to the user's bound Telegram chat.

**Request:**

```json
{
  "user_id": "uuid-string",
  "code": "123456",
  "ttl_minutes": 5
}
```

**Response:**

```json
{
  "success": true,
  "message_id": 12345
}
```

#### SendEventReminder

Sends an event reminder with an inline keyboard button.

**Request:**

```json
{
  "user_id": "uuid-string",
  "event_id": "event-uuid",
  "event_title": "Team Meeting",
  "event_time": "2025-01-15T14:00:00Z",
  "event_url": "https://app.example.com/events/123"
}
```

**Response:**

```json
{
  "success": true,
  "message_id": 12346
}
```

#### SendMessage

Sends a custom text message with optional Markdown/HTML formatting.

**Request:**

```json
{
  "user_id": "uuid-string",
  "text": "Hello, *world*!",
  "parse_mode": "Markdown"
}
```

**Response:**

```json
{
  "success": true,
  "message_id": 12347
}
```

#### IsUserBound

Checks if a user has an active Telegram binding.

**Request:**

```json
{
  "user_id": "uuid-string"
}
```

**Response:**

```json
{
  "is_bound": true,
  "status": "active"
}
```

#### GetBindingStatus

Returns detailed information about a user's Telegram binding.

**Request:**

```json
{
  "user_id": "uuid-string"
}
```

**Response:**

```json
{
  "status": "active",
  "chat_id": 123456789,
  "username": "johndoe",
  "first_name": "John",
  "last_name": "Doe",
  "updated_at": "2025-01-15T10:30:00Z"
}
```

#### UnbindUser

Removes the Telegram binding for a user.

**Request:**

```json
{
  "user_id": "uuid-string",
  "reason": "user requested"
}
```

**Response:**

```json
{
  "success": true
}
```

## Bot Commands

The service handles the following Telegram bot commands:

| Command        | Description                                     |
| -------------- | ----------------------------------------------- |
| `/start`       | Show welcome message and available commands     |
| `/start TOKEN` | Complete account binding with token (deep link) |
| `/unsubscribe` | Remove Telegram binding                         |
| `/help`        | Show help message                               |
| `6-CHAR CODE`  | Complete account binding with short code        |

### Command Flow Examples

**Binding via Deep Link:**

```
User clicks: https://t.me/BotName?start=eyJhbGciOiJIUzI1NiIs...
Bot receives: /start eyJhbGciOiJIUzI1NiIs...
Bot responds: ✅ Аккаунт успешно привязан!
```

**Binding via Short Code:**

```
User sends: A3X9K2
Bot checks if matches 6-char pattern
Bot responds: ✅ Аккаунт успешно привязан!
```

**Unsubscribe:**

```
User sends: /unsubscribe
Bot removes binding
Bot responds: ✅ Вы успешно отписались от уведомлений.
```

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

**Important: DNS Configuration**

When running in Docker, the container needs external DNS to resolve `api.telegram.org`. Add this to your `docker-compose.yml`:

```yaml
telegram-service:
  dns:
    - 8.8.8.8
    - 8.8.4.4
```

Without this, the service may fail to connect to Telegram API.

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
| `TELEGRAM_RATE_LIMIT`          | Rate limit requests per second       | `30`      |

## Database Schema

The service maintains its own database with the following tables:

### telegram_bindings

Stores user-to-Telegram-chat mappings.

| Column       | Type         | Description                   |
| ------------ | ------------ | ----------------------------- |
| `id`         | UUID         | Primary key                   |
| `user_id`    | UUID         | Application user ID (unique)  |
| `chat_id`    | BIGINT       | Telegram chat ID (unique)     |
| `username`   | VARCHAR(255) | Telegram username             |
| `first_name` | VARCHAR(255) | Telegram first name           |
| `last_name`  | VARCHAR(255) | Telegram last name            |
| `status`     | VARCHAR(50)  | active, blocked, unsubscribed |
| `created_at` | TIMESTAMPTZ  | Creation timestamp            |
| `updated_at` | TIMESTAMPTZ  | Last update timestamp         |

### telegram_binding_tokens

Single-use tokens for account binding.

| Column       | Type        | Description                |
| ------------ | ----------- | -------------------------- |
| `id`         | UUID        | Primary key                |
| `user_id`    | UUID        | Application user ID        |
| `token`      | TEXT        | JWT token for deep link    |
| `code`       | VARCHAR(6)  | Short alphanumeric code    |
| `used`       | BOOLEAN     | Whether token was consumed |
| `expires_at` | TIMESTAMPTZ | Token expiration time      |
| `created_at` | TIMESTAMPTZ | Creation timestamp         |

### telegram_delivery

Message delivery tracking for audit and retry logic.

| Column       | Type        | Description             |
| ------------ | ----------- | ----------------------- |
| `id`         | UUID        | Primary key             |
| `user_id`    | UUID        | Recipient user ID       |
| `message_id` | BIGINT      | Telegram message ID     |
| `status`     | VARCHAR(50) | sent, failed, blocked   |
| `error`      | TEXT        | Error message if failed |
| `created_at` | TIMESTAMPTZ | Send timestamp          |

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

## Troubleshooting

### Deep link doesn't work for existing users

**Problem**: When a user who has previously chatted with the bot clicks a deep link (`t.me/bot?start=TOKEN`), Telegram doesn't pass the `start` parameter.

**Solution**: Use the short code system instead. The `GenerateBindingLink` response includes a `code` field containing a 6-character code that users can send directly to the bot.

### Cannot connect to Telegram API from Docker

**Problem**: Service logs show `dial tcp: lookup api.telegram.org: no such host`.

**Solution**: Add DNS configuration to your docker-compose.yml:

```yaml
telegram-service:
  dns:
    - 8.8.8.8
```

### gRPC connection refused

**Problem**: `connection refused` errors when connecting from event-api.

**Solution**:

1. Ensure telegram-service is running: `docker-compose ps`
2. Check the address format: `telegram-service:50051` (Docker) or `localhost:50051` (local)
3. Verify gRPC port is exposed in docker-compose.yml

### User binding fails with "chat_id already bound"

**Problem**: A Telegram account is already bound to another user.

**Solution**: The service automatically handles this by unbinding the previous user when the same chat_id attempts to bind to a new user_id.

### Messages not delivered (status: blocked)

**Problem**: User blocked the bot, messages return 403 Forbidden.

**Solution**: The service automatically updates binding status to `blocked`. User must unblock the bot and re-initiate binding.

## Project Structure

```
telegram-service/
├── cmd/
│   └── server/
│       └── main.go           # Entry point
├── internal/
│   ├── config/
│   │   └── config.go         # Configuration management
│   ├── database/
│   │   ├── pool.go           # PostgreSQL connection pool
│   │   └── store.go          # Database operations
│   ├── grpcserver/
│   │   ├── server.go         # gRPC service implementation
│   │   └── types.go          # Request/response types
│   ├── poller/
│   │   └── poller.go         # Long-polling implementation
│   ├── service/
│   │   ├── telegram_service.go  # Core business logic
│   │   └── token_encoder.go     # JWT token handling
│   ├── telegram/
│   │   └── client.go         # Telegram Bot API client
│   └── webhook/
│       └── handler.go        # Webhook HTTP handler
├── migrations/
│   └── *.sql                 # Database migrations
├── docker-compose.yml
├── Dockerfile
└── Makefile
```

## License

Internal use only.
