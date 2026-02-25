package integration

import (
	"context"
	"testing"
	"time"

	borrowerdomain "github.com/loangraph/backend/internal/domain/borrower"
	lenderdomain "github.com/loangraph/backend/internal/domain/lender"
	loandomain "github.com/loangraph/backend/internal/domain/loan"
	passportdomain "github.com/loangraph/backend/internal/domain/passport"
	pooldomain "github.com/loangraph/backend/internal/domain/pool"
	"github.com/loangraph/backend/internal/repository/postgres"
	"github.com/loangraph/backend/test/integration/testutil"
)

func TestPostgresRepositoriesCoreDomainFlow(t *testing.T) {
	pool := testutil.NewTestPool(t)
	defer pool.Close()
	testutil.ApplyMigrations(t, pool)
	testutil.ResetTables(t, pool)

	ctx := context.Background()
	lenderRepo := postgres.NewLenderRepository(pool)
	borrowerRepo := postgres.NewBorrowerRepository(pool)
	loanRepo := postgres.NewLoanRepository(pool)
	poolRepo := postgres.NewPoolRepository(pool)
	passportRepo := postgres.NewPassportRepository(pool)

	lender, err := lenderRepo.Create(ctx, borrowLenderInput())
	if err != nil {
		t.Fatalf("create lender: %v", err)
	}

	lenderByWallet, err := lenderRepo.GetByWallet(ctx, lender.WalletAddress)
	if err != nil {
		t.Fatalf("get lender by wallet: %v", err)
	}
	if lenderByWallet.ID != lender.ID {
		t.Fatalf("lender mismatch: got %s want %s", lenderByWallet.ID, lender.ID)
	}

	borrowerHash := []byte{0x10, 0x20, 0x30, 0x40}
	borrower, err := borrowerRepo.Create(ctx, borrowerdomain.CreateInput{
		BorrowerHash: borrowerHash,
		LenderID:     lender.ID,
		CountryCode:  "NG",
		Sector:       "retail",
	})
	if err != nil {
		t.Fatalf("create borrower: %v", err)
	}

	borrowerByHash, err := borrowerRepo.GetByHash(ctx, borrowerHash)
	if err != nil {
		t.Fatalf("get borrower by hash: %v", err)
	}
	if borrowerByHash.ID != borrower.ID {
		t.Fatalf("borrower mismatch: got %s want %s", borrowerByHash.ID, borrower.ID)
	}

	loanHash := []byte{0xaa, 0xbb, 0xcc, 0xdd}
	loanItem, err := loanRepo.Create(ctx, loandomain.CreateInput{
		LoanHash:        loanHash,
		LenderID:        lender.ID,
		BorrowerID:      borrower.ID,
		PrincipalMinor:  500000,
		CurrencyCode:    "NGN",
		InterestRateBPS: 2200,
		StartDate:       time.Now().UTC(),
		MaturityDate:    time.Now().UTC().Add(180 * 24 * time.Hour),
		RiskGrade:       "A",
		Metadata:        []byte(`{"source":"integration-test"}`),
	})
	if err != nil {
		t.Fatalf("create loan: %v", err)
	}

	loanByHash, err := loanRepo.GetByHash(ctx, loanHash)
	if err != nil {
		t.Fatalf("get loan by hash: %v", err)
	}
	if loanByHash.ID != loanItem.ID {
		t.Fatalf("loan mismatch: got %s want %s", loanByHash.ID, loanItem.ID)
	}

	loans, err := loanRepo.List(ctx, loandomain.ListFilter{LenderID: lender.ID, Status: "active", Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("list loans: %v", err)
	}
	if len(loans) != 1 {
		t.Fatalf("expected 1 loan, got %d", len(loans))
	}

	poolItem, err := poolRepo.Create(ctx, pooldomain.CreateInput{
		LenderID:      lender.ID,
		Name:          "Starter Pool",
		PoolTokenAddr: "0x114dac28b091F2d73dF6509E4063D6553eB36fa2",
		TargetAPYBPS:  1600,
		CurrencyCode:  "NGN",
		Status:        "open",
	})
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}

	pools, err := poolRepo.ListByLender(ctx, lender.ID)
	if err != nil {
		t.Fatalf("list pools by lender: %v", err)
	}
	if len(pools) != 1 || pools[0].ID != poolItem.ID {
		t.Fatalf("pool mismatch")
	}

	tokenID := int64(1)
	cache, err := passportRepo.Upsert(ctx, passportdomain.UpsertInput{
		BorrowerID:         borrower.ID,
		TokenID:            &tokenID,
		TotalLoans:         1,
		TotalRepaid:        0,
		TotalDefaulted:     0,
		CumulativeBorrowed: 500000,
		CumulativeRepaid:   0,
		CreditScore:        680,
	})
	if err != nil {
		t.Fatalf("upsert passport cache: %v", err)
	}

	cacheByBorrower, err := passportRepo.GetByBorrowerID(ctx, borrower.ID)
	if err != nil {
		t.Fatalf("get passport cache: %v", err)
	}
	if cacheByBorrower.BorrowerID != cache.BorrowerID || cacheByBorrower.CreditScore != 680 {
		t.Fatalf("passport cache mismatch")
	}
}

func borrowLenderInput() lenderdomain.CreateInput {
	return lenderdomain.CreateInput{
		Name:          "Test Lender",
		CountryCode:   "NG",
		WalletAddress: "0x1111111111111111111111111111111111111111",
		KYCStatus:     "approved",
		Tier:          "starter",
	}
}
