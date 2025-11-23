package main

import (
	"database/sql"
	"fmt"
	"log"

	"event-api/internal/config"

	_ "github.com/lib/pq"
)

func main() {
	cfg := config.Load()
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBSSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT column_name, data_type, character_maximum_length FROM information_schema.columns WHERE table_name = 'users'")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("Users table columns:")
	for rows.Next() {
		var name, dataType string
		var charLen sql.NullInt64
		if err := rows.Scan(&name, &dataType, &charLen); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("- %s: %s (len: %v)\n", name, dataType, charLen)
	}

	rows2, err := db.Query("SELECT tc.constraint_name, tc.constraint_type FROM information_schema.table_constraints tc WHERE tc.table_name = 'users'")
	if err != nil {
		log.Fatal(err)
	}
	defer rows2.Close()

	fmt.Println("\nUsers table constraints:")
	for rows2.Next() {
		var name, cType string
		if err := rows2.Scan(&name, &cType); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("- %s: %s\n", name, cType)
	}
}
