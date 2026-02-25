package integration

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"mime/multipart"
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

type fakeLoanService struct {
	result *loandomain.UploadResult
	err    error
}

func (s *fakeLoanService) ProcessCSVUpload(_ context.Context, _ string, _ io.Reader) (*loandomain.UploadResult, error) {
	return s.result, s.err
}

func TestLoanUploadRouteRequiresAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtManager := auth.NewJWTManager("issuer", "aud", "super-secret")
	loanHandler := handlers.NewLoanHandler(&fakeLoanService{result: &loandomain.UploadResult{LoanIDs: []string{"l1"}, Processed: 1}})
	r := server.NewRouter(config.Config{Env: "test"}, slog.Default(), server.Dependencies{JWTManager: jwtManager, LoanHandler: loanHandler})

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	_ = w.WriteField("lender_id", "lender-1")
	fw, _ := w.CreateFormFile("file", "loan.csv")
	_, _ = fw.Write([]byte("borrower_kyc_id,gov_id_hash,principal_minor,currency,interest_rate_bps,maturity_date,loan_reference\n"))
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/v1/loans/upload", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404 without auth routes enabled, got %d", resp.Code)
	}
}

func TestLoanUploadSuccess(t *testing.T) {
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

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	_ = w.WriteField("lender_id", "lender-1")
	fw, _ := w.CreateFormFile("file", "loan.csv")
	_, _ = fw.Write([]byte("borrower_kyc_id,gov_id_hash,principal_minor,currency,interest_rate_bps,maturity_date,loan_reference\nsmile:NG-BVN:1,abc123,500000,NGN,2200,2030-12-31T00:00:00Z,LOAN-001\n"))
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/v1/loans/upload", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(accessCookie)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}
