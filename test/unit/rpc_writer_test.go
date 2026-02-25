package unit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/loangraph/backend/internal/blockchain"
)

func TestRPCWriterSendTransaction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req["method"] != "eth_sendTransaction" {
			t.Fatalf("unexpected method: %v", req["method"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "result": "0x123"})
	}))
	defer srv.Close()

	w, err := blockchain.NewRPCWriter(
		srv.URL,
		"0x1111111111111111111111111111111111111111",
		"0x2222222222222222222222222222222222222222",
		300000,
	)
	if err != nil {
		t.Fatalf("new rpc writer: %v", err)
	}

	tx, err := w.RegisterLoan(context.Background(), "loan-1")
	if err != nil {
		t.Fatalf("register loan: %v", err)
	}
	if tx != "0x123" {
		t.Fatalf("unexpected tx hash: %s", tx)
	}
}
