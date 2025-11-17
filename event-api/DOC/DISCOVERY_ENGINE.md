# Narrow Time-Slot Discovery Engine

This document describes the domain model, queue lifecycle, and integration points for the discovery engine that powers `/v1/api/discovery/*`.

## Concept Overview

Users explore a curated list of events that all fit inside a short time window (currently six hours). The engine guarantees deterministic ordering, conflict-aware booking, and full action history per user. Reactions never mutate the source catalog; state lives only inside queue snapshots and history logs.

Key properties:

- **Single active item** – the client must always act on the event returned by `GET /next` before requesting the next item.
- **Stable ordering** – neutral skips append to the tail, dislikes remove items, and conflicts always trail non-conflicting events.
- **Conflict propagation** – booking immediately tags all overlapping events so they appear after every non-conflicting item.
- **Idempotency** – repeated requests with the same event/action pair are no-ops and simply return the last recorded history entry.

## Domain Model

| Entity         | Responsibility                                                                         |
| -------------- | -------------------------------------------------------------------------------------- |
| `TimeSlot`     | Closed-open interval (`start` inclusive, `end` exclusive) used to detect overlaps.     |
| `Event`        | Immutable event metadata loaded from the approved catalog.                             |
| `QueueState`   | Per-user snapshot containing primary queue, conflict queue, and the “current” pointer. |
| `ConflictFlag` | Tracks why an event was delayed (booked event ID, reason, timestamp).                  |
| `HistoryEntry` | Append-only record of every action (like, dislike, neutral, book).                     |

## Action Semantics

| Endpoint                      | Effect on queue                                                                                          | Notes |
| ----------------------------- | -------------------------------------------------------------------------------------------------------- | ----- |
| `POST /action` with `like`    | Removes the current event from the head and records the preference.                                      |
| `POST /action` with `dislike` | Removes the event and ensures it never re-enters the queue.                                              |
| `POST /action` with `neutral` | Moves the event to the tail of the primary queue unless it already carries a conflict flag.              |
| `POST /book`                  | Removes the event, marks all overlapping events as `conflict`, and pushes them behind the primary queue. |

Neutral on a conflict event does **not** upgrade it back into the primary queue; it stays within the conflict tail to respect booking commitments.

## Booking Flow

1. Client fetches the current candidate via `/next`.
2. Client sends `POST /book` for the same `eventId`.
3. Engine identifies all remaining events whose `TimeSlot` overlaps the booked slot.
4. Each conflicting event receives an active `ConflictFlag` and repositions into the `Conflicts` slice.
5. The booking response returns both the booked event snapshot and an ordered list of conflict IDs so the UI can notify the user.

## Queue Lifecycle

```
Primary Queue   --->   [current]   --->   actions   --->   history
                                          |  |  |
             conflict tagging  <---  booking  neutral  dislike
```

- `QueueState.CurrentEventID` is populated lazily when `/next` runs. It is cleared automatically once an action is processed.
- When both primary and conflict queues are empty, `/next` returns `404 queue_empty`.
- `ResetQueue` (internal helper) discards the snapshot, forcing a rebuild from the latest catalog the next time the user requests `/next`.

## API Reference Recap

| Method | Route                       | Description                                                       |
| ------ | --------------------------- | ----------------------------------------------------------------- |
| `GET`  | `/v1/api/discovery/next`    | Fetch next event respecting current queue ordering and conflicts. |
| `POST` | `/v1/api/discovery/action`  | Apply `like`, `dislike`, or `neutral` to the current event.       |
| `POST` | `/v1/api/discovery/book`    | Confirm an event and propagate conflicts.                         |
| `GET`  | `/v1/api/discovery/history` | Retrieve chronological action logs.                               |

All endpoints require Bearer JWT authentication. The handlers infer `userId` from the `X-User-ID` header injected by `AuthMiddleware`.

## Storage and Concurrency

- **Event repository** – in-memory, refreshed via `ReplaceEvents`. Only approved events within the discovery window are loaded.
- **Queue repository** – stores serialized `QueueState` per user. Access is serialized via per-user locks in `Engine.lock` to guarantee thread safety.
- **History repository** – append-only list plus a hash map to short-circuit idempotent checks.

## Failure Modes

| Error                 | HTTP Code | Resolution                                                    |
| --------------------- | --------- | ------------------------------------------------------------- |
| `ErrQueueEmpty`       | 404       | No more events remain. Client should stop requesting `/next`. |
| `ErrOutOfOrderAction` | 409       | Client acted on a stale event; fetch `/next` again.           |
| `ErrInvalidAction`    | 400       | Action was missing or unsupported.                            |
| `ErrEventNotFound`    | 404       | Event catalog changed; client should refresh.                 |

## Testing Strategy

The test suite in `internal/discovery/engine_test.go` covers:

- Reaction semantics (like, dislike, neutral).
- Booking conflict propagation ordering.
- Queue exhaustion and idempotent retries.
- Concurrency safety via multi-goroutine stress tests.

Use `go test ./internal/discovery` for focused runs during development.
