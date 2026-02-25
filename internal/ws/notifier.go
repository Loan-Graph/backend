package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type RealtimeEvent struct {
	ID          int64
	LoanID      string
	LenderID    string
	AmountMinor int64
	Currency    string
	RecordedAt  time.Time
}

type RealtimeRepository interface {
	ListRepaymentEventsSince(ctx context.Context, lastID int64, limit int32) ([]RealtimeEvent, error)
	ListPoolsByLender(ctx context.Context, lenderID string) ([]string, error)
}

type Notifier struct {
	repo         RealtimeRepository
	hub          *Hub
	pollInterval time.Duration
	lastID       int64
}

func NewNotifier(repo RealtimeRepository, hub *Hub, pollInterval time.Duration) *Notifier {
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}
	return &Notifier{repo: repo, hub: hub, pollInterval: pollInterval}
}

func (n *Notifier) Run(ctx context.Context) error {
	ticker := time.NewTicker(n.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := n.tick(ctx); err != nil {
				return err
			}
		}
	}
}

func (n *Notifier) tick(ctx context.Context) error {
	events, err := n.repo.ListRepaymentEventsSince(ctx, n.lastID, 100)
	if err != nil {
		return err
	}
	for _, ev := range events {
		if ev.ID > n.lastID {
			n.lastID = ev.ID
		}
		pools, err := n.repo.ListPoolsByLender(ctx, ev.LenderID)
		if err != nil {
			return err
		}
		payload, _ := json.Marshal(map[string]any{
			"event": "repayment_recorded",
			"data": map[string]any{
				"loan_id":      ev.LoanID,
				"lender_id":    ev.LenderID,
				"amount_minor": ev.AmountMinor,
				"currency":     ev.Currency,
				"recorded_at":  ev.RecordedAt.UTC().Format(time.RFC3339),
			},
		})
		for _, poolID := range pools {
			n.hub.Publish("pool:repayments:"+poolID, payload)
		}

		portfolioPayload, _ := json.Marshal(map[string]any{
			"event": "portfolio_updated",
			"data": map[string]any{
				"lender_id": ev.LenderID,
				"source":    "repayment_recorded",
			},
		})
		n.hub.Publish(fmt.Sprintf("lender:portfolio:%s", ev.LenderID), portfolioPayload)
	}
	return nil
}
