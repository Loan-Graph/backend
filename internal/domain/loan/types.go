package loan

import (
	"context"
	"time"
)

type Entity struct {
	ID               string
	LoanHash         []byte
	LenderID         string
	BorrowerID       string
	PrincipalMinor   int64
	CurrencyCode     string
	InterestRateBPS  int32
	StartDate        time.Time
	MaturityDate     time.Time
	AmountRepaid     int64
	Status           string
	OnChainTX        string
	OnChainConfirmed bool
	RiskGrade        string
	Metadata         []byte
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type CreateInput struct {
	LoanHash        []byte
	LenderID        string
	BorrowerID      string
	PrincipalMinor  int64
	CurrencyCode    string
	InterestRateBPS int32
	StartDate       time.Time
	MaturityDate    time.Time
	RiskGrade       string
	Metadata        []byte
}

type ListFilter struct {
	LenderID  string
	Status    string
	RiskGrade string
	Limit     int32
	Offset    int32
}

type Repository interface {
	Create(ctx context.Context, in CreateInput) (*Entity, error)
	GetByID(ctx context.Context, id string) (*Entity, error)
	GetByHash(ctx context.Context, loanHash []byte) (*Entity, error)
	List(ctx context.Context, f ListFilter) ([]Entity, error)
	SetOnChainSubmission(ctx context.Context, loanID, txHash string, confirmed bool) error
}
