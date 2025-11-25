# Testing Event API with Redis

## Quick Test Guide

### 1. Start Services

```tex

## Start PostgreSQL and Redis

## Start PostgreSQL and Redis
docker-compose up -d

## Start application

## Start application
make run
```

### 2. Test Registration (Redis Verification Code)

```tex
curl -X POST http://localhost:8080/v1/api/auth/registe
  -H "Content-Type: application/json"
  -d '{"email":"test@example.com","phone":"+79991234567","password":"test123"}'

## Response:

## Response:

## {"user":{...},"verify_code":"123456"}

## {"user":{...},"verify_code":"123456"}
```

**Verify in Redis:**

```tex
docker exec -it event_api_redis redis-cli -a devpass GET verify:test@example.com

## Returns: "123456"

## Returns: "123456"

docker exec -it event_api_redis redis-cli -a devpass TTL verify:test@example.com

## Returns: ~600 (10 minutes in seconds)

## Returns: ~600 (10 minutes in seconds)
```

### 3. Test Verification

```tex
curl -X POST http://localhost:8080/v1/api/auth/verify
  -H "Content-Type: application/json"
  -d '{"email":"test@example.com","code":"123456"}'

## Response:

## Response:

## {"access_token":"eyJ...","user":{...verified:true...}}

## {"access_token":"eyJ...","user":{...verified:true...}}
```

**Verify code deleted from Redis:**

```tex
docker exec -it event_api_redis redis-cli -a devpass EXISTS verify:test@example.com

## Returns: 0 (key deleted)

## Returns: 0 (key deleted)
```

### 4. Test Login

```tex
curl -X POST http://localhost:8080/v1/api/auth/login
  -H "Content-Type: application/json"
  -d '{"email":"test@example.com","password":"test123"}'

## Response:

## Response:

## {"access_token":"eyJ...","user":{...}}

## {"access_token":"eyJ...","user":{...}}
```

### 5. SMTP Email Integration Test (real email)

Requires a reachable SMTP server and credentials. Set the following env vars,
then run the focused test:

```tex
export SMTP_HOST="smtp.example.com"
export SMTP_PORT="587"        # or 465 for SSL
export SMTP_USERNAME="user@example.com"
export SMTP_PASSWORD="your_app_password"
export EMAIL_FROM="from@example.com"
export EMAIL_FROM_NAME="Event API"
export EMAIL_TO="recipient@example.com"

## Run only the SMTP email tes

## Run only the SMTP email tes
make test-email-smtp

## or directly

## or directly
go test -v ./internal/email -run TestSMTPIntegrationSendEmail
```

Notes:

- Port 465 uses SSL; 587 typically uses STARTTLS (handled automatically).
- Some providers require app passwords (e.g., Gmail) or SMTP enabled in account settings.
- The test is skipped if required env vars are not set.

### 5. Test Protected Endpoin

```tex
TOKEN="eyJ..."  # Your JWT token

curl -X GET http://localhost:8080/v1/api/auth/me
  -H "Authorization: Bearer $TOKEN"

## Response:

## Response:

## {"user":{...}}

## {"user":{...}}
```

### 6. Test Logout (Redis Token Blacklist)

```tex
curl -X POST http://localhost:8080/v1/api/auth/logou
  -H "Authorization: Bearer $TOKEN"

## Response:

## Response:

## {"message":"Успешно вышли из системы"}

## {"message":"Успешно вышли из системы"}
```

**Verify token in blacklist:**

```tex
docker exec -it event_api_redis redis-cli -a devpass EXISTS "blacklist:$TOKEN"

## Returns: 1 (token blacklisted)

## Returns: 1 (token blacklisted)

docker exec -it event_api_redis redis-cli -a devpass TTL "blacklist:$TOKEN"

## Returns: ~3600 (1 hour - token expiration time)

## Returns: ~3600 (1 hour - token expiration time)
```

**Try to use blacklisted token:**

```tex
curl -X GET http://localhost:8080/v1/api/auth/me
  -H "Authorization: Bearer $TOKEN"

## Response:

## Response:

## {"error":"unauthorized","message":"токен был отозван","code":401}

## {"error":"unauthorized","message":"токен был отозван","code":401}
```

### 7. Test Events CRUD

**Get all events:**

```tex
curl -X GET http://localhost:8080/v1/api/events
```

**Get specific event:**

```tex
curl -X GET http://localhost:8080/v1/api/events/{id}
```

**Create event (protected):**

```tex
curl -X POST http://localhost:8080/v1/api/events
  -H "Authorization: Bearer $NEW_TOKEN"
  -H "Content-Type: application/json"
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

```tex
curl -X DELETE http://localhost:8080/v1/api/events/{id}
  -H "Authorization: Bearer $NEW_TOKEN"
```

## Redis Key Patterns

### Verification Codes

```tex

Key Pattern: verify:{email}
Example: verify:user@example.com
Value: "123456"
TTL: 600 seconds (10 minutes)

```

### Token Blacklis

```tex

Key Pattern: blacklist:{jwt_token}
Example: blacklist:eyJhbGc...
Value: "1"
TTL: 3600 seconds (1 hour - same as token expiration)

```

## Monitoring Redis

### Connect to Redis CLI

```tex
docker exec -it event_api_redis redis-cli -a devpass
```

### Useful Commands

```tex

## List all keys

## List all keys
KEYS *

## Get all verification codes

## Get all verification codes
KEYS verify:*

## Get all blacklisted tokens

## Get all blacklisted tokens
KEYS blacklist:*

## Monitor all commands in real-time

## Monitor all commands in real-time
MONITOR

## Get database size

## Get database size
DBSIZE

## Get database info

## Get database info
INFO

## Get memory usage

## Get memory usage
INFO memory

## Get specific key info

## Get specific key info
TYPE verify:user@example.com
TTL verify:user@example.com
GET verify:user@example.com

## Delete specific key

## Delete specific key
DEL verify:user@example.com

## Flush all keys (CAUTION!)

## Flush all keys (CAUTION!)
FLUSHDB
```

## Expected Redis State

### After Registration

```tex
127.0.0.1:6379> KEYS *
1) "verify:user@example.com"

127.0.0.1:6379> GET verify:user@example.com
"123456"

127.0.0.1:6379> TTL verify:user@example.com
(integer) 598
```

### After Verification

```tex
127.0.0.1:6379> KEYS verify:*
(empty array)
```

### After Logou

```tex
127.0.0.1:6379> KEYS *
1) "blacklist:eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

127.0.0.1:6379> GET "blacklist:eyJ..."
"1"

127.0.0.1:6379> TTL "blacklist:eyJ..."
(integer) 3598
```

## Performance Testing

### Concurrent Registrations

```tex

## Install Apache Bench

## Install Apache Bench
brew install httpd  # macOS

## Test 100 concurrent requests

## Test 100 concurrent requests
ab -n 100 -c 10 -p register.json -T application/json
  http://localhost:8080/v1/api/auth/registe
```

### Redis Performance

```tex

## Connect to Redis

## Connect to Redis
docker exec -it event_api_redis redis-cli -a devpass

## Run benchmark

## Run benchmark
redis-benchmark -a devpass -q -n 10000
```

## Cleanup

### Clear Redis Data

```tex
docker exec -it event_api_redis redis-cli -a devpass FLUSHDB
```

### Clear Database

<!-- markdownlint-disable MD013 -->

```tex
docker exec -it event_api_postgres psql -U devuser -d event_api -c "TRUNCATE users, events, verification_codes CASCADE;"
```

<!-- markdownlint-enable MD013 -->

### Restart Services

```tex
docker-compose resta
make run
```

## Troubleshooting

### Redis Connection Failed

```tex

## Check Redis is running

## Check Redis is running
docker ps | grep redis

## Check Redis logs

## Check Redis logs
docker logs event_api_redis

## Test connection

## Test connection
docker exec -it event_api_redis redis-cli -a devpass ping

## Expected: PONG

## Expected: PONG
```

### Verification Code Not Found

```tex

## Check if code exists

## Check if code exists
docker exec -it event_api_redis redis-cli -a devpass GET verify:user@example.com

## Check TTL

## Check TTL
docker exec -it event_api_redis redis-cli -a devpass TTL verify:user@example.com

## If returns -2: key expired or doesn't exis

## If returns -2: key expired or doesn't exis
```

### Token Always Rejected

```tex

## Check if token is blacklisted

## Check if token is blacklisted
docker exec -it event_api_redis redis-cli -a devpass KEYS blacklist:*

## Get token from blacklis

## Get token from blacklis
docker exec -it event_api_redis redis-cli -a devpass GET "blacklist:YOUR_TOKEN"
```

## Success Indicators

✅ **Registration works** - Verification code stored in Redis with TTL ✅
**Verification works** - Code removed from Redis after verification ✅ **Logou
works** - Token added to blacklist in Redis ✅ **Blacklist works** - Blacklisted
token rejected ✅ **TTL works** - Keys automatically deleted after expiration ✅
**PostgreSQL works** - Users and events persisted in database ✅ **Swagge
works** - API documentation accessible at /swagger/index.html

## Next Steps

1. **Production Setup**: Configure Redis Sentinel for high availability
2. **Monitoring**: Add Prometheus metrics for Redis operations
3. **Caching**: Add caching layer for frequently accessed events
4. **Rate Limiting**: Implement rate limiting using Redis
5. **Session Management**: Store user sessions in Redis
