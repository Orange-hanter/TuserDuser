# ✨ Улучшение вывода ошибок в консоль Event API

## 📋 Что было сделано

### 1. Расширенный логгер с цветным форматированием

**Файл:** `internal/logger/logger.go`

Добавлены 4 функции форматирования с поддержкой ANSI цветов и Unicode символов:
- `FormatError()` — красные ошибки с ❌
- `FormatSuccess()` — зелёные успехи с ✅
- `FormatWarning()` — жёлтые предупреждения с ⚠️
- `FormatInfo()` — синяя информация с ℹ️

**Ключевые улучшения:**
- Красивые рамки в консоли
- Автоматическое определение окружения (development/production)
- Цветной вывод в development режиме
- Поддержка деталей (до 5 дополнительных строк на сообщение)
- ANSI цветовая поддержка во всех терминалах

### 2. Демонстрационное приложение

**Файл:** `cmd/logger-demo/main.go`

Пример использования всех типов форматирования ошибок.

**Запуск:**
```bash
make logger-demo
# или
go run ./cmd/logger-demo
```

### 3. Документация

**Файл:** `LOGGING.md`

Полное руководство с примерами использования.

### 4. Обновлен главный файл

**Файл:** `cmd/server/main.go`

Интегрировано новое форматирование при запуске сервера:
- Ошибки подключения к БД показывают детали
- Успешный запуск показывает конфигурацию
- Ошибки миграций показывают сообщение об ошибке

---

## 🎨 Примеры использования

### Пример 1: Ошибка подключения к БД

```go
fmt.Println(logger.FormatError(
    "Failed to Connect to Database",
    err,
    "Host: "+cfg.DBHost,
    "Port: "+cfg.DBPort,
    "Database: "+cfg.DBName,
))
```

**Вывод:**
```
╔════════════════════════════════════════════════════════════╗
║ ❌ Failed to Connect to Database
╠════════════════════════════════════════════════════════════╣
║ Error: connection refused
║ → Host: localhost
║ → Port: 5432
║ → Database: event_api
╚════════════════════════════════════════════════════════════╝
```

### Пример 2: Успешная регистрация

```go
logger.Log.Info(logger.FormatSuccess(
    "User Registered",
    "Email: user@example.com",
    "User ID: " + user.ID,
    "Verification Code Sent",
))
```

---

## 🔧 Интеграция в вашем коде

### В handlers

```go
if err != nil {
    logger.Log.Error(logger.FormatError(
        "User Registration Failed",
        err,
        "Email: " + req.Email,
        "Reason: Invalid format",
    ))
    return
}
```

### В services

```go
logger.Log.Info(logger.FormatSuccess(
    "Payment Processed",
    "Amount: $99.99",
    "Order ID: " + orderID,
    "Status: Completed",
))
```

### В migrations

```go
logger.Log.Info(logger.FormatInfo(
    "Migrations Applied",
    "Total: 5 migrations",
    "Applied: 5",
    "Pending: 0",
))
```

---

## 📊 Цветовая схема

| Тип | Цвет | Символ | Использование |
|-----|------|--------|---------------|
| **Error** | 🔴 Красный | ❌ | Ошибки приложения |
| **Success** | 🟢 Зелёный | ✅ | Успешные операции |
| **Warning** | 🟡 Жёлтый | ⚠️ | Предупреждения |
| **Info** | 🔵 Синий | ℹ️ | Информационные сообщения |

---

## 🚀 Команды в Makefile

```bash
# Запуск демонстрации логирования
make logger-demo

# Запуск сервера
make run

# Запуск тестов
make test

# Генерация отчёта покрытия
make test-coverage
```

---

## 🔍 Когда использовать каждый тип

### ❌ Ошибки (FormatError)

```go
// Ошибки подключения к БД
// Ошибки валидации данных
// Ошибки аутентификации
// Ошибки обработки платежей
fmt.Println(logger.FormatError("Operation Failed", err, details...))
```

### ✅ Успех (FormatSuccess)

```go
// Пользователь успешно зарегистрирован
// Платёж обработан
// Данные синхронизированы
// Сервер запущен
fmt.Println(logger.FormatSuccess("Operation Completed", details...))
```

### ⚠️ Предупреждения (FormatWarning)

```go
// .env файл не найден
// Deprecated API используется
// Высокое использование памяти
// Синхронизация отсрочена
fmt.Println(logger.FormatWarning("Warning Message", details...))
```

### ℹ️ Информация (FormatInfo)

```go
// Конфигурация загружена
// Миграции выполнены
// Сервер готов к работе
// Подключение установлено
fmt.Println(logger.FormatInfo("Information", details...))
```

---

## 🌍 Поддержка окружений

### Development (красивый вывод)

```bash
ENV=development go run ./cmd/server
```

Вывод с цветами и Unicode символами ✨

### Production (JSON)

```bash
ENV=production go run ./cmd/server
```

Стандартный JSON формат Zap для парсирования логами системы 📊

---

## 📝 Заметки

- Функции форматирования работают с `fmt.Println()`
- Можно использовать вместе с `logger.Log.Error()`, `logger.Log.Info()` и т.д.
- ANSI цвета автоматически выключаются в production режиме
- Все детали опциональны - минимум можно передать только название и ошибку

---

## 🎯 Быстрый старт

1. **Просмотрите примеры:**
   ```bash
   make logger-demo
   ```

2. **Интегрируйте в свой код:**
   ```go
   import "event-api/internal/logger"
   
   fmt.Println(logger.FormatError("Error Title", err, "Detail 1", "Detail 2"))
   ```

3. **Запустите сервер:**
   ```bash
   make run
   ```

Готово! 🚀 Теперь вам будут показаны красивые сообщения об ошибках в консоли.
