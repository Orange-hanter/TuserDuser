package telegram

import "errors"

var (
	// ErrBindingNotFound indicates that the user has no Telegram chat bound.
	ErrBindingNotFound = errors.New("telegram binding not found")

	// ErrBindingBlocked indicates attempts to send to a blocked binding.
	ErrBindingBlocked = errors.New("telegram binding blocked")

	// ErrTransportUnavailable is returned when Telegram sink is disabled or missing data.
	ErrTransportUnavailable = errors.New("telegram transport unavailable")

	// ErrInvalidToken indicates a malformed, expired, or reused binding token.
	ErrInvalidToken = errors.New("invalid or expired binding token")
)
