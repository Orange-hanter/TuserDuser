# Redis Quick Start

## Installation

### Using Docker Compose (Recommended)

Already set up in project's `docker-compose.yml`:

```bash
docker-compose up redis
```

Verify it's running:

```bash
redis-cli PING
# Output: PONG
```

### Using Homebrew (macOS)

```bash
brew install redis
brew services start redis

# Verify
redis-cli PING
```

### Using Docker

```bash
docker run -d -p 6379:6379 --name redis redis:7-alpine
docker exec redis redis-cli PING
```

## Configure Server

### 1. Update `.env` file

```bash
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=devpass
REDIS_DB=0

# Optional: Customize TTL (in seconds)
DISCOVERY_HISTORY_TTL=604800    # 7 days
DISCOVERY_QUEUE_TTL=2592000     # 30 days
```

### 2. Verify Configuration

```bash
# Check if Redis is accessible from your app
telnet localhost 6379
# or
redis-cli PING
```

## Start Server

```bash
cd event-api
go build -o bin/server ./cmd/server
./bin/server
```

Watch for these logs:

```
✅ Redis queue repository initialized (ttl_seconds: 2592000)
✅ Redis history repository initialized (ttl_seconds: 604800)
```

## Quick Testing

### 1. Create a Discovery Session

```bash
curl -X GET http://localhost:8080/v1/api/discovery/next \
  -H "Authorization: Bearer YOUR_TOKEN"
```

This will:

- Create a queue state in Redis
- Store action history in Redis

### 2. Check Redis

```bash
redis-cli
> KEYS "*"                           # See all keys
> GET queue:user:abc123              # Get user's queue
> LRANGE history:user:abc123 0 10    # Get history
> TTL queue:user:abc123              # Check expiration
```

### 3. Monitor Real-Time

```bash
# Terminal 1: Watch Redis keys
redis-cli MONITOR

# Terminal 2: Make API calls
curl -X GET http://localhost:8080/v1/api/discovery/next \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## Common Commands

### View Queue State

```bash
redis-cli GET queue:user:USER_ID
```

### View History (Last 5 Actions)

```bash
redis-cli LRANGE history:user:USER_ID 0 5
```

### Check Memory Usage

```bash
redis-cli INFO memory | grep used_memory_human
```

### Delete User's Data

```bash
redis-cli DEL queue:user:USER_ID
redis-cli DEL history:user:USER_ID
redis-cli DEL last-action:user:USER_ID:event:*
```

### Clear All Discovery Data

```bash
redis-cli DEL $(redis-cli KEYS "queue:user:*" | tr '\n' ' ')
redis-cli DEL $(redis-cli KEYS "history:user:*" | tr '\n' ' ')
redis-cli DEL $(redis-cli KEYS "last-action:*" | tr '\n' ' ')
```

### Monitor Live Activity

```bash
redis-cli MONITOR
```

## Troubleshooting

### "connection refused"

```bash
# Check if Redis is running
redis-cli PING

# If not running:
brew services start redis
# or
docker-compose up redis
```

### "WRONGPASS invalid password"

- Check REDIS_PASSWORD in `.env`
- Make sure it matches Redis configuration
- Default: `devpass`

### "i/o timeout"

- Check REDIS_HOST and REDIS_PORT
- Verify network connectivity
- Try: `telnet localhost 6379`

### Memory Growing Too Fast

- Check TTL values: `redis-cli TTL queue:user:*`
- Reduce `DISCOVERY_QUEUE_TTL` or `DISCOVERY_HISTORY_TTL`
- Monitor: `redis-cli INFO memory`

## Next Steps

- [Full Migration Guide](./REDIS_MIGRATION.md)
- [Redis Commands Reference](./REDIS_COMMANDS.md)
- [Production Deployment](./PRODUCTION_DEPLOYMENT.md)
