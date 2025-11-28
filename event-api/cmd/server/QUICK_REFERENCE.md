# Quick Reference - Event API Server Architecture

## File Map

### 🎯 Start Here

**`main.go`** - Application entry point

- Find: Where the app starts
- Look for: `main()`, `run()`, initialization sequence
- Change: Startup parameters, version handling

### 🏗️ The Hub

**`container.go`** - Dependency injection & service management

- Find: All services and handlers
- Look for: `AppContainer` struct, service initialization
- Change: Adding new services, modifying dependencies

### 🔧 Setup

**`bootstrap.go`** - Infrastructure initialization

- Find: Database, Redis, SMS, Email setup
- Look for: `initDatabase()`, `initRedis()`, etc.
- Change: Configuration, connection parameters

### 🔄 Discovery Engine

**`discovery.go`** - Event discovery subsystem

- Find: Discovery service, workers, event updates
- Look for: `StartDiscoveryWorkers()`, update handling
- Change: Discovery logic, event synchronization

### 🛣️ Routes

**`routes.go`** - HTTP endpoint organization

- Find: All API endpoints grouped by role
- Look for: `buildHTTPHandler()`, route registration
- Change: Adding endpoints, modifying access control

### 🛑 Cleanup

**`shutdown.go`** - Graceful application shutdown

- Find: Signal handling, resource cleanup
- Look for: `ExecuteGracefulShutdown()`, cleanup order
- Change: Shutdown behavior, cleanup sequence

### 📦 Version

**`version_info.go`** - Build information

- Find: Version, commit, build time
- Look for: `VersionInfo` struct, version flag
- Change: Version handling, build metadata

---

## Common Tasks

### ➕ Add a New Service

**Step 1:** Create initialization in `bootstrap.go`

```go
func initMyService(cfg *config.Config) (*myservice.Service, error) {
    // initialization logic
}
```

**Step 2:** Add to `AppContainer` in `container.go`

```go
type AppContainer struct {
    // ... existing fields
    MyService *myservice.Service
}
```

**Step 3:** Initialize in `container.initServices()`

```go
c.MyService, err = initMyService(c.Config)
```

### 🛣️ Add a New Route

**In `routes.go`:**

```go
func registerNewFeatureRoutes(r chi.Router, handler *handlers.NewHandler) {
    r.Post("/api/new-feature", handler.Create)
    r.Get("/api/new-feature/{id}", handler.Get)
}

// Add to buildHTTPHandler:
authenticated.Route("/api/new-feature", func(r chi.Router) {
    registerNewFeatureRoutes(r, newHandler)
})
```

### 👷 Add a Background Worker

**In `discovery.go` (or create `workers.go`):**

```go
func (c *AppContainer) startMyWorker(ctx context.Context) {
    go func() {
        // worker logic
    }()
}

// Call from container.go:
c.StartDiscoveryWorkers(appCtx)  // add to this or create new method
```

---

## Debugging Tips

| Issue                     | Look in                    | Why                            |
| ------------------------- | -------------------------- | ------------------------------ |
| Service not initialized   | `container.go`             | Central place for all services |
| Route not found           | `routes.go`                | All routes registered here     |
| Database connection fails | `bootstrap.go`             | DB init logic                  |
| Worker not running        | `discovery.go`             | Worker startup logic           |
| Shutdown hangs            | `shutdown.go`              | Cleanup sequence               |
| Missing handler           | `container.initHandlers()` | Handler initialization         |

---

## Architecture at a Glance

```
┌─────────────┐
│   main()    │ Entry point
└──────┬──────┘
       │
       ▼
┌─────────────────────┐
│   run()             │ Orchestration
│ (154 lines)         │
└──────┬──────────────┘
       │
       ├──────────────────────┬──────────────────┬────────────────┐
       │                      │                  │                │
       ▼                      ▼                  ▼                ▼
  ┌─────────┐         ┌────────────┐     ┌──────────┐     ┌──────────┐
  │container│         │ bootstrap  │     │discovery │     │ routes   │
  │  .go    │         │   .go      │     │   .go    │     │  .go     │
  │(288)    │         │ (106)      │     │ (219)    │     │ (199)    │
  └─────────┘         └────────────┘     └──────────┘     └──────────┘
       │                   │                   │                │
   Services           Infrastructure      Workers &         Routes &
   & Handlers         Setup              Updates           Handlers
```

---

## Style Guide

### Naming

- **Files:** lowercase with underscore (`bootstrap.go`, `discovery.go`)
- **Types:** PascalCase (`AppContainer`, `GracefulShutdownConfig`)
- **Functions:** camelCase (`initDatabase`, `startLockCleanupWorker`)
- **Constants:** UPPER_SNAKE_CASE

### Comments

- Package-level: Describe module purpose
- Functions: Explain "why" not "what"
- Inline: Used sparingly for non-obvious logic

### Organization

- Imports: stdlib → internal → external
- Types first, then functions
- Public before private
- Related functions grouped

---

## Performance Notes

- **Container creation:** One-time, ~100-200ms
- **Route registration:** ~10-20ms
- **Discovery bootstrap:** Depends on event count
- **Shutdown:** Graceful, respects timeout

No performance critical code here - all heavy lifting in internal packages.
