package blockchain

import (
	"context"
	"fmt"
	"time"
)

type LoanRegistryWriter interface {
	RegisterLoan(ctx context.Context, loanID string) (string, error)
}

type StubWriter struct{}

func NewStubWriter() *StubWriter {
	return &StubWriter{}
}

func (w *StubWriter) RegisterLoan(_ context.Context, loanID string) (string, error) {
	if loanID == "" {
		return "", fmt.Errorf("missing loan id")
	}
	prefix := loanID
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}
	return fmt.Sprintf("0xstub%s%x", prefix, time.Now().UTC().UnixNano()), nil
}
