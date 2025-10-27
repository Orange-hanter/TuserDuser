# Event API

REST API для управления событиями с аутентификацией на основе JWT.

## 🚀 Возможности

- ✅ Регистрация пользователей
- ✅ Верификация email по коду
- ✅ JWT аутентификация
- ✅ Управление сессиями (logout)
- ✅ CORS поддержка
- ✅ Security headers
- ✅ Структурированное логирование

## 📋 Технологический стек

- **Go** 1.25.0
- **Chi** v5 - HTTP роутер
- **JWT-GO** - JWT токены
- **Bcrypt** - Хеширование паролей
- **Zap** - Логирование
- **CORS** - Cross-origin поддержка

## 📦 Зависимости

```
github.com/go-chi/chi/v5 v5.2.3         # HTTP роутер
github.com/golang-jwt/jwt/v5 v5.3.0     # JWT
golang.org/x/crypto v0.43.0             # Bcrypt
github.com/rs/cors v1.11.1              # CORS
go.uber.org/zap v1.27.0                 # Логирование
github.com/joho/godotenv v1.5.1         # .env загрузка
```

## 🏗️ Структура проекта

```
event-api/
├── cmd/
│   └── server/
│       └── main.go                 # Точка входа
├── internal/
│   ├── config/
│   │   └── config.go              # Конфигурация
│   ├── handlers/
│   │   ├── health.go              # Health check
│   │   └── auth.go                # Auth endpoints
│   ├── middleware/
│   │   ├── security.go            # Security headers
│   │   └── auth.go                # JWT middleware
│   ├── models/
│   │   └── auth.go                # Data models
│   ├── service/
│   │   └── auth.go                # Business logic
│   └── logger/
│       └── logger.go              # Logging setup
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── go.mod
├── go.sum
├── .env
└── API_DOCUMENTATION.md
```

## ⚙️ Конфигурация

Создайте файл `.env` в корне проекта:

```env
PORT=8080
ENV=development
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000
JWT_SECRET=super-secret-key-change-in-production-please
```

## 🚀 Запуск

### Локально

```bash
# Установка зависимостей
go mod download

# Компиляция
make build

# Запуск
make run

# Или простой запуск
go run ./cmd/server
```

### Docker

```bash
# Сборка образа
make docker-build

# Запуск контейнера
make docker-run
```

## 📝 API Endpoints

### Public endpoints

- `POST /api/auth/register` - Регистрация пользователя
- `POST /api/auth/verify` - Верификация email
- `POST /api/auth/login` - Вход в систему
- `GET /health` - Проверка статуса

### Protected endpoints

- `GET /api/auth/me` - Получить текущего пользователя
- `POST /api/auth/logout` - Выход из системы

Полная документация: [API_DOCUMENTATION.md](./API_DOCUMENTATION.md)

## 🧪 Тестирование

### Регистрация

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "phone": "+79991234567",
    "password": "password123"
  }'
```

### Верификация

```bash
curl -X POST http://localhost:8080/api/auth/verify \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "code": "436194"
  }'
```

### Вход

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'
```

### Получение профиля

```bash
curl -X GET http://localhost:8080/api/auth/me \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

## 🛠️ Makefile команды

```bash
make build         # Компилирует бинарный файл
make run           # Компилирует и запускает локально
make test          # Запускает тесты
make docker-build  # Создает Docker образ
make docker-run    # Запускает контейнер
```

## 🔒 Безопасность

- ✅ Пароли хешируются с bcrypt
- ✅ JWT с HMAC-SHA256
- ✅ Black list токенов для logout
- ✅ Security headers (HSTS, X-Frame-Options, X-Content-Type-Options)
- ✅ CORS настроен
- ✅ Валидация input данных

## 📝 Логирование

Приложение использует Zap для структурированного логирования:

```json
{"level":"info","ts":1761425592.289552,"caller":"server/main.go:57","msg":"Сервер запущен","port":":8080","env":"development"}
```

## 🚧 Планы развития

- [ ] Интеграция с БД (PostgreSQL)
- [ ] Refresh tokens
- [ ] 2FA (двухфакторная аутентификация)
- [ ] Social login (Google, GitHub)
- [ ] Email notifications
- [ ] Rate limiting
- [ ] Tests coverage

## 📄 Лицензия

MIT

## 👨‍💻 Автор

Event API Team

---

**Примечание**: Текущая реализация использует in-memory хранилище. Для production используйте настоящую базу данных и Redis для кеша.
