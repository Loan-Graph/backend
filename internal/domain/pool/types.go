package pool

import (
	"context"
	"time"
)

type Entity struct {
	ID            string
	LenderID      string
	Name          string
	PoolTokenAddr string
	TargetAPYBPS  int32
	CurrencyCode  string
	Status        string
	TotalDeployed int64
	CreatedAt     time.Time
}

type CreateInput struct {
	LenderID      string
	Name          string
	PoolTokenAddr string
	TargetAPYBPS  int32
	CurrencyCode  string
	Status        string
}

type Repository interface {
	Create(ctx context.Context, in CreateInput) (*Entity, error)
	GetByID(ctx context.Context, id string) (*Entity, error)
	ListByLender(ctx context.Context, lenderID string) ([]Entity, error)
}
