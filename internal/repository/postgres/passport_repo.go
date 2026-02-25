package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/loangraph/backend/internal/domain/passport"
)

type PassportRepository struct {
	pool *pgxpool.Pool
}

func NewPassportRepository(pool *pgxpool.Pool) *PassportRepository {
	return &PassportRepository{pool: pool}
}

func (r *PassportRepository) Upsert(ctx context.Context, in passport.UpsertInput) (*passport.Cache, error) {
	q := `
INSERT INTO passport_cache (
  borrower_id, token_id, total_loans, total_repaid, total_defaulted,
  cumulative_borrowed, cumulative_repaid, credit_score, last_updated
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())
ON CONFLICT (borrower_id)
DO UPDATE SET
  token_id = EXCLUDED.token_id,
  total_loans = EXCLUDED.total_loans,
  total_repaid = EXCLUDED.total_repaid,
  total_defaulted = EXCLUDED.total_defaulted,
  cumulative_borrowed = EXCLUDED.cumulative_borrowed,
  cumulative_repaid = EXCLUDED.cumulative_repaid,
  credit_score = EXCLUDED.credit_score,
  last_updated = NOW()
RETURNING borrower_id, token_id, total_loans, total_repaid, total_defaulted,
          cumulative_borrowed, cumulative_repaid, credit_score, last_updated
`
	out := &passport.Cache{}
	err := r.pool.QueryRow(ctx, q,
		in.BorrowerID, in.TokenID, in.TotalLoans, in.TotalRepaid, in.TotalDefaulted,
		in.CumulativeBorrowed, in.CumulativeRepaid, in.CreditScore,
	).Scan(
		&out.BorrowerID, &out.TokenID, &out.TotalLoans, &out.TotalRepaid, &out.TotalDefaulted,
		&out.CumulativeBorrowed, &out.CumulativeRepaid, &out.CreditScore, &out.LastUpdated,
	)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *PassportRepository) GetByBorrowerID(ctx context.Context, borrowerID string) (*passport.Cache, error) {
	q := `
SELECT borrower_id, token_id, total_loans, total_repaid, total_defaulted,
       cumulative_borrowed, cumulative_repaid, credit_score, last_updated
FROM passport_cache
WHERE borrower_id = $1
`
	out := &passport.Cache{}
	err := r.pool.QueryRow(ctx, q, borrowerID).Scan(
		&out.BorrowerID, &out.TokenID, &out.TotalLoans, &out.TotalRepaid, &out.TotalDefaulted,
		&out.CumulativeBorrowed, &out.CumulativeRepaid, &out.CreditScore, &out.LastUpdated,
	)
	if err != nil {
		return nil, err
	}
	return out, nil
}
