# Redis Commands Reference

## Key Patterns Used in Discovery

```
queue:user:{userID}                    - Queue state for user
history:user:{userID}                  - Action history list
last-action:user:{userID}:event:{eventID}  - Last action for event
```

## Basic Commands

### Connection

```bash
redis-cli              # Connect to Redis
redis-cli PING         # Check connection
redis-cli INFO         # Server info
```

### Keys Operations

```bash
KEYS pattern           # Find all keys matching pattern
DEL key [key ...]      # Delete one or more keys
EXISTS key [key ...]   # Check if keys exist
TTL key                # Get time to live (seconds)
EXPIRE key seconds     # Set expiration
PERSIST key            # Remove expiration
TYPE key               # Get key type
RANDOMKEY              # Get random key
SCAN cursor [MATCH pattern] [COUNT count]  # Scan keys iteratively
```

## Queue Storage (String)

### Get Queue

```bash
GET queue:user:abc123
# Returns JSON string with queue state
```

### Save Queue

```bash
SET queue:user:abc123 '{"user_id":"abc123",...}' EX 2592000
# EX: expiration in seconds (30 days)
```

### Delete Queue

```bash
DEL queue:user:abc123
```

### Check Expiration

```bash
TTL queue:user:abc123
# Returns: -1 (no expiration), -2 (key doesn't exist), or seconds remaining
```

## History Storage (List)

### Add Action (Append)

```bash
LPUSH history:user:abc123 '{"action":"like","event_id":"ev1",...}'
# LPUSH adds to HEAD (newest first)
```

### Get All History

```bash
LRANGE history:user:abc123 0 -1
# 0 = start, -1 = end (all entries)
```

### Get First 10 Actions

```bash
LRANGE history:user:abc123 0 9
```

### Get Count

```bash
LLEN history:user:abc123
```

### Trim to Keep Last 100

```bash
LTRIM history:user:abc123 0 99
# Keeps entries 0-99, deletes rest
```

### Remove All History

```bash
DEL history:user:abc123
```

## Last Action Storage (String)

### Get Last Action for Event

```bash
GET last-action:user:abc123:event:ev1
# Returns: JSON string with last action
```

### Update Last Action

```bash
SET last-action:user:abc123:event:ev1 '{"action":"like",...}' EX 604800
```

### Remove

```bash
DEL last-action:user:abc123:event:ev1
```

## Batch Operations

### Get All User Queues

```bash
KEYS "queue:user:*" | wc -l
```

### Get All User Histories

```bash
KEYS "history:user:*"
```

### Delete All Discovery Data

```bash
SCAN 0 MATCH "queue:user:*" COUNT 100   # Gets batches of 100
SCAN 0 MATCH "history:user:*" COUNT 100
SCAN 0 MATCH "last-action:*" COUNT 100
```

Or in one command:

```bash
redis-cli FLUSHDB  # Clear entire database (careful!)
```

## Expiration Management

### Set TTL

```bash
EXPIRE queue:user:abc123 2592000
# Set to expire in 30 days
```

### Persist (Remove TTL)

```bash
PERSIST queue:user:abc123
```

### Check Remaining TTL

```bash
TTL queue:user:abc123
# -1 = never expires
# -2 = doesn't exist
# N = seconds remaining
```

### Set with TTL (One Command)

```bash
SET queue:user:abc123 value EX 2592000
# EX = expire seconds
# PX = expire milliseconds
```

## Transaction Commands

### Begin Transaction

```bash
MULTI
SET key1 value1
SET key2 value2
EXEC
```

### Watch for Changes

```bash
WATCH key1 key2
MULTI
SET key1 value1
EXEC
```

## Monitoring & Debugging

### Monitor All Commands

```bash
MONITOR
```

### Get Server Info

```bash
INFO
INFO memory        # Memory usage
INFO stats         # Statistics
INFO replication   # Replication info
INFO keyspace      # Database sizes
```

### Memory Analysis

```bash
INFO memory | grep used_memory_human
INFO memory | grep peak_memory_human
```

### Check Database Size

```bash
DBSIZE
```

### Get Command Stats

```bash
INFO commandstats
```

## Python/Go Client Examples

### Get Queue (Go)

```go
data, err := redisClient.Get(ctx, "queue:user:abc123").Result()
var state QueueState
json.Unmarshal([]byte(data), &state)
```

### Save Queue (Go)

```go
data, _ := json.Marshal(state)
redisClient.Set(ctx, "queue:user:abc123", data, 30*24*time.Hour)
```

### Append History (Go)

```go
entry, _ := json.Marshal(historyEntry)
redisClient.LPush(ctx, "history:user:abc123", entry)
redisClient.LTrim(ctx, "history:user:abc123", 0, 99)
```

## Performance Tips

### Use SCAN Instead of KEYS

```bash
# Bad (blocks Redis for large datasets):
KEYS pattern

# Good (non-blocking, iterative):
SCAN cursor MATCH pattern COUNT 100
```

### Batch Operations with Pipeline

```bash
# Multiple commands in one round trip
redis.Pipeline() {
    SET key1 value1
    SET key2 value2
    SET key3 value3
}
```

### Monitor Memory

```bash
MEMORY USAGE key
MEMORY DOCTOR
MEMORY HELP
```

## Cleanup Commands

### Delete All Keys of Pattern

```bash
SCAN 0 MATCH "queue:user:*" COUNT 100 | while read key; do DEL $key; done
```

### Flush Database (WARNING!)

```bash
FLUSHDB      # Clear current DB
FLUSHALL     # Clear all DBs
```

### Clear Expired Keys

```bash
# Redis automatically removes expired keys
# Force cleanup if needed:
redis-cli --rdb /tmp/dump.rdb  # Create RDB snapshot (triggers cleanup)
```

## Persistence Commands

### Save Database

```bash
SAVE      # Synchronous save (blocks)
BGSAVE    # Background save
BGREWRITEAOF  # Optimize AOF file
```

### Check Last Save

```bash
LASTSAVE
```

## See Also

- [Redis Quick Start](./REDIS_QUICKSTART.md)
- [Redis Migration Guide](./REDIS_MIGRATION.md)
