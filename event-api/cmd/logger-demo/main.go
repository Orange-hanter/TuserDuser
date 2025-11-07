// cmd/logger-demo/main.go
package main

import (
	"fmt"
	"os"
	"strings"

	"event-api/internal/logger"
)

func main() {
	logger.Init()
	defer func() {
		if err := logger.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to sync logger: %v\n", err)
		}
	}()

	separator := strings.Repeat("=", 60)

	fmt.Println("\n" + separator)
	fmt.Println("📊 Event API - Logging Examples")
	fmt.Println(separator)

	// ========== ОШИБКИ ==========
	fmt.Println("\n1️⃣  ERROR EXAMPLES:")
	fmt.Println(logger.FormatError(
		"Database Connection Failed",
		fmt.Errorf("connection refused"),
		"Host: localhost",
		"Port: 5432",
		"Database: event_api",
	))

	fmt.Println(logger.FormatError(
		"User Registration Failed",
		fmt.Errorf("invalid email format"),
		"Email: invalid-email@",
		"Reason: Email validation failed",
	))

	// ========== УСПЕХ ==========
	fmt.Println("\n2️⃣  SUCCESS EXAMPLES:")
	fmt.Println(logger.FormatSuccess(
		"Server Started Successfully",
		"Port: 8080",
		"Environment: development",
		"CORS Origins: 2",
	))

	fmt.Println(logger.FormatSuccess(
		"User Registered",
		"Email: user@example.com",
		"ID: 123e4567-e89b-12d3-a456-426614174000",
		"Verification Code Sent",
	))

	// ========== ПРЕДУПРЕЖДЕНИЯ ==========
	fmt.Println("\n3️⃣  WARNING EXAMPLES:")
	fmt.Println(logger.FormatWarning(
		".env file not found",
		"Using system environment variables",
		"Some values may use defaults",
	))

	// ========== ИНФОРМАЦИЯ ==========
	fmt.Println("\n4️⃣  INFO EXAMPLES:")
	fmt.Println(logger.FormatInfo(
		"Application Configuration",
		"Port: 8080",
		"Environment: development",
		"JWT Expiration: 3600 seconds",
	))

	fmt.Println("\n" + separator)
	fmt.Println("✨ Logging examples completed!")
	fmt.Println(separator)
}
