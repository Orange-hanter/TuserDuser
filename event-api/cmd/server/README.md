# Event API Server - Modular Architecture

## Overview

The server code has been refactored from a monolithic `main.go` (667 lines) into a clean, modular architecture that significantly reduces cognitive load during development.

## Module Structure

### 1. **main.go** (154 lines)

**Entry point** - High-level orchestration only.

- `main()` - Standard entry point
- `run()` - Application initialization and lifecycle
- Coordinate startup/shutdown sequence

**Key responsibility:** Clean, readable flow showing the complete startup process at a glance.

---

### 2. **container.go** (288 lines)

**Dependency Injection Container** - Central place for all services.

- `AppContainer` struct - Holds all services, handlers, and infrastructure
- `NewAppContainer()` - Initializes everything in logical order
- Service initialization methods (`initInfrastructure()`, `initServices()`, `initHandlers()`)

**Key responsibility:** Single source of truth for all application components.

---

### 3. **bootstrap.go** (106 lines)

**Infrastructure Initialization** - Low-level setup.

- Database connection and migrations
- Redis connection
- SMS and Email service initialization

**Key responsibility:** Setup external dependencies with clear error handling.

---

### 4. **discovery.go** (219 lines)

**Discovery Engine Management** - Event discovery subsystem.

- Discovery service initialization
- Background workers for event synchronization
- Redis pub/sub event update listeners
- Event transformation and catalog refresh logic

**Key responsibility:** Isolate all discovery-related complexity.

---

### 5. **routes.go** (199 lines)

**HTTP Router Configuration** - Clear, organized endpoint structure.

- Route registration by category:
  - Public auth routes
  - Public event endpoints
  - Authenticated user routes
  - Discovery routes
  - Creator/Admin routes
  - Admin-only routes
- CORS configuration

**Key responsibility:** Router clarity without the cognitive overhead.

---

### 6. **shutdown.go** (82 lines)

**Graceful Shutdown** - Clean resource cleanup.

- Signal handling
- Graceful shutdown orchestration
- Resource cleanup in correct order

**Key responsibility:** Proper application lifecycle management.

---

### 7. **version_info.go** (44 lines)

**Version Information** - Build metadata.

- Version struct
- Version flag handling

**Key responsibility:** Build information tracking.

---

## Architecture Benefits

### Before (667 lines in main.go)

- Multiple concerns mixed together
- Hard to find specific functionality
- Difficult to onboard new developers
- Cognitive load when reading the code
- Testing would require mocking entire main function

### After (6 focused modules)

- **Separation of Concerns:** Each module has a single responsibility
- **Readability:** Average module 100-200 lines (manageable size)
- **Maintainability:** Related code lives together
- **Testability:** Components can be tested independently
- **Scalability:** Easy to add new features without expanding existing files
- **Code Navigation:** Quick mental model of what goes where

## Initialization Flow

```
main()
  └─> run()
       ├─> NewAppContainer(cfg)
       │   ├─> initInfrastructure()
       │   │   ├─> initDatabase()
       │   │   ├─> initRedis()
       │   │   ├─> initSMSService()
       │   │   └─> initEmailService()
       │   ├─> initServices()
       │   │   ├─> AuthService
       │   │   ├─> EventService
       │   │   ├─> DiscoveryService
       │   │   └─> ...
       │   └─> initHandlers()
       │       ├─> AuthHandler
       │       ├─> EventHandler
       │       └─> ...
       ├─> StartDiscoveryWorkers()
       ├─> BuildHTTPRouter()
       ├─> CreateHTTPServer()
       └─> WaitForShutdownSignal()
            └─> ExecuteGracefulShutdown()
```

## Adding New Features

### Example: Adding a new service

1. **If adding external service:**
   - Add initialization to `bootstrap.go` → `initNewService()`
   - Add field to `AppContainer` in `container.go`
   - Initialize in `container.initServices()`

2. **If adding HTTP routes:**
   - Create `registerNewRoutes()` function in `routes.go`
   - Call from `buildHTTPHandler()`

3. **If adding background worker:**
   - Add function to `discovery.go` or create `workers.go` module
   - Call from `AppContainer.StartDiscoveryWorkers()`

## Maintenance Guidelines

- **main.go:** Keep clean - only orchestration logic
- **container.go:** The hub - if a new component exists, it belongs here
- **bootstrap.go:** Place infrastructure setup here
- **discovery.go:** Isolate discovery-specific workers
- **routes.go:** Router configuration grows here
- **shutdown.go:** Cleanup logic centralized

## Future Improvements

- Consider extracting handlers initialization to separate module if it grows
- Could create `workers.go` if background processing becomes complex
- Could add `middleware.go` for HTTP middleware setup if needed
