# Email Verification Integration - Usage Guide

## Overview

The email verification system is now fully integrated into the registration flow. Users can choose to receive verification codes via:

- **Email only** (`verification_type: "email"`)
- **SMS only** (`verification_type: "sms"`)
- **Both email and SMS** (`verification_type: "both"` or omit the field)

## Configuration

Add these environment variables to your `.env` file:

```env
# Email Service Configuration
EMAIL_PROVIDER=mock          # Options: mock, smtp, sendgrid, mailgun
EMAIL_FROM=noreply@tuserduser.online
EMAIL_FROM_NAME=TuserDuser

# For SMTP (e.g., Gmail)
# EMAIL_PROVIDER=smtp
# SMTP_HOST=smtp.gmail.com
# SMTP_PORT=587
# SMTP_USERNAME=your-email@gmail.com
# SMTP_PASSWORD=your-app-password

# For SendGrid
# EMAIL_PROVIDER=sendgrid
# EMAIL_API_KEY=SG.xxxxxxxxxxxxx

# For Mailgun
# EMAIL_PROVIDER=mailgun
# EMAIL_API_KEY=key-xxxxxxxxxxxxx
```

## API Usage Examples

### 1. Register with Email Verification Only

```bash
curl -X POST https://api.tuserduser.online/v1/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "phone": "+79991234567",
    "password": "securepassword123",
    "verification_type": "email"
  }'
```

**Response:**

```json
{
  "user": {
    "id": "usr_abc123",
    "email": "user@example.com",
    "phone": "+79991234567",
    "verified": false,
    "created_at": "2025-11-11T18:00:00Z",
    "updated_at": "2025-11-11T18:00:00Z"
  },
  "verify_code": "123456"
}
```

**Email sent to user:**

```
Subject: Код верификации

Здравствуйте!

Для завершения регистрации введите следующий код:

  1 2 3 4 5 6

Код действителен в течение 10 минут.

С уважением,
Команда TuserDuser
```

### 2. Register with SMS Verification Only

```bash
curl -X POST https://api.tuserduser.online/v1/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "phone": "+79991234567",
    "password": "securepassword123",
    "verification_type": "sms"
  }'
```

### 3. Register with Both Email and SMS (Default)

```bash
# verification_type can be "both" or omitted entirely
curl -X POST https://api.tuserduser.online/v1/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "phone": "+79991234567",
    "password": "securepassword123",
    "verification_type": "both"
  }'
```

Or simply omit the field:

```bash
curl -X POST https://api.tuserduser.online/v1/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "phone": "+79991234567",
    "password": "securepassword123"
  }'
```

## Email Templates

The system includes beautifully styled HTML email templates:

### Verification Code Email

The HTML template includes:

- Responsive design (mobile-friendly)
- Clean, professional styling
- Large, easy-to-read verification code
- Clear instructions and expiration time
- Company branding

You can customize the template in `internal/email/email.go`:

- Colors: Change `#4CAF50` to your brand color
- Text: Modify Russian or add multi-language support
- Layout: Adjust spacing, fonts, and structure

## Flow Diagram

```
User Registration Request
         ↓
    Validate Input
         ↓
    Create User in DB
         ↓
    Generate 6-digit Code
         ↓
    Store Code in Redis (10 min TTL)
         ↓
   Check verification_type
         ↓
    ┌────┴────┬────────┐
    ↓         ↓        ↓
  Email     SMS     Both
    ↓         ↓        ↓
Send HTML  Send    Send Both
  Email     SMS    (Async)
    ↓         ↓        ↓
    └─────────┴────────┘
         ↓
   Return Response
   (with verify_code in dev)
```

## Development vs Production

### Development Mode (Mock Provider)

```env
EMAIL_PROVIDER=mock
```

- Emails are logged to console
- No actual emails sent
- Perfect for local testing
- Check logs: `make logs-follow-api | grep "📧"`

**Log Output:**

```
INFO  email/mock.go:33  📧 [MOCK] HTML Email отправлен
  to=user@example.com
  subject=Код верификации
  html_length=1523
```

### Production Mode (Real SMTP/API)

```env
EMAIL_PROVIDER=smtp
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
EMAIL_FROM=noreply@tuserduser.online
EMAIL_FROM_NAME=TuserDuser
```

- Real emails sent via SMTP/API
- Async processing via worker pool
- Error handling and retries
- Full logging for monitoring

## Testing

### 1. Test with Mock Provider (Development)

```bash
# Start server in development
make run

# Register a user
curl -X POST http://localhost:8080/v1/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "phone": "+79991234567",
    "password": "testpass123",
    "verification_type": "email"
  }'

# Check logs
make logs-follow-api | grep "📧"
```

### 2. Test with Real Gmail SMTP

1. Enable 2FA on your Google account
2. Generate App Password: https://myaccount.google.com/apppasswords
3. Update `.env`:
   ```env
   EMAIL_PROVIDER=smtp
   SMTP_HOST=smtp.gmail.com
   SMTP_PORT=587
   SMTP_USERNAME=your-email@gmail.com
   SMTP_PASSWORD=your-16-char-app-password
   ```
4. Restart server and test registration

## Monitoring

### Check Email Sending Status

```bash
# Watch email logs in real-time
make logs-follow-api | grep "email"

# Filter for errors
make logs-access-api | grep "error.*email"

# Count successful email sends today
make logs-access-api | grep "Email успешно отправлен" | wc -l
```

### Metrics to Monitor

- **Email delivery rate**: Success vs failure ratio
- **Queue processing time**: Worker pool performance
- **Redis operations**: Code storage/retrieval
- **SMTP connection errors**: Network/auth issues

## Troubleshooting

### Email Not Sent

**Check 1: Email service initialized?**

```bash
make logs-follow-api | grep "Email service initialized"
```

**Check 2: Worker pool working?**

```bash
make logs-follow-api | grep "Worker pool"
```

**Check 3: SMTP credentials correct?**

```bash
make logs-follow-api | grep "Ошибка при отправке email"
```

### Common Issues

1. **"Email сервис не инициализирован"**
   - Check EMAIL_PROVIDER in .env
   - Verify email package imported in main.go
   - Restart server

2. **"SMTP authentication failed"**
   - Use App Password for Gmail (not regular password)
   - Check username/password are correct
   - Verify SMTP_HOST and SMTP_PORT

3. **"Connection timeout"**
   - Check firewall rules
   - Try different port (587, 465, 25)
   - Verify network connectivity

## Advanced Usage

### Custom Email Templates

Create a custom email template function:

```go
// In internal/email/email.go

func (s *Service) SendWelcomeEmail(ctx context.Context, to, userName string) error {
    subject := "Добро пожаловать в TuserDuser!"
    htmlBody := fmt.Sprintf(`
    <!DOCTYPE html>
    <html>
    <body>
        <h1>Привет, %s!</h1>
        <p>Спасибо за регистрацию...</p>
    </body>
    </html>
    `, userName)

    return s.SendHTMLEmail(ctx, to, subject, htmlBody)
}
```

### Rate Limiting

Consider implementing rate limiting for email sends:

```go
// Example in auth service
const maxEmailsPerHour = 10

func (s *AuthService) canSendEmail(email string) bool {
    key := fmt.Sprintf("email_rate:%s", email)
    count, _ := s.redis.Get(ctx, key)
    // Check count and increment...
}
```

## Production Recommendations

1. **Use Professional Email Service**: SendGrid or Mailgun for better deliverability
2. **Implement Retry Logic**: Retry failed sends with exponential backoff
3. **Add Unsubscribe Links**: For marketing emails (legal requirement)
4. **Monitor Bounce Rates**: Track and handle email bounces
5. **Queue Management**: Consider RabbitMQ/Kafka for high volume
6. **SPF/DKIM/DMARC**: Configure DNS records for email authentication
7. **Template Management**: Store templates in database for easy updates

## Security Notes

- ✅ Verification codes expire in 10 minutes
- ✅ Codes stored securely in Redis
- ✅ Async processing prevents blocking
- ✅ Email addresses validated before sending
- ✅ SMTP credentials stored in environment variables
- ⚠️ Consider rate limiting to prevent abuse
- ⚠️ Add CAPTCHA for public registration endpoints

## Next Steps

1. ✅ Email sending implemented
2. ✅ Verification type selection added
3. ✅ HTML templates created
4. 🔄 Add email bounce handling
5. 🔄 Implement email preferences
6. 🔄 Add password reset via email
7. 🔄 Create email notification system
