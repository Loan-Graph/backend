package integration

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/loangraph/backend/internal/config"
	"github.com/loangraph/backend/internal/server"
)

type fakePinger struct {
	err error
}

func (p fakePinger) Ping(_ context.Context) error {
	return p.err
}

func TestHealthEndpoint(t *testing.T) {
	r := server.NewRouter(config.Config{Env: "test"}, slog.Default(), fakePinger{})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestReadyEndpointOK(t *testing.T) {
	r := server.NewRouter(config.Config{Env: "test"}, slog.Default(), fakePinger{})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestReadyEndpointDBFailure(t *testing.T) {
	r := server.NewRouter(config.Config{Env: "test"}, slog.Default(), fakePinger{err: errors.New("db down")})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}
