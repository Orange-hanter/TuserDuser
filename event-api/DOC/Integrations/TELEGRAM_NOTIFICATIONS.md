# Telegram Notification Integration

## 🎯 Goals

- Provide a secure Telegram-based notification sink that plugs into the existing `ReminderWorker` with zero changes to core business logic.
- Keep user identifiers private by exchanging only signed, expiring binding tokens.
- Guarantee auditability, graceful degradation, and replay-friendly logging from MVP (single instance, SQLite) to 50k+ users (PostgreSQL, HA workers).
- Maintain parity for non-Telegram users: integrating this sink must not leak transport-specific concepts into domain code.

## 🧩 Component Overview

| Component             | Responsibility                                                                                                    | Availability Tier |
| --------------------- | ----------------------------------------------------------------------------------------------------------------- | ----------------- |
| `ReminderWorker`      | Emits `ReminderJob` payloads (user ID, trigger metadata, deadline). Calls the Telegram sink via interface.        | Existing          |
| Telegram Sink Adapter | Converts `ReminderJob` → `TelegramMessageAttempt`, enqueues work and records delivery intent before dispatch.     | New               |
| Binding Service       | Issues signed deep-link tokens and verifies inbound webhooks to map Telegram chat IDs to internal user IDs.       | New               |
| Webhook Handler       | Terminates Telegram webhook, validates signatures, updates delivery statuses, processes `/unsubscribe`/`/status`. | New               |
| Storage Layer         | Persists bindings, delivery attempts, and state transitions. SQLite for MVP, PostgreSQL + replicas in production. | New               |
| Delivery Worker       | Pulls pending attempts, serializes rate-limited sends, applies retry / backoff based on Telegram responses.       | New               |
| Ops Toolkit           | CLI / admin endpoints to inspect logs, replay failed attempts, and unblock/ban users.                             | New               |

## 🔐 Secure Binding Flow

1. **Token Request**: Mobile/web app calls `POST /api/notifications/telegram/link` with auth JWT.
2. **Token Minting**: Service issues a short-lived (e.g., 10 min) token: `base64url( user_id | nonce | exp )`, signed with HMAC-SHA256. No server-side session state is stored.
3. **Deep Link**: App presents `https://t.me/<bot>?start=<token>`. Token includes a one-time `nonce` stored hashed in DB to prevent replay.
4. **Telegram Start**: User hits `/start <token>` via bot. Telegram forwards to webhook.
5. **Verification**:
   - Webhook parses token, verifies signature & expiry.
   - Checks hashed nonce against `telegram_binding_tokens` table (upserted when token minted). If unused, marks consumed.
   - Stores binding into `telegram_bindings` (user_id, chat_id, status=active, telegram_username, last_seen, blocked=false).
6. **Acknowledgement**: Bot responds with confirmation message plus `/unsubscribe` instructions.
7. **Zero-Trust Guarantees**: Even if token leaks, it cannot be reused (nonce consumed + expiry) and reveals no raw identifiers without signature key.

### Binding Data Structures (MVP in SQLite)

````sql
CREATE TABLE telegram_binding_tokens (
    nonce_hash TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE telegram_bindings (
    user_id TEXT PRIMARY KEY,
    chat_id TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active','blocked','pending','revoked')),
    blocked_reason TEXT,
    last_error_code INTEGER,
    last_error_at TIMESTAMP,
    updated_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL
);
```bash
- SQLite ensures low-cost MVP. For production, migrate to PostgreSQL (same schema) with indexes on `status`, `chat_id`.

## 📤 Outbound Delivery Pipeline

1. `ReminderWorker` emits `ReminderJob` and invokes `telegramSink.Enqueue(job)`.
2. Adapter resolves user binding:
   - If none, returns `ErrTransportUnavailable` so worker can fall back to email/push.
   - If status != `active`, job is dropped with audit log.
3. Adapter writes a `telegram_delivery` row **before** sending:

```sql
CREATE TABLE telegram_delivery (
    id UUID PRIMARY KEY,
    user_id TEXT NOT NULL,
    chat_id TEXT NOT NULL,
    reminder_id TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('scheduled','sending','sent','failed','blocked','abandoned')),
    attempt_count INTEGER NOT NULL DEFAULT 0,
    last_error_code INTEGER,
    last_error_msg TEXT,
    next_attempt_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
````

1. Delivery worker dequeues `status='scheduled'` rows ordered by `next_attempt_at` and `chat_id` to respect Telegram 30 msg/sec per bot limit (configurable).
1. **Send Attempt**:
   - Row is transitioned to `sending` via optimistic locking (ETag column) to prevent duplicates.
   - HTTP request to Telegram Bot API uses per-bot token stored in secrets manager.
   - Response codes map to outcomes:
     - `200`: mark `sent`, store Telegram `message_id`, add audit trail entry.
     - `429`: mark `failed`, compute exponential backoff with jitter, set `next_attempt_at`, emit alert.
     - `403`/`blocked`: mark user binding `status='blocked'`, delivery `status='blocked'`, and stop retries.
     - Network errors: `failed`, schedule retry (max N attempts, e.g., 5) with jitter.
1. **Logging**: Each transition writes to `telegram_delivery_log` (append-only) for replayability.

## 📥 Webhook Handling & Commands

- Endpoint `POST /webhooks/telegram/<botAlias>` exposed via HTTPS. MVP uses `cloudflared` or `ngrok` tunnel; production sits behind ALB + WAF.
- Validates Telegram signature header and rejects mismatches.
- Routes updates:
  - `/start <token>` → binding flow.
  - `/unsubscribe` → sets `telegram_bindings.status='revoked'`, acknowledges idempotently.
  - `/status` → echoes current binding state.
  - Delivery receipts (if enabled) update `telegram_delivery` rows with `message_id`.
- All inbound payloads persisted to `telegram_webhook_events` for audit (can prune with TTL job).

## ♻️ Lifecycle Summary (Event Like/Dislike Reminder)

````text
ReminderWorker trigger
    ↓ Enqueue telegramSink job
telegram_delivery row (status=scheduled)
    ↓ Delivery worker send attempt
telegram_delivery_log append (status=sent or failed)
    ↓ Binding status update (if error)
History available for replay via admin CLI
```bash
- Data persists until retention job archives old rows (e.g., 90 days) to cold storage. Deletion uses batched `DELETE ... WHERE created_at < cutoff` plus optional export to object storage.

## 🛡️ Security & Reliability Controls

- **Zero trust tokens**: expiring, signed, nonce-tracked, no shared session state.
- **Secret hygiene**: Bot tokens only loaded from env/secret store; never logged.
- **Per-user mutex**: Delivery worker locks by `chat_id` to prevent interleaved retries.
- **Graceful degradation**: If Telegram API unreachable, worker stops new sends, marks pending rows `abandoned`, and raises Prometheus alert.
- **Blocking semantics**: Any `403` auto-blocks binding; ReminderWorker sees `ErrBlocked` and skips retries.
- **Observability**: Metrics for send latency, failure codes, backlog depth, token issuance. Structured logs contain correlation IDs.
- **Cost efficiency**: MVP stack = single Go binary + SQLite + `cloudflared` tunnel. Scaling path = Postgres, Redis rate-limit tokens, HA queue (e.g., NATS or RabbitMQ).

## 🔁 Recovery & Replay

- `telegram_delivery_log` keeps immutable history (id, attempt, status, error, timestamp).
- Admin CLI `telegram replay --delivery-id <id>` loads payload and requeues if status in (`failed`,`abandoned`).
- `/webhooks/telegram` can be temporarily paused; pending deliveries remain in DB for replay.
- Backfill script can resync bindings from exported chats if needed (requires explicit ops approval).

## 🧪 Testing Strategy

- **Unit**: Mock Telegram API + time controls for token expiry, rate limiter, retry policies.
- **Integration**: Spin up SQLite + httptest webhook. Use `TestMain` to register local tunnel callback.
- **E2E**: Replay fixture updates to ensure `/start`, `/unsubscribe`, `/status`, reminder send, and failure paths work without external network by mocking Telegram HTTP server.
- **Load**: Use k6 to simulate 10k users; verify rate limiter prevents API bans and backlog stays < 1s.

## 🚀 Scaling Path

| Stage             | Storage                    | Workers                                        | Rate limiting                     | Notes                            |
| ----------------- | -------------------------- | ---------------------------------------------- | --------------------------------- | -------------------------------- |
| MVP               | SQLite WAL                 | 1 delivery goroutine                           | Token bucket in-memory            | Free; relies on process uptime   |
| Pilot (5k users)  | PostgreSQL single          | 2 workers (active/passive)                     | Redis-based distributed limiter   | Add Prometheus exporter          |
| Production (50k+) | Postgres HA + read replica | Worker pool + job queue (e.g., NATS JetStream) | Sharded limiter + circuit breaker | Bot tokens rotated automatically |

## 🔗 ReminderWorker Integration Contract

```go
type NotificationSink interface {
    EnqueueReminder(ctx context.Context, job ReminderJob) error
}

// ReminderWorker already loops over ReminderJob(s) → call sink.
telegramSink := telegram.NewSink(store, limiter, telegramClient)
worker.RegisterSink("telegram", telegramSink)
````

- All Telegram-specific logic is encapsulated; the worker just observes `ErrTransportUnavailable` or `ErrBlocked` to fallback gracefully.
- Non-Telegram users never touch this pathway because `ReminderWorker` selects sinks based on user preferences.

## ✅ Success Criteria Mapping

- **One-click binding**: `/start <token>` completes within one round-trip, token expires after first use.
- **Low latency**: Delivery worker monitors queue depth; alerts if >2s between schedule and send.
- **10k+ users**: Rate limiter + sharded workers prevent burst bans.
- **Blocked users**: Binding status enforced before enqueue + on send; no retries once blocked.
- **Testability**: All external calls abstracted behind interfaces; local mocks cover webhook and API.

With this integration, the system gains a security-hardened Telegram channel
that scales from laptop MVP to production load while keeping ReminderWorker
untouched and ensuring every notification attempt is traceable, replayable, and
safe.
