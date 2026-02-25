package unit

import (
	"context"
	"testing"

	investordomain "github.com/loangraph/backend/internal/domain/investor"
	lenderdomain "github.com/loangraph/backend/internal/domain/lender"
	loandomain "github.com/loangraph/backend/internal/domain/loan"
	pooldomain "github.com/loangraph/backend/internal/domain/pool"
)

type investorPoolRepoMock struct {
	pools []pooldomain.Entity
}

func (m *investorPoolRepoMock) List(_ context.Context, _, _, _ string, _, _ int32) ([]pooldomain.Entity, error) {
	return m.pools, nil
}

func (m *investorPoolRepoMock) GetByID(_ context.Context, id string) (*pooldomain.Entity, error) {
	for _, p := range m.pools {
		if p.ID == id {
			cp := p
			return &cp, nil
		}
	}
	return nil, context.Canceled
}

type investorLenderRepoMock struct {
	lender *lenderdomain.Entity
}

func (m *investorLenderRepoMock) GetByID(_ context.Context, _ string) (*lenderdomain.Entity, error) {
	return m.lender, nil
}

type investorLoanRepoMock struct {
	series    []loandomain.PerformancePoint
	analytics *loandomain.PortfolioAnalytics
	health    *loandomain.PortfolioHealth
}

func (m *investorLoanRepoMock) GetRepaymentTimeSeriesByLender(_ context.Context, _ string, _ int32) ([]loandomain.PerformancePoint, error) {
	return m.series, nil
}

func (m *investorLoanRepoMock) GetPortfolioAnalytics(_ context.Context, _ string) (*loandomain.PortfolioAnalytics, error) {
	return m.analytics, nil
}

func (m *investorLoanRepoMock) GetPortfolioHealth(_ context.Context, _ string) (*loandomain.PortfolioHealth, error) {
	return m.health, nil
}

func TestInvestorServiceGetPoolPerformanceAndLenderProfile(t *testing.T) {
	svc := investordomain.NewService(
		&investorPoolRepoMock{pools: []pooldomain.Entity{{ID: "pool-1", LenderID: "lender-1", Name: "Starter Pool"}}},
		&investorLenderRepoMock{lender: &lenderdomain.Entity{ID: "lender-1", Name: "Lender A"}},
		&investorLoanRepoMock{
			series:    []loandomain.PerformancePoint{{Date: "2026-02-25", RepaymentCount: 3, RepaidAmountMinor: 10000}},
			analytics: &loandomain.PortfolioAnalytics{LenderID: "lender-1", TotalLoans: 10},
			health:    &loandomain.PortfolioHealth{LenderID: "lender-1", UniqueBorrowers: 5},
		},
	)

	points, err := svc.GetPoolPerformance(context.Background(), "pool-1", 30)
	if err != nil {
		t.Fatalf("pool performance error: %v", err)
	}
	if len(points) != 1 || points[0].RepaymentCount != 3 {
		t.Fatalf("unexpected performance points")
	}

	profile, err := svc.GetLenderProfile(context.Background(), "lender-1")
	if err != nil {
		t.Fatalf("lender profile error: %v", err)
	}
	if profile.Lender.ID != "lender-1" || profile.Analytics.TotalLoans != 10 || profile.Health.UniqueBorrowers != 5 {
		t.Fatalf("unexpected lender profile payload")
	}
}
