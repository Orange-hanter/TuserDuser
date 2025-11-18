package telegram

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

// TokenEncoder handles generation and validation of binding tokens.
type TokenEncoder struct {
	secret []byte
	ttl    time.Duration
}

// NewTokenEncoder builds a new encoder.
func NewTokenEncoder(secret string, ttlSeconds int) *TokenEncoder {
	if ttlSeconds <= 0 {
		ttlSeconds = 600
	}
	return &TokenEncoder{
		secret: []byte(secret),
		ttl:    time.Duration(ttlSeconds) * time.Second,
	}
}

// Mint returns a signed token and the associated nonce metadata.
func (e *TokenEncoder) Mint(userID string) (token string, nonce string, expiresAt time.Time, err error) {
	if userID == "" {
		return "", "", time.Time{}, errors.New("user id required")
	}

	nonceBytes := make([]byte, 16)
	if _, err = rand.Read(nonceBytes); err != nil {
		return "", "", time.Time{}, fmt.Errorf("generate nonce: %w", err)
	}
	nonce = base64.RawURLEncoding.EncodeToString(nonceBytes)
	expiresAt = time.Now().Add(e.ttl)

	payload := fmt.Sprintf("%s|%s|%d", userID, nonce, expiresAt.Unix())
	signature := e.sign(payload)
	composite := fmt.Sprintf("%s|%s", payload, signature)
	token = base64.RawURLEncoding.EncodeToString([]byte(composite))
	return token, nonce, expiresAt, nil
}

// Parse validates a token returning its components.
func (e *TokenEncoder) Parse(token string) (userID, nonce string, expiresAt time.Time, err error) {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("decode token: %w", err)
	}
	parts := strings.Split(string(decoded), "|")
	if len(parts) != 4 {
		return "", "", time.Time{}, errors.New("malformed token")
	}
	payload := strings.Join(parts[:3], "|")
	expected := e.sign(payload)
	if !hmac.Equal([]byte(expected), []byte(parts[3])) {
		return "", "", time.Time{}, errors.New("signature mismatch")
	}
	expUnix, err := parseUnix(parts[2])
	if err != nil {
		return "", "", time.Time{}, err
	}
	expiresAt = time.Unix(expUnix, 0)
	if time.Now().After(expiresAt) {
		return "", "", time.Time{}, errors.New("token expired")
	}
	return parts[0], parts[1], expiresAt, nil
}

func (e *TokenEncoder) sign(payload string) string {
	h := hmac.New(sha256.New, e.secret)
	h.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func parseUnix(value string) (int64, error) {
	var result int64
	_, err := fmt.Sscanf(value, "%d", &result)
	if err != nil {
		return 0, fmt.Errorf("parse unix: %w", err)
	}
	return result, nil
}

// HashNonce returns a deterministic hash for nonce storage.
func HashNonce(nonce string) string {
	sum := sha256.Sum256([]byte(nonce))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
