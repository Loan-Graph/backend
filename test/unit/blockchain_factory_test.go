package unit

import (
	"testing"

	"github.com/loangraph/backend/internal/blockchain"
	"github.com/loangraph/backend/internal/config"
)

func TestWriterFactoryReturnsStubByDefault(t *testing.T) {
	cfg := config.Config{ChainWriterMode: ""}
	w, err := blockchain.NewWriterFromConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := w.(*blockchain.StubWriter); !ok {
		t.Fatalf("expected stub writer by default")
	}
}

func TestWriterFactoryRealModeRequiresConfig(t *testing.T) {
	cfg := config.Config{ChainWriterMode: "real"}
	_, err := blockchain.NewWriterFromConfig(cfg)
	if err == nil {
		t.Fatalf("expected error for missing real writer config")
	}
}
