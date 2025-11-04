# Event API

[![CI/CD Pipeline](https://github.com/Orange-hanter/TuserDuser/actions/workflows/ci.yml/badge.svg)](https://github.com/Orange-hanter/TuserDuser/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.25.0-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

REST API для управления событиями с аутентификацией на основе JWT.

## 🚀 Возможности

- ✅ Регистрация пользователей
- ✅ Верификация email по коду
- ✅ JWT аутентификация
- ✅ Управление сессиями (logout)
- ✅ CORS поддержка
- ✅ Security headers
- ✅ Структурированное логирование
- ✅ **Swagger API документация**
- ✅ **API versioning** (`/v1`)
- ✅ **Graceful shutdown**
- ✅ **PostgreSQL** - персистентное хранилище пользователей
- ✅ **Redis** - коды верификации и token blacklist
- ✅ **Worker Pool** - асинхронная обработка задач
- ✅ **Events CRUD** - управление событиями
- ✅ **SMS Service** - отправка SMS через несколько провайдеров
- ✅ **CI/CD** - автоматическое тестирование и деплой

## 📋 Технологический стек

- **Go** 1.25.0
- **Chi** v5 - HTTP роутер
- **JWT-GO** - JWT токены
- **Bcrypt** - Хеширование паролей
- **Zap** - Логирование
- **CORS** - Cross-origin поддержка
- **Swagger** - API документация
- **PostgreSQL** 17 - Основная БД
- **Redis** 7 - Кеш и временные данные
- **Docker** - Контейнеризация
- **GitHub Actions** - CI/CD
- **PostgreSQL** 18 - Основная БД
- **Redis** 7 - Кеш и временные данные

## 📦 Зависимости

```
github.com/go-chi/chi/v5 v5.2.3         # HTTP роутер
github.com/golang-jwt/jwt/v5 v5.3.0     # JWT
golang.org/x/crypto v0.43.0             # Bcrypt
github.com/rs/cors v1.11.1              # CORS
go.uber.org/zap v1.27.0                 # Логирование
github.com/joho/godotenv v1.5.1         # .env загрузка
github.com/swaggo/http-swagger v1.3.4   # Swagger UI
github.com/swaggo/swag v1.16.6          # Swagger генератор
github.com/lib/pq v1.10.9               # PostgreSQL драйвер
github.com/redis/go-redis/v9 v9.x       # Redis клиент
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
# Server
PORT=8080
ENV=development
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5174
JWT_SECRET=super-secret-key-change-in-production-please
SHUTDOWN_TIMEOUT=30

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=devuser
DB_PASSWORD=devpass
DB_NAME=event_api
DB_SSLMODE=disable

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=devpass
REDIS_DB=0
```

## 🚀 Запуск

### С Docker Compose (рекомендуется)

```bash
# Запустить PostgreSQL и Redis
docker-compose up -d

# Запустить приложение
make run

# Или все вместе
docker-compose up -d && make run
```

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

- `POST /v1/api/auth/register` - Регистрация пользователя
- `POST /v1/api/auth/verify` - Верификация email
- `POST /v1/api/auth/login` - Вход в систему
- `GET /health` - Проверка статуса
- `GET /swagger/*` - **Swagger UI документация**

### Protected endpoints

- `GET /v1/api/auth/me` - Получить текущего пользователя
- `POST /v1/api/auth/logout` - Выход из системы

Полная документация: [API_DOCUMENTATION.md](./API_DOCUMENTATION.md)

## 📖 Swagger документация

Интерактивная API документация доступна по адресу:
**http://localhost:8080/swagger/index.html**

Для генерации обновленной документации:
```bash
make swagger
```

Документация включает:
- ✅ Все доступные endpoints
- ✅ Модели запросов и ответов
- ✅ Примеры использования
- ✅ Возможность тестирования API прямо в браузере

## 🔢 API Versioning

API использует семантическое версионирование через URL префиксы:

- **Текущая версия**: `v1` (`/v1/api/...`)
- **Предыдущие версии**: Поддерживаются для обратной совместимости
- **Будущие версии**: Новые версии будут добавляться как `v2`, `v3` и т.д.

Пример:
```
GET /v1/api/auth/me     # Текущая версия
GET /v2/api/auth/me     # Будущая версия (когда будет готова)
```

## 🧪 Тестирование

### Регистрация

```bash
curl -X POST http://localhost:8080/v1/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "phone": "+79991234567",
    "password": "password123"
  }'
```

### Верификация

```bash
curl -X POST http://localhost:8080/v1/api/auth/verify \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "code": "436194"
  }'
```

### Вход

```bash
curl -X POST http://localhost:8080/v1/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'
```

### Получение профиля

```bash
curl -X GET http://localhost:8080/v1/api/auth/me \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

## 🛠️ Makefile команды

```bash
make build         # Компилирует бинарный файл
make run           # Компилирует и запускает локально
make test          # Запускает тесты
make swagger       # Генерирует Swagger документацию
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

## 🛑 Graceful Shutdown

Приложение поддерживает graceful shutdown для корректного завершения работы:

- **Обработка сигналов**: `SIGTERM` и `SIGINT` (Ctrl+C)
- **Таймаут**: 30 секунд (настраивается через `SHUTDOWN_TIMEOUT`)
- **Закрытие ресурсов**: БД соединения, активные запросы
- **Логирование**: Детальная информация о процессе shutdown

```bash
# Отправка сигнала для graceful shutdown
kill -TERM <pid>
# или
pkill -TERM -f "bin/server"
```

## 📝 Логирование

Приложение использует Zap для структурированного логирования:

```json
{"level":"info","ts":1761425592.289552,"caller":"server/main.go:57","msg":"Сервер запущен","port":":8080","env":"development"}
```

## � Хранилища данных

### PostgreSQL
- **Пользователи**: id, email, phone, password (bcrypt), verified, timestamps
- **События**: id, type, start_time, end_time, duration, place, price_type, need_registration, details (JSONB)
- **Миграции**: Автоматическое применение схемы при старте

См. [DOC/DATABASE.md](DOC/DATABASE.md) для подробностей

### Redis
- **Коды верификации**: `verify:{email}` → `{code}` (TTL: 10 минут)
- **Token Blacklist**: `blacklist:{jwt}` → `"1"` (TTL: время жизни токена)
- **Автоматическое удаление**: Истекшие ключи удаляются автоматически

См. [DOC/REDIS.md](DOC/REDIS.md) для подробностей

### Мониторинг Redis

```bash
# Подключиться к Redis CLI
docker exec -it event_api_redis redis-cli -a devpass

# Посмотреть все ключи
KEYS *

# Мониторинг в реальном времени
MONITOR
```

## 🔄 CI/CD

Проект использует GitHub Actions для автоматизации:

### Workflows

- **CI Pipeline** - Lint, Test, Build, Security Scan, Deploy
- **Staging Pipeline** - Автоматический деплой на staging
- **Release Pipeline** - Релизы с бинарниками для всех платформ

### Быстрый старт

1. Push в `master` → Автоматический деплой на production
2. Push в `develop` → Автоматический деплой на staging
3. Tag `v1.0.0` → Автоматический релиз с бинарниками

### Команды

```bash
# Локальное тестирование как в CI
make ci-test

# Проверка кода
make check

# Деплой
./scripts/deploy.sh production

# Бэкап БД
./scripts/backup.sh
```

Подробнее:
- 📖 [CI/CD Quick Start](./CI_CD_QUICKSTART.md)
- 📚 [Полная документация CI/CD](./CI_CD.md)

## 📱 SMS Service

Интегрированный SMS сервис для отправки кодов верификации:

- **Mock Provider** - для разработки и тестирования
- **SMS.RU** - российский SMS провайдер
- **SMSC.RU** - российский SMS провайдер
- **Twilio** - международный SMS провайдер

Подробнее: [SMS_SERVICE.md](./SMS_SERVICE.md)

## 🚧 Планы развития

- [x] Интеграция с БД (PostgreSQL) ✅
- [x] Redis для кеша и временных данных ✅
- [x] Worker Pool для асинхронных задач ✅
- [x] Events CRUD с JSONB полями ✅
- [x] SMS Service с несколькими провайдерами ✅
- [x] CI/CD с GitHub Actions ✅
- [ ] Refresh tokens
- [ ] 2FA (двухфакторная аутентификация)
- [ ] Social login (Google, GitHub)
- [ ] Email notifications (SMTP/SendGrid)
- [ ] Rate limiting (через Redis)
- [ ] Tests coverage >80%
- [ ] Metrics (Prometheus + Grafana)
- [ ] Kubernetes deployment

## 📚 Документация

- [API Documentation](./API_DOCUMENTATION.md) - Полная документация API
- [CI/CD Quick Start](./CI_CD_QUICKSTART.md) - Быстрый старт с CI/CD
- [CI/CD Documentation](./CI_CD.md) - Подробная документация CI/CD
- [SMS Service](./SMS_SERVICE.md) - Документация SMS сервиса
- [Redis Documentation](./DOC/REDIS.md) - Redis интеграция
- [Testing Guide](./docs/TESTING.md) - Руководство по тестированию

## 📄 Лицензия

MIT

## 👨‍💻 Автор

Event API Team
## �🚧 Планы развития

- [x] Интеграция с БД (PostgreSQL) ✅
- [x] Redis для кеша и временных данных ✅
- [x] Worker Pool для асинхронных задач ✅
- [x] Events CRUD с JSONB полями ✅
- [ ] Refresh tokens
- [ ] 2FA (двухфакторная аутентификация)
- [ ] Social login (Google, GitHub)
- [ ] Email notifications (SMTP/SendGrid)
- [ ] Rate limiting (через Redis)
- [ ] Tests coverage
- [ ] Metrics (Prometheus)

## 📄 Лицензия

MIT

## 👨‍💻 Автор

Event API Team
