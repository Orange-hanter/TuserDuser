# Telegram Service - Руководство по развёртыванию

Пошаговая инструкция по настройке и развёртыванию telegram-service на production сервере.

## Содержание

1. [Требования](#требования)
2. [Первоначальная настройка сервера](#первоначальная-настройка-сервера)
3. [Настройка базы данных](#настройка-базы-данных)
4. [Конфигурация сервиса](#конфигурация-сервиса)
5. [Настройка Telegram бота](#настройка-telegram-бота)
6. [Запуск сервиса](#запуск-сервиса)
7. [CI/CD автоматический деплой](#cicd-автоматический-деплой)
8. [Мониторинг и логи](#мониторинг-и-логи)
9. [Устранение неполадок](#устранение-неполадок)

---

## Требования

### Сервер

- **ОС**: Ubuntu 22.04+ / Debian 11+
- **RAM**: минимум 512 MB
- **CPU**: 1 vCPU
- **Диск**: 1 GB свободного места

### Программное обеспечение

- PostgreSQL 14+
- systemd
- curl (для health checks)

### Сеть

- Открытый порт `50051` (gRPC) — только для внутренней сети
- Открытый порт `9090` (HTTP метрики) — только для внутренней сети
- Доступ к `api.telegram.org` (исходящий HTTPS)

---

## Первоначальная настройка сервера

### Шаг 1: Подключение к серверу

```bash
ssh user@your-server
```

### Шаг 2: Создание системного пользователя

```bash
sudo useradd -r -s /bin/false -d /opt/telegram-service telegramservice
```

### Шаг 3: Создание структуры директорий

```bash
sudo mkdir -p /opt/telegram-service/{bin,logs,scripts}
sudo chown -R telegramservice:telegramservice /opt/telegram-service
sudo chmod 755 /opt/telegram-service
sudo chmod 750 /opt/telegram-service/logs
```

### Шаг 4: Установка systemd сервиса

Создайте файл `/etc/systemd/system/telegram-service.service`:

```ini
[Unit]
Description=Telegram Service - gRPC microservice for Telegram Bot API
Documentation=https://github.com/Orange-hanter/TuserDuser
After=network-online.target postgresql.service
Wants=network-online.target

[Service]
Type=simple
User=telegramservice
Group=telegramservice
WorkingDirectory=/opt/telegram-service

ExecStart=/opt/telegram-service/bin/telegram-service
ExecReload=/bin/kill -HUP $MAINPID

Restart=on-failure
RestartSec=5
TimeoutStartSec=30
TimeoutStopSec=30

# Environment
EnvironmentFile=-/opt/telegram-service/.env

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=/opt/telegram-service/logs

# Logging
StandardOutput=append:/opt/telegram-service/logs/telegram-service.log
StandardError=append:/opt/telegram-service/logs/telegram-service.log

# Limits
LimitNOFILE=65535
LimitNPROC=4096

[Install]
WantedBy=multi-user.target
```

Активируйте сервис:

```bash
sudo systemctl daemon-reload
sudo systemctl enable telegram-service
```

### Шаг 5: Настройка ротации логов

Создайте файл `/etc/logrotate.d/telegram-service`:

```
/opt/telegram-service/logs/*.log {
    daily
    missingok
    rotate 14
    compress
    delaycompress
    notifempty
    create 0640 telegramservice telegramservice
    sharedscripts
    postrotate
        systemctl reload telegram-service > /dev/null 2>&1 || true
    endscript
}
```

### Альтернатива: Автоматическая настройка

Используйте готовый скрипт:

```bash
# На локальной машине
scp telegram-service/scripts/setup-server.sh user@server:/tmp/
scp telegram-service/telegram-service.service user@server:/tmp/

# На сервере
ssh user@server "sudo bash /tmp/setup-server.sh"
```

---

## Настройка базы данных

### Шаг 1: Создание пользователя PostgreSQL

```bash
# Генерация безопасного пароля
PASSWORD=$(openssl rand -base64 24)
echo "Пароль: $PASSWORD"

# Создание пользователя
sudo -u postgres psql -c "CREATE USER telegramservice WITH PASSWORD '$PASSWORD';"
```

### Шаг 2: Создание базы данных

```bash
sudo -u postgres psql -c "CREATE DATABASE telegram_service OWNER telegramservice;"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE telegram_service TO telegramservice;"
```

### Шаг 3: Проверка подключения

```bash
psql -U telegramservice -h localhost -d telegram_service -c "SELECT 1;"
```

### Важно: Миграции

Миграции выполняются **автоматически** при запуске сервиса. Сервис создаёт необходимые таблицы:

- `telegram_bindings` — связь пользователей с Telegram аккаунтами
- `binding_tokens` — токены для привязки
- `delivery_log` — журнал отправленных сообщений

---

## Конфигурация сервиса

### Шаг 1: Создание файла конфигурации

```bash
sudo nano /opt/telegram-service/.env
```

### Шаг 2: Заполнение конфигурации

```bash
# ===================================
# Telegram Service Configuration
# ===================================

# Окружение (development / production)
ENV=production

# ===================================
# Серверы
# ===================================

# gRPC порт (для связи с event-api)
GRPC_PORT=50051

# HTTP порт (метрики и health check)
HTTP_PORT=9090

# ===================================
# База данных
# ===================================

DATABASE_URL=postgres://telegramservice:YOUR_PASSWORD@localhost:5432/telegram_service?sslmode=disable

# ===================================
# Telegram Bot
# ===================================

# Токен бота от @BotFather
TELEGRAM_BOT_TOKEN=1234567890:ABCdefGHIjklMNOpqrsTUVwxyz

# Username бота (без @)
TELEGRAM_BOT_USERNAME=YourBotName_bot

# Режим получения обновлений: polling или webhook
TELEGRAM_UPDATE_MODE=polling

# ===================================
# Polling (если TELEGRAM_UPDATE_MODE=polling)
# ===================================

# Таймаут long polling в секундах
TELEGRAM_POLLING_TIMEOUT=30

# Задержка при ошибке (секунды)
TELEGRAM_POLLING_RETRY_DELAY=3

# ===================================
# Webhook (если TELEGRAM_UPDATE_MODE=webhook)
# ===================================

# Секрет для верификации webhook запросов
TELEGRAM_WEBHOOK_SECRET=your-random-secret-string

# Публичный URL для webhook (Telegram будет отправлять обновления сюда)
# TELEGRAM_WEBHOOK_URL=https://your-domain.com/webhook

# ===================================
# Безопасность привязки
# ===================================

# Секрет для подписи токенов привязки
TELEGRAM_BINDING_SECRET=another-random-secret-string

# Время жизни токена привязки (секунды)
TELEGRAM_BINDING_TTL=600

# ===================================
# Rate Limiting
# ===================================

# Максимум сообщений в секунду
TELEGRAM_RATE_LIMIT=30

# Максимум повторных попыток
TELEGRAM_MAX_RETRIES=5

# Базовая задержка для exponential backoff (секунды)
TELEGRAM_RETRY_BASE_SECONDS=5
```

### Шаг 3: Установка прав доступа

```bash
sudo chown telegramservice:telegramservice /opt/telegram-service/.env
sudo chmod 600 /opt/telegram-service/.env
```

---

## Настройка Telegram бота

### Шаг 1: Создание бота

1. Откройте Telegram и найдите [@BotFather](https://t.me/BotFather)
2. Отправьте команду `/newbot`
3. Введите имя бота (например: "TuserDuser Events")
4. Введите username бота (например: `tuserduser_bot`)
5. Сохраните полученный токен

### Шаг 2: Настройка команд бота

Отправьте @BotFather:

```
/setcommands
```

Выберите вашего бота и отправьте:

```
start - Начать работу с ботом
help - Показать справку
status - Проверить статус привязки
unlink - Отвязать аккаунт
```

### Шаг 3: Настройка описания

```
/setdescription
```

```
Бот для уведомлений о событиях TuserDuser.
Получайте напоминания о мероприятиях в Telegram!
```

### Шаг 4: Выбор режима работы

#### Polling Mode (рекомендуется для начала)

Не требует публичного IP или домена. Сервис сам опрашивает Telegram.

```bash
TELEGRAM_UPDATE_MODE=polling
```

#### Webhook Mode (для production)

Требует публичный HTTPS endpoint.

```bash
TELEGRAM_UPDATE_MODE=webhook
TELEGRAM_WEBHOOK_URL=https://api.your-domain.com/telegram/webhook
TELEGRAM_WEBHOOK_SECRET=your-secret
```

---

## Запуск сервиса

### Шаг 1: Копирование бинарника

Если вы деплоите вручную:

```bash
# Сборка на локальной машине
cd telegram-service
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/telegram-service ./cmd/server

# Копирование на сервер
scp dist/telegram-service user@server:/tmp/

# На сервере
sudo install -o telegramservice -g telegramservice -m 755 /tmp/telegram-service /opt/telegram-service/bin/
```

### Шаг 2: Запуск

```bash
sudo systemctl start telegram-service
```

### Шаг 3: Проверка статуса

```bash
# Статус сервиса
sudo systemctl status telegram-service

# Логи
sudo journalctl -u telegram-service -f

# Health check
curl http://localhost:9090/health
```

### Шаг 4: Проверка gRPC

```bash
# Если установлен grpcurl
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check
```

---

## CI/CD автоматический деплой

После настройки сервера, CI/CD будет автоматически деплоить при push в `master`.

### Что делает CI/CD:

1. **Build**: Компилирует бинарник под Linux AMD64
2. **Deploy**:
   - Копирует бинарник на сервер в `/tmp/`
   - Запускает `install.sh` скрипт
   - Скрипт останавливает сервис, обновляет бинарник, запускает сервис
   - Выполняет health check

### GitHub Secrets (необходимые):

| Secret           | Описание                                |
| ---------------- | --------------------------------------- |
| `DEPLOY_HOST`    | IP или домен сервера                    |
| `DEPLOY_USER`    | SSH пользователь                        |
| `DEPLOY_SSH_KEY` | Приватный SSH ключ                      |
| `DEPLOY_PORT`    | SSH порт (опционально, по умолчанию 22) |

### Ручной запуск деплоя:

1. Перейдите в **Actions** → **Delivery**
2. Нажмите **Run workflow**
3. Выберите ветку и нажмите **Run workflow**

---

## Мониторинг и логи

### Логи сервиса

```bash
# Последние 100 строк
tail -100 /opt/telegram-service/logs/telegram-service.log

# В реальном времени
tail -f /opt/telegram-service/logs/telegram-service.log

# Через journalctl
sudo journalctl -u telegram-service -f
```

### Prometheus метрики

Доступны на `http://localhost:9090/metrics`:

```
# Количество gRPC запросов
grpc_server_handled_total

# Время обработки
grpc_server_handling_seconds

# Telegram API вызовы
telegram_api_requests_total
telegram_api_request_duration_seconds

# Отправленные сообщения
telegram_messages_sent_total
telegram_messages_failed_total
```

### Health Check

```bash
curl http://localhost:9090/health
```

Ответ:

```json
{ "status": "healthy", "service": "telegram-service" }
```

---

## Устранение неполадок

### Сервис не запускается

```bash
# Проверьте логи
sudo journalctl -u telegram-service -n 50 --no-pager

# Частые причины:
# 1. Неверный DATABASE_URL
# 2. Отсутствует TELEGRAM_BOT_TOKEN
# 3. Бинарник не имеет прав на выполнение
```

### Ошибка подключения к БД

```bash
# Проверьте доступность PostgreSQL
sudo systemctl status postgresql

# Проверьте права пользователя
sudo -u postgres psql -c "\du telegramservice"

# Проверьте подключение
psql -U telegramservice -h localhost -d telegram_service -c "SELECT 1;"
```

### Telegram не получает сообщения

```bash
# Проверьте токен бота
curl "https://api.telegram.org/bot<TOKEN>/getMe"

# Если используете webhook, проверьте его статус
curl "https://api.telegram.org/bot<TOKEN>/getWebhookInfo"

# Переключитесь на polling для отладки
```

### gRPC недоступен

```bash
# Проверьте, что порт слушается
ss -tlnp | grep 50051

# Проверьте firewall
sudo ufw status

# Разрешите порт (если нужно)
sudo ufw allow 50051/tcp
```

### Высокое потребление памяти

```bash
# Проверьте использование
ps aux | grep telegram-service

# Перезапустите сервис
sudo systemctl restart telegram-service
```

---

## Полезные команды

```bash
# Статус сервиса
sudo systemctl status telegram-service

# Перезапуск
sudo systemctl restart telegram-service

# Остановка
sudo systemctl stop telegram-service

# Логи в реальном времени
sudo journalctl -u telegram-service -f

# Проверка конфигурации
cat /opt/telegram-service/.env

# Проверка прав
ls -la /opt/telegram-service/

# Проверка портов
ss -tlnp | grep -E '50051|9090'
```

---

## Чек-лист развёртывания

- [ ] Создан системный пользователь `telegramservice`
- [ ] Созданы директории `/opt/telegram-service/{bin,logs,scripts}`
- [ ] Установлен systemd сервис
- [ ] Создана база данных `telegram_service`
- [ ] Создан пользователь БД `telegramservice`
- [ ] Создан и настроен файл `.env`
- [ ] Получен токен бота от @BotFather
- [ ] Загружен бинарник в `/opt/telegram-service/bin/`
- [ ] Сервис запущен и работает
- [ ] Health check проходит успешно
- [ ] Настроены GitHub Secrets для CI/CD
