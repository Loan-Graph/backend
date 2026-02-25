package integration

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/loangraph/backend/internal/auth"
	"github.com/loangraph/backend/internal/config"
	investordomain "github.com/loangraph/backend/internal/domain/investor"
	lenderdomain "github.com/loangraph/backend/internal/domain/lender"
	loandomain "github.com/loangraph/backend/internal/domain/loan"
	pooldomain "github.com/loangraph/backend/internal/domain/pool"
	"github.com/loangraph/backend/internal/http/handlers"
	"github.com/loangraph/backend/internal/server"
)

type fakeInvestorService struct{}

func (s *fakeInvestorService) ListPools(_ context.Context, _ investordomain.PoolFilter) ([]pooldomain.Entity, error) {
	return []pooldomain.Entity{{ID: "pool-1", Name: "Starter Pool"}}, nil
}

func (s *fakeInvestorService) GetPool(_ context.Context, poolID string) (*pooldomain.Entity, error) {
	return &pooldomain.Entity{ID: poolID, Name: "Starter Pool"}, nil
}

func (s *fakeInvestorService) GetPoolPerformance(_ context.Context, _ string, _ int32) ([]loandomain.PerformancePoint, error) {
	return []loandomain.PerformancePoint{{Date: "2026-02-25", RepaymentCount: 2, RepaidAmountMinor: 5000}}, nil
}

func (s *fakeInvestorService) GetLenderProfile(_ context.Context, lenderID string) (*investordomain.LenderProfile, error) {
	return &investordomain.LenderProfile{Lender: &lenderdomain.Entity{ID: lenderID, Name: "Lender A"}}, nil
}

func TestInvestorRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := newFakeRepo()
	jwtManager := auth.NewJWTManager("issuer", "aud", "super-secret")
	authSvc := auth.NewService(repo, jwtManager, fakeVerifier{}, 15*time.Minute, 24*time.Hour, "")
	authHandler := handlers.NewAuthHandler(authSvc, auth.CookieConfig{}, 15*time.Minute, 24*time.Hour)
	loanHandler := handlers.NewLoanHandler(&fakeLoanService{result: &loandomain.UploadResult{LoanIDs: []string{"l1"}, Processed: 1}})
	passportHandler := handlers.NewPassportHandler(&fakePassportService{})
	investorHandler := handlers.NewInvestorHandler(&fakeInvestorService{})

	r := server.NewRouter(config.Config{Env: "test"}, slog.Default(), server.Dependencies{
		AuthHandler:     authHandler,
		LoanHandler:     loanHandler,
		PassportHandler: passportHandler,
		InvestorHandler: investorHandler,
		JWTManager:      jwtManager,
	})

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

	paths := []string{
		"/v1/pools",
		"/v1/pools/pool-1",
		"/v1/pools/pool-1/performance?days=30",
		"/v1/lenders/lender-1/profile",
	}
	for _, path := range paths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(accessCookie)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d", path, w.Code)
		}
	}
}
