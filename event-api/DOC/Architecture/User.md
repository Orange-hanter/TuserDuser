I have implemented the requested RESTful endpoints for authenticated users, including the necessary database migrations, service logic, and handler wiring.

### Summary of Changes

1.  **Data Models (`internal/models/user_profile.go`)**: Defined structures for `UserProfile`, `TelegramInfo`, `EventSubscription`, and `EventWithSubscription` to ensure type-safe and structured JSON responses.
2.  **Database Migration (migrations.go)**: Added a new migration `007_create_event_subscriptions` to create the `event_subscriptions` table for tracking user participation.
3.  **Discovery Engine Update (`internal/discovery/`)**: Added `RegisterBooking` method to the discovery engine to handle conflict resolution for external subscriptions (side-effect requirement).
4.  **User Service (`internal/service/user.go`)**: Implemented `UserService` to handle:
    - Fetching user profiles with Telegram info.
    - Retrieving upcoming and historical events.
    - Handling subscriptions with capacity checks and conflict resolution.
5.  **User Handler (`internal/handlers/user.go`)**: Created HTTP handlers for the new endpoints, enforcing authentication via `X-User-ID` header (populated by middleware).
6.  **Server Wiring (main.go)**: Registered the new `UserService` and `UserHandler`, and mapped the endpoints to `/v1/api/users/...`.

### Endpoints Implemented

| Method | Endpoint                                       | Description                                                          |
| :----- | :--------------------------------------------- | :------------------------------------------------------------------- |
| `GET`  | `/v1/api/users/me`                             | Returns full user profile and status flags.                          |
| `GET`  | `/v1/api/users/me/events/upcoming`             | Returns confirmed upcoming events.                                   |
| `GET`  | `/v1/api/users/me/events/history`              | Returns past events (attended/missed).                               |
| `POST` | `/v1/api/users/me/events/{event_id}/subscribe` | Idempotent subscription with capacity check and conflict resolution. |

### Verification

You can verify the new endpoints using the following `curl` commands (assuming you have a valid JWT token):

```bash
# Get User Profile
curl -H "Authorization: Bearer <token>" http://localhost:8080/v1/api/users/me

# Subscribe to an Event
curl -X POST -H "Authorization: Bearer <token>" \
     -H "Content-Type: application/json" \
     -d '{"metadata": {"dietary": "vegan"}}' \
     http://localhost:8080/v1/api# Get User Profile
curl -H "Authorization: Bearer <token>" http://localhost:8080/v1/api/users/me

# Subscribe to an Event
curl -X POST -H "Authorization: Bearer <token>" \
     -H "Content-Type: application/json" \
     -d '{"metadata": {"dietary": "vegan"}}' \
     http://localhost:8080/v1/api/users/me/events/<event_id>/subscribe

# Get Upcoming Events
curl -H "Authorization: Bearer <token>" http://localhost:8080/v1/api/users/me/events/upcoming


Made changes.
```
