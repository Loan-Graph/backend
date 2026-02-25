package unit

import (
	"testing"
	"time"

	"github.com/loangraph/backend/internal/auth"
)

func TestJWTMintAndParse(t *testing.T) {
	m := auth.NewJWTManager("issuer", "aud", "secret")
	tok, err := m.Mint("u1", "s1", "access", 5*time.Minute)
	if err != nil {
		t.Fatalf("mint error: %v", err)
	}

	claims, err := m.Parse(tok)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if claims.UserID != "u1" || claims.SessionID != "s1" || claims.Type != "access" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}
