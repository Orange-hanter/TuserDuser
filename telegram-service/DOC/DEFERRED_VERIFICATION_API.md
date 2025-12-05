# Deferred Telegram Verification API

## Рабочая записка для интеграции

Данный документ описывает API для реализации двухэтапной верификации через Telegram.

---

## Обзор схемы

```
┌─────────────────┐                           ┌──────────────┐                          ┌──────────────────┐
│    Frontend     │                           │   event-api  │                          │ telegram-service │
│                 │                           │              │                          │                  │
│ 1. Регистрация  │──POST /auth/register────► │ 2. Создаёт   │──gRPC: RegisterPending──►│ 3. Сохраняет     │
│    с telegram   │   verification_type:      │    pending   │   Verification           │    pending code  │
│                 │   "telegram"              │    user      │                          │    + binding     │
│                 │◄──────────────────────────│◄─────────────│──────────────────────────│    token         │
│                 │   {user_id,               │              │   {deeplink, code,       │                  │
│                 │    telegram_deeplink,     │              │    expires_at}           │                  │
│                 │    telegram_code}         │              │                          │                  │
└────────┬────────┘                           └──────────────┘                          └─────────┬────────┘
         │                                                                                        │
         │ 4. Показывает пользователю:                                                           │
         │    - Deeplink (кнопка "Открыть Telegram")                                             │
         │    - 6-символьный код для ручного ввода                                               │
         │                                                                                        │
         │                                                                                        │
         ▼                                                                                        │
┌─────────────────┐                                                                              │
│    Telegram     │                                                                              │
│      App        │                                                                              │
│                 │──────────────── 5. /start TOKEN или код ─────────────────────────────────────┤
│                 │◄─────────────── 6. Бот отправляет verify code ──────────────────────────────│
│                 │                   "🔐 Ваш код: 123456"                                       │
└─────────────────┘                                                                              │
         │                                                                                        │
         │ 7. Пользователь видит код                                                             │
         │                                                                                        │
         ▼                                                                                        │
┌─────────────────┐                           ┌──────────────┐                                   │
│    Frontend     │                           │   event-api  │                                   │
│                 │──POST /auth/verify───────►│              │                                   │
│ 8. Вводит код   │   {email, code}           │ 9. Проверяет │                                   │
│                 │◄──────────────────────────│    код       │                                   │
│                 │   {access_token, user}    │    Активирует│                                   │
│                 │                           │    аккаунт   │                                   │
└─────────────────┘                           └──────────────┘
```

---

## API telegram-service (gRPC)

### Подключение

```
Host: localhost:50051
Service: telegram.v1.TelegramService
Codec: JSON (не protobuf)
```

---

### 1. RegisterPendingVerification

Регистрирует verification code, который будет автоматически отправлен в Telegram после привязки.

**Request:**

```json
{
  "user_id": "usr_abc123",
  "verification_code": "123456",
  "ttl_minutes": 10
}
```

| Поле                | Тип    | Обязательное | Описание                                           |
| ------------------- | ------ | ------------ | -------------------------------------------------- |
| `user_id`           | string | ✅           | ID пользователя из event-api                       |
| `verification_code` | string | ✅           | 6-значный код верификации                          |
| `ttl_minutes`       | int32  | ❌           | TTL для pending verification (по умолчанию 10 мин) |

**Response:**

```json
{
  "success": true,
  "deeplink": "https://t.me/BrestEvents_bot?start=eyJhbGciOi...",
  "token": "eyJhbGciOi...",
  "code": "A3X9K2",
  "expires_at_unix": 1733400000
}
```

| Поле              | Тип    | Описание                                  |
| ----------------- | ------ | ----------------------------------------- |
| `success`         | bool   | Результат операции                        |
| `deeplink`        | string | Полная ссылка для открытия в Telegram     |
| `token`           | string | Raw binding token (для deeplink)          |
| `code`            | string | 6-символьный код для ручного ввода в бота |
| `expires_at_unix` | int64  | Время истечения (Unix timestamp)          |
| `error_code`      | string | Код ошибки (если success=false)           |
| `error_message`   | string | Описание ошибки                           |

**Ошибки:**

- `invalid_user_id` — пустой user_id
- `invalid_code` — пустой verification_code
- `service_unavailable` — внутренняя ошибка

---

### 2. GetPendingVerificationStatus

Проверяет, есть ли у пользователя ожидающая отправки верификация.

**Request:**

```json
{
  "user_id": "usr_abc123"
}
```

**Response:**

```json
{
  "success": true,
  "has_pending": true,
  "expires_at_unix": 1733400000
}
```

| Поле              | Тип   | Описание                                   |
| ----------------- | ----- | ------------------------------------------ |
| `has_pending`     | bool  | Есть ли pending verification               |
| `expires_at_unix` | int64 | Время истечения (0 если has_pending=false) |

---

### 3. IsUserBound

Проверяет, привязан ли Telegram к пользователю.

**Request:**

```json
{
  "user_id": "usr_abc123"
}
```

**Response:**

```json
{
  "success": true,
  "is_bound": true,
  "status": "active"
}
```

| status    | Описание                         |
| --------- | -------------------------------- |
| `active`  | Привязка активна                 |
| `blocked` | Пользователь заблокировал бота   |
| `revoked` | Привязка отключена пользователем |
| `pending` | Ожидает активации                |

---

### 4. GetBindingStatus

Возвращает детальную информацию о привязке.

**Request:**

```json
{
  "user_id": "usr_abc123"
}
```

**Response:**

```json
{
  "success": true,
  "status": "active",
  "telegram_username": "john_doe",
  "telegram_first_name": "John",
  "telegram_last_name": "Doe",
  "chat_id": 123456789,
  "created_at_unix": 1733300000,
  "updated_at_unix": 1733350000,
  "blocked_reason": ""
}
```

---

## Интеграция с event-api

### Изменения в POST /v1/api/auth/register

**Новый параметр `verification_type`:**

```json
{
  "email": "user@example.com",
  "password": "securepassword",
  "phone": "+79001234567",
  "verification_type": "telegram"
}
```

| verification_type | Описание                                      |
| ----------------- | --------------------------------------------- |
| `email`           | Отправить код на email                        |
| `sms`             | Отправить код по SMS                          |
| `both`            | Отправить и email, и SMS (по умолчанию)       |
| `telegram`        | **Новое:** Отложенная отправка через Telegram |

### Ответ при verification_type=telegram

```json
{
  "user": {
    "id": "usr_abc123",
    "email": "user@example.com",
    "verified": false
  },
  "telegram_binding": {
    "deeplink": "https://t.me/BrestEvents_bot?start=eyJhbGciOi...",
    "code": "A3X9K2",
    "expires_at": "2025-12-05T15:00:00Z"
  }
}
```

**Важно:** При `verification_type=telegram` не возвращается JWT токен — пользователь должен сначала привязать Telegram и получить код.

---

## Пример использования (Frontend)

### 1. Регистрация с Telegram

```typescript
async function registerWithTelegram(
  email: string,
  password: string,
  phone: string,
) {
  const response = await fetch("/v1/api/auth/register", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      email,
      password,
      phone,
      verification_type: "telegram",
    }),
  });

  const data = await response.json();

  if (data.telegram_binding) {
    // Показать модальное окно с инструкцией
    showTelegramBindingModal({
      deeplink: data.telegram_binding.deeplink,
      code: data.telegram_binding.code,
      expiresAt: data.telegram_binding.expires_at,
      userId: data.user.id,
    });
  }
}
```

### 2. Модальное окно привязки

```tsx
function TelegramBindingModal({ deeplink, code, expiresAt, userId }) {
  const [codeReceived, setCodeReceived] = useState(false);

  // Polling статуса привязки
  useEffect(() => {
    const interval = setInterval(async () => {
      const status = await checkBindingStatus(userId);
      if (status.is_bound) {
        setCodeReceived(true);
        clearInterval(interval);
      }
    }, 3000); // каждые 3 секунды

    return () => clearInterval(interval);
  }, [userId]);

  return (
    <Modal>
      <h2>Привязка Telegram</h2>

      {!codeReceived ? (
        <>
          <p>Откройте Telegram и привяжите аккаунт:</p>

          <a href={deeplink} target="_blank" className="btn-primary">
            📱 Открыть в Telegram
          </a>

          <Divider>или</Divider>

          <p>Отправьте этот код боту @BrestEvents_bot:</p>
          <CodeDisplay>{code}</CodeDisplay>

          <Timer expiresAt={expiresAt} />
        </>
      ) : (
        <>
          <SuccessIcon />
          <p>Telegram привязан! Код верификации отправлен.</p>
          <p>Введите полученный код:</p>
          <CodeInput onSubmit={handleVerify} />
        </>
      )}
    </Modal>
  );
}
```

### 3. Проверка статуса привязки

```typescript
async function checkBindingStatus(userId: string): Promise<BindingStatus> {
  // Через event-api proxy или напрямую к telegram-service
  const response = await fetch(
    `/v1/api/telegram/binding/status?user_id=${userId}`,
  );
  return response.json();
}
```

### 4. Верификация кода

```typescript
async function verifyCode(email: string, code: string) {
  const response = await fetch("/v1/api/auth/verify", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, code }),
  });

  const data = await response.json();

  if (data.access_token) {
    // Успешная верификация
    saveToken(data.access_token);
    redirect("/dashboard");
  }
}
```

---

## Таймлайны и TTL

| Объект               | TTL                   | Описание                  |
| -------------------- | --------------------- | ------------------------- |
| Binding token        | 1 час                 | Токен в deeplink          |
| Binding code         | 1 час                 | 6-символьный код          |
| Pending verification | 10 мин (настраиваемо) | Код верификации           |
| Verification code    | 10 мин                | Код для POST /auth/verify |

---

## Обработка ошибок

### Истёк срок deeplink/кода

Если пользователь не успел привязать Telegram:

1. Frontend показывает "Время истекло"
2. Кнопка "Получить новый код"
3. Повторный POST `/auth/register` с теми же данными (или специальный endpoint для resend)

### Пользователь заблокировал бота

При попытке отправить код в заблокированный чат:

1. telegram-service помечает binding как `blocked`
2. При следующей попытке возвращает ошибку `blocked`
3. Если пользователь разблокирует бота и напишет `/start`, binding автоматически активируется

### Telegram недоступен

Если telegram-service недоступен:

1. event-api возвращает ошибку
2. Frontend предлагает альтернативные способы верификации (email/sms)

---

## Метрики Prometheus

Новые метрики для мониторинга:

```
# Количество зарегистрированных pending verification
telegram_pending_verifications_registered_total

# Количество отправленных кодов после привязки
telegram_pending_verifications_sent_total{status="sent|failed"}

# Количество привязок по статусу
telegram_bindings_total{status="active|blocked|revoked"}

# Количество сгенерированных binding links
telegram_binding_links_generated_total
```

---

## Чеклист интеграции

### event-api ✅

- [x] Добавить обработку `verification_type: "telegram"` в `Register`
- [x] Вызывать `telegramClient.RegisterPendingVerification()` вместо отправки кода
- [x] Возвращать `telegram_binding` в ответе
- [x] Добавить endpoint `/v1/api/telegram/binding/status` (proxy к gRPC)
- [x] Обновить `/v1/api/auth/resend` для поддержки telegram

### Frontend

- [ ] Добавить выбор способа верификации на экране регистрации
- [ ] Реализовать модальное окно привязки Telegram
- [ ] Polling статуса привязки
- [ ] Обработка истечения срока действия
- [ ] Fallback на email/sms при ошибках

### telegram-service ✅

- [x] `RegisterPendingVerification` gRPC метод
- [x] `GetPendingVerificationStatus` gRPC метод
- [x] Автоматическая отправка кода после `HandleStartCommand`
- [x] Автоматическая отправка кода после `HandleBindingCode`
- [x] Миграции для `telegram_pending_verifications`
- [x] Интеграционные тесты
