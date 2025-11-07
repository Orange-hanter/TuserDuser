package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"event-api/internal/config"
	"event-api/internal/models"
	redisClient "event-api/internal/redis"
	"event-api/internal/sms"
	"event-api/internal/worker"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// AuthService управляет аутентификацией и авторизацией.
type AuthService struct {
	cfg        *config.Config
	db         *sql.DB
	redis      *redisClient.Client
	sms        *sms.Service
	workerPool *worker.Pool
	logger     *zap.Logger
}

// VerificationCode хранит информацию о коде верификации.
type VerificationCode struct {
	ExpiresAt time.Time
	Code      string
	Email     string
}

// NewAuthService создает новый сервис аутентификации.
func NewAuthService(cfg *config.Config, db *sql.DB, redis *redisClient.Client, sms *sms.Service, workerPool *worker.Pool, logger *zap.Logger) *AuthService {
	return &AuthService{
		cfg:        cfg,
		db:         db,
		redis:      redis,
		sms:        sms,
		workerPool: workerPool,
		logger:     logger,
	}
}

// Register регистрирует нового пользователя.
func (s *AuthService) Register(req *models.RegisterRequest) (*models.User, string, error) {
	// Проверяем, не существует ли уже пользователь с этим email
	var existingUserID string
	err := s.db.QueryRow("SELECT id FROM users WHERE email = $1", req.Email).Scan(&existingUserID)
	if err == nil {
		return nil, "", fmt.Errorf("пользователь с таким email уже существует")
	} else if err != sql.ErrNoRows {
		return nil, "", fmt.Errorf("ошибка при проверке существования пользователя: %w", err)
	}

	// Хешируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("ошибка при хешировании пароля: %w", err)
	}

	// Создаем пользователя в БД
	userID, err := generateID()
	if err != nil {
		return nil, "", fmt.Errorf("не удалось сгенерировать идентификатор пользователя: %w", err)
	}
	user := &models.User{
		ID:        userID,
		Email:     req.Email,
		Phone:     req.Phone,
		Password:  string(hashedPassword),
		Verified:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	query := `
		INSERT INTO users (id, email, phone, password, verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = s.db.Exec(query, user.ID, user.Email, user.Phone, user.Password, user.Verified, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return nil, "", fmt.Errorf("ошибка при сохранении пользователя: %w", err)
	}

	// Генерируем код верификации
	code, err := generateVerificationCode()
	if err != nil {
		return nil, "", fmt.Errorf("не удалось сгенерировать код верификации: %w", err)
	}
	fmt.Printf("\nGenerated verification code for %s: %s\n", req.Email, code)

	// Сохраняем код верификации в Redis с TTL 10 минут
	ctx := context.Background()
	verifyKey := fmt.Sprintf("verify:%s", req.Email)
	err = s.redis.Set(ctx, verifyKey, code, 10*time.Minute)
	if err != nil {
		return nil, "", fmt.Errorf("ошибка при сохранении кода верификации в Redis: %w", err)
	}

	// Асинхронная отправка кодов верификации (email и SMS)
	email := req.Email
	phone := req.Phone

	// Отправка email кода
	if err := s.workerPool.Submit(func(ctx context.Context) error {
		return s.sendVerificationCode(ctx, email, code)
	}); err != nil {
		s.logger.Error("Не удалось добавить задачу отправки email кода в очередь", zap.Error(err))
	}

	// Отправка SMS кода
	if err := s.workerPool.Submit(func(ctx context.Context) error {
		return s.sendSMSVerificationCode(ctx, phone, code)
	}); err != nil {
		s.logger.Error("Не удалось добавить задачу отправки SMS кода в очередь", zap.Error(err))
	}

	return user, code, nil
}

// VerifyCode проверяет код верификации и подтверждает email.
func (s *AuthService) VerifyCode(email, code string) error {
	ctx := context.Background()
	verifyKey := fmt.Sprintf("verify:%s", email)

	// Получаем код верификации из Redis
	storedCode, err := s.redis.Get(ctx, verifyKey)
	if err != nil {
		return fmt.Errorf("код верификации не найден или истек срок действия")
	}

	// Проверяем правильность кода
	if storedCode != code {
		return fmt.Errorf("неверный код верификации")
	}

	// Обновляем статус верификации пользователя в БД
	_, err = s.db.Exec(
		"UPDATE users SET verified = true, updated_at = $1 WHERE email = $2",
		time.Now(), email,
	)
	if err != nil {
		return fmt.Errorf("ошибка при обновлении статуса верификации: %w", err)
	}

	// Удаляем использованный код из Redis
	if err := s.redis.Del(ctx, verifyKey); err != nil {
		s.logger.Error("Ошибка при удалении кода верификации из Redis", zap.Error(err))
		// Не возвращаем ошибку, так как верификация уже прошла успешно
	}

	return nil
}

// VerifyAndIssueToken проверяет код верификации и возвращает JWT токен.
func (s *AuthService) VerifyAndIssueToken(email, code string) (*models.AuthResponse, error) {
	// Проверяем код верификации
	if err := s.VerifyCode(email, code); err != nil {
		return nil, err
	}

	// Получаем пользователя из БД
	var user models.User
	err := s.db.QueryRow(
		"SELECT id, email, phone, password, verified, created_at, updated_at FROM users WHERE email = $1",
		email,
	).Scan(&user.ID, &user.Email, &user.Phone, &user.Password, &user.Verified, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("пользователь не найден после верификации")
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении пользователя: %w", err)
	}

	// Генерируем JWT токен
	token, expiresAt, err := s.GenerateJWT(&user)
	if err != nil {
		return nil, fmt.Errorf("ошибка при генерации токена: %w", err)
	}

	return &models.AuthResponse{
		AccessToken: token,
		User: &models.User{
			ID:        user.ID,
			Email:     user.Email,
			Phone:     user.Phone,
			Verified:  user.Verified,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
		ExpiresIn: s.cfg.JWTExpiration,
		ExpiresAt: expiresAt,
	}, nil
}

// Login аутентифицирует пользователя и выдает JWT токен.
func (s *AuthService) Login(req *models.LoginRequest) (*models.AuthResponse, error) {
	// Получаем пользователя из БД
	var user models.User
	err := s.db.QueryRow(
		"SELECT id, email, phone, password, verified, created_at, updated_at FROM users WHERE email = $1",
		req.Email,
	).Scan(&user.ID, &user.Email, &user.Phone, &user.Password, &user.Verified, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("пользователь не найден")
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении пользователя: %w", err)
	}

	// Проверяем пароль
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return nil, fmt.Errorf("неверный пароль")
	}

	// Генерируем JWT токен
	token, expiresAt, err := s.GenerateJWT(&user)
	if err != nil {
		return nil, fmt.Errorf("ошибка при генерации токена: %w", err)
	}

	return &models.AuthResponse{
		AccessToken: token,
		User: &models.User{
			ID:        user.ID,
			Email:     user.Email,
			Phone:     user.Phone,
			Verified:  user.Verified,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
		ExpiresIn: s.cfg.JWTExpiration,
		ExpiresAt: expiresAt,
	}, nil
}

// Logout добавляет токен в черный список.
func (s *AuthService) Logout(token string) error {
	ctx := context.Background()
	blacklistKey := fmt.Sprintf("blacklist:%s", token)

	// Добавляем токен в blacklist с TTL равным времени жизни токена
	ttl := time.Duration(s.cfg.JWTExpiration) * time.Second
	err := s.redis.Set(ctx, blacklistKey, "1", ttl)
	if err != nil {
		return fmt.Errorf("ошибка при добавлении токена в blacklist: %w", err)
	}

	return nil
}

// GetUserByID получает пользователя по ID.
func (s *AuthService) GetUserByID(userID string) (*models.User, error) {
	var user models.User
	err := s.db.QueryRow(
		"SELECT id, email, phone, verified, created_at, updated_at FROM users WHERE id = $1",
		userID,
	).Scan(&user.ID, &user.Email, &user.Phone, &user.Verified, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("пользователь не найден")
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении пользователя: %w", err)
	}

	return &models.User{
		ID:        user.ID,
		Email:     user.Email,
		Phone:     user.Phone,
		Verified:  user.Verified,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

// IsTokenBlacklisted проверяет, находится ли токен в черном списке.
func (s *AuthService) IsTokenBlacklisted(token string) bool {
	ctx := context.Background()
	blacklistKey := fmt.Sprintf("blacklist:%s", token)

	exists, err := s.redis.Exists(ctx, blacklistKey)
	if err != nil {
		s.logger.Error("Ошибка при проверке токена в blacklist", zap.Error(err))
		return false
	}

	return exists
}

// GenerateJWT генерирует JWT токен.
func (s *AuthService) GenerateJWT(user *models.User) (string, time.Time, error) {
	expiresAt := time.Now().Add(time.Duration(s.cfg.JWTExpiration) * time.Second)

	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"email":    user.Email,
		"phone":    user.Phone,
		"verified": user.Verified,
		"exp":      expiresAt.Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// ValidateJWT валидирует JWT токен и возвращает claims.
func (s *AuthService) ValidateJWT(tokenString string) (jwt.MapClaims, error) {
	// Проверяем черный список
	if s.IsTokenBlacklisted(tokenString) {
		return nil, fmt.Errorf("токен был отозван")
	}

	token, err := jwt.ParseWithClaims(tokenString, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("неожиданный метод подписи: %v", token.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("ошибка при парсинге токена: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("недействительный токен")
	}

	return claims, nil
}

// sendVerificationCode отправляет код верификации по email
// В будущем здесь будет реальная интеграция с email сервисом.
func (s *AuthService) sendVerificationCode(ctx context.Context, email, code string) error {
	s.logger.Info("Отправка кода верификации (асинхронно)",
		zap.String("email", email),
		zap.String("code", code),
	)

	// Симуляция отправки email
	time.Sleep(100 * time.Millisecond)

	// В production здесь будет:
	// - SMTP отправка
	// - SendGrid/Mailgun API
	// - Очередь сообщений (RabbitMQ/Kafka)

	s.logger.Info("Код верификации отправлен (асинхронно)",
		zap.String("email", email),
	)

	return nil
}

// sendSMSVerificationCode отправляет код верификации по SMS.
func (s *AuthService) sendSMSVerificationCode(ctx context.Context, phone, code string) error {
	s.logger.Info("Отправка кода верификации по SMS (асинхронно)",
		zap.String("phone", phone),
		zap.String("code", code),
	)

	// Отправка SMS через SMS сервис
	if err := s.sms.SendVerificationCode(ctx, phone, code); err != nil {
		s.logger.Error("Ошибка при отправке SMS кода верификации",
			zap.String("phone", phone),
			zap.Error(err),
		)
		return err
	}

	s.logger.Info("SMS код верификации успешно отправлен",
		zap.String("phone", phone),
	)

	return nil
}

// Вспомогательные функции

// generateID генерирует уникальный ID.
func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand.Read: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// generateVerificationCode генерирует 6-значный код верификации.
func generateVerificationCode() (string, error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand.Read: %w", err)
	}
	code := fmt.Sprintf("%06d", int(b[0])<<16|int(b[1])<<8|int(b[2]))
	return code[:6], nil
}
