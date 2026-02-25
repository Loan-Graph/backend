package borrower

import (
	"context"
	"time"
)

type Entity struct {
	ID           string
	BorrowerHash []byte
	LenderID     string
	CountryCode  string
	Sector       string
	CreatedAt    time.Time
}

type CreateInput struct {
	BorrowerHash []byte
	LenderID     string
	CountryCode  string
	Sector       string
}

type Repository interface {
	Create(ctx context.Context, in CreateInput) (*Entity, error)
	GetByID(ctx context.Context, id string) (*Entity, error)
	GetByHash(ctx context.Context, borrowerHash []byte) (*Entity, error)
}
