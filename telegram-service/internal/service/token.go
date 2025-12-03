// Package service provides token generation and validation.
package service

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

// TokenEncoder handles generation and validation of HMAC-signed binding tokens.
type TokenEncoder struct {
	secret []byte
	ttl    time.Duration
}

// NewTokenEncoder creates a new encoder with the given secret and TTL.
func NewTokenEncoder(secret string, ttlSeconds int) *TokenEncoder {
	if ttlSeconds <= 0 {
		ttlSeconds = 3600 // 1 hour default
	}
	return &TokenEncoder{
		secret: []byte(secret),
		ttl:    time.Duration(ttlSeconds) * time.Second,
	}
}

// Mint generates a new signed token for the given user ID.
// Returns: token, nonce, expiresAt, error
func (e *TokenEncoder) Mint(userID string) (token string, nonce string, expiresAt time.Time, err error) {
	if userID == "" {
		return "", "", time.Time{}, errors.New("user id required")
	}

	// Generate random nonce
	nonceBytes := make([]byte, 16)
	if _, err = rand.Read(nonceBytes); err != nil {
		return "", "", time.Time{}, fmt.Errorf("generate nonce: %w", err)
	}
	nonce = base64.RawURLEncoding.EncodeToString(nonceBytes)
	expiresAt = time.Now().Add(e.ttl)

	// Create payload: userID|nonce|expiresAt
	payload := fmt.Sprintf("%s|%s|%d", userID, nonce, expiresAt.Unix())
	signature := e.sign(payload)

	// Combine payload and signature
	composite := fmt.Sprintf("%s|%s", payload, signature)
	token = base64.RawURLEncoding.EncodeToString([]byte(composite))

	return token, nonce, expiresAt, nil
}

// Parse validates and extracts data from a token.
// Returns: userID, nonce, expiresAt, error
func (e *TokenEncoder) Parse(token string) (userID, nonce string, expiresAt time.Time, err error) {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("decode token: %w", err)
	}

	parts := strings.Split(string(decoded), "|")
	if len(parts) != 4 {
		return "", "", time.Time{}, errors.New("malformed token")
	}

	// Verify signature
	payload := strings.Join(parts[:3], "|")
	expected := e.sign(payload)
	if !hmac.Equal([]byte(expected), []byte(parts[3])) {
		return "", "", time.Time{}, errors.New("signature mismatch")
	}

	// Parse expiration
	var expUnix int64
	if _, err := fmt.Sscanf(parts[2], "%d", &expUnix); err != nil {
		return "", "", time.Time{}, fmt.Errorf("parse expiration: %w", err)
	}
	expiresAt = time.Unix(expUnix, 0)

	// Check if expired
	if time.Now().After(expiresAt) {
		return "", "", time.Time{}, errors.New("token expired")
	}

	return parts[0], parts[1], expiresAt, nil
}

// sign creates an HMAC-SHA256 signature for the payload.
func (e *TokenEncoder) sign(payload string) string {
	h := hmac.New(sha256.New, e.secret)
	h.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// HashNonce returns a deterministic SHA256 hash for nonce storage.
func HashNonce(nonce string) string {
	sum := sha256.Sum256([]byte(nonce))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
