# RBAC Quick Start Guide

## Initial Setup

### 1. Run Database Migrations

The migration `005_add_role_to_users` will automatically run on server start,
adding the `role` column to existing users with a default value of `'user'`.

````bash
## Start the server - migrations run automatically
## Start the server - migrations run automatically
./server
```bash
### 2. Default Admin Seeding (Automatic & Idempotent)

The application automatically ensures there is at least one admin user after
migrations. If no admin exists, it will create a default admin user. This
process is idempotent: if an admin already exists, nothing is changed.

- Defaults (can be overridden using env vars):
  - `ADMIN_EMAIL` → default `admin@example.com`
  - `ADMIN_PHONE` → default `+70000000000`
  - `ADMIN_PASSWORD` → if not set, a strong random password is generated and printed once to logs on startup

Set these before starting the server to control the seeded admin:

```bash
export ADMIN_EMAIL="admin@yourdomain.com"
export ADMIN_PHONE="+10000000000"
export ADMIN_PASSWORD="Strong_Admin_Passw0rd!"
./server
````

If you prefer manual promotion instead of auto-seeding, you can still do it via
SQL:

````sql
UPDATE users SET role = 'admin' WHERE email = 'your-admin@example.com';
```bash
### 3. Verify Admin Access

```bash
## Login as admin
## Login as admin
curl -X POST http://localhost:8080/v1/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "your-admin@example.com",
    "password": "your-password"
  }'

## You should receive a JWT token with "role": "admin" in the response
## You should receive a JWT token with "role": "admin" in the response
````

## Managing User Roles

### Grant Creator Role

Only admins can grant creator or support roles:

````bash
curl -X PUT http://localhost:8080/v1/api/admin/users/role \
  -H "Authorization: Bearer <admin_jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "uuid-of-user",
    "role": "creator"
  }'
```bash
### List All Users

```bash
curl -X GET http://localhost:8080/v1/api/admin/users \
  -H "Authorization: Bearer <admin_jwt_token>"
````

## Testing Role Permissions

### As Regular User (Default)

````bash
## Can view events
## Can view events
curl http://localhost:8080/v1/api/events

## Cannot create events (403 Forbidden)
## Cannot create events (403 Forbidden)
curl -X POST http://localhost:8080/v1/api/events \
  -H "Authorization: Bearer <user_jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{"type": "test"}'
```bash
### As Creator

```bash
## Can create events (goes to pending for admin review)
## Can create events (goes to pending for admin review)
curl -X POST http://localhost:8080/v1/api/events \
  -H "Authorization: Bearer <creator_jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "meetup",
    "start": "2025-12-01T18:00:00Z",
    "end": "2025-12-01T20:00:00Z",
    "duration": 120,
    "place": "Online",
    "priceType": "free",
    "needReg": false
  }'

## Cannot review events (403 Forbidden)
## Cannot review events (403 Forbidden)
curl -X GET http://localhost:8080/v1/api/events/pending \
  -H "Authorization: Bearer <creator_jwt_token>"
````

### As Admin

````bash
## View pending events
## View pending events
curl -X GET http://localhost:8080/v1/api/events/pending \
  -H "Authorization: Bearer <admin_jwt_token>"

## Approve event
## Approve event
curl -X POST http://localhost:8080/v1/api/events/{event-id}/review \
  -H "Authorization: Bearer <admin_jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "action": "approve"
  }'

## Reject event (requires comment)
## Reject event (requires comment)
curl -X POST http://localhost:8080/v1/api/events/{event-id}/review \
  -H "Authorization: Bearer <admin_jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "action": "reject",
    "comment": "Does not meet quality standards"
  }'
```bash
## Role Hierarchy

```text
admin          → Full system access
  ↓
creator        → Can create events + user permissions
  ↓
user (default) → Read-only access to approved events
  ↓
support        → Reserved for future features
````

## Common Scenarios

### Scenario 1: User Wants to Create Events

1. User registers → Gets `user` role by default
2. User contacts admin or requests creator access
3. Admin grants `creator` role via API
4. User can now create events (pending admin approval)

### Scenario 2: Event Moderation Workflow

1. Creator creates event → Stored in `events_pending` table
2. Admin views pending events via API
3. Admin approves → Event moves to `events` table (publicly visible)
4. OR Admin rejects → Event moves to `events_rejected` table

### Scenario 3: Multi-Admin Setup

1. First admin promotes another user to admin
2. Both admins can now manage roles and moderate events
3. No limit on number of admins

## Troubleshooting

### "Forbidden" Error When Accessing Endpoint

- Check that JWT token is included in `Authorization: Bearer <token>` header
- Verify token hasn't expired
- Confirm user has required role by decoding JWT at <https://jwt.io>
- Check role field in JWT claims

### User Role Not Updated After API Call

- Ensure you're using admin credentials
- User needs to login again to get new JWT with updated role
- Old tokens still have old role until they expire

### Cannot Create First Admin

- Use direct database query to promote first user
- Ensure user exists before trying to update role
- Check that migration `005_add_role_to_users` ran successfully

## Security Notes

- Roles are embedded in JWT tokens - users must re-authenticate after role changes
- JWT tokens expire based on `JWT_EXPIRATION` config setting
- Role field is validated at database level with CHECK constraint
- Admin role should be granted sparingly
- Consider implementing 2FA for admin accounts in production

### Notes About Admin Seeding

- The seeding runs only after all migrations complete.
- If `ADMIN_PASSWORD` is not provided, a random password is generated and logged with a warning. Change it immediately after first login.
- If `ADMIN_EMAIL` collides with an existing user but still no admin exists, the system tries a fallback email like `admin+<suffix>@local` and logs it.
