package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/loangraph/backend/internal/domain/lender"
)

type LenderRepository struct {
	pool *pgxpool.Pool
}

func NewLenderRepository(pool *pgxpool.Pool) *LenderRepository {
	return &LenderRepository{pool: pool}
}

func (r *LenderRepository) Create(ctx context.Context, in lender.CreateInput) (*lender.Entity, error) {
	q := `
INSERT INTO lenders (name, country_code, wallet_address, kyc_status, tier)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, name, country_code, wallet_address, kyc_status, tier, created_at, updated_at
`
	out := &lender.Entity{}
	err := r.pool.QueryRow(ctx, q, in.Name, in.CountryCode, in.WalletAddress, in.KYCStatus, in.Tier).
		Scan(&out.ID, &out.Name, &out.CountryCode, &out.WalletAddress, &out.KYCStatus, &out.Tier, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *LenderRepository) GetByID(ctx context.Context, id string) (*lender.Entity, error) {
	q := `SELECT id, name, country_code, wallet_address, kyc_status, tier, created_at, updated_at FROM lenders WHERE id = $1`
	out := &lender.Entity{}
	err := r.pool.QueryRow(ctx, q, id).
		Scan(&out.ID, &out.Name, &out.CountryCode, &out.WalletAddress, &out.KYCStatus, &out.Tier, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *LenderRepository) GetByWallet(ctx context.Context, walletAddress string) (*lender.Entity, error) {
	q := `SELECT id, name, country_code, wallet_address, kyc_status, tier, created_at, updated_at FROM lenders WHERE wallet_address = $1`
	out := &lender.Entity{}
	err := r.pool.QueryRow(ctx, q, walletAddress).
		Scan(&out.ID, &out.Name, &out.CountryCode, &out.WalletAddress, &out.KYCStatus, &out.Tier, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return out, nil
}
