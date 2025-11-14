// Package service предоставляет сервисы для аутентификации и авторизации.
package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"event-api/internal/config"
	"event-api/internal/email"
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
	email      *email.Service
	workerPool *worker.Pool
	logger     *zap.Logger
}

// VerificationCode хранит информацию о коде верификации.
type VerificationCode struct {
	ExpiresAt time.Time
	Code      string
	Email     string
}

const (
	pendingUserTTL   = 10 * time.Minute
	errGetUserFormat = "ошибка при получении пользователя: %w"
	errEmailRequired = "email не указан"
)

var errPendingUserNotFound = errors.New("pending user not found")

type pendingUser struct {
	ID                string    `json:"id"`
	Email             string    `json:"email"`
	Phone             string    `json:"phone"`
	Password          string    `json:"password"`
	VerificationType  string    `json:"verification_type"`
	VerificationCode  string    `json:"verification_code"`
	CreatedAt         time.Time `json:"created_at"`
	OriginalUpdatedAt time.Time `json:"updated_at"`
}

// NewAuthService создает новый сервис аутентификации.
func NewAuthService(cfg *config.Config, db *sql.DB, redis *redisClient.Client, sms *sms.Service, email *email.Service, workerPool *worker.Pool, logger *zap.Logger) *AuthService {
	return &AuthService{
		cfg:        cfg,
		db:         db,
		redis:      redis,
		sms:        sms,
		email:      email,
		workerPool: workerPool,
		logger:     logger,
	}
}

// Register регистрирует нового пользователя.
func (s *AuthService) Register(req *models.RegisterRequest) (*models.User, string, error) {
	if err := s.ensureRegisterDependencies(); err != nil {
		return nil, "", err
	}

	email := normalizeEmail(req.Email)
	if email == "" {
		return nil, "", errors.New(errEmailRequired)
	}

	ctx := context.Background()
	if err := s.ensureUserDoesNotExist(ctx, email); err != nil {
		return nil, "", err
	}
	if err := s.ensureNoPendingRegistration(ctx, email); err != nil {
		return nil, "", err
	}

	pending, code, err := s.buildPendingUser(req, email)
	if err != nil {
		return nil, "", err
	}

	if err := s.savePendingUser(ctx, pending); err != nil {
		return nil, "", err
	}

	s.dispatchVerificationTasks(req.Email, req.Phone, pending.VerificationType, code)
	return pending.toModel(false), code, nil
}

func (s *AuthService) ensureRegisterDependencies() error {
	if s.db == nil {
		return fmt.Errorf("database клиент не инициализирован")
	}
	if s.redis == nil {
		return fmt.Errorf("redis клиент не инициализирован")
	}
	return nil
}

func (s *AuthService) ensureUserDoesNotExist(ctx context.Context, email string) error {
	var existingUserID string
	err := s.db.QueryRowContext(ctx, "SELECT id FROM users WHERE email = $1", email).Scan(&existingUserID)
	if err == nil {
		return fmt.Errorf("пользователь с таким email уже существует")
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("ошибка при проверке существования пользователя: %w", err)
	}
	return nil
}

func (s *AuthService) ensureNoPendingRegistration(ctx context.Context, email string) error {
	pendingKey := pendingUserKey(email)
	pendingExists, err := s.redis.Exists(ctx, pendingKey)
	if err != nil {
		return fmt.Errorf("ошибка при проверке незавершенной регистрации: %w", err)
	}
	if pendingExists {
		return fmt.Errorf("регистрация уже ожидает подтверждения")
	}
	return nil
}

func (s *AuthService) buildPendingUser(req *models.RegisterRequest, email string) (*pendingUser, string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("ошибка при хешировании пароля: %w", err)
	}
	userID, err := generateID()
	if err != nil {
		return nil, "", fmt.Errorf("не удалось сгенерировать идентификатор пользователя: %w", err)
	}
	verificationType := req.VerificationType
	if verificationType == "" {
		verificationType = "both"
	}
	code, err := generateVerificationCode()
	if err != nil {
		return nil, "", fmt.Errorf("не удалось сгенерировать код верификации: %w", err)
	}
	pending := &pendingUser{
		ID:                userID,
		Email:             email,
		Phone:             strings.TrimSpace(req.Phone),
		Password:          string(hashedPassword),
		VerificationType:  verificationType,
		VerificationCode:  code,
		CreatedAt:         time.Now(),
		OriginalUpdatedAt: time.Now(),
	}
	return pending, code, nil
}

// dispatchVerificationTasks schedules async verification deliveries so registration flow stays fast.
func (s *AuthService) dispatchVerificationTasks(email, phone, verificationType, code string) {
	if verificationType == "email" || verificationType == "both" {
		if err := s.workerPool.Submit(func(ctx context.Context) error {
			return s.sendEmailVerificationCode(ctx, email, code)
		}); err != nil {
			s.logger.Error("Не удалось добавить задачу отправки email кода в очередь", zap.Error(err))
		}
	}
	if verificationType == "sms" || verificationType == "both" {
		if err := s.workerPool.Submit(func(ctx context.Context) error {
			return s.sendSMSVerificationCode(ctx, phone, code)
		}); err != nil {
			s.logger.Error("Не удалось добавить задачу отправки SMS кода в очередь", zap.Error(err))
		}
	}
}

// VerifyCode проверяет код верификации и подтверждает email.
func (s *AuthService) VerifyCode(email, code string) error {
	if s.redis == nil {
		return fmt.Errorf("redis клиент не инициализирован")
	}

	ctx := context.Background()
	normalizedEmail := normalizeEmail(email)

	if normalizedEmail == "" {
		return errors.New(errEmailRequired)
	}

	// Пробуем обработать незавершенную регистрацию
	if pending, err := s.loadPendingUser(ctx, normalizedEmail); err == nil {
		if pending.VerificationCode != code {
			return fmt.Errorf("неверный код верификации")
		}
		pending.OriginalUpdatedAt = time.Now()
		if err := s.persistPendingUser(ctx, pending); err != nil {
			return err
		}
		s.deletePendingUser(ctx, normalizedEmail)
		return nil
	} else if err != errPendingUserNotFound {
		return err
	}

	// Fallback: поддержка старого сценария, когда пользователь уже существовал в БД
	verifyKey := fmt.Sprintf("verify:%s", normalizedEmail)
	storedCode, err := s.redis.Get(ctx, verifyKey)
	if err != nil {
		return fmt.Errorf("код верификации не найден или истек срок действия")
	}
	if storedCode != code {
		return fmt.Errorf("неверный код верификации")
	}
	_, err = s.db.ExecContext(
		ctx,
		"UPDATE users SET verified = true, updated_at = $1 WHERE email = $2",
		time.Now(), normalizedEmail,
	)
	if err != nil {
		return fmt.Errorf("ошибка при обновлении статуса верификации: %w", err)
	}
	if err := s.redis.Del(ctx, verifyKey); err != nil {
		s.logger.Error("Ошибка при удалении кода верификации из Redis", zap.Error(err))
	}
	return nil
}

// VerifyAndIssueToken проверяет код верификации и возвращает JWT токен.
func (s *AuthService) VerifyAndIssueToken(email, code string) (*models.AuthResponse, error) {
	normalizedEmail := normalizeEmail(email)
	if normalizedEmail == "" {
		return nil, errors.New(errEmailRequired)
	}

	// Проверяем код верификации
	if err := s.VerifyCode(normalizedEmail, code); err != nil {
		return nil, err
	}

	// Получаем пользователя из БД
	var user models.User
	ctx := context.Background()
	err := s.db.QueryRowContext(
		ctx,
		"SELECT id, email, phone, password, verified, created_at, updated_at FROM users WHERE email = $1",
		normalizedEmail,
	).Scan(&user.ID, &user.Email, &user.Phone, &user.Password, &user.Verified, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("пользователь не найден после верификации")
	}
	if err != nil {
		return nil, fmt.Errorf(errGetUserFormat, err)
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
	email := normalizeEmail(req.Email)
	if email == "" {
		return nil, errors.New(errEmailRequired)
	}

	// Получаем пользователя из БД
	var user models.User
	ctx := context.Background()
	err := s.db.QueryRowContext(
		ctx,
		"SELECT id, email, phone, password, verified, created_at, updated_at FROM users WHERE email = $1",
		email,
	).Scan(&user.ID, &user.Email, &user.Phone, &user.Password, &user.Verified, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("пользователь не найден")
	}
	if err != nil {
		return nil, fmt.Errorf(errGetUserFormat, err)
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
	ctx := context.Background()
	err := s.db.QueryRowContext(
		ctx,
		"SELECT id, email, phone, verified, created_at, updated_at FROM users WHERE id = $1",
		userID,
	).Scan(&user.ID, &user.Email, &user.Phone, &user.Verified, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("пользователь не найден")
	}
	if err != nil {
		return nil, fmt.Errorf(errGetUserFormat, err)
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

// sendEmailVerificationCode отправляет код верификации по email.
func (s *AuthService) sendEmailVerificationCode(ctx context.Context, emailAddr, code string) error {
	s.logger.Info("Отправка кода верификации по email (асинхронно)",
		zap.String("email", emailAddr),
		zap.String("code", code),
	)

	// Проверяем, инициализирован ли email сервис
	if s.email == nil {
		s.logger.Warn("Email сервис не инициализирован, код не отправлен",
			zap.String("email", emailAddr),
		)
		return fmt.Errorf("email сервис не инициализирован")
	}

	// Отправка email через email сервис
	if err := s.email.SendVerificationHTMLEmail(ctx, emailAddr, code); err != nil {
		s.logger.Error("Ошибка при отправке email кода верификации",
			zap.String("email", emailAddr),
			zap.Error(err),
		)
		return err
	}

	s.logger.Info("Email код верификации успешно отправлен",
		zap.String("email", emailAddr),
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

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func pendingUserKey(email string) string {
	return fmt.Sprintf("pending:user:%s", normalizeEmail(email))
}

func (p *pendingUser) toModel(verified bool) *models.User {
	if p == nil {
		return nil
	}
	return &models.User{
		ID:        p.ID,
		Email:     p.Email,
		Phone:     p.Phone,
		Password:  p.Password,
		Verified:  verified,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.OriginalUpdatedAt,
	}
}

func (s *AuthService) savePendingUser(ctx context.Context, pending *pendingUser) error {
	data, err := json.Marshal(pending)
	if err != nil {
		return fmt.Errorf("не удалось сериализовать данные пользователя для Redis: %w", err)
	}
	if err := s.redis.Set(ctx, pendingUserKey(pending.Email), data, pendingUserTTL); err != nil {
		return fmt.Errorf("ошибка при сохранении данных регистрации в Redis: %w", err)
	}
	return nil
}

func (s *AuthService) loadPendingUser(ctx context.Context, email string) (*pendingUser, error) {
	key := pendingUserKey(email)
	exists, err := s.redis.Exists(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("ошибка при проверке незавершенной регистрации: %w", err)
	}
	if !exists {
		return nil, errPendingUserNotFound
	}
	raw, err := s.redis.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении незавершенной регистрации: %w", err)
	}
	var pending pendingUser
	if err := json.Unmarshal([]byte(raw), &pending); err != nil {
		return nil, fmt.Errorf("не удалось распарсить данные незавершенной регистрации: %w", err)
	}
	return &pending, nil
}

func (s *AuthService) deletePendingUser(ctx context.Context, email string) {
	if err := s.redis.Del(ctx, pendingUserKey(email)); err != nil {
		s.logger.Warn("Не удалось удалить незавершенную регистрацию", zap.String("email", email), zap.Error(err))
	}
}

func (s *AuthService) persistPendingUser(ctx context.Context, pending *pendingUser) error {
	now := time.Now()
	query := `
		INSERT INTO users (id, email, phone, password, verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := s.db.ExecContext(ctx, query, pending.ID, pending.Email, pending.Phone, pending.Password, true, pending.CreatedAt, now)
	if err != nil {
		return fmt.Errorf("ошибка при сохранении пользователя после верификации: %w", err)
	}
	return nil
}

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
