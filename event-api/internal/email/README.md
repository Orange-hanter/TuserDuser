# Email Service

Email сервис для отправки уведомлений пользователям.

## Возможности

- ✅ Поддержка нескольких провайдеров (Mock, SMTP, SendGrid, Mailgun)
- ✅ Отправка текстовых и HTML email
- ✅ Готовые шаблоны для верификации и сброса пароля
- ✅ Логирование всех операций
- ✅ Graceful обработка ошибок

## Провайдеры

### 1. Mock Provider (для разработки)

Только логирует email, ничего не отправляет.

```env
EMAIL_PROVIDER=mock
```

### 2. SMTP Provider

Универсальный провайдер для любого SMTP сервера.

```env
EMAIL_PROVIDER=smtp
EMAIL_FROM=noreply@tuserduser.online
EMAIL_FROM_NAME=TuserDuser
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
```

**Популярные SMTP серверы:**

- Gmail: `smtp.gmail.com:587` (требует App Password)
- Yandex: `smtp.yandex.ru:465`
- Mail.ru: `smtp.mail.ru:465`
- SendGrid SMTP: `smtp.sendgrid.net:587`

### 3. SendGrid Provider

Использует SendGrid API.

```env
EMAIL_PROVIDER=sendgrid
EMAIL_API_KEY=SG.xxxxxxxxxxxxx
EMAIL_FROM=noreply@tuserduser.online
EMAIL_FROM_NAME=TuserDuser
```

### 4. Mailgun Provider

Использует Mailgun API.

```env
EMAIL_PROVIDER=mailgun
EMAIL_API_KEY=key-xxxxxxxxxxxxx
EMAIL_FROM=noreply@tuserduser.online
EMAIL_FROM_NAME=TuserDuser
```

## Использование

### Инициализация сервиса

```go
import (
    "event-api/internal/config"
    "event-api/internal/email"
    "event-api/internal/logger"
)

// Создаем конфигурацию
cfg := config.Load()
emailConfig := &email.Config{
    Provider:     cfg.EmailProvider,
    APIKey:       cfg.EmailAPIKey,
    SMTPHost:     cfg.SMTPHost,
    SMTPPort:     cfg.SMTPPort,
    SMTPUsername: cfg.SMTPUsername,
    SMTPPassword: cfg.SMTPPassword,
    From:         cfg.EmailFrom,
    FromName:     cfg.EmailFromName,
}

// Создаем сервис
emailService, err := email.NewService(emailConfig, logger.Log)
if err != nil {
    log.Fatal("Ошибка инициализации email сервиса:", err)
}
```

### Отправка простого email

```go
ctx := context.Background()
err := emailService.SendEmail(
    ctx,
    "user@example.com",
    "Тестовое письмо",
    "Привет! Это тестовое письмо.",
)
```

### Отправка HTML email

```go
htmlBody := `
<html>
<body>
    <h1>Привет!</h1>
    <p>Это <strong>HTML</strong> письмо.</p>
</body>
</html>
`

err := emailService.SendHTMLEmail(
    ctx,
    "user@example.com",
    "HTML письмо",
    htmlBody,
)
```

### Отправка кода верификации

```go
// Простой текстовый формат
err := emailService.SendVerificationEmail(
    ctx,
    "user@example.com",
    "123456",
)

// Красивый HTML формат
err := emailService.SendVerificationHTMLEmail(
    ctx,
    "user@example.com",
    "123456",
)
```

### Отправка ссылки для сброса пароля

```go
resetLink := "https://tuserduser.online/reset-password?token=abc123"
err := emailService.SendPasswordResetEmail(
    ctx,
    "user@example.com",
    resetLink,
)
```

## Интеграция с AuthService

Добавьте email сервис в `AuthService`:

```go
type AuthService struct {
    db           *database.DB
    redis        *redis.Client
    sms          *sms.Service
    email        *email.Service  // Добавить
    jwtSecret    string
    jwtExpiresIn int64
}

func (s *AuthService) Register(req *models.RegisterRequest) (*models.User, string, error) {
    // ... существующий код регистрации ...

    // Отправляем email с кодом верификации
    if s.email != nil {
        go func() {
            ctx := context.Background()
            if err := s.email.SendVerificationHTMLEmail(ctx, user.Email, verifyCode); err != nil {
                logger.Log.Error("Ошибка отправки email", zap.Error(err))
            }
        }()
    }

    return user, verifyCode, nil
}
```

## Настройка Gmail SMTP

1. Включите 2FA в настройках Google аккаунта
2. Создайте App Password:
   - Перейдите на https://myaccount.google.com/apppasswords
   - Выберите "Mail" и "Other (Custom name)"
   - Сохраните сгенерированный пароль
3. Используйте его в `.env`:

```env
EMAIL_PROVIDER=smtp
EMAIL_FROM=your-email@gmail.com
EMAIL_FROM_NAME=Your Name
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=xxxx xxxx xxxx xxxx
```

## Тестирование

```bash
# В режиме разработки используйте mock
EMAIL_PROVIDER=mock

# Проверка email в логах
make logs-follow-api | grep "📧"
```

## Структура файлов

```
internal/email/
├── email.go      # Основной сервис и интерфейс
├── mock.go       # Mock провайдер для тестирования
├── smtp.go       # SMTP провайдер (Gmail, Yandex, etc.)
├── sendgrid.go   # SendGrid API провайдер
├── mailgun.go    # Mailgun API провайдер
└── README.md     # Документация
```

## Добавление нового провайдера

1. Создайте файл `provider_name.go`
2. Реализуйте интерфейс `Provider`:

```go
type Provider interface {
    SendEmail(ctx context.Context, to, subject, body string) error
    SendHTMLEmail(ctx context.Context, to, subject, htmlBody string) error
    GetName() string
}
```

3. Добавьте case в `NewService()` в `email.go`

## Best Practices

1. **Async отправка**: Отправляйте email в горутинах, чтобы не блокировать запросы
2. **Retry логика**: Для production добавьте повторные попытки при ошибках
3. **Rate limiting**: Соблюдайте лимиты провайдеров
4. **Валидация**: Проверяйте email адреса перед отправкой
5. **Unsubscribe**: Добавляйте ссылку отписки для массовых рассылок
6. **Monitoring**: Отслеживайте delivery rate и ошибки

## Troubleshooting

### SMTP ошибки

**"535 Authentication failed"**

- Проверьте username/password
- Для Gmail используйте App Password

**"Connection timeout"**

- Проверьте firewall правила
- Попробуйте другой порт (587, 465, 25)

**"TLS handshake failed"**

- Обновите Go до последней версии
- Проверьте SSL сертификаты сервера

### SendGrid/Mailgun ошибки

**"Unauthorized"**

- Проверьте API ключ
- Убедитесь, что домен верифицирован

**"Daily sending quota exceeded"**

- Проверьте лимиты плана
- Обновите план или подождите до следующего дня
