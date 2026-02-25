package integration

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/loangraph/backend/internal/auth"
	"github.com/loangraph/backend/internal/config"
	loandomain "github.com/loangraph/backend/internal/domain/loan"
	"github.com/loangraph/backend/internal/http/handlers"
	"github.com/loangraph/backend/internal/server"
)

func TestLoanLifecycleRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := newFakeRepo()
	jwtManager := auth.NewJWTManager("issuer", "aud", "super-secret")
	authSvc := auth.NewService(repo, jwtManager, fakeVerifier{}, 15*time.Minute, 24*time.Hour, "")
	authHandler := handlers.NewAuthHandler(authSvc, auth.CookieConfig{}, 15*time.Minute, 24*time.Hour)
	loanHandler := handlers.NewLoanHandler(&fakeLoanService{result: &loandomain.UploadResult{LoanIDs: []string{"l1"}, Processed: 1}})
	r := server.NewRouter(config.Config{Env: "test"}, slog.Default(), server.Dependencies{AuthHandler: authHandler, LoanHandler: loanHandler, JWTManager: jwtManager})

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

	t.Run("list loans", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/loans?lender_id=lender-1", nil)
		req.AddCookie(accessCookie)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200 got %d", resp.Code)
		}
	})

	t.Run("get loan", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/loans/loan-1", nil)
		req.AddCookie(accessCookie)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200 got %d", resp.Code)
		}
	})

	t.Run("repay", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"amount_minor": 1000, "currency": "NGN"})
		req := httptest.NewRequest(http.MethodPost, "/v1/loans/loan-1/repay", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(accessCookie)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200 got %d", resp.Code)
		}
	})

	t.Run("default", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"reason": "missed payments"})
		req := httptest.NewRequest(http.MethodPost, "/v1/loans/loan-1/default", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(accessCookie)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200 got %d", resp.Code)
		}
	})

	t.Run("portfolio analytics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/portfolio/analytics?lender_id=lender-1", nil)
		req.AddCookie(accessCookie)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200 got %d", resp.Code)
		}
	})
}
