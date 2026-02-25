package blockchain

import (
	"context"
	"fmt"
	"time"
)

type LoanRegistryWriter interface {
	RegisterLoan(ctx context.Context, loanID string) (string, error)
	RecordRepayment(ctx context.Context, loanID string, amountMinor int64, currency string) (string, error)
	MarkDefault(ctx context.Context, loanID string, reason string) (string, error)
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

func (w *StubWriter) RecordRepayment(_ context.Context, loanID string, amountMinor int64, currency string) (string, error) {
	if loanID == "" || amountMinor <= 0 || len(currency) != 3 {
		return "", fmt.Errorf("invalid repayment args")
	}
	return fmt.Sprintf("0xrepay%s%x", loanID[:min(8, len(loanID))], time.Now().UTC().UnixNano()), nil
}

func (w *StubWriter) MarkDefault(_ context.Context, loanID string, reason string) (string, error) {
	if loanID == "" {
		return "", fmt.Errorf("invalid default args")
	}
	return fmt.Sprintf("0xdef%s%x", loanID[:min(8, len(loanID))], time.Now().UTC().UnixNano()), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
