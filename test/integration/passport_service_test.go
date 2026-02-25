package integration

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	borrowerdomain "github.com/loangraph/backend/internal/domain/borrower"
	lenderdomain "github.com/loangraph/backend/internal/domain/lender"
	loandomain "github.com/loangraph/backend/internal/domain/loan"
	passportdomain "github.com/loangraph/backend/internal/domain/passport"
	postgresrepo "github.com/loangraph/backend/internal/repository/postgres"
	"github.com/loangraph/backend/test/integration/testutil"
)

func TestPassportServiceWithPostgres(t *testing.T) {
	pool := testutil.NewTestPool(t)
	defer pool.Close()
	testutil.ApplyMigrations(t, pool)
	testutil.ResetTables(t, pool)

	ctx := context.Background()
	lenderRepo := postgresrepo.NewLenderRepository(pool)
	borrowerRepo := postgresrepo.NewBorrowerRepository(pool)
	loanRepo := postgresrepo.NewLoanRepository(pool)
	passportRepo := postgresrepo.NewPassportRepository(pool)

	lender, err := lenderRepo.Create(ctx, lenderdomain.CreateInput{
		Name:          "Passport Lender",
		CountryCode:   "NG",
		WalletAddress: "0x6666666666666666666666666666666666666666",
		KYCStatus:     "approved",
		Tier:          "starter",
	})
	if err != nil {
		t.Fatalf("create lender: %v", err)
	}

	borrowerHash := []byte{0xde, 0xad, 0xbe, 0xef}
	borrower, err := borrowerRepo.Create(ctx, borrowerdomain.CreateInput{BorrowerHash: borrowerHash, LenderID: lender.ID, CountryCode: "NG"})
	if err != nil {
		t.Fatalf("create borrower: %v", err)
	}

	_, err = loanRepo.Create(ctx, loandomain.CreateInput{
		LoanHash:        []byte{0x12, 0x34},
		LenderID:        lender.ID,
		BorrowerID:      borrower.ID,
		PrincipalMinor:  100000,
		CurrencyCode:    "NGN",
		InterestRateBPS: 1800,
		StartDate:       time.Now().UTC(),
		MaturityDate:    time.Now().UTC().Add(120 * 24 * time.Hour),
		RiskGrade:       "A",
		Metadata:        []byte(`{"source":"passport-test"}`),
	})
	if err != nil {
		t.Fatalf("create loan: %v", err)
	}

	_, err = passportRepo.Upsert(ctx, passportdomain.UpsertInput{
		BorrowerID:         borrower.ID,
		TotalLoans:         1,
		TotalRepaid:        0,
		TotalDefaulted:     0,
		CumulativeBorrowed: 100000,
		CumulativeRepaid:   0,
		CreditScore:        640,
	})
	if err != nil {
		t.Fatalf("upsert passport: %v", err)
	}

	svc := passportdomain.NewService(borrowerRepo, passportRepo, loanRepo)
	hashHex := "0x" + hex.EncodeToString(borrowerHash)

	cache, err := svc.GetPassportByBorrowerHash(ctx, hashHex)
	if err != nil {
		t.Fatalf("get passport by hash: %v", err)
	}
	if cache.CreditScore != 640 {
		t.Fatalf("expected score 640, got %d", cache.CreditScore)
	}

	history, err := svc.GetHistoryByBorrowerHash(ctx, hashHex, 10, 0)
	if err != nil {
		t.Fatalf("get history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 history item, got %d", len(history))
	}

	health, err := svc.GetPortfolioHealth(ctx, lender.ID)
	if err != nil {
		t.Fatalf("get portfolio health: %v", err)
	}
	if health.UniqueBorrowers != 1 {
		t.Fatalf("expected unique borrowers=1, got %d", health.UniqueBorrowers)
	}
}
