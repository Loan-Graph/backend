package unit

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/loangraph/backend/internal/auth"
)

func TestSetAndClearAuthCookies(t *testing.T) {
	r := httptest.NewRecorder()
	cfg := auth.CookieConfig{Secure: false}

	auth.SetAuthCookies(r, cfg, "access", "refresh", 15*time.Minute, 24*time.Hour)
	if len(r.Result().Cookies()) < 2 {
		t.Fatalf("expected auth cookies")
	}

	r2 := httptest.NewRecorder()
	auth.ClearAuthCookies(r2, cfg)
	if len(r2.Result().Cookies()) < 2 {
		t.Fatalf("expected clear cookies")
	}
}
