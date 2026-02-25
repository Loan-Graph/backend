package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/loangraph/backend/internal/auth"
	"github.com/loangraph/backend/internal/config"
	"github.com/loangraph/backend/internal/db"
	"github.com/loangraph/backend/internal/http/handlers"
	"github.com/loangraph/backend/internal/server"
)

type fakeVerifier struct{}

func (fakeVerifier) VerifyAccessToken(_ context.Context, _ string) (*auth.Identity, error) {
	return &auth.Identity{Subject: "did:privy:test-user", Email: "user@example.com", EmailVerified: true, WalletAddress: "0xabc"}, nil
}

type fakeRepo struct {
	users    map[string]*db.User
	sessions map[string]*db.Session
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{users: map[string]*db.User{}, sessions: map[string]*db.Session{}}
}

func (r *fakeRepo) UpsertUser(_ context.Context, privySubject, email string, emailVerified bool, walletAddress string) (*db.User, error) {
	if u, ok := r.users[privySubject]; ok {
		u.Email = email
		u.EmailVerified = emailVerified
		u.WalletAddress = walletAddress
		return u, nil
	}
	u := &db.User{ID: "u-1", PrivySubject: privySubject, Email: email, EmailVerified: emailVerified, WalletAddress: walletAddress}
	r.users[privySubject] = u
	return u, nil
}

func (r *fakeRepo) GetUserByID(_ context.Context, userID string) (*db.User, error) {
	for _, u := range r.users {
		if u.ID == userID {
			return u, nil
		}
	}
	return nil, context.Canceled
}

func (r *fakeRepo) CreateSession(_ context.Context, userID, refreshHash, userAgent, ipAddress string, expiresAt time.Time) (*db.Session, error) {
	s := &db.Session{ID: "s-" + time.Now().UTC().Format("150405.000000"), UserID: userID, RefreshTokenHash: refreshHash, UserAgent: userAgent, IPAddress: ipAddress, ExpiresAt: expiresAt}
	r.sessions[s.ID] = s
	return s, nil
}

func (r *fakeRepo) GetSessionByID(_ context.Context, sessionID string) (*db.Session, error) {
	if s, ok := r.sessions[sessionID]; ok {
		return s, nil
	}
	return nil, context.Canceled
}

func (r *fakeRepo) RevokeSession(_ context.Context, sessionID string) error {
	if s, ok := r.sessions[sessionID]; ok {
		now := time.Now().UTC()
		s.RevokedAt = &now
	}
	return nil
}

func (r *fakeRepo) UpdateSessionRefreshHash(_ context.Context, sessionID, refreshHash string) error {
	if s, ok := r.sessions[sessionID]; ok {
		s.RefreshTokenHash = refreshHash
	}
	return nil
}

func TestAuthLoginSetsCookies(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := newFakeRepo()
	jwtManager := auth.NewJWTManager("issuer", "aud", "super-secret")
	svc := auth.NewService(repo, jwtManager, fakeVerifier{}, 15*time.Minute, 24*time.Hour)
	h := handlers.NewAuthHandler(svc, auth.CookieConfig{}, 15*time.Minute, 24*time.Hour)

	r := server.NewRouter(config.Config{Env: "test"}, slog.Default(), server.Dependencies{AuthHandler: h, JWTManager: jwtManager})

	body, _ := json.Marshal(map[string]string{"privy_access_token": "token"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/privy/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	cookies := w.Result().Cookies()
	if len(cookies) < 2 {
		t.Fatalf("expected auth cookies to be set")
	}
}
