package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/tudorAbrudan/tracelog/internal/models"
)

func (s *Store) CreateUser(ctx context.Context, username, password string) (*models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	id, err := generateUserID()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339)

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO users (id, username, password_hash, created_at)
		VALUES (?, ?, ?, ?)`,
		id, username, string(hash), now,
	)
	if err != nil {
		return nil, fmt.Errorf("create user %q: %w", username, err)
	}

	return &models.User{
		ID:        id,
		Username:  username,
		CreatedAt: time.Now(),
	}, nil
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	var u models.User
	var created string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, username, password_hash, created_at
		FROM users WHERE username = ?`, username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &created)
	if err != nil {
		return nil, fmt.Errorf("user %q not found: %w", username, err)
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &u, nil
}

func (s *Store) ListUsers(ctx context.Context) ([]models.User, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, username, created_at FROM users ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		var created string
		if err := rows.Scan(&u.ID, &u.Username, &created); err != nil {
			return nil, err
		}
		u.CreatedAt, _ = time.Parse(time.RFC3339, created)
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *Store) UpdatePassword(ctx context.Context, username, newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	result, err := s.db.ExecContext(ctx, `UPDATE users SET password_hash = ? WHERE username = ?`, string(hash), username)
	if err != nil {
		return fmt.Errorf("update password for %q: %w", username, err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user %q not found", username)
	}
	return nil
}

func (s *Store) UserCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

// Sessions

func (s *Store) CreateSession(ctx context.Context, token, userID string, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (token, user_id, expires_at)
		VALUES (?, ?, ?)`,
		token, userID, expiresAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *Store) GetUserBySession(ctx context.Context, token string) (*models.User, error) {
	var u models.User
	var created, expires string
	err := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.username, u.created_at, s.expires_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token = ?`, token,
	).Scan(&u.ID, &u.Username, &created, &expires)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	expiresAt, _ := time.Parse(time.RFC3339, expires)
	if time.Now().After(expiresAt) {
		_ = s.DeleteSession(ctx, token) //nolint:errcheck // best-effort cleanup of expired session row
		return nil, fmt.Errorf("session expired")
	}

	u.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &u, nil
}

func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE token = ?", token)
	return err
}

func (s *Store) CleanExpiredSessions(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at < ?", time.Now().UTC().Format(time.RFC3339))
	return err
}

func generateUserID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate user id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func GeneratePassword() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate password: %w", err)
	}
	const charset = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789!@#$"
	password := make([]byte, 16)
	for i := range password {
		password[i] = charset[int(b[i%len(b)])%len(charset)]
	}
	return string(password), nil
}
