package investor

import (
	"context"
	"fmt"
	"strings"

	lenderdomain "github.com/loangraph/backend/internal/domain/lender"
	loandomain "github.com/loangraph/backend/internal/domain/loan"
	pooldomain "github.com/loangraph/backend/internal/domain/pool"
)

type PoolFilter struct {
	LenderID     string
	CurrencyCode string
	Status       string
	Limit        int32
	Offset       int32
}

type LenderProfile struct {
	Lender    *lenderdomain.Entity           `json:"lender"`
	Analytics *loandomain.PortfolioAnalytics `json:"analytics"`
	Health    *loandomain.PortfolioHealth    `json:"health"`
}

type PoolRepository interface {
	List(ctx context.Context, lenderID, currencyCode, status string, limit, offset int32) ([]pooldomain.Entity, error)
	GetByID(ctx context.Context, id string) (*pooldomain.Entity, error)
}

type LenderRepository interface {
	GetByID(ctx context.Context, id string) (*lenderdomain.Entity, error)
}

type LoanRepository interface {
	GetRepaymentTimeSeriesByLender(ctx context.Context, lenderID string, days int32) ([]loandomain.PerformancePoint, error)
	GetPortfolioAnalytics(ctx context.Context, lenderID string) (*loandomain.PortfolioAnalytics, error)
	GetPortfolioHealth(ctx context.Context, lenderID string) (*loandomain.PortfolioHealth, error)
}

type Service struct {
	poolRepo   PoolRepository
	lenderRepo LenderRepository
	loanRepo   LoanRepository
}

func NewService(poolRepo PoolRepository, lenderRepo LenderRepository, loanRepo LoanRepository) *Service {
	return &Service{poolRepo: poolRepo, lenderRepo: lenderRepo, loanRepo: loanRepo}
}

func (s *Service) ListPools(ctx context.Context, f PoolFilter) ([]pooldomain.Entity, error) {
	return s.poolRepo.List(ctx, f.LenderID, f.CurrencyCode, f.Status, f.Limit, f.Offset)
}

func (s *Service) GetPool(ctx context.Context, poolID string) (*pooldomain.Entity, error) {
	if strings.TrimSpace(poolID) == "" {
		return nil, fmt.Errorf("missing_pool_id")
	}
	return s.poolRepo.GetByID(ctx, poolID)
}

func (s *Service) GetPoolPerformance(ctx context.Context, poolID string, days int32) ([]loandomain.PerformancePoint, error) {
	pool, err := s.GetPool(ctx, poolID)
	if err != nil {
		return nil, err
	}
	return s.loanRepo.GetRepaymentTimeSeriesByLender(ctx, pool.LenderID, days)
}

func (s *Service) GetLenderProfile(ctx context.Context, lenderID string) (*LenderProfile, error) {
	if strings.TrimSpace(lenderID) == "" {
		return nil, fmt.Errorf("missing_lender_id")
	}
	lender, err := s.lenderRepo.GetByID(ctx, lenderID)
	if err != nil {
		return nil, err
	}
	analytics, err := s.loanRepo.GetPortfolioAnalytics(ctx, lenderID)
	if err != nil {
		return nil, err
	}
	health, err := s.loanRepo.GetPortfolioHealth(ctx, lenderID)
	if err != nil {
		return nil, err
	}
	return &LenderProfile{Lender: lender, Analytics: analytics, Health: health}, nil
}
