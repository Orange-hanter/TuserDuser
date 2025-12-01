# Redis Integration - Migration Guide

## Overview

Redis integration provides persistent storage for Discovery queue states and hot history data, enabling:

- ✅ Persistence across server restarts
- ✅ Horizontal scaling (multiple server instances)
- ✅ Hot data caching with TTL
- ✅ Graceful fallback to in-memory if Redis unavailable

## Architecture

### Before Redis

```
┌─ Server 1 ─┐    ┌─ Server 2 ─┐
│ Mem Queue  │    │ Mem Queue  │
│ Mem History│    │ Mem History│
└────────────┘    └────────────┘
   (isolated)        (isolated)
       ▼                 ▼
    PostgreSQL (persistence only)
```

### After Redis

```
┌─ Server 1 ─┐    ┌─ Server 2 ─┐
│  Engine    │    │  Engine    │
└─────┬──────┘    └──────┬─────┘
      │                  │
      └──────┬───────────┘
             ▼
      ┌─────────────┐
      │    Redis    │  ← Shared queue state + hot history
      └─────────────┘
             ▼
      ┌─────────────┐
      │ PostgreSQL  │  ← Analytics persistence
      └─────────────┘
```

## Configuration

### Environment Variables

```bash
# Redis Connection
REDIS_HOST=localhost              # Default: localhost
REDIS_PORT=6379                   # Default: 6379
REDIS_PASSWORD=devpass            # Default: empty
REDIS_DB=0                        # Default: 0

# Discovery Redis TTL (in seconds)
DISCOVERY_HISTORY_TTL=604800      # 7 days (default: 7*24*3600)
DISCOVERY_QUEUE_TTL=2592000       # 30 days (default: 30*24*3600)
```

### Default .env

```
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=devpass
REDIS_DB=0
DISCOVERY_HISTORY_TTL=604800
DISCOVERY_QUEUE_TTL=2592000
```

## Repositories

### RedisQueueRepository

- **Purpose**: Stores per-user queue state (current event, Primary queue, Secondary queue)
- **Key Format**: `queue:user:{userID}`
- **TTL**: 30 days (configurable)
- **Type**: String (JSON serialized)

Example data:

```json
{
  "user_id": "abc123",
  "filter": {"category": "music"},
  "primary": ["event1", "event2", ...],
  "secondary": ["event5", "event6", ...],
  "conflict_flags": {...}
}
```

### RedisHistoryRepository

- **Purpose**: Hot history cache for recent user actions
- **Key Formats**:
  - `history:user:{userID}` — Redis List (newest first, max 100 entries)
  - `last-action:user:{userID}:event:{eventID}` — Last action for event
- **TTL**: 7 days (configurable)
- **Operations**: O(1) for all operations

Example operations:

```
LPUSH history:user:abc123 {...}     # Add new action (newest first)
LRANGE history:user:abc123 0 -1     # Get all (O(1) for limited list)
LTRIM history:user:abc123 0 99      # Keep only 100 entries
```

## Fallback Behavior

If Redis is unavailable:

```go
if redis != nil {
    // Use Redis repositories
    discoveryQueueRepo = NewRedisQueueRepository(...)
    discoveryHistoryRepo = NewRedisHistoryRepository(...)
} else {
    // Fallback to in-memory
    discoveryQueueRepo = NewInMemoryQueueRepository()
    discoveryHistoryRepo = NewPostgresHistoryRepository(...) // PostgreSQL fallback
    logger.Warn("Redis not available, using in-memory repositories")
}
```

### Graceful Degradation

- ✅ Continues working without Redis
- ✅ Uses in-memory queue (lost on restart)
- ✅ Falls back to PostgreSQL history
- ⚠️ No persistence across restarts
- ⚠️ No horizontal scaling

## Migration Steps

### 1. Update Configuration

Add Redis environment variables to `.env`:

```bash
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=devpass
DISCOVERY_HISTORY_TTL=604800
DISCOVERY_QUEUE_TTL=2592000
```

### 2. Ensure Redis is Running

```bash
docker-compose up redis
# or
brew services start redis
```

### 3. Rebuild & Restart Server

```bash
cd event-api
go build -o bin/server ./cmd/server
./bin/server
```

### 4. Monitor Redis

```bash
# Check connected clients
redis-cli info clients

# Monitor queue storage
redis-cli KEYS "queue:user:*" | wc -l

# Check memory usage
redis-cli INFO memory
```

## Performance Characteristics

| Operation      | In-Memory   | Redis          | PostgreSQL     |
| -------------- | ----------- | -------------- | -------------- |
| Get Queue      | O(1)        | O(1) + network | O(1) + network |
| Save Queue     | O(1)        | O(1) + network | O(1) + network |
| Get History    | O(n) scan   | O(1) LRANGE    | O(log n) index |
| Last Action    | O(n) search | O(1) GET       | O(1) index     |
| Server Restart | ❌ Lost     | ✅ Persisted   | ✅ Persisted   |
| Multi-Instance | ❌ Isolated | ✅ Shared      | ✅ Shared      |

## Monitoring & Debugging

### Check Queue Storage

```bash
redis-cli
> KEYS "queue:user:*"       # List all user queues
> GET queue:user:abc123     # Get specific queue
> TTL queue:user:abc123     # Check TTL (seconds)
> DEL queue:user:abc123     # Delete queue
```

### Check History Storage

```bash
redis-cli
> KEYS "history:user:*"                    # List all histories
> LLEN history:user:abc123                 # Count entries
> LRANGE history:user:abc123 0 10          # Get first 10 entries
> TTL history:user:abc123                  # Check TTL
> GET last-action:user:abc123:event:ev1   # Get last action for event
```

### Memory Usage

```bash
redis-cli INFO memory
# Check used_memory_human, peak memory usage
```

### Clear All Data

```bash
redis-cli
> FLUSHDB           # Clear current Redis DB
> FLUSHALL          # Clear all DBs (careful!)
```

## Production Considerations

### 1. Enable Persistence

Configure RDB or AOF in `redis.conf`:

```conf
# RDB Snapshot
save 900 1          # Save after 900s if 1+ keys changed
save 300 10
save 60 10000

# or AOF (Append-Only File)
appendonly yes
appendfsync everysec
```

### 2. Replication

For High Availability, set up Redis Sentinel or Cluster:

```conf
# Redis Sentinel monitors primary and handles failover
sentinel monitor mymaster 127.0.0.1 6379 1
sentinel down-after-milliseconds mymaster 5000
sentinel failover-timeout mymaster 30000
```

### 3. Memory Management

Configure max memory policy:

```conf
maxmemory 1gb
maxmemory-policy allkeys-lru  # Evict least-recently-used keys
```

### 4. Monitoring

Use Prometheus + Grafana with Redis exporter:

```bash
docker run -p 9121:9121 oliver006/redis_exporter
```

## Troubleshooting

### Redis Not Available

Error: `redis get error: i/o timeout`

- Check Redis is running: `redis-cli PING`
- Check network connectivity
- Check credentials/password
- Verify REDIS_HOST, REDIS_PORT, REDIS_PASSWORD

### Memory Issues

- Monitor `redis-cli INFO memory`
- Reduce TTL values if memory grows
- Enable maxmemory eviction policy

### Data Persistence Issues

- Verify RDB/AOF is enabled
- Check disk space
- Monitor `redis-cli INFO persistence`

## See Also

- [Redis Quick Start](./REDIS_QUICKSTART.md)
- [Redis Commands Reference](./REDIS_COMMANDS.md)
