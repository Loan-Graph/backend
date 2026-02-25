package postgres

import (
	"context"
	"math"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/loangraph/backend/internal/indexer"
)

type IndexerRepository struct {
	pool *pgxpool.Pool
}

func NewIndexerRepository(pool *pgxpool.Pool) *IndexerRepository {
	return &IndexerRepository{pool: pool}
}

func (r *IndexerRepository) ListUnprocessed(ctx context.Context, limit int32) ([]indexer.ChainEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	q := `
SELECT id, event_name, tx_hash, raw_data::text
FROM chain_events
WHERE processed = FALSE
ORDER BY id
LIMIT $1
`
	rows, err := r.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]indexer.ChainEvent, 0)
	for rows.Next() {
		var ev indexer.ChainEvent
		var rawText string
		if err := rows.Scan(&ev.ID, &ev.EventName, &ev.TXHash, &rawText); err != nil {
			return nil, err
		}
		ev.RawData = []byte(rawText)
		out = append(out, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *IndexerRepository) MarkProcessed(ctx context.Context, eventID int64) error {
	_, err := r.pool.Exec(ctx, `UPDATE chain_events SET processed = TRUE WHERE id = $1`, eventID)
	return err
}

func (r *IndexerRepository) ApplyLoanRegistered(ctx context.Context, loanID, txHash string) error {
	_, err := r.pool.Exec(ctx, `UPDATE loans SET on_chain_tx = $2, on_chain_confirmed = TRUE, updated_at = NOW() WHERE id = $1`, loanID, txHash)
	return err
}

func (r *IndexerRepository) ApplyRepayment(ctx context.Context, loanID string, amountMinor int64) error {
	q := `
UPDATE loans
SET amount_repaid_minor = amount_repaid_minor + $2,
    status = CASE WHEN (amount_repaid_minor + $2) >= principal_minor THEN 'repaid' ELSE status END,
    updated_at = NOW()
WHERE id = $1 AND status != 'defaulted'
`
	_, err := r.pool.Exec(ctx, q, loanID, amountMinor)
	return err
}

func (r *IndexerRepository) ApplyDefault(ctx context.Context, loanID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE loans SET status = 'defaulted', updated_at = NOW() WHERE id = $1`, loanID)
	return err
}

func (r *IndexerRepository) RefreshPassportCacheByLoan(ctx context.Context, loanID string) error {
	var borrowerID string
	if err := r.pool.QueryRow(ctx, `SELECT borrower_id FROM loans WHERE id = $1`, loanID).Scan(&borrowerID); err != nil {
		return err
	}

	var totalLoans, totalRepaid, totalDefaulted int32
	var cumulativeBorrowed, cumulativeRepaid int64
	q := `
SELECT
  COUNT(*)::int,
  COUNT(*) FILTER (WHERE status = 'repaid')::int,
  COUNT(*) FILTER (WHERE status = 'defaulted')::int,
  COALESCE(SUM(principal_minor), 0)::bigint,
  COALESCE(SUM(amount_repaid_minor), 0)::bigint
FROM loans
WHERE borrower_id = $1
`
	if err := r.pool.QueryRow(ctx, q, borrowerID).Scan(&totalLoans, &totalRepaid, &totalDefaulted, &cumulativeBorrowed, &cumulativeRepaid); err != nil {
		return err
	}

	score := computeScore(cumulativeBorrowed, cumulativeRepaid, totalDefaulted)
	upsert := `
INSERT INTO passport_cache (
  borrower_id, token_id, total_loans, total_repaid, total_defaulted,
  cumulative_borrowed, cumulative_repaid, credit_score, last_updated
) VALUES ($1, NULL, $2, $3, $4, $5, $6, $7, NOW())
ON CONFLICT (borrower_id)
DO UPDATE SET
  total_loans = EXCLUDED.total_loans,
  total_repaid = EXCLUDED.total_repaid,
  total_defaulted = EXCLUDED.total_defaulted,
  cumulative_borrowed = EXCLUDED.cumulative_borrowed,
  cumulative_repaid = EXCLUDED.cumulative_repaid,
  credit_score = EXCLUDED.credit_score,
  last_updated = NOW()
`
	_, err := r.pool.Exec(ctx, upsert, borrowerID, totalLoans, totalRepaid, totalDefaulted, cumulativeBorrowed, cumulativeRepaid, score)
	return err
}

func computeScore(cumulativeBorrowed, cumulativeRepaid int64, totalDefaulted int32) int32 {
	score := 300.0
	if cumulativeBorrowed > 0 {
		ratio := float64(cumulativeRepaid) / float64(cumulativeBorrowed)
		score += ratio * 550.0
	}
	score -= float64(totalDefaulted) * 40.0
	if score < 300.0 {
		score = 300.0
	}
	if score > 850.0 {
		score = 850.0
	}
	return int32(math.Round(score))
}
