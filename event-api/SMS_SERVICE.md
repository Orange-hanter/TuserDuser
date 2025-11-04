# SMS Service Documentation

## Обзор

SMS сервис для отправки SMS-сообщений через различные провайдеры. Поддерживает несколько SMS API с единым интерфейсом.

## Поддерживаемые провайдеры

### 1. Mock Provider (для разработки и тестирования)
- **Описание**: Имитирует отправку SMS без реальных API-вызовов
- **Задержка**: 100ms (симуляция реального API)
- **Логирование**: Выводит SMS в логи с эмодзи 📱
- **Тестирование ошибок**: Номера с префиксом +7000 вернут ошибку

```env
SMS_PROVIDER=mock
```

### 2. SMS.RU
- **Описание**: Российский SMS-провайдер
- **API**: https://sms.ru
- **Поддержка**: Россия, СНГ
- **Требования**: API ключ (api_id)

```env
SMS_PROVIDER=smsru
SMS_API_KEY=your_smsru_api_id
SMS_FROM=YourSenderName
```

### 3. SMSC.RU
- **Описание**: Российский SMS-провайдер
- **API**: https://smsc.ru
- **Поддержка**: Россия, СНГ, международные
- **Требования**: Логин и пароль

```env
SMS_PROVIDER=smsc
SMS_API_KEY=your_smsc_login
SMS_API_TOKEN=your_smsc_password
SMS_FROM=YourSenderName
```

### 4. Twilio
- **Описание**: Международный SMS-провайдер
- **API**: https://twilio.com
- **Поддержка**: 180+ стран
- **Требования**: Account SID и Auth Token

```env
SMS_PROVIDER=twilio
SMS_API_KEY=your_twilio_account_sid
SMS_API_TOKEN=your_twilio_auth_token
SMS_FROM=+1234567890
```

## Конфигурация

### Переменные окружения (.env)

```env
# SMS Provider Configuration
# Поддерживаемые провайдеры: mock, smsru, smsc, twilio
SMS_PROVIDER=mock

# API ключи (зависит от провайдера)
# SMS.RU: API ключ (api_id)
# SMSC.RU: Логин
# Twilio: Account SID
SMS_API_KEY=

# API токен/пароль (для некоторых провайдеров)
# SMSC.RU: Пароль
# Twilio: Auth Token
SMS_API_TOKEN=

# Отправитель SMS (имя или номер)
# SMS.RU/SMSC.RU: Буквенное имя отправителя
# Twilio: Номер телефона в формате E.164 (+1234567890)
SMS_FROM=EventAPI
```

## Использование

### Базовая отправка SMS

```go
// Инициализация SMS сервиса
smsService, err := sms.NewService(ctx, smsConfig, logger)
if err != nil {
    log.Fatal(err)
}

// Отправка SMS
err = smsService.SendSMS(ctx, "+79991234567", "Привет, это тестовое сообщение!")
if err != nil {
    log.Printf("Ошибка отправки SMS: %v", err)
}
```

### Отправка кода верификации

```go
// Генерация и отправка 6-значного кода
code, err := smsService.SendVerificationCode(ctx, "+79991234567")
if err != nil {
    log.Printf("Ошибка отправки кода: %v", err)
}
log.Printf("Отправлен код: %s", code)
```

### Отправка кода сброса пароля

```go
err := smsService.SendPasswordReset(ctx, "+79991234567", "123456")
if err != nil {
    log.Printf("Ошибка отправки SMS: %v", err)
}
```

### Асинхронная отправка через Worker Pool

```go
// В AuthService регистрация
go func() {
    if err := s.sendSMSVerificationCode(ctx, phone, code); err != nil {
        s.logger.Error("Не удалось отправить SMS с кодом верификации",
            zap.String("phone", phone),
            zap.Error(err))
    }
}()
```

## Интеграция в AuthService

SMS сервис интегрирован в `AuthService` и автоматически отправляет SMS при:

1. **Регистрации пользователя** (`Register`)
   - Отправляет код верификации на email И на телефон
   - Асинхронно через worker pool
   - Код действителен 10 минут

2. **Сбросе пароля** (будущее)
   - Будет отправлять код сброса пароля по SMS

## Логирование

### Mock Provider
```
2025-10-30T21:46:58.185+0300 INFO sms/sms.go:69 📱 [MOCK SMS] Отправка сообщения
    {"phone": "+79991112255", "message": "Ваш код верификации: 121368\nКод действителен 10 минут."}
2025-10-30T21:46:58.286+0300 INFO sms/sms.go:69 ✅ [MOCK SMS] Сообщение успешно отправлено
    {"phone": "+79991112255"}
```

### Реальные провайдеры
```
2025-10-30T21:46:58.185+0300 INFO sms/sms.go:89 Отправка SMS
    {"phone": "+79991112255", "provider": "SMS.RU"}
2025-10-30T21:46:58.286+0300 INFO sms/sms.go:89 SMS успешно отправлена
    {"phone": "+79991112255", "provider": "SMS.RU"}
```

## Обработка ошибок

```go
err := smsService.SendSMS(ctx, phone, message)
if err != nil {
    // Ошибки могут быть:
    // - Контекст отменен
    // - Неверный формат номера
    // - Ошибка API провайдера
    // - Недостаточно средств
    // - Сеть недоступна
    
    s.logger.Error("Ошибка отправки SMS",
        zap.String("phone", phone),
        zap.Error(err))
}
```

## Тестирование

### Локальная разработка

1. Используйте Mock провайдер:
```env
SMS_PROVIDER=mock
```

2. Зарегистрируйте пользователя:
```bash
curl -X POST http://localhost:8080/v1/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "phone": "+79991234567",
    "password": "password123"
  }'
```

3. Проверьте логи - увидите SMS с кодом верификации:
```bash
tail -f /tmp/server_new.log | grep "📱"
```

### Тестирование ошибок с Mock провайдером

Номера телефонов с префиксом `+7000` будут возвращать ошибку:
```bash
curl -X POST http://localhost:8080/v1/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "phone": "+70001234567",
    "password": "password123"
  }'
```

### Production тестирование

1. Настройте реального провайдера (SMS.RU/SMSC.RU/Twilio)
2. Добавьте API ключи в `.env`
3. Перезапустите сервер
4. Отправьте тестовую SMS на свой номер

## Производительность

- **Асинхронная отправка**: SMS не блокирует регистрацию
- **Worker Pool**: 5 воркеров обрабатывают задачи параллельно
- **Timeout**: 30 секунд на отправку одной SMS
- **Retry**: Нет автоматических повторов (TODO)

## Безопасность

1. **API ключи**: Хранятся в переменных окружения
2. **Не логируем**: API ключи не попадают в логи
3. **Rate limiting**: TODO - добавить ограничение частоты отправки
4. **Валидация номеров**: Базовая проверка формата

## Roadmap

- [ ] Автоматические повторы при ошибках (retry with backoff)
- [ ] Rate limiting для защиты от спама
- [ ] Поддержка шаблонов сообщений
- [ ] Статистика отправок (успешные/неудачные)
- [ ] Unit тесты для всех провайдеров
- [ ] Integration тесты
- [ ] Поддержка множественной отправки (bulk SMS)
- [ ] Webhook для отслеживания статуса доставки
- [ ] Поддержка других провайдеров (Vonage, MessageBird, etc.)

## Troubleshooting

### SMS не отправляются

1. Проверьте логи:
```bash
tail -f /tmp/server_new.log | grep -E "(SMS|📱|error)"
```

2. Убедитесь, что провайдер правильно настроен:
```bash
echo $SMS_PROVIDER
echo $SMS_API_KEY
```

3. Проверьте, что worker pool запущен:
```
Worker pool started {"workers": 5}
```

### Mock провайдер не логирует SMS

Убедитесь, что:
- `SMS_PROVIDER=mock` в .env
- Сервер перезапущен после изменения .env
- Логи перенаправлены в файл правильно

### Реальный провайдер возвращает ошибку

Проверьте:
- API ключи правильные и действительные
- Достаточно средств на балансе
- Номер телефона в правильном формате
- Имя отправителя зарегистрировано (для SMS.RU/SMSC.RU)

## Примеры использования

### Пример 1: Регистрация с SMS верификацией

```bash
# Регистрация
curl -X POST http://localhost:8080/v1/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "phone": "+79991234567",
    "password": "SecurePass123"
  }'

# Ответ:
{
  "user": {
    "id": "...",
    "email": "user@example.com",
    "phone": "+79991234567",
    "verified": false
  },
  "verify_code": "123456"  # Только для dev окружения
}

# Верификация (TODO)
curl -X POST http://localhost:8080/v1/api/auth/verify \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "+79991234567",
    "code": "123456"
  }'
```

### Пример 2: Смена провайдера

```bash
# Остановка сервера
kill $(lsof -ti:8080)

# Изменение провайдера в .env
echo "SMS_PROVIDER=smsru" > .env
echo "SMS_API_KEY=your_api_key" >> .env
echo "SMS_FROM=YourApp" >> .env

# Перезапуск
./bin/server
```

## Архитектура

```
internal/sms/
├── sms.go       # Основной сервис и интерфейс Provider
├── mock.go      # Mock провайдер для тестирования
├── smsru.go     # SMS.RU провайдер
├── smsc.go      # SMSC.RU провайдер
└── twilio.go    # Twilio провайдер
```

Интерфейс `Provider` позволяет легко добавлять новые провайдеры:

```go
type Provider interface {
    SendSMS(ctx context.Context, phone, message string) error
    Name() string
}
```

## Контакты и поддержка

Если возникли вопросы или проблемы:
1. Проверьте логи сервера
2. Убедитесь, что конфигурация правильная
3. Попробуйте Mock провайдер для тестирования
4. Проверьте документацию выбранного SMS API провайдера
