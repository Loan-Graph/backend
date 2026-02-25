package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID            string
	PrivySubject  string
	Email         string
	EmailVerified bool
	WalletAddress string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Session struct {
	ID               string
	UserID           string
	RefreshTokenHash string
	UserAgent        string
	IPAddress        string
	ExpiresAt        time.Time
	RevokedAt        *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type AuthRepository struct {
	pool *pgxpool.Pool
}

func NewAuthRepository(pool *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{pool: pool}
}

func (r *AuthRepository) UpsertUser(ctx context.Context, privySubject, email string, emailVerified bool, walletAddress string) (*User, error) {
	q := `
INSERT INTO users (privy_subject, email, email_verified, wallet_address)
VALUES ($1, $2, $3, $4)
ON CONFLICT (privy_subject)
DO UPDATE SET
  email = EXCLUDED.email,
  email_verified = EXCLUDED.email_verified,
  wallet_address = EXCLUDED.wallet_address,
  updated_at = NOW()
RETURNING id, privy_subject, email, email_verified, wallet_address, created_at, updated_at
`
	u := &User{}
	err := r.pool.QueryRow(ctx, q, privySubject, email, emailVerified, walletAddress).
		Scan(&u.ID, &u.PrivySubject, &u.Email, &u.EmailVerified, &u.WalletAddress, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *AuthRepository) GetUserByID(ctx context.Context, userID string) (*User, error) {
	q := `SELECT id, privy_subject, email, email_verified, wallet_address, created_at, updated_at FROM users WHERE id = $1`
	u := &User{}
	err := r.pool.QueryRow(ctx, q, userID).
		Scan(&u.ID, &u.PrivySubject, &u.Email, &u.EmailVerified, &u.WalletAddress, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *AuthRepository) CreateSession(ctx context.Context, userID, refreshHash, userAgent, ipAddress string, expiresAt time.Time) (*Session, error) {
	q := `
INSERT INTO auth_sessions (user_id, refresh_token_hash, user_agent, ip_address, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, refresh_token_hash, user_agent, ip_address, expires_at, revoked_at, created_at, updated_at
`
	s := &Session{}
	err := r.pool.QueryRow(ctx, q, userID, refreshHash, userAgent, ipAddress, expiresAt).
		Scan(&s.ID, &s.UserID, &s.RefreshTokenHash, &s.UserAgent, &s.IPAddress, &s.ExpiresAt, &s.RevokedAt, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *AuthRepository) GetSessionByID(ctx context.Context, sessionID string) (*Session, error) {
	q := `
SELECT id, user_id, refresh_token_hash, user_agent, ip_address, expires_at, revoked_at, created_at, updated_at
FROM auth_sessions
WHERE id = $1
`
	s := &Session{}
	err := r.pool.QueryRow(ctx, q, sessionID).
		Scan(&s.ID, &s.UserID, &s.RefreshTokenHash, &s.UserAgent, &s.IPAddress, &s.ExpiresAt, &s.RevokedAt, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *AuthRepository) RevokeSession(ctx context.Context, sessionID string) error {
	q := `UPDATE auth_sessions SET revoked_at = NOW(), updated_at = NOW() WHERE id = $1 AND revoked_at IS NULL`
	_, err := r.pool.Exec(ctx, q, sessionID)
	return err
}

func (r *AuthRepository) UpdateSessionRefreshHash(ctx context.Context, sessionID, refreshHash string) error {
	q := `UPDATE auth_sessions SET refresh_token_hash = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, q, sessionID, refreshHash)
	return err
}
