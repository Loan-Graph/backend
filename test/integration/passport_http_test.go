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
	loandomain "github.com/loangraph/backend/internal/domain/loan"
	passportdomain "github.com/loangraph/backend/internal/domain/passport"
	"github.com/loangraph/backend/internal/http/handlers"
	"github.com/loangraph/backend/internal/server"
)

type fakePassportService struct{}

func (s *fakePassportService) GetPassportByBorrowerHash(_ context.Context, _ string) (*passportdomain.Cache, error) {
	return &passportdomain.Cache{BorrowerID: "b-1", CreditScore: 690}, nil
}

func (s *fakePassportService) GetHistoryByBorrowerHash(_ context.Context, _ string, _ int32, _ int32) ([]loandomain.Entity, error) {
	return []loandomain.Entity{{ID: "loan-1", Status: "active"}}, nil
}

func (s *fakePassportService) GetNFTByBorrowerHash(_ context.Context, _ string) (map[string]any, error) {
	return map[string]any{"token_id": 1, "token_uri": map[string]any{"credit_score": 690}}, nil
}

func (s *fakePassportService) GetPortfolioHealth(_ context.Context, lenderID string) (*loandomain.PortfolioHealth, error) {
	return &loandomain.PortfolioHealth{LenderID: lenderID, UniqueBorrowers: 1}, nil
}

func TestPassportRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := newFakeRepo()
	jwtManager := auth.NewJWTManager("issuer", "aud", "super-secret")
	authSvc := auth.NewService(repo, jwtManager, fakeVerifier{}, 15*time.Minute, 24*time.Hour, "")
	authHandler := handlers.NewAuthHandler(authSvc, auth.CookieConfig{}, 15*time.Minute, 24*time.Hour)
	loanHandler := handlers.NewLoanHandler(&fakeLoanService{result: &loandomain.UploadResult{LoanIDs: []string{"l1"}, Processed: 1}})
	passportHandler := handlers.NewPassportHandler(&fakePassportService{})
	r := server.NewRouter(config.Config{Env: "test"}, slog.Default(), server.Dependencies{AuthHandler: authHandler, LoanHandler: loanHandler, PassportHandler: passportHandler, JWTManager: jwtManager})

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

	cases := []string{
		"/v1/passport/0xdeadbeef",
		"/v1/passport/0xdeadbeef/history",
		"/v1/passport/0xdeadbeef/nft",
		"/v1/portfolio/health?lender_id=lender-1",
	}
	for _, path := range cases {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(accessCookie)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d", path, resp.Code)
		}
	}
}
