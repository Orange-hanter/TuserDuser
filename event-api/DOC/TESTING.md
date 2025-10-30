# Testing Event API with Redis

## Quick Test Guide

### 1. Start Services

```bash
# Start PostgreSQL and Redis
docker-compose up -d

# Start application
make run
```

### 2. Test Registration (Redis Verification Code)

```bash
curl -X POST http://localhost:8080/v1/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","phone":"+79991234567","password":"test123"}'

# Response:
# {"user":{...},"verify_code":"123456"}
```

**Verify in Redis:**
```bash
docker exec -it event_api_redis redis-cli -a devpass GET verify:test@example.com
# Returns: "123456"

docker exec -it event_api_redis redis-cli -a devpass TTL verify:test@example.com
# Returns: ~600 (10 minutes in seconds)
```

### 3. Test Verification

```bash
curl -X POST http://localhost:8080/v1/api/auth/verify \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","code":"123456"}'

# Response:
# {"access_token":"eyJ...","user":{...verified:true...}}
```

**Verify code deleted from Redis:**
```bash
docker exec -it event_api_redis redis-cli -a devpass EXISTS verify:test@example.com
# Returns: 0 (key deleted)
```

### 4. Test Login

```bash
curl -X POST http://localhost:8080/v1/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"test123"}'

# Response:
# {"access_token":"eyJ...","user":{...}}
```

### 5. Test Protected Endpoint

```bash
TOKEN="eyJ..."  # Your JWT token

curl -X GET http://localhost:8080/v1/api/auth/me \
  -H "Authorization: Bearer $TOKEN"

# Response:
# {"user":{...}}
```

### 6. Test Logout (Redis Token Blacklist)

```bash
curl -X POST http://localhost:8080/v1/api/auth/logout \
  -H "Authorization: Bearer $TOKEN"

# Response:
# {"message":"Успешно вышли из системы"}
```

**Verify token in blacklist:**
```bash
docker exec -it event_api_redis redis-cli -a devpass EXISTS "blacklist:$TOKEN"
# Returns: 1 (token blacklisted)

docker exec -it event_api_redis redis-cli -a devpass TTL "blacklist:$TOKEN"
# Returns: ~3600 (1 hour - token expiration time)
```

**Try to use blacklisted token:**
```bash
curl -X GET http://localhost:8080/v1/api/auth/me \
  -H "Authorization: Bearer $TOKEN"

# Response:
# {"error":"unauthorized","message":"токен был отозван","code":401}
```

### 7. Test Events CRUD

**Get all events:**
```bash
curl -X GET http://localhost:8080/v1/api/events
```

**Get specific event:**
```bash
curl -X GET http://localhost:8080/v1/api/events/{id}
```

**Create event (protected):**
```bash
curl -X POST http://localhost:8080/v1/api/events \
  -H "Authorization: Bearer $NEW_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type":"Митап",
    "start":"2025-12-01T19:00:00Z",
    "end":"2025-12-01T21:00:00Z",
    "duration":120,
    "place":"Офис",
    "price_type":"free",
    "need_registration":true,
    "details":{"topic":"Go Programming"}
  }'
```

**Delete event (protected):**
```bash
curl -X DELETE http://localhost:8080/v1/api/events/{id} \
  -H "Authorization: Bearer $NEW_TOKEN"
```

## Redis Key Patterns

### Verification Codes
```
Key Pattern: verify:{email}
Example: verify:user@example.com
Value: "123456"
TTL: 600 seconds (10 minutes)
```

### Token Blacklist
```
Key Pattern: blacklist:{jwt_token}
Example: blacklist:eyJhbGc...
Value: "1"
TTL: 3600 seconds (1 hour - same as token expiration)
```

## Monitoring Redis

### Connect to Redis CLI
```bash
docker exec -it event_api_redis redis-cli -a devpass
```

### Useful Commands

```bash
# List all keys
KEYS *

# Get all verification codes
KEYS verify:*

# Get all blacklisted tokens
KEYS blacklist:*

# Monitor all commands in real-time
MONITOR

# Get database size
DBSIZE

# Get database info
INFO

# Get memory usage
INFO memory

# Get specific key info
TYPE verify:user@example.com
TTL verify:user@example.com
GET verify:user@example.com

# Delete specific key
DEL verify:user@example.com

# Flush all keys (CAUTION!)
FLUSHDB
```

## Expected Redis State

### After Registration
```bash
127.0.0.1:6379> KEYS *
1) "verify:user@example.com"

127.0.0.1:6379> GET verify:user@example.com
"123456"

127.0.0.1:6379> TTL verify:user@example.com
(integer) 598
```

### After Verification
```bash
127.0.0.1:6379> KEYS verify:*
(empty array)
```

### After Logout
```bash
127.0.0.1:6379> KEYS *
1) "blacklist:eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

127.0.0.1:6379> GET "blacklist:eyJ..."
"1"

127.0.0.1:6379> TTL "blacklist:eyJ..."
(integer) 3598
```

## Performance Testing

### Concurrent Registrations
```bash
# Install Apache Bench
brew install httpd  # macOS

# Test 100 concurrent requests
ab -n 100 -c 10 -p register.json -T application/json \
  http://localhost:8080/v1/api/auth/register
```

### Redis Performance
```bash
# Connect to Redis
docker exec -it event_api_redis redis-cli -a devpass

# Run benchmark
redis-benchmark -a devpass -q -n 10000
```

## Cleanup

### Clear Redis Data
```bash
docker exec -it event_api_redis redis-cli -a devpass FLUSHDB
```

### Clear Database
```bash
docker exec -it event_api_postgres psql -U devuser -d event_api -c "TRUNCATE users, events, verification_codes CASCADE;"
```

### Restart Services
```bash
docker-compose restart
make run
```

## Troubleshooting

### Redis Connection Failed
```bash
# Check Redis is running
docker ps | grep redis

# Check Redis logs
docker logs event_api_redis

# Test connection
docker exec -it event_api_redis redis-cli -a devpass ping
# Expected: PONG
```

### Verification Code Not Found
```bash
# Check if code exists
docker exec -it event_api_redis redis-cli -a devpass GET verify:user@example.com

# Check TTL
docker exec -it event_api_redis redis-cli -a devpass TTL verify:user@example.com
# If returns -2: key expired or doesn't exist
```

### Token Always Rejected
```bash
# Check if token is blacklisted
docker exec -it event_api_redis redis-cli -a devpass KEYS blacklist:*

# Get token from blacklist
docker exec -it event_api_redis redis-cli -a devpass GET "blacklist:YOUR_TOKEN"
```

## Success Indicators

✅ **Registration works** - Verification code stored in Redis with TTL
✅ **Verification works** - Code removed from Redis after verification
✅ **Logout works** - Token added to blacklist in Redis
✅ **Blacklist works** - Blacklisted token rejected
✅ **TTL works** - Keys automatically deleted after expiration
✅ **PostgreSQL works** - Users and events persisted in database
✅ **Swagger works** - API documentation accessible at /swagger/index.html

## Next Steps

1. **Production Setup**: Configure Redis Sentinel for high availability
2. **Monitoring**: Add Prometheus metrics for Redis operations
3. **Caching**: Add caching layer for frequently accessed events
4. **Rate Limiting**: Implement rate limiting using Redis
5. **Session Management**: Store user sessions in Redis
