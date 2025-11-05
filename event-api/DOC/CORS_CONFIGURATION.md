# CORS Configuration Guide

## Overview

Cross-Origin Resource Sharing (CORS) is configured to allow API calls from specified frontend domains.

## Current Configuration

### Allowed Origins

The following origins are allowed to make requests to the API:

```text
- https://api.tuserduser.online     (API domain itself)
- https://tuserduser.online          (Production frontend)
- https://www.tuserduser.online      (WWW variant)
- http://localhost:3000              (Local development - React/Vue/Next.js)
- http://localhost:8080              (Local development - alternative port)
```

## HTTP Headers

### Request Headers Accepted

```text
- Accept
- Authorization
- Content-Type
- X-CSRF-Token
- X-Requested-With
```

### Response Headers Exposed

```text
- Content-Length
- X-Json-Response
```

## CORS Methods

Allowed HTTP methods for CORS requests:

```text
- GET       (Retrieve data)
- POST      (Create data)
- PUT       (Update data)
- DELETE    (Remove data)
- PATCH     (Partial update)
- OPTIONS   (Preflight requests)
```

## Configuration Files

### Server-side (.env)

Location: `/opt/event-api/.env`

```bash
CORS_ALLOWED_ORIGINS=https://api.tuserduser.online,https://tuserduser.online,https://www.tuserduser.online,http://localhost:3000,http://localhost:8080
```

### Application Code

Location: `cmd/server/main.go`

```go
c := cors.New(cors.Options{
    AllowedOrigins:   cfg.CORSAllowedOrigins,
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
    AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Requested-With"},
    ExposedHeaders:   []string{"Content-Length", "X-JSON-Response"},
    AllowCredentials: true,
    MaxAge:           3600, // 1 hour
})
```

## How to Test CORS

### 1. Preflight Request (OPTIONS)

```bash
curl -i -X OPTIONS https://api.tuserduser.online/v1/api/health \
  -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: GET"
```

Expected response headers:

```text
Access-Control-Allow-Origin: http://localhost:3000
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, PATCH, OPTIONS
Access-Control-Allow-Credentials: true
Access-Control-Max-Age: 3600
```

### 2. Actual Request (GET)

```bash
curl -i https://api.tuserduser.online/health \
  -H "Origin: http://localhost:3000"
```

### 3. Actual Request (POST)

```bash
curl -i -X POST https://api.tuserduser.online/v1/api/auth/register \
  -H "Origin: http://localhost:3000" \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","phone":"+79991234567","password":"password123"}'
```

Expected response headers:

```text
Access-Control-Allow-Origin: http://localhost:3000
Access-Control-Allow-Credentials: true
Access-Control-Expose-Headers: Content-Length, X-Json-Response
```

## Frontend Usage

### JavaScript/TypeScript

```javascript
// Fetch API
fetch("https://api.tuserduser.online/v1/api/auth/register", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
  },
  credentials: "include", // Important: send credentials
  body: JSON.stringify({
    email: "user@example.com",
    phone: "+79991234567",
    password: "password123",
  }),
})
  .then((response) => response.json())
  .then((data) => console.log(data))
  .catch((error) => console.error("Error:", error));
```

### Axios

```javascript
import axios from "axios";

const apiClient = axios.create({
  baseURL: "https://api.tuserduser.online/v1/api",
  withCredentials: true, // Important: include credentials
  headers: {
    "Content-Type": "application/json",
  },
});

apiClient
  .post("/auth/register", {
    email: "user@example.com",
    phone: "+79991234567",
    password: "password123",
  })
  .then((response) => console.log(response.data))
  .catch((error) => console.error(error));
```

### React

```jsx
useEffect(() => {
  fetch("https://api.tuserduser.online/v1/api/health", {
    credentials: "include", // Important for CORS with cookies/auth
  })
    .then((res) => res.json())
    .then((data) => setHealth(data))
    .catch((err) => console.error(err));
}, []);
```

## Adding New Origins

To add a new allowed origin, update the `.env` file on the server:

```bash
ssh tuser

# Edit the .env file
sudo nano /opt/event-api/.env

# Find and update CORS_ALLOWED_ORIGINS
# Example - add https://myapp.com
CORS_ALLOWED_ORIGINS=https://api.tuserduser.online,https://tuserduser.online,https://www.tuserduser.online,https://myapp.com,http://localhost:3000,http://localhost:8080

# Save and restart the service
sudo systemctl restart event-api

# Verify
curl -i https://api.tuserduser.online/health -H "Origin: https://myapp.com" | grep -i "access-control"
```

## Common CORS Issues & Solutions

### Issue: "No 'Access-Control-Allow-Origin' header"

**Cause**: Origin not in allowed list

**Solution**:

1. Check the `CORS_ALLOWED_ORIGINS` in `.env`
2. Ensure origin matches exactly (protocol, domain, port)
3. Restart the service: `sudo systemctl restart event-api`

### Issue: "Access-Control-Allow-Credentials: false"

**Cause**: `AllowCredentials` is false

**Solution**:

- Check that `AllowCredentials: true` is set in CORS config
- Update and redeploy if needed

### Issue: "Preflight request rejected"

**Cause**: Method or headers not allowed

**Solution**:

1. Check `AllowedMethods` includes your method (GET, POST, PUT, DELETE, PATCH)
2. Check `AllowedHeaders` includes required headers
3. Update code/config if needed and redeploy

### Issue: Authentication token not being sent

**Cause**: CORS credentials not configured in frontend

**Solution**: Add `credentials: 'include'` to fetch or axios requests

## Security Considerations

- ✅ **HTTPS required** for production origins
- ✅ **Specific origins** are whitelisted (not `*`)
- ✅ **Credentials allowed** only with specific origins
- ✅ **Methods limited** to necessary ones
- ✅ **Headers validated** for security

## Monitoring

Check CORS errors in logs:

```bash
# View recent logs
ssh tuser "sudo tail -f /opt/event-api/logs/event-api.log" | grep -i cors

# Or check HTTP access logs
ssh tuser "sudo tail -f /var/log/nginx/event-api-access.log"
```

## Troubleshooting

### Test specific origin

```bash
# Test if origin is allowed
curl -i https://api.tuserduser.online/health \
  -H "Origin: YOUR_ORIGIN_HERE" | grep -i "access-control-allow-origin"

# Should see: Access-Control-Allow-Origin: YOUR_ORIGIN_HERE
```

### View current CORS config on server

```bash
ssh tuser "grep CORS_ALLOWED /opt/event-api/.env"
```

### Verify service has new config

```bash
ssh tuser "sudo systemctl restart event-api && sleep 2 && curl -i https://api.tuserduser.online/v1/api/health -H 'Origin: http://localhost:3000' | grep -i access-control"
```

## References

- [MDN - CORS](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)
- [rs/cors Documentation](https://github.com/rs/cors)
