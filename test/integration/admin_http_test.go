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
	lenderdomain "github.com/loangraph/backend/internal/domain/lender"
	"github.com/loangraph/backend/internal/http/handlers"
	"github.com/loangraph/backend/internal/server"
)

type fakeAdminService struct{}

func (s *fakeAdminService) OnboardLender(_ context.Context, _ string, in lenderdomain.CreateInput) (*lenderdomain.Entity, error) {
	return &lenderdomain.Entity{ID: "lender-1", Name: in.Name, CountryCode: in.CountryCode, WalletAddress: in.WalletAddress, KYCStatus: "pending", Tier: "starter"}, nil
}

func (s *fakeAdminService) UpdateLenderStatus(_ context.Context, _ string, _ string, _ string) error {
	return nil
}

func TestAdminRoutesRequireAdminRoleAndWork(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := newFakeRepo()
	jwtManager := auth.NewJWTManager("issuer", "aud", "super-secret")
	subject := "did:privy:test-user"
	authSvc := auth.NewService(repo, jwtManager, fakeVerifier{}, 15*time.Minute, 24*time.Hour, subject)
	authHandler := handlers.NewAuthHandler(authSvc, auth.CookieConfig{}, 15*time.Minute, 24*time.Hour)
	adminHandler := handlers.NewAdminHandler(&fakeAdminService{})

	r := server.NewRouter(config.Config{Env: "test"}, slog.Default(), server.Dependencies{AuthHandler: authHandler, AdminHandler: adminHandler, JWTManager: jwtManager})

	loginReq := httptest.NewRequest(http.MethodPost, "/v1/auth/privy/login", bytes.NewBufferString(`{"privy_access_token":"token"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	r.ServeHTTP(loginW, loginReq)
	if loginW.Code != http.StatusOK {
		t.Fatalf("expected login 200, got %d", loginW.Code)
	}

	var accessCookie *http.Cookie
	for _, c := range loginW.Result().Cookies() {
		if c.Name == auth.AccessCookieName {
			accessCookie = c
			break
		}
	}
	if accessCookie == nil {
		t.Fatalf("missing access cookie")
	}

	body, _ := json.Marshal(map[string]any{
		"name":           "Admin Added Lender",
		"country_code":   "NG",
		"wallet_address": "0x8888888888888888888888888888888888888888",
	})
	req := httptest.NewRequest(http.MethodPost, "/admin/lenders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(accessCookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	statusBody, _ := json.Marshal(map[string]any{"kyc_status": "approved"})
	statusReq := httptest.NewRequest(http.MethodPatch, "/admin/lenders/lender-1/status", bytes.NewReader(statusBody))
	statusReq.Header.Set("Content-Type", "application/json")
	statusReq.AddCookie(accessCookie)
	statusW := httptest.NewRecorder()
	r.ServeHTTP(statusW, statusReq)
	if statusW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", statusW.Code)
	}
}
