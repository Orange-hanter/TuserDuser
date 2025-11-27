# Admin Comments API Fix

## Problem

The original `/v1/api/creator/events/{eventId}/comments` endpoint was checking that the event `creator_id` matches the requesting `user_id`, which prevented admins from viewing comments for events they didn't create.

## Solution

Created a new admin-only endpoint `/v1/api/admin/events/{eventId}/comments` that:

1. Does NOT check `creator_id` ownership
2. Only checks that the event exists in any status table (pending, active, rejected, blocked)
3. Returns all comments for that event

## Backend Changes

### 1. New Handler Method in `creator.go`

```go
func (h *CreatorHandler) GetEventCommentsAsAdmin(w http.ResponseWriter, r *http.Request) {
    // Gets event comments without creator ownership check
}
```

### 2. New Service Method in `creator.go`

```go
func (s *CreatorService) GetEventCommentsForAdmin(ctx context.Context, eventID string) ([]models.ReviewComment, error) {
    // Service layer for admin comment retrieval
}
```

### 3. New Route in `main.go`

```go
adminOnly.Get("/api/admin/events/{eventId}/comments", creatorHandler.GetEventCommentsAsAdmin)
```

## Frontend Changes

### Updated API Call in `admin-panel/src/services/api.js`

```javascript
export const getEventComments = async (eventId) => {
  const response = await api.get(`/v1/api/admin/events/${eventId}/comments`);
  return response.data;
};
```

Previously used: `/v1/api/creator/events/{eventId}/comments`
Now uses: `/v1/api/admin/events/{eventId}/comments`

## API Endpoints Summary

| Endpoint                                    | Method | Role          | Purpose                     |
| ------------------------------------------- | ------ | ------------- | --------------------------- |
| `/v1/api/creator/events/{eventId}/comments` | GET    | Creator       | Get comments for own events |
| `/v1/api/admin/events/{eventId}/comments`   | GET    | Admin         | Get comments for any event  |
| `/v1/api/creator/events/{eventId}/comments` | POST   | Creator/Admin | Add comment to any event    |

## Testing

After rebuilding the server with `go build -o bin/server ./cmd/server`, the admin-panel can now:

1. View event comments in the chat modal
2. Admins see comments for all pending events (not restricted to owned events)
3. Comment history loads automatically every 5 seconds

## Files Modified

- ✅ `/event-api/internal/handlers/creator.go` - Added `GetEventCommentsAsAdmin`
- ✅ `/event-api/internal/service/creator.go` - Added `GetEventCommentsForAdmin`
- ✅ `/event-api/cmd/server/main.go` - Added new route
- ✅ `/admin-panel/src/services/api.js` - Updated endpoint URL
