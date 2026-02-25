package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/loangraph/backend/internal/db"
)

type Repository interface {
	UpsertUser(ctx context.Context, privySubject, email string, emailVerified bool, walletAddress string) (*db.User, error)
	GetUserByID(ctx context.Context, userID string) (*db.User, error)
	CreateSession(ctx context.Context, userID, refreshHash, userAgent, ipAddress string, expiresAt time.Time) (*db.Session, error)
	GetSessionByID(ctx context.Context, sessionID string) (*db.Session, error)
	RevokeSession(ctx context.Context, sessionID string) error
	UpdateSessionRefreshHash(ctx context.Context, sessionID, refreshHash string) error
}

type Service struct {
	repo       Repository
	jwt        *JWTManager
	verifier   PrivyVerifier
	accessTTL  time.Duration
	refreshTTL time.Duration
}

type AuthTokens struct {
	AccessToken  string
	RefreshToken string
	SessionID    string
	User         *db.User
}

func NewService(repo Repository, jwt *JWTManager, verifier PrivyVerifier, accessTTL, refreshTTL time.Duration) *Service {
	return &Service{repo: repo, jwt: jwt, verifier: verifier, accessTTL: accessTTL, refreshTTL: refreshTTL}
}

func (s *Service) LoginWithPrivy(ctx context.Context, privyAccessToken, userAgent, ipAddress string) (*AuthTokens, error) {
	identity, err := s.verifier.VerifyAccessToken(ctx, privyAccessToken)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.UpsertUser(ctx, identity.Subject, identity.Email, identity.EmailVerified, identity.WalletAddress)
	if err != nil {
		return nil, err
	}

	bundle, err := s.createSessionAndTokens(ctx, user.ID, userAgent, ipAddress)
	if err != nil {
		return nil, err
	}

	return &AuthTokens{AccessToken: bundle.AccessToken, RefreshToken: bundle.RefreshToken, SessionID: bundle.SessionID, User: user}, nil
}

type sessionBundle struct {
	AccessToken  string
	RefreshToken string
	SessionID    string
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ipAddress string) (*AuthTokens, error) {
	claims, err := s.jwt.Parse(refreshToken)
	if err != nil {
		return nil, err
	}
	if claims.Type != "refresh" {
		return nil, errors.New("invalid token type")
	}

	session, err := s.repo.GetSessionByID(ctx, claims.SessionID)
	if err != nil {
		return nil, err
	}
	if session.RevokedAt != nil {
		return nil, errors.New("session revoked")
	}
	if time.Now().UTC().After(session.ExpiresAt) {
		return nil, errors.New("session expired")
	}
	if session.RefreshTokenHash != hashToken(refreshToken) {
		return nil, errors.New("refresh token mismatch")
	}

	if err := s.repo.RevokeSession(ctx, session.ID); err != nil {
		return nil, err
	}

	bundle, err := s.createSessionAndTokens(ctx, session.UserID, userAgent, ipAddress)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil {
		return nil, err
	}

	return &AuthTokens{AccessToken: bundle.AccessToken, RefreshToken: bundle.RefreshToken, SessionID: bundle.SessionID, User: user}, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	claims, err := s.jwt.Parse(refreshToken)
	if err != nil {
		return nil
	}
	if claims.Type != "refresh" || claims.SessionID == "" {
		return nil
	}
	return s.repo.RevokeSession(ctx, claims.SessionID)
}

func (s *Service) Me(ctx context.Context, userID string) (*db.User, error) {
	return s.repo.GetUserByID(ctx, userID)
}

func (s *Service) createSessionAndTokens(ctx context.Context, userID, userAgent, ipAddress string) (*sessionBundle, error) {
	expiresAt := time.Now().UTC().Add(s.refreshTTL)
	sessionSeed := uuid.NewString()
	session, err := s.repo.CreateSession(ctx, userID, hashToken(sessionSeed), userAgent, ipAddress, expiresAt)
	if err != nil {
		return nil, err
	}

	accessToken, err := s.jwt.Mint(userID, session.ID, "access", s.accessTTL)
	if err != nil {
		return nil, err
	}
	refreshToken, err := s.jwt.Mint(userID, session.ID, "refresh", s.refreshTTL)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateSessionRefreshHash(ctx, session.ID, hashToken(refreshToken)); err != nil {
		return nil, err
	}

	return &sessionBundle{AccessToken: accessToken, RefreshToken: refreshToken, SessionID: session.ID}, nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func ClientIP(r *http.Request) string {
	xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
