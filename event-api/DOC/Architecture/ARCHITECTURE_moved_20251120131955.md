# Event API - Архитектура

## 🏗️ Архитектурная диаграмма

```text
┌─────────────────────────────────────────────────────────────┐
│                   HTTP Client (Frontend)                     │
└────────────────────────┬────────────────────────────────────┘
                         │
                    HTTP Requests
                         │
         ┌───────────────▼───────────────┐
         │  Chi Router (v5)              │
         │  + CORS Middleware            │
         │  + Security Headers           │
         └───────────────┬───────────────┘
                         │
        ┌────────────────┴───────────────┬─────────────┐
        │                                │             │
   ┌────▼─────┐    ┌──────────────────┬──▼─────┐  ┌────▼────┐
   │  Auth    │    │  Auth Routes     │        │  │ Health  │
   │Middleware│    │                  │        │  │ Endpoint│
   │(JWT)     │    │ - /register      │        │  └─────────┘
   └────┬─────┘    │ - /verify        │        │
        │          │ - /login         │        │
        │          │ - /logout        │        │
        │          │ - /me (protected)│        │
        │          └─────────────────┬┘        │
        │                            │         │
        └─────────────────┬──────────┴─────────┘
                          │
            ┌─────────────▼────────────────┐
            │   Auth Handler               │
            │  (internal/handlers/auth.go) │
            │                              │
            │ - Register(...)              │
            │ - Verify(...)                │
            │ - Login(...)                 │
            │ - Logout(...)                │
            │ - GetMe(...)                 │
            └─────────────────┬────────────┘
                              │
            ┌─────────────────▼────────────────┐
            │   Auth Service                   │
            │  (internal/service/auth.go)      │
            │                                  │
            │  Бизнес-логика:                  │
            │  - Register + password hash      │
            │  - Verify code validation        │
            │  - Login + JWT generation        │
            │  - Token blacklist (logout)      │
            │  - User retrieval                │
            │  - JWT validation                │
            └─────────────────┬────────────────┘
                              │
            ┌─────────────────┼────────────────┐
            │                 │                │
     ┌──────▼───────┐  ┌─────▼──────┐  ┌─────▼──────┐
     │ User Storage │  │ Verify Code│  │Token       │
     │ (In-Memory)  │  │ Cache      │  │Blacklist   │
     │              │  │ (In-Memory)│  │(In-Memory) │
     │ map[ID]User  │  │            │  │            │
     └──────────────┘  └────────────┘  └────────────┘
```

---

## 🔄 Поток данных

### Регистрация (Register Flow)

```text
1. Client POST /api/auth/register
   ├── Email, Phone, Password
   │
2. Handler ←── Validate input
   ├── Check duplicate email
   │
3. Service ←── Hash password (bcrypt)
   ├── Generate user ID
   ├── Generate verify code
   │
4. Storage ← Save user
   ├── Save verify code
   │
5. Response → 201 Created
   └── User + Verify Code
```

### Верификация (Verify Flow)

```text
1. Client POST /api/auth/verify
   ├── Email, Code
   │
2. Handler ←── Validate input
   │
3. Service ←── Get code from storage
   ├── Compare codes
   ├── Mark user as verified
   │
4. Storage ← Update user
   ├── Delete verify code
   │
5. Response → 200 OK
   └── Success message
```

### Логин (Login Flow)

```text
1. Client POST /api/auth/login
   ├── Email, Password
   │
2. Handler ←── Validate input
   │
3. Service ←── Find user by email
   ├── Compare password (bcrypt)
   ├── Generate JWT token
   │
4. Storage ← (read-only)
   │
5. Response → 200 OK
   └── Access Token + User
```

### Защищенный запрос (Protected Flow)

```text
1. Client GET /api/auth/me
   ├── Authorization: Bearer &lt;token&gt;
   │
2. Middleware ←── Extract token
   ├── Validate JWT signature
   ├── Check token expiry
   ├── Check blacklist
   │
3. Handler ←── Extract user ID
   │
4. Service ←── Get user by ID
   │
5. Storage ← (read-only)
   │
6. Response → 200 OK
   └── User data
```

### Выход (Logout Flow)

```text
1. Client POST /api/auth/logout
   ├── Authorization: Bearer &lt;token&gt;
   │
2. Handler ←── Extract token
   │
3. Service ←── Add to blacklist
   │
4. Storage ← Save token in blacklist
   │
5. Response → 200 OK
   └── Success message
```

---

## 📦 Компоненты и их ответственность

### 1. **Models** (`internal/models/auth.go`)

- Определяет структуры данных
- User, RegisterRequest, VerifyRequest, LoginRequest
- AuthResponse, VerifyResponse, ErrorResponse
- Claims для JWT

### 2. **Service** (`internal/service/auth.go`)

- Бизнес-логика приложения
- Управление пользователями
- Генерация и валидация JWT
- Хеширование паролей
- Управление кодами верификации

### 3. **Handlers** (`internal/handlers/auth.go`)

- HTTP endpoint обработчики
- Валидация input/output
- Error handling
- Логирование
- Преобразование между HTTP и service layer

### 4. **Middleware** (`internal/middleware/auth.go`)

- JWT валидация
- Extraction токенов
- Передача данных в контекст

### 5. **Config** (`internal/config/config.go`)

- Загрузка конфигурации из .env
- Централизованное управление параметрами

### 6. **Logger** (`internal/logger/logger.go`)

- Инициализация Zap логгера
- Структурированное логирование

---

## 🔐 Слои безопасности

```text
┌─────────────────────────────────────────────┐
│ 1. Transport Layer (HTTPS)                  │
│    - SSL/TLS encryption                     │
│    - CORS validation                        │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│ 2. Authentication Layer                     │
│    - Input validation                       │
│    - JWT verification                       │
│    - Token blacklist check                  │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│ 3. Business Logic Layer                     │
│    - Password hashing (bcrypt)              │
│    - User verification                      │
│    - Session management                     │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│ 4. Data Layer (In-Memory)                   │
│    - User storage                           │
│    - Session storage                        │
│    - Verification code storage              │
└─────────────────────────────────────────────┘
```

---

## 🔌 Интеграция точек

```text
main.go
  │
  ├── Config.Load() → конфигурация
  ├── Logger.Init() → логирование
  ├── AuthService.New() → бизнес-логика
  ├── AuthHandler.New() → обработчики
  │
  └── Chi Router
      ├── SecurityHeaders middleware
      ├── CORS middleware
      │
      ├── POST /api/auth/register → Handler.Register()
      ├── POST /api/auth/verify → Handler.Verify()
      ├── POST /api/auth/login → Handler.Login()
      ├── POST /api/auth/logout → Handler.Logout()
      ├── GET /api/auth/me → AuthMiddleware → Handler.GetMe()
      │
      └── GET /health → Handler.HealthCheck()
```

## 🎢 Discovery Engine Overview

Наряду с auth-модулем сервис содержит двигатель «narrow time-slot discovery», позволяющий пользователю проходить через очередь событий и фиксировать реакции.

```text
┌───────────────────────────────────────────────────────────────┐
│                       Authenticated Client                     │
└───────────────┬───────────────────────────────────────────────┘
      │  Authorization: Bearer &lt;token&gt;
   ┌───────▼────────┐
   │  Chi Router    │
   └───────┬────────┘
      │ /v1/api/discovery/*
   ┌───────▼──────────────────────────┐
   │ Discovery Handler                │
   │ (internal/handlers/discovery.go) │
   └───────┬──────────────────────────┘
      │ service API (Next, Action, Book, History)
   ┌───────▼──────────────────────────┐
   │ Discovery Service                │
   │ (internal/discovery/service.go)  │
   └───────┬──────────────────────────┘
      │ orchestrates
   ┌───────▼──────────────────────────┐
   │ Discovery Engine                 │
   │ (internal/discovery/engine.go)   │
   └───────┬──────────────────────────┘
    ┌───────────▼───────────┐ ┌───────────▼──────────┐ ┌───────────▼──────────┐
    │ Event Repository      │ │ Queue Repository      │ │ History Repository    │
    │ (ReplaceAll/List/Get) │ │ (per-user QueueState) │ │ (append-only log)     │
    └───────────────────────┘ └───────────────────────┘ └───────────────────────┘
```

**Основные принципы:**

- **Детерминированность**: на каждого пользователя хранится `QueueState` (primary + conflict очереди). Только `NextEvent` назначает текущий элемент.
- **Конфликт-менеджмент**: `BookEvent` помечает пересекающиеся `TimeSlot` события флагом `ConflictFlag` и переносит их в хвост.
- **Idempotency**: история хранит последнее действие по паре (user, event), что позволяет безопасно повторять запросы.
- **Concurrency safety**: движок удерживает per-user mutex, поэтому параллельные запросы одного пользователя не ломают порядок очереди.

---

## 📊 Структура данных

### User

```go
type User struct {
    ID        string    // Уникальный идентификатор
    Email     string    // Email адрес
    Phone     string    // Телефон
    Password  string    // Хеш пароля (bcrypt)
    Verified  bool      // Верифицирован ли
    CreatedAt time.Time // Время создания
    UpdatedAt time.Time // Время обновления
}
```

### JWT Claims

```go
type Claims struct {
    UserID   string `json:"user_id"`
    Email    string `json:"email"`
    Phone    string `json:"phone"`
    Verified bool   `json:"verified"`
    Exp      int64  `json:"exp"`      // Expiry time
    Iat      int64  `json:"iat"`      // Issued at
}
```

---

## 🚀 Масштабируемость

Текущая архитектура позволяет легко:

1. **Заменить In-Memory Storage на БД:**
   - Создать interface Storage
   - Реализовать PostgreSQL version

2. **Добавить кеширование:**
   - Redis для token blacklist
   - Redis для verification codes

3. **Добавить новые методы аутентификации:**
   - Social login (OAuth2)
   - Passwordless auth
   - Multi-factor auth

4. **Расширить функционал:**
   - Email notifications
   - SMS notifications
   - Audit logging

5. **Улучшить мониторинг:**
   - Metrics (Prometheus)
   - Tracing (OpenTelemetry)
   - Health checks

---

## 📝 Зависимости между компонентами

```text
main.go
  ↓
Chi Router ← Config, Logger
  ↓
Handlers ← Service, Logger
  ↓
Service ← Models, Config
  ↓
Storage (In-Memory)
  ↓
Models
```

---

## 🔧 Технологии и их роли

| Компонент | Назначение    | Альтернативы      |
| --------- | ------------- | ----------------- |
| Chi v5    | HTTP роутер   | Gin, Echo         |
| JWT-GO    | JWT токены    | jose              |
| Bcrypt    | Хеширование   | Argon2, scrypt    |
| Zap       | Логирование   | Logrus, slog      |
| CORS      | Cross-origin  | Middleware custom |
| godotenv  | .env загрузка | viper, envconfig  |

---

## 📈 Performance Considerations

- **In-Memory storage**: Хороша для development, нужна оптимизация для production
- **JWT validation**: O(1) operationя (без DB lookup)
- **Password hashing**: ~100ms (bcrypt cost=12)
- **Token blacklist**: O(1) lookup (map-based)

---

## 🎯 Future Architecture

```text
┌─────────────────────────────────────────┐
│ Load Balancer (nginx/HAProxy)           │
└────────┬────────────────────────────┬───┘
         │                            │
    ┌────▼────┐              ┌───────▼──┐
    │ App 1   │              │ App N    │
    │ :8080   │              │ :8080    │
    └────┬────┘              └───────┬──┘
         │                            │
    ┌────▼────────────────────────────▼──┐
    │ PostgreSQL Database                 │
    └─────────────────────────────────────┘
         │
    ┌────▼──────────────────────────┐
    │ Redis Cache                    │
    │ (Token blacklist, sessions)    │
    └────────────────────────────────┘
         │
    ┌────▼──────────────────────────┐
    │ Message Queue (RabbitMQ/Kafka)│
    │ (Email/SMS notifications)      │
    └────────────────────────────────┘
```

---

Архитектура разработана с учетом:

- ✅ Чистого кода (Clean Architecture)
- ✅ Принципов SOLID
- ✅ Масштабируемости
- ✅ Безопасности
- ✅ Тестируемости
