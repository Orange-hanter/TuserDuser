package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"event-api/internal/config"
	"event-api/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthService управляет аутентификацией и авторизацией
type AuthService struct {
	cfg        *config.Config
	users      map[string]*models.User
	verifyCode map[string]string // email -> code
	tokens     map[string]bool    // blacklist для токенов
	mu         sync.RWMutex
}

// VerificationCode хранит информацию о коде верификации
type VerificationCode struct {
	Code      string
	Email     string
	ExpiresAt time.Time
}

// NewAuthService создает новый сервис аутентификации
func NewAuthService(cfg *config.Config) *AuthService {
	return &AuthService{
		cfg:        cfg,
		users:      make(map[string]*models.User),
		verifyCode: make(map[string]string),
		tokens:     make(map[string]bool),
	}
}

// Register регистрирует нового пользователя
func (s *AuthService) Register(req *models.RegisterRequest) (*models.User, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Проверяем, не существует ли уже пользователь с этим email
	for _, user := range s.users {
		if user.Email == req.Email {
			return nil, "", fmt.Errorf("пользователь с таким email уже существует")
		}
	}

	// Хешируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("ошибка при хешировании пароля: %w", err)
	}

	// Создаем пользователя
	userID := generateID()
	user := &models.User{
		ID:        userID,
		Email:     req.Email,
		Phone:     req.Phone,
		Password:  string(hashedPassword),
		Verified:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.users[userID] = user

	// Генерируем код верификации
	code := generateVerificationCode()
	s.verifyCode[req.Email] = code

	return user, code, nil
}

// VerifyCode проверяет код верификации и подтверждает email
func (s *AuthService) VerifyCode(email, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	storedCode, exists := s.verifyCode[email]
	if !exists {
		return fmt.Errorf("код верификации не найден")
	}

	if storedCode != code {
		return fmt.Errorf("неверный код верификации")
	}

	// Находим пользователя по email и отмечаем как верифицированного
	for _, user := range s.users {
		if user.Email == email {
			user.Verified = true
			user.UpdatedAt = time.Now()
			break
		}
	}

	// Удаляем использованный код
	delete(s.verifyCode, email)

	return nil
}

// Login аутентифицирует пользователя и выдает JWT токен
func (s *AuthService) Login(req *models.LoginRequest) (*models.AuthResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Ищем пользователя по email
	var user *models.User
	for _, u := range s.users {
		if u.Email == req.Email {
			user = u
			break
		}
	}

	if user == nil {
		return nil, fmt.Errorf("пользователь не найден")
	}

	// Проверяем пароль
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return nil, fmt.Errorf("неверный пароль")
	}

	// Генерируем JWT токен
	token, expiresAt, err := s.GenerateJWT(user)
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

// Logout добавляет токен в черный список
func (s *AuthService) Logout(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tokens[token] = true
	return nil
}

// GetUserByID получает пользователя по ID
func (s *AuthService) GetUserByID(userID string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[userID]
	if !exists {
		return nil, fmt.Errorf("пользователь не найден")
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

// IsTokenBlacklisted проверяет, находится ли токен в черном списке
func (s *AuthService) IsTokenBlacklisted(token string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.tokens[token]
}

// GenerateJWT генерирует JWT токен
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

// ValidateJWT валидирует JWT токен и возвращает claims
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

// Вспомогательные функции

// generateID генерирует уникальный ID
func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// generateVerificationCode генерирует 6-значный код верификации
func generateVerificationCode() string {
	b := make([]byte, 3)
	rand.Read(b)
	return fmt.Sprintf("%06d", int32(b[0])<<16|int32(b[1])<<8|int32(b[2]))[:6]
}
