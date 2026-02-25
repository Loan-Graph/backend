package lender

import (
	"context"
	"time"
)

type Entity struct {
	ID            string
	Name          string
	CountryCode   string
	WalletAddress string
	KYCStatus     string
	Tier          string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type CreateInput struct {
	Name          string
	CountryCode   string
	WalletAddress string
	KYCStatus     string
	Tier          string
}

type Repository interface {
	Create(ctx context.Context, in CreateInput) (*Entity, error)
	GetByID(ctx context.Context, id string) (*Entity, error)
	GetByWallet(ctx context.Context, walletAddress string) (*Entity, error)
	UpdateKYCStatus(ctx context.Context, lenderID, kycStatus string) error
}
