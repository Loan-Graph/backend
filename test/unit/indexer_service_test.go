package unit

import (
	"context"
	"testing"

	"github.com/loangraph/backend/internal/indexer"
)

type fakeEventRepo struct {
	events      []indexer.ChainEvent
	processedID []int64
}

func (r *fakeEventRepo) ListUnprocessed(_ context.Context, _ int32) ([]indexer.ChainEvent, error) {
	return r.events, nil
}

func (r *fakeEventRepo) MarkProcessed(_ context.Context, eventID int64) error {
	r.processedID = append(r.processedID, eventID)
	return nil
}

type fakeProjectionRepo struct {
	registered []string
	repayments []string
	defaults   []string
	refreshed  []string
}

func (r *fakeProjectionRepo) ApplyLoanRegistered(_ context.Context, loanID, _ string) error {
	r.registered = append(r.registered, loanID)
	return nil
}

func (r *fakeProjectionRepo) ApplyRepayment(_ context.Context, loanID string, _ int64) error {
	r.repayments = append(r.repayments, loanID)
	return nil
}

func (r *fakeProjectionRepo) ApplyDefault(_ context.Context, loanID string) error {
	r.defaults = append(r.defaults, loanID)
	return nil
}

func (r *fakeProjectionRepo) RefreshPassportCacheByLoan(_ context.Context, loanID string) error {
	r.refreshed = append(r.refreshed, loanID)
	return nil
}

func TestIndexerRunOnceProcessesSupportedEvents(t *testing.T) {
	evRepo := &fakeEventRepo{events: []indexer.ChainEvent{
		{ID: 1, EventName: "LoanRegistered", TXHash: "0x1", RawData: []byte(`{"loan_id":"11111111-1111-1111-1111-111111111111"}`)},
		{ID: 2, EventName: "RepaymentRecorded", TXHash: "0x2", RawData: []byte(`{"loan_id":"22222222-2222-2222-2222-222222222222","amount_minor":5000}`)},
		{ID: 3, EventName: "LoanDefaulted", TXHash: "0x3", RawData: []byte(`{"loan_id":"33333333-3333-3333-3333-333333333333"}`)},
	}}
	proj := &fakeProjectionRepo{}
	svc := indexer.NewService(evRepo, proj)

	if err := svc.RunOnce(context.Background(), 10); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if len(evRepo.processedID) != 3 {
		t.Fatalf("expected 3 processed events")
	}
	if len(proj.registered) != 1 || proj.registered[0] != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("loan registered projection mismatch")
	}
	if len(proj.repayments) != 1 || proj.repayments[0] != "22222222-2222-2222-2222-222222222222" {
		t.Fatalf("repayment projection mismatch")
	}
	if len(proj.defaults) != 1 || proj.defaults[0] != "33333333-3333-3333-3333-333333333333" {
		t.Fatalf("default projection mismatch")
	}
	if len(proj.refreshed) != 2 {
		t.Fatalf("expected passport refresh for repayment/default")
	}
}

func TestIndexerRunOnceIgnoresUnknownEvent(t *testing.T) {
	evRepo := &fakeEventRepo{events: []indexer.ChainEvent{{ID: 9, EventName: "UnknownEvent", RawData: []byte(`{}`)}}}
	proj := &fakeProjectionRepo{}
	svc := indexer.NewService(evRepo, proj)

	if err := svc.RunOnce(context.Background(), 10); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if len(evRepo.processedID) != 1 || evRepo.processedID[0] != 9 {
		t.Fatalf("expected unknown event to be marked processed")
	}
}

func TestIndexerRunOnceSkipsProjectionForNonUUIDLoanID(t *testing.T) {
	evRepo := &fakeEventRepo{events: []indexer.ChainEvent{
		{ID: 10, EventName: "LoanRegistered", TXHash: "0x10", RawData: []byte(`{"loan_id":"0xabc123"}`)},
		{ID: 11, EventName: "RepaymentRecorded", TXHash: "0x11", RawData: []byte(`{"loan_id":"0xabc123","amount_minor":100}`)},
		{ID: 12, EventName: "LoanDefaulted", TXHash: "0x12", RawData: []byte(`{"loan_id":"0xabc123"}`)},
	}}
	proj := &fakeProjectionRepo{}
	svc := indexer.NewService(evRepo, proj)

	if err := svc.RunOnce(context.Background(), 10); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if len(evRepo.processedID) != 3 {
		t.Fatalf("expected 3 processed events")
	}
	if len(proj.registered) != 0 || len(proj.repayments) != 0 || len(proj.defaults) != 0 || len(proj.refreshed) != 0 {
		t.Fatalf("expected no projections for non-uuid loan ids")
	}
}
