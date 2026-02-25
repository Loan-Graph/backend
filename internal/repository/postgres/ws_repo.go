package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/loangraph/backend/internal/ws"
)

type WSRepository struct {
	pool *pgxpool.Pool
}

func NewWSRepository(pool *pgxpool.Pool) *WSRepository {
	return &WSRepository{pool: pool}
}

func (r *WSRepository) ListRepaymentEventsSince(ctx context.Context, lastID int64, limit int32) ([]ws.RealtimeEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	q := `
SELECT
  ce.id,
  (ce.raw_data->>'loan_id') AS loan_id,
  l.lender_id,
  COALESCE((ce.raw_data->>'amount_minor')::bigint, 0) AS amount_minor,
  l.currency_code,
  ce.indexed_at
FROM chain_events ce
JOIN loans l ON l.id = ((ce.raw_data->>'loan_id')::uuid)
WHERE ce.id > $1
  AND ce.event_name = 'RepaymentRecorded'
ORDER BY ce.id ASC
LIMIT $2
`
	rows, err := r.pool.Query(ctx, q, lastID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ws.RealtimeEvent, 0)
	for rows.Next() {
		var ev ws.RealtimeEvent
		var recordedAt time.Time
		if err := rows.Scan(&ev.ID, &ev.LoanID, &ev.LenderID, &ev.AmountMinor, &ev.Currency, &recordedAt); err != nil {
			return nil, err
		}
		ev.RecordedAt = recordedAt
		out = append(out, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *WSRepository) ListPoolsByLender(ctx context.Context, lenderID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT id FROM pools WHERE lender_id = $1`, lenderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]string, 0)
	for rows.Next() {
		var poolID string
		if err := rows.Scan(&poolID); err != nil {
			return nil, err
		}
		out = append(out, poolID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
