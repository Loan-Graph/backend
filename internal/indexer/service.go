package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type ChainEvent struct {
	ID        int64
	EventName string
	TXHash    string
	RawData   []byte
}

type EventRepository interface {
	ListUnprocessed(ctx context.Context, limit int32) ([]ChainEvent, error)
	MarkProcessed(ctx context.Context, eventID int64) error
}

type ProjectionRepository interface {
	ApplyLoanRegistered(ctx context.Context, loanID, txHash string) error
	ApplyRepayment(ctx context.Context, loanID string, amountMinor int64) error
	ApplyDefault(ctx context.Context, loanID string) error
	RefreshPassportCacheByLoan(ctx context.Context, loanID string) error
}

type Service struct {
	eventRepo EventRepository
	projRepo  ProjectionRepository
}

func NewService(eventRepo EventRepository, projRepo ProjectionRepository) *Service {
	return &Service{eventRepo: eventRepo, projRepo: projRepo}
}

func (s *Service) RunOnce(ctx context.Context, batchSize int32) error {
	events, err := s.eventRepo.ListUnprocessed(ctx, batchSize)
	if err != nil {
		return err
	}

	for _, ev := range events {
		if err := s.processEvent(ctx, ev); err != nil {
			return err
		}
		if err := s.eventRepo.MarkProcessed(ctx, ev.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) processEvent(ctx context.Context, ev ChainEvent) error {
	name := strings.TrimSpace(ev.EventName)
	switch name {
	case "LoanRegistered":
		var payload struct {
			LoanID string `json:"loan_id"`
		}
		if err := json.Unmarshal(ev.RawData, &payload); err != nil {
			return fmt.Errorf("invalid LoanRegistered payload: %w", err)
		}
		if strings.TrimSpace(payload.LoanID) == "" {
			return fmt.Errorf("missing loan_id in LoanRegistered")
		}
		return s.projRepo.ApplyLoanRegistered(ctx, payload.LoanID, ev.TXHash)

	case "RepaymentRecorded":
		var payload struct {
			LoanID      string `json:"loan_id"`
			AmountMinor int64  `json:"amount_minor"`
		}
		if err := json.Unmarshal(ev.RawData, &payload); err != nil {
			return fmt.Errorf("invalid RepaymentRecorded payload: %w", err)
		}
		if strings.TrimSpace(payload.LoanID) == "" || payload.AmountMinor <= 0 {
			return fmt.Errorf("invalid RepaymentRecorded payload values")
		}
		if err := s.projRepo.ApplyRepayment(ctx, payload.LoanID, payload.AmountMinor); err != nil {
			return err
		}
		return s.projRepo.RefreshPassportCacheByLoan(ctx, payload.LoanID)

	case "LoanDefaulted":
		var payload struct {
			LoanID string `json:"loan_id"`
		}
		if err := json.Unmarshal(ev.RawData, &payload); err != nil {
			return fmt.Errorf("invalid LoanDefaulted payload: %w", err)
		}
		if strings.TrimSpace(payload.LoanID) == "" {
			return fmt.Errorf("missing loan_id in LoanDefaulted")
		}
		if err := s.projRepo.ApplyDefault(ctx, payload.LoanID); err != nil {
			return err
		}
		return s.projRepo.RefreshPassportCacheByLoan(ctx, payload.LoanID)

	default:
		return nil
	}
}
