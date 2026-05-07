package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // never serialised to API responses
	CreatedAt    time.Time `json:"created_at"`
}

func CreateUser(ctx context.Context, pool *pgxpool.Pool, email, passwordHash string) (*User, error) {
	var u User
	err := pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, password_hash, created_at
	`, email, passwordHash).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("CreateUser: %w", err)
	}
	return &u, nil
}

// GetUserByEmail returns nil, nil when the email is not found.
func GetUserByEmail(ctx context.Context, pool *pgxpool.Pool, email string) (*User, error) {
	var u User
	err := pool.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at
		FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetUserByEmail: %w", err)
	}
	return &u, nil
}
