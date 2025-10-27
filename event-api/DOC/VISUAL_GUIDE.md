# 🎨 Визуальный гайд форматирования ошибок

## 📊 Полная демонстрация

### ❌ ОШИБКИ

```
╔════════════════════════════════════════════════════════════╗
║ ❌ Database Connection Failed
╠════════════════════════════════════════════════════════════╣
║ Error: connection refused
║ → Host: localhost
║ → Port: 5432
║ → Database: event_api
╚════════════════════════════════════════════════════════════╝

╔════════════════════════════════════════════════════════════╗
║ ❌ User Registration Failed
╠════════════════════════════════════════════════════════════╣
║ Error: invalid email format
║ → Email: invalid-email@
║ → Reason: Email validation failed
╚════════════════════════════════════════════════════════════╝

╔════════════════════════════════════════════════════════════╗
║ ❌ Payment Processing Failed
╠════════════════════════════════════════════════════════════╣
║ Error: insufficient funds
║ → Amount: $500.00
║ → Available: $100.00
║ → Card: ****1234
╚════════════════════════════════════════════════════════════╝
```

**Код:**
```go
fmt.Println(logger.FormatError(
    "Database Connection Failed",
    err,
    "Host: localhost",
    "Port: 5432",
    "Database: event_api",
))
```

---

### ✅ УСПЕХ

```
╔════════════════════════════════════════════════════════════╗
║ ✅ Server Started Successfully
╠════════════════════════════════════════════════════════════╣
║ → Port: 8080
║ → Environment: development
║ → CORS Origins: 2
╚════════════════════════════════════════════════════════════╝

╔════════════════════════════════════════════════════════════╗
║ ✅ User Registered
╠════════════════════════════════════════════════════════════╣
║ → Email: user@example.com
║ → User ID: 123e4567-e89b-12d3-a456-426614174000
║ → Verification Code Sent
╚════════════════════════════════════════════════════════════╝

╔════════════════════════════════════════════════════════════╗
║ ✅ Database Connected
╠════════════════════════════════════════════════════════════╣
║ → Host: localhost:5432
║ → Database: event_api
║ → Pool: 10/25 connections
╚════════════════════════════════════════════════════════════╝
```

**Код:**
```go
fmt.Println(logger.FormatSuccess(
    "Server Started Successfully",
    "Port: 8080",
    "Environment: development",
    "CORS Origins: 2",
))
```

---

### ⚠️ ПРЕДУПРЕЖДЕНИЯ

```
╔════════════════════════════════════════════════════════════╗
║ ⚠️  .env file not found
╠════════════════════════════════════════════════════════════╣
║ → Using system environment variables
║ → Some values may use defaults
╚════════════════════════════════════════════════════════════╝

╔════════════════════════════════════════════════════════════╗
║ ⚠️  High Memory Usage
╠════════════════════════════════════════════════════════════╣
║ → Current: 85% of available memory
║ → Connections: 950/1000
║ → Consider scaling horizontally
╚════════════════════════════════════════════════════════════╝

╔════════════════════════════════════════════════════════════╗
║ ⚠️  Deprecated API Endpoint
╠════════════════════════════════════════════════════════════╣
║ → Endpoint: /api/v1/users
║ → Use: /api/v2/users instead
║ → Removal date: 2026-01-01
╚════════════════════════════════════════════════════════════╝
```

**Код:**
```go
fmt.Println(logger.FormatWarning(
    ".env file not found",
    "Using system environment variables",
    "Some values may use defaults",
))
```

---

### ℹ️ ИНФОРМАЦИЯ

```
╔════════════════════════════════════════════════════════════╗
║ ℹ️  Application Configuration
╠════════════════════════════════════════════════════════════╣
║ → Port: 8080
║ → Environment: development
║ → JWT Expiration: 3600 seconds
║ → CORS Origins: http://localhost:3000
╚════════════════════════════════════════════════════════════╝

╔════════════════════════════════════════════════════════════╗
║ ℹ️  Migration Status
╠════════════════════════════════════════════════════════════╣
║ → Total migrations: 5
║ → Applied: 5
║ → Pending: 0
║ → Status: ✓ Up to date
╚════════════════════════════════════════════════════════════╝

╔════════════════════════════════════════════════════════════╗
║ ℹ️  Performance Metrics
╠════════════════════════════════════════════════════════════╣
║ → Request/sec: 1,234
║ → Avg latency: 45ms
║ → Error rate: 0.01%
╚════════════════════════════════════════════════════════════╝
```

**Код:**
```go
fmt.Println(logger.FormatInfo(
    "Application Configuration",
    "Port: 8080",
    "Environment: development",
    "JWT Expiration: 3600 seconds",
))
```

---

## 🎯 Таблица использования по типам операций

### Аутентификация

| Операция | Функция | Пример |
|----------|---------|---------|
| Успешный вход | ✅ Success | "User Logged In" |
| Ошибка входа | ❌ Error | "Login Failed: Invalid password" |
| Токен истёк | ⚠️ Warning | "JWT token expired" |
| Новая попытка | ℹ️ Info | "Token refresh initiated" |

### База данных

| Операция | Функция | Пример |
|----------|---------|---------|
| Успешное подключение | ✅ Success | "Database Connected" |
| Ошибка подключения | ❌ Error | "Connection refused" |
| Медленный запрос | ⚠️ Warning | "Query took 5s" |
| Миграция | ℹ️ Info | "Migrations applied" |

### Платежи

| Операция | Функция | Пример |
|----------|---------|---------|
| Платёж обработан | ✅ Success | "Payment Processed" |
| Ошибка платежа | ❌ Error | "Payment failed: Insufficient funds" |
| Подозрительная активность | ⚠️ Warning | "High transaction frequency" |
| Платёж инициирован | ℹ️ Info | "Processing payment..." |

---

## 💻 Примеры кода для копирования

### Ejemplo 1: Ошибка валидации

```go
if !isValidEmail(req.Email) {
    fmt.Println(logger.FormatError(
        "Email Validation Failed",
        fmt.Errorf("invalid email format"),
        "Email: " + req.Email,
        "Required format: user@example.com",
    ))
    return
}
```

### Пример 2: Успешная операция

```go
user, _ := authService.Register(req)
fmt.Println(logger.FormatSuccess(
    "User Registered Successfully",
    "Email: " + user.Email,
    "ID: " + user.ID,
    "Verification code sent to email",
))
```

### Пример 3: Предупреждение о производительности

```go
if queryTime > 5*time.Second {
    fmt.Println(logger.FormatWarning(
        "Slow Database Query",
        "Query time: "+queryTime.String(),
        "Threshold: 5s",
        "Consider adding index",
    ))
}
```

### Пример 4: Информационное сообщение о статусе

```go
fmt.Println(logger.FormatInfo(
    "Service Health Check",
    "Database: ✓ Connected",
    "Cache: ✓ Available",
    "Disk space: 45% used",
))
```

---

## 🌈 Визуальная иерархия приоритета

```
🔴 ERROR     ← Критичные, требуют немедленного внимания
🟡 WARNING   ← Важные, требуют проверки
🔵 INFO      ← Информационные, для отслеживания
🟢 SUCCESS   ← Подтверждение выполнения
```

---

## 📱 Совместимость

✅ **Поддерживаемые терминалы:**
- macOS Terminal
- macOS iTerm2
- Linux Terminal
- VS Code Integrated Terminal
- Windows Terminal
- PowerShell (Windows 10+)

⚠️ **Ограничения:**
- Старые версии Windows cmd.exe не поддерживают ANSI цвета
- Production режим использует JSON (без цветов)

---

## 🔧 Кастомизация

Все цвета определены в `internal/logger/logger.go`:

```go
const (
    ColorRed     = "\033[31m"
    ColorGreen   = "\033[32m"
    ColorYellow  = "\033[33m"
    ColorBlue    = "\033[34m"
    ColorCyan    = "\033[36m"
    ColorReset   = "\033[0m"
)
```

Можете изменить под свои предпочтения 🎨

---

## 📞 Поддержка

**Вопросы о форматировании?**
- Смотрите `LOGGING.md` для полного руководства
- Смотрите `ERROR_FORMATTING_GUIDE.md` для практических примеров
- Запустите `make logger-demo` для демонстрации

---

**Версия:** 1.0  
**Дата:** 27 Октября 2025  
**Статус:** ✅ Production Ready 🚀
