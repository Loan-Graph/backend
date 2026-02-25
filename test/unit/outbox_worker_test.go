package unit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/loangraph/backend/internal/jobs"
)

type fakeOutboxRepo struct {
	jobs      []jobs.OutboxJob
	doneIDs   []int64
	retryIDs  []int64
	failedIDs []int64
}

func (r *fakeOutboxRepo) ClaimPending(_ context.Context, _ int32) ([]jobs.OutboxJob, error) {
	return r.jobs, nil
}

func (r *fakeOutboxRepo) MarkDone(_ context.Context, jobID int64) error {
	r.doneIDs = append(r.doneIDs, jobID)
	return nil
}

func (r *fakeOutboxRepo) MarkRetry(_ context.Context, jobID int64, _ time.Time, _ string) error {
	r.retryIDs = append(r.retryIDs, jobID)
	return nil
}

func (r *fakeOutboxRepo) MarkFailed(_ context.Context, jobID int64, _ string) error {
	r.failedIDs = append(r.failedIDs, jobID)
	return nil
}

type fakeLoanRepo struct {
	updated map[string]string
}

func (r *fakeLoanRepo) SetOnChainSubmission(_ context.Context, loanID, txHash string, _ bool) error {
	if r.updated == nil {
		r.updated = map[string]string{}
	}
	r.updated[loanID] = txHash
	return nil
}

type fakeWriter struct {
	txHash string
	err    error
}

func (w *fakeWriter) RegisterLoan(_ context.Context, _ string) (string, error) {
	if w.err != nil {
		return "", w.err
	}
	return w.txHash, nil
}

func TestWorkerRunOnceSuccess(t *testing.T) {
	outbox := &fakeOutboxRepo{jobs: []jobs.OutboxJob{{ID: 1, Topic: "register_loan", Attempts: 1, Payload: []byte(`{"loan_id":"loan-1"}`)}}}
	loanRepo := &fakeLoanRepo{}
	worker := jobs.NewWorker(outbox, loanRepo, &fakeWriter{txHash: "0xtx"})

	if err := worker.RunOnce(context.Background(), 10); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if len(outbox.doneIDs) != 1 || outbox.doneIDs[0] != 1 {
		t.Fatalf("expected job marked done")
	}
	if loanRepo.updated["loan-1"] != "0xtx" {
		t.Fatalf("expected loan on-chain tx update")
	}
}

func TestWorkerRunOnceRetryOnWriterError(t *testing.T) {
	outbox := &fakeOutboxRepo{jobs: []jobs.OutboxJob{{ID: 1, Topic: "register_loan", Attempts: 1, Payload: []byte(`{"loan_id":"loan-1"}`)}}}
	loanRepo := &fakeLoanRepo{}
	worker := jobs.NewWorker(outbox, loanRepo, &fakeWriter{err: errors.New("rpc down")})

	if err := worker.RunOnce(context.Background(), 10); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if len(outbox.retryIDs) != 1 || outbox.retryIDs[0] != 1 {
		t.Fatalf("expected job marked retry")
	}
}

func TestWorkerRunOnceTerminalFailure(t *testing.T) {
	outbox := &fakeOutboxRepo{jobs: []jobs.OutboxJob{{ID: 9, Topic: "register_loan", Attempts: 5, Payload: []byte(`{"loan_id":"loan-1"}`)}}}
	loanRepo := &fakeLoanRepo{}
	worker := jobs.NewWorker(outbox, loanRepo, &fakeWriter{err: errors.New("rpc down")})

	if err := worker.RunOnce(context.Background(), 10); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if len(outbox.failedIDs) != 1 || outbox.failedIDs[0] != 9 {
		t.Fatalf("expected job marked failed")
	}
}
