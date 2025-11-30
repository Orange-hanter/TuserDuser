package repository

import (
	"database/sql"
	"errors"
	_ "github.com/lib/pq"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(dataSourceName string) (*PostgresRepository, error) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresRepository{db: db}, nil
}

func (r *PostgresRepository) CreateUser(userID int64, username string) error {
	query := "INSERT INTO users (user_id, username) VALUES ($1, $2) ON CONFLICT (user_id) DO NOTHING"
	_, err := r.db.Exec(query, userID, username)
	return err
}

func (r *PostgresRepository) GetUser(userID int64) (string, error) {
	var username string
	query := "SELECT username FROM users WHERE user_id = $1"
	err := r.db.QueryRow(query, userID).Scan(&username)
	if err == sql.ErrNoRows {
		return "", errors.New("user not found")
	}
	return username, err
}

func (r *PostgresRepository) CreateMessage(userID int64, message string) error {
	query := "INSERT INTO messages (user_id, message) VALUES ($1, $2)"
	_, err := r.db.Exec(query, userID, message)
	return err
}

func (r *PostgresRepository) GetMessages(userID int64) ([]string, error) {
	rows, err := r.db.Query("SELECT message FROM messages WHERE user_id = $1", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []string
	for rows.Next() {
		var message string
		if err := rows.Scan(&message); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, nil
}
