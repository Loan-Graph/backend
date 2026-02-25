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

type PortfolioAnalytics struct {
	LenderID             string  `json:"lender_id"`
	TotalLoans           int64   `json:"total_loans"`
	ActiveLoans          int64   `json:"active_loans"`
	RepaidLoans          int64   `json:"repaid_loans"`
	DefaultedLoans       int64   `json:"defaulted_loans"`
	TotalPrincipalMinor  int64   `json:"total_principal_minor"`
	TotalRepaidMinor     int64   `json:"total_repaid_minor"`
	RepaymentRatePercent float64 `json:"repayment_rate_percent"`
}

type ScoreBand struct {
	Label string `json:"label"`
	Count int64  `json:"count"`
}

type PortfolioHealth struct {
	LenderID        string      `json:"lender_id"`
	UniqueBorrowers int64       `json:"unique_borrowers"`
	AverageScore    float64     `json:"average_score"`
	ScoreBands      []ScoreBand `json:"score_bands"`
}

type Repository interface {
	Create(ctx context.Context, in CreateInput) (*Entity, error)
	GetByID(ctx context.Context, id string) (*Entity, error)
	GetByHash(ctx context.Context, loanHash []byte) (*Entity, error)
	List(ctx context.Context, f ListFilter) ([]Entity, error)
	SetOnChainSubmission(ctx context.Context, loanID, txHash string, confirmed bool) error
	RecordRepayment(ctx context.Context, loanID string, amountMinor int64) error
	MarkDefault(ctx context.Context, loanID string) error
	GetPortfolioAnalytics(ctx context.Context, lenderID string) (*PortfolioAnalytics, error)
	ListByBorrower(ctx context.Context, borrowerID string, limit, offset int32) ([]Entity, error)
	GetPortfolioHealth(ctx context.Context, lenderID string) (*PortfolioHealth, error)
}
