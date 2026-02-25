package passport

import (
	"context"
	"time"
)

type Cache struct {
	BorrowerID         string
	TokenID            *int64
	TotalLoans         int32
	TotalRepaid        int32
	TotalDefaulted     int32
	CumulativeBorrowed int64
	CumulativeRepaid   int64
	CreditScore        int32
	LastUpdated        time.Time
}

type UpsertInput struct {
	BorrowerID         string
	TokenID            *int64
	TotalLoans         int32
	TotalRepaid        int32
	TotalDefaulted     int32
	CumulativeBorrowed int64
	CumulativeRepaid   int64
	CreditScore        int32
}

type Repository interface {
	Upsert(ctx context.Context, in UpsertInput) (*Cache, error)
	GetByBorrowerID(ctx context.Context, borrowerID string) (*Cache, error)
}
