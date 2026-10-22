package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/KeKe45/resourcelocate/internal/auth"
	"github.com/KeKe45/resourcelocate/internal/domain"
)

// SessionLength is how long a login lasts.
const SessionLength = 30 * 24 * time.Hour

const userCols = `id, name, email, role, area, created_at`

func scanUser(r scanner, extra ...any) (domain.User, error) {
	var u domain.User
	var created string
	err := r.Scan(append([]any{&u.ID, &u.Name, &u.Email, &u.Role, &u.Area, &created}, extra...)...)
	if errors.Is(err, sql.ErrNoRows) {
		return u, ErrNotFound
	}
	if err != nil {
		return u, err
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return u, nil
}

// CreateUser stores a new account. passwordHash comes from auth.HashPassword.
// An email that is already registered is a conflict.
func (s *Store) CreateUser(ctx context.Context, u *domain.User, passwordHash string) error {
	if u.ID == "" {
		u.ID = NewID()
	}
	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO users (`+userCols+`, password_hash) VALUES (?,?,?,?,?,?,?)`,
		u.ID, u.Name, u.Email, u.Role, u.Area, ts(u.CreatedAt), passwordHash)
	if err != nil && strings.Contains(err.Error(), "UNIQUE") {
		return fmt.Errorf("%w: an account with that email already exists, log in instead", domain.ErrConflict)
	}
	return err
}

// UserByEmail returns the account and its password hash, for logging in.
func (s *Store) UserByEmail(ctx context.Context, email string) (domain.User, string, error) {
	var hash string
	u, err := scanUser(s.db.QueryRowContext(ctx, `SELECT `+userCols+`, password_hash FROM users WHERE email = ?`,
		domain.NormalizeEmail(email)), &hash)
	return u, hash, err
}

// CreateSession starts a login for userID and returns the token for the cookie.
func (s *Store) CreateSession(ctx context.Context, userID string) (string, time.Time, error) {
	token, hash, err := auth.NewToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expires := time.Now().UTC().Add(SessionLength)
	// Clearing expired sessions here keeps the table small without a job.
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < ?`, ts(time.Now().UTC())); err != nil {
		return "", time.Time{}, err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO sessions (token_hash, user_id, expires_at) VALUES (?,?,?)`,
		hash, userID, ts(expires))
	return token, expires, err
}

// UserForSession returns the user a session token belongs to, or
// ErrNotFound if the token is unknown or expired.
func (s *Store) UserForSession(ctx context.Context, token string) (domain.User, error) {
	return scanUser(s.db.QueryRowContext(ctx, `SELECT `+prefixed("u.", userCols)+` FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = ? AND s.expires_at > ?`, auth.HashToken(token), ts(time.Now().UTC())))
}

// DeleteSession logs a token out.
func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, auth.HashToken(token))
	return err
}
