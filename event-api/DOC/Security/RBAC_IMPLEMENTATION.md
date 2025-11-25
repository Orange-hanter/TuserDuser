# Role-Based Access Control (RBAC) Implementation Summary

## Overview

Successfully implemented a comprehensive role-based access control system with
four user roles: `admin`, `creator`, `user`, and `support`.

## Roles and Permissions

### User (Default Role)

- **Permissions:**
  - View published events (`events.read`)
  - Interact with events (view, like, dislike)
- **API Access:**
  - `GET /api/events` - View all approved events
  - `GET /api/events/{id}` - View specific event details

### Creator

- **Permissions:**
  - All user permissions
  - Create events (`events.create`)
  - Update own events (`events.update_own`)
  - Delete own events (`events.delete_own`)
- **API Access:**
  - All user endpoints
  - `POST /api/events` - Create new events (pending review)
  - `DELETE /api/events/{id}` - Delete own events

### Support

- **Permissions:**
  - View events (`events.read`)
  - View users (`users.read`)
- **Status:** Reserved for future functionality
- **API Access:**
  - Currently same as regular users
  - Role exists for future feature extension

### Admin

- **Permissions:**
  - All system permissions (`*`)
  - Full access to all endpoints
- **Capabilities:**
  - Review and approve/reject pending events
  - Grant `creator` and `support` roles to users
  - Manage all users
  - Full event management
- **API Access:**
  - All creator endpoints
  - `GET /api/events/pending` - View events awaiting moderation
  - `POST /api/events/{id}/review` - Approve or reject events
  - `GET /api/admin/users` - List all users
  - `PUT /api/admin/users/role` - Update user roles

## Implementation Details

### 1. Database Schema

**Migration: `005_add_role_to_users`**

````sql
ALTER TABLE users
ADD COLUMN IF NOT EXISTS role VARCHAR(20) NOT NULL DEFAULT 'user'
CHECK (role IN ('user', 'creator', 'support', 'admin'));

CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
```bash
### 2. User Model Updates

**File:** `internal/models/auth.go`

- Added role constants: `RoleUser`, `RoleCreator`, `RoleSupport`, `RoleAdmin`
- Extended `User` struct with `Role` field
- Updated `Claims` struct to include role in JWT tokens

### 3. RBAC Middleware

**File:** `internal/middleware/rbac.go`

- `RequireRole(...roles)` - Generic role checker
- `RequireAdmin(handler)` - Admin-only access
- `RequireCreatorOrAdmin(handler)` - Creator or admin access
- `HasPermission(role, permission)` - Permission checking utility
- `getRolePermissions(role)` - Returns permissions for each role

### 4. Authentication Updates

**File:** `internal/middleware/auth.go`

- Extracts role from JWT claims
- Sets `X-User-Role` header for downstream middleware
- Defaults to `user` role if not specified in token

### 5. Service Layer

**File:** `internal/service/auth.go`

- `UpdateUserRole(userID, role)` - Change user role (admin only)
- `GetAllUsers()` - List all users (admin only)
- Updated registration to assign default `user` role
- JWT generation includes role claim
- All user queries include role field

### 6. Handler Layer

**File:** `internal/handlers/auth.go`

- `UpdateUserRole` - Admin endpoint to change roles
- `GetAllUsers` - Admin endpoint to list users
- Updated `AuthService` interface with role management methods

### 7. Route Protection

**File:** `cmd/server/main.go`

```go
// Public endpoints (no authentication)
r.Get("/api/events", eventHandler.GetApprovedEvents)

// Authenticated users
authenticated := r.With(middleware.AuthMiddleware(authService))
authenticated.Get("/api/auth/me", authHandler.GetMe)

// Creator or Admin only
creatorOrAdmin := authenticated.With(middleware.RequireCreatorOrAdmin)
creatorOrAdmin.Post("/api/events", eventHandler.CreateEvent)

// Admin only
adminOnly := authenticated.With(middleware.RequireAdmin)
adminOnly.Get("/api/events/pending", eventHandler.GetPendingEvents)
adminOnly.Post("/api/events/{id}/review", eventHandler.ReviewPendingEvent)
adminOnly.Get("/api/admin/users", authHandler.GetAllUsers)
adminOnly.Put("/api/admin/users/role", authHandler.UpdateUserRole)
````

## API Endpoints by Role

| Endpoint                  | Method | User | Creator | Support | Admin |
| ------------------------- | ------ | ---- | ------- | ------- | ----- |
| `/api/events`             | GET    | ✅   | ✅      | ✅      | ✅    |
| `/api/events/{id}`        | GET    | ✅   | ✅      | ✅      | ✅    |
| `/api/auth/me`            | GET    | ✅   | ✅      | ✅      | ✅    |
| `/api/events`             | POST   | ❌   | ✅      | ❌      | ✅    |
| `/api/events/{id}`        | DELETE | ❌   | ✅      | ❌      | ✅    |
| `/api/events/pending`     | GET    | ❌   | ❌      | ❌      | ✅    |
| `/api/events/{id}/review` | POST   | ❌   | ❌      | ❌      | ✅    |
| `/api/admin/users`        | GET    | ❌   | ❌      | ❌      | ✅    |
| `/api/admin/users/role`   | PUT    | ❌   | ❌      | ❌      | ✅    |

## Testing

### Unit Tests

- **Handlers:** `internal/handlers/auth_test.go` - Mock-based handler tests
- **Middleware:** `internal/middleware/rbac_test.go` - Role checking tests
- **Event Handlers:** `internal/handlers/event_test.go` - Event API tests

### Test Coverage

✅ Role-based access control enforcement ✅ Permission checking for each role ✅
JWT token includes role claim ✅ Middleware properly extracts and validates roles
✅ Handler tests updated with role management methods

## Usage Examples

### 1. Register New User (Default: user role)

````bash
curl -X POST http://localhost:8080/v1/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "phone": "+1234567890",
    "password": "securepass123"
  }'
```bash
### 2. Admin Grants Creator Role

```bash
curl -X PUT http://localhost:8080/v1/api/admin/users/role \
  -H "Authorization: Bearer <admin_jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user-uuid",
    "role": "creator"
  }'
````

### 3. Creator Creates Event

````bash
curl -X POST http://localhost:8080/v1/api/events \
  -H "Authorization: Bearer <creator_jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "meetup",
    "start": "2025-12-01T18:00:00Z",
    "end": "2025-12-01T20:00:00Z",
    "duration": 120,
    "place": "Conference Room A",
    "priceType": "free",
    "needReg": true
  }'
```bash
### 4. Admin Reviews Pending Event

```bash
curl -X POST http://localhost:8080/v1/api/events/{event-id}/review \
  -H "Authorization: Bearer <admin_jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "action": "approve",
    "comment": "Looks good!"
  }'
````

## Security Features

1. **JWT Token Security:** Role stored in JWT claims, verified on each request
2. **Middleware Chain:** Auth → Role check → Handler
3. **Database Constraints:** CHECK constraint ensures only valid roles
4. **Default Safe:** New users get minimal `user` role by default
5. **Admin Privilege:** Only admins can modify user roles
6. **Endpoint Protection:** Sensitive endpoints require appropriate roles

## Migration Path

To update an existing system:

1. Run database migrations: `005_add_role_to_users`
2. Existing users automatically get `user` role
3. Manually promote first admin via direct database update:

   ```sql
   UPDATE users SET role = 'admin' WHERE email = 'admin@example.com';
   ```

4. Admin can then promote other users through API

## Future Enhancements

- [ ] Support role functionality definition
- [ ] Fine-grained permissions per resource
- [ ] Role hierarchy and inheritance
- [ ] Audit logging for role changes
- [ ] Rate limiting per role
- [ ] Custom role creation
