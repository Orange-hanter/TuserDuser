// Example: How to integrate Email Service into the application

// 1. Update cmd/server/main.go - Add email service initialization after SMS service

/\*
// Инициализируем Email сервис
emailConfig := &email.Config{
Provider: cfg.EmailProvider,
APIKey: cfg.EmailAPIKey,
SMTPHost: cfg.SMTPHost,
SMTPPort: cfg.SMTPPort,
SMTPUsername: cfg.SMTPUsername,
SMTPPassword: cfg.SMTPPassword,
From: cfg.EmailFrom,
FromName: cfg.EmailFromName,
}

emailService, err := email.NewService(emailConfig, logger.Log)
if err != nil {
fmt.Println(logger.FormatError(
"Failed to Initialize Email Service",
err,
"Provider: "+cfg.EmailProvider,
))
os.Exit(1)
}

logger.Log.Info("✅ Email service initialized",
zap.String("provider", cfg.EmailProvider),
zap.String("from", cfg.EmailFrom),
)
\*/

// 2. Update internal/service/auth.go - Add email field to AuthService

/*
type AuthService struct {
cfg *config.Config
db *sql.DB
redis *redisClient.Client
sms *sms.Service
email *email.Service // Add this
workerPool *worker.Pool
logger *zap.Logger
}

func NewAuthService(
cfg *config.Config,
db *sql.DB,
redis *redisClient.Client,
sms *sms.Service,
email *email.Service, // Add this parameter
workerPool *worker.Pool,
logger *zap.Logger,
) *AuthService {
return &AuthService{
cfg: cfg,
db: db,
redis: redis,
sms: sms,
email: email, // Add this
workerPool: workerPool,
logger: logger,
}
}
\*/

// 3. Update Register method in internal/service/auth.go - Send verification email

/*
func (s *AuthService) Register(req *models.RegisterRequest) (*models.User, string, error) {
// ... existing code for user creation ...
// После создания пользователя, отправляем email с кодом верификации
if s.email != nil {
s.workerPool.Submit(func() {
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
// Отправляем красивый HTML email
if err := s.email.SendVerificationHTMLEmail(ctx, user.Email, verifyCode); err != nil {
s.logger.Error("Ошибка отправки verification email",
zap.String("email", user.Email),
zap.Error(err),
)
} else {
s.logger.Info("Verification email отправлен",
zap.String("email", user.Email),
)
}
})
}
return user, verifyCode, nil
}
*/

// 4. Update cmd/server/main.go - Pass email service to AuthService

/\*
// Было:
authService := service.NewAuthService(cfg, db.DB, redis, smsService, workerPool, logger.Log)

// Стало:
authService := service.NewAuthService(cfg, db.DB, redis, smsService, emailService, workerPool, logger.Log)
\*/

// 5. Add to .env file

/\*

# Email Configuration

EMAIL_PROVIDER=mock
EMAIL_FROM=noreply@tuserduser.online
EMAIL_FROM_NAME=TuserDuser

# For SMTP (Gmail example)

#EMAIL_PROVIDER=smtp
#SMTP_HOST=smtp.gmail.com
#SMTP_PORT=587
#SMTP_USERNAME=your-email@gmail.com
#SMTP_PASSWORD=your-app-password

# For SendGrid

#EMAIL_PROVIDER=sendgrid
#EMAIL_API_KEY=SG.xxxxxxxxxxxxx

# For Mailgun

#EMAIL_PROVIDER=mailgun
#EMAIL_API_KEY=key-xxxxxxxxxxxxx
\*/

// 6. Usage examples in different scenarios

/\*
// Отправка простого текстового email
err := emailService.SendEmail(ctx, user.Email, "Welcome!", "Welcome to TuserDuser!")

// Отправка HTML email
err := emailService.SendHTMLEmail(ctx, user.Email, "Welcome!", "<h1>Welcome!</h1>")

// Отправка кода верификации (простой текст)
err := emailService.SendVerificationEmail(ctx, user.Email, "123456")

// Отправка кода верификации (красивый HTML)
err := emailService.SendVerificationHTMLEmail(ctx, user.Email, "123456")

// Отправка ссылки для сброса пароля
resetLink := "https://tuserduser.online/reset-password?token=abc123"
err := emailService.SendPasswordResetEmail(ctx, user.Email, resetLink)
\*/

// 7. Testing with mock provider

/\*

# Set in .env for development

EMAIL_PROVIDER=mock

# Check logs to see email content

make logs-follow-api | grep "📧"

# Output example:

# 📧 [MOCK] HTML Email отправлен to=user@example.com subject=Код верификации

\*/
