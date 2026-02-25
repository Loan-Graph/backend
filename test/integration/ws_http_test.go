package integration

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/loangraph/backend/internal/auth"
	"github.com/loangraph/backend/internal/config"
	"github.com/loangraph/backend/internal/http/handlers"
	"github.com/loangraph/backend/internal/server"
	internalws "github.com/loangraph/backend/internal/ws"
	"golang.org/x/net/websocket"
)

func TestWebSocketSubscribeAndReceive(t *testing.T) {
	repo := newFakeRepo()
	jwtManager := auth.NewJWTManager("issuer", "aud", "super-secret")
	authSvc := auth.NewService(repo, jwtManager, fakeVerifier{}, 15*time.Minute, 24*time.Hour, "")
	authHandler := handlers.NewAuthHandler(authSvc, auth.CookieConfig{}, 15*time.Minute, 24*time.Hour)

	hub := internalws.NewHub()
	wsHandler := internalws.NewHandler(hub)

	r := server.NewRouter(config.Config{Env: "test"}, slog.Default(), server.Dependencies{
		AuthHandler: authHandler,
		WSHandler:   wsHandler,
		JWTManager:  jwtManager,
	})
	ts := httptest.NewServer(r)
	defer ts.Close()

	loginReqBody, _ := json.Marshal(map[string]string{"privy_access_token": "token"})
	loginResp, err := http.Post(ts.URL+"/v1/auth/privy/login", "application/json", bytes.NewReader(loginReqBody))
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	defer loginResp.Body.Close()
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("expected login 200, got %d", loginResp.StatusCode)
	}

	var accessCookie *http.Cookie
	for _, c := range loginResp.Cookies() {
		if c.Name == auth.AccessCookieName {
			accessCookie = c
			break
		}
	}
	if accessCookie == nil {
		t.Fatalf("missing access cookie")
	}

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/v1/ws"
	cfg, err := websocket.NewConfig(wsURL, ts.URL)
	if err != nil {
		t.Fatalf("new ws config: %v", err)
	}
	cfg.Header = make(http.Header)
	cfg.Header.Set("Cookie", accessCookie.String())
	conn, err := websocket.DialConfig(cfg)
	if err != nil {
		t.Fatalf("dial ws: %v", err)
	}
	defer conn.Close()

	sub := `{"action":"subscribe","channel":"pool:repayments","poolId":"pool-1"}`
	if err := websocket.Message.Send(conn, sub); err != nil {
		t.Fatalf("send subscribe: %v", err)
	}
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	var ack string
	if err := websocket.Message.Receive(conn, &ack); err != nil {
		t.Fatalf("receive ws ack: %v", err)
	}
	if !strings.Contains(ack, "subscribed") {
		t.Fatalf("unexpected ws ack: %s", ack)
	}
	time.Sleep(50 * time.Millisecond)

	hub.Publish("pool:repayments:pool-1", []byte(`{"event":"repayment_recorded","data":{"loan_id":"loan-1"}}`))

	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	var msg string
	if err := websocket.Message.Receive(conn, &msg); err != nil {
		t.Fatalf("receive ws message: %v", err)
	}
	if !strings.Contains(msg, "repayment_recorded") {
		t.Fatalf("unexpected ws payload: %s", msg)
	}
}
