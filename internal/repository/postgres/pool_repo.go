package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/loangraph/backend/internal/domain/pool"
)

type PoolRepository struct {
	pool *pgxpool.Pool
}

func NewPoolRepository(db *pgxpool.Pool) *PoolRepository {
	return &PoolRepository{pool: db}
}

func (r *PoolRepository) Create(ctx context.Context, in pool.CreateInput) (*pool.Entity, error) {
	q := `
INSERT INTO pools (lender_id, name, pool_token_addr, target_apybps, currency_code, status)
VALUES ($1,$2,$3,$4,$5,$6)
RETURNING id, lender_id, name, pool_token_addr, target_apybps, currency_code, status, total_deployed, created_at
`
	out := &pool.Entity{}
	err := r.pool.QueryRow(ctx, q, in.LenderID, in.Name, in.PoolTokenAddr, in.TargetAPYBPS, in.CurrencyCode, in.Status).
		Scan(&out.ID, &out.LenderID, &out.Name, &out.PoolTokenAddr, &out.TargetAPYBPS, &out.CurrencyCode, &out.Status, &out.TotalDeployed, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *PoolRepository) GetByID(ctx context.Context, id string) (*pool.Entity, error) {
	q := `SELECT id, lender_id, name, pool_token_addr, target_apybps, currency_code, status, total_deployed, created_at FROM pools WHERE id = $1`
	out := &pool.Entity{}
	err := r.pool.QueryRow(ctx, q, id).
		Scan(&out.ID, &out.LenderID, &out.Name, &out.PoolTokenAddr, &out.TargetAPYBPS, &out.CurrencyCode, &out.Status, &out.TotalDeployed, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *PoolRepository) ListByLender(ctx context.Context, lenderID string) ([]pool.Entity, error) {
	q := `SELECT id, lender_id, name, pool_token_addr, target_apybps, currency_code, status, total_deployed, created_at FROM pools WHERE lender_id = $1 ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, q, lenderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]pool.Entity, 0)
	for rows.Next() {
		var item pool.Entity
		if err := rows.Scan(&item.ID, &item.LenderID, &item.Name, &item.PoolTokenAddr, &item.TargetAPYBPS, &item.CurrencyCode, &item.Status, &item.TotalDeployed, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
