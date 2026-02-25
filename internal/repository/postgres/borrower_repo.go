package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/loangraph/backend/internal/domain/borrower"
)

type BorrowerRepository struct {
	pool *pgxpool.Pool
}

func NewBorrowerRepository(pool *pgxpool.Pool) *BorrowerRepository {
	return &BorrowerRepository{pool: pool}
}

func (r *BorrowerRepository) Create(ctx context.Context, in borrower.CreateInput) (*borrower.Entity, error) {
	q := `
INSERT INTO borrowers (borrower_hash, lender_id, country_code, sector)
VALUES ($1, $2, $3, $4)
RETURNING id, borrower_hash, lender_id, country_code, sector, created_at
`
	out := &borrower.Entity{}
	err := r.pool.QueryRow(ctx, q, in.BorrowerHash, in.LenderID, in.CountryCode, in.Sector).
		Scan(&out.ID, &out.BorrowerHash, &out.LenderID, &out.CountryCode, &out.Sector, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *BorrowerRepository) GetByID(ctx context.Context, id string) (*borrower.Entity, error) {
	q := `SELECT id, borrower_hash, lender_id, country_code, sector, created_at FROM borrowers WHERE id = $1`
	out := &borrower.Entity{}
	err := r.pool.QueryRow(ctx, q, id).
		Scan(&out.ID, &out.BorrowerHash, &out.LenderID, &out.CountryCode, &out.Sector, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *BorrowerRepository) GetByHash(ctx context.Context, borrowerHash []byte) (*borrower.Entity, error) {
	q := `SELECT id, borrower_hash, lender_id, country_code, sector, created_at FROM borrowers WHERE borrower_hash = $1`
	out := &borrower.Entity{}
	err := r.pool.QueryRow(ctx, q, borrowerHash).
		Scan(&out.ID, &out.BorrowerHash, &out.LenderID, &out.CountryCode, &out.Sector, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return out, nil
}
