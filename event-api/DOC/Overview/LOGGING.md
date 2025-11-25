# 📊 Улучшенное логирование в Event API

## ✨ Новые возможности

Логгер теперь поддерживает красивое форматирование ошибок, успехов,
предупреждений и информационных сообщений с использованием ANSI цветов и Unicode
символов.

## 🎨 Виды сообщений

### 1️⃣ Ошибки (Errors)

````go
fmt.Println(logger.FormatError(
    "Database Connection Failed",
    err,
    "Host: localhost",
    "Port: 5432",
    "Database: event_api",
))
```bash
**Вывод:**

````

╔════════════════════════════════════════════════════════════╗
║ ❌ Database Connection Failed
╠════════════════════════════════════════════════════════════╣
║ Error: connection refused
║ → Host: localhost
║ → Port: 5432
║ → Database: event_api
╚════════════════════════════════════════════════════════════╝

````bash
### 2️⃣ Успех (Success)

```go
fmt.Println(logger.FormatSuccess(
    "User Registered Successfully",
    "Email: user@example.com",
    "User ID: 123e4567-e89b-12d3-a456-426614174000",
    "Verification Code Sent",
))
````

**Вывод:**

```bash
╔════════════════════════════════════════════════════════════╗
║ ✅ User Registered Successfully
╠════════════════════════════════════════════════════════════╣
║ → Email: user@example.com
║ → User ID: 123e4567-e89b-12d3-a456-426614174000
║ → Verification Code Sent
╚════════════════════════════════════════════════════════════╝
```

### 3️⃣ Предупреждения (Warnings)

````go
fmt.Println(logger.FormatWarning(
    ".env file not found",
    "Using system environment variables",
    "Some values may use defaults",
))
```bash
**Вывод:**

````

╔════════════════════════════════════════════════════════════╗
║ ⚠️ .env file not found
╠════════════════════════════════════════════════════════════╣
║ → Using system environment variables
║ → Some values may use defaults
╚════════════════════════════════════════════════════════════╝

````bash
### 4️⃣ Информация (Info)

```go
fmt.Println(logger.FormatInfo(
    "Server Configuration",
    "Port: 8080",
    "Environment: development",
    "CORS Origins: 2",
))
````

**Вывод:**

```bash
╔════════════════════════════════════════════════════════════╗
║ ℹ️  Server Configuration
╠════════════════════════════════════════════════════════════╣
║ → Port: 8080
║ → Environment: development
║ → CORS Origins: 2
╚════════════════════════════════════════════════════════════╝
```

## 🔧 Использование в коде

### В main.go

````go
package main

import (
    "fmt"
    "event-api/internal/logger"
)

func main() {
    logger.Init()
    defer logger.Sync()

    // Успешное подключение к БД
    fmt.Println(logger.FormatSuccess(
        "Database Connected",
        "Host: localhost",
        "Port: 5432",
        "Pool Size: 10/25",
    ))
}
```go
### В handlers

```go
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
    // ... код ...

    if err != nil {
        logger.Log.Error(logger.FormatError(
            "User Registration Failed",
            err,
            "Email: " + req.Email,
            "Reason: Invalid email format",
        ))
        return
    }

    logger.Log.Info(logger.FormatSuccess(
        "User Registered",
        "Email: " + user.Email,
        "ID: " + user.ID,
    ))
}
````

### В services

````go
func (s *AuthService) Login(req *models.LoginRequest) (*models.AuthResponse, error) {
    user, err := s.GetUserByEmail(req.Email)
    if err != nil {
        logger.Log.Error(logger.FormatError(
            "Login Attempt Failed",
            err,
            "Email: " + req.Email,
            "Reason: User not found",
        ))
        return nil, err
    }

    return &models.AuthResponse{...}, nil
}
```bash
## 🎯 Когда использовать

| Тип         | Когда использовать       | Пример                             |
| ----------- | ------------------------ | ---------------------------------- |
| **Error**   | Ошибки приложения        | БД не найдена, валидация не прошла |
| **Success** | Успешные операции        | Пользователь зарегистрирован       |
| **Warning** | Предупреждения           | .env не найден, deprecated API     |
| **Info**    | Информационные сообщения | Server started, config loaded      |

## 🎨 Цветовая схема

- 🔴 **Ошибки** — красный цвет (`ColorRed`)
- 🟢 **Успех** — зелёный цвет (`ColorGreen`)
- 🟡 **Предупреждения** — жёлтый цвет (`ColorYellow`)
- 🔵 **Информация** — синий цвет (`ColorBlue`)

## 📝 Интеграция с Zap Logger

Новые функции форматирования работают вместе с существующим Zap логгером:

```go
// Для важных ошибок используйте оба:
fmt.Println(logger.FormatError("Critical Error", err, "details..."))
logger.Log.Error("Critical Error", zap.Error(err))

// Для обычных сообщений:
logger.Log.Info(logger.FormatSuccess("Operation Completed"))
````

## 🚀 Автоматическое определение окружения

Логгер автоматически определяет окружение из переменной `ENV`:

- **development** — красивое форматирование с цветами
- **production** — стандартный JSON формат Zap

```bash
## Development (красивый вывод)
## Development (красивый вывод)
ENV=development go run ./cmd/server

## Production (JSON)
## Production (JSON)
ENV=production go run ./cmd/server
```
