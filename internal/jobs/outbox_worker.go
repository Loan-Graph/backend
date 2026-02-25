package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/loangraph/backend/internal/blockchain"
)

const registerLoanTopic = "register_loan"

type OutboxJob struct {
	ID          int64
	Topic       string
	Payload     []byte
	Status      string
	Attempts    int32
	LastError   string
	AvailableAt time.Time
}

type OutboxRepository interface {
	ClaimPending(ctx context.Context, limit int32) ([]OutboxJob, error)
	MarkDone(ctx context.Context, jobID int64) error
	MarkRetry(ctx context.Context, jobID int64, nextAvailableAt time.Time, lastError string) error
	MarkFailed(ctx context.Context, jobID int64, lastError string) error
}

type LoanRepository interface {
	SetOnChainSubmission(ctx context.Context, loanID, txHash string, confirmed bool) error
}

type Worker struct {
	outboxRepo   OutboxRepository
	loanRepo     LoanRepository
	writer       blockchain.LoanRegistryWriter
	maxAttempts  int32
	now          func() time.Time
	retryBackoff func(attempt int32) time.Duration
}

func NewWorker(outboxRepo OutboxRepository, loanRepo LoanRepository, writer blockchain.LoanRegistryWriter) *Worker {
	return &Worker{
		outboxRepo:  outboxRepo,
		loanRepo:    loanRepo,
		writer:      writer,
		maxAttempts: 5,
		now:         func() time.Time { return time.Now().UTC() },
		retryBackoff: func(attempt int32) time.Duration {
			if attempt < 1 {
				attempt = 1
			}
			return time.Duration(attempt*15) * time.Second
		},
	}
}

func (w *Worker) RunOnce(ctx context.Context, batchSize int32) error {
	jobs, err := w.outboxRepo.ClaimPending(ctx, batchSize)
	if err != nil {
		return err
	}

	for _, job := range jobs {
		if err := w.processJob(ctx, job); err != nil {
			return err
		}
	}

	return nil
}

func (w *Worker) processJob(ctx context.Context, job OutboxJob) error {
	switch job.Topic {
	case registerLoanTopic:
		return w.processRegisterLoan(ctx, job)
	default:
		if job.Attempts >= w.maxAttempts {
			return w.outboxRepo.MarkFailed(ctx, job.ID, "unsupported_topic")
		}
		next := w.now().Add(w.retryBackoff(job.Attempts))
		return w.outboxRepo.MarkRetry(ctx, job.ID, next, "unsupported_topic")
	}
}

type registerLoanPayload struct {
	LoanID string `json:"loan_id"`
}

func (w *Worker) processRegisterLoan(ctx context.Context, job OutboxJob) error {
	var payload registerLoanPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return w.handleJobError(ctx, job, fmt.Errorf("invalid_payload"))
	}
	if payload.LoanID == "" {
		return w.handleJobError(ctx, job, errors.New("missing_loan_id"))
	}

	txHash, err := w.writer.RegisterLoan(ctx, payload.LoanID)
	if err != nil {
		return w.handleJobError(ctx, job, err)
	}

	if err := w.loanRepo.SetOnChainSubmission(ctx, payload.LoanID, txHash, false); err != nil {
		return w.handleJobError(ctx, job, err)
	}

	return w.outboxRepo.MarkDone(ctx, job.ID)
}

func (w *Worker) handleJobError(ctx context.Context, job OutboxJob, err error) error {
	msg := err.Error()
	if job.Attempts >= w.maxAttempts {
		return w.outboxRepo.MarkFailed(ctx, job.ID, msg)
	}
	next := w.now().Add(w.retryBackoff(job.Attempts))
	return w.outboxRepo.MarkRetry(ctx, job.ID, next, msg)
}
