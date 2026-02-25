package integration

import (
	"context"
	"testing"
	"time"

	borrowerdomain "github.com/loangraph/backend/internal/domain/borrower"
	lenderdomain "github.com/loangraph/backend/internal/domain/lender"
	loandomain "github.com/loangraph/backend/internal/domain/loan"
	"github.com/loangraph/backend/internal/indexer"
	postgresrepo "github.com/loangraph/backend/internal/repository/postgres"
	"github.com/loangraph/backend/test/integration/testutil"
)

func TestIndexerAppliesEventProjections(t *testing.T) {
	pool := testutil.NewTestPool(t)
	defer pool.Close()
	testutil.ApplyMigrations(t, pool)
	testutil.ResetTables(t, pool)

	ctx := context.Background()
	lenderRepo := postgresrepo.NewLenderRepository(pool)
	borrowerRepo := postgresrepo.NewBorrowerRepository(pool)
	loanRepo := postgresrepo.NewLoanRepository(pool)
	idxRepo := postgresrepo.NewIndexerRepository(pool)

	lender, err := lenderRepo.Create(ctx, lenderdomain.CreateInput{
		Name:          "Indexer Lender",
		CountryCode:   "NG",
		WalletAddress: "0x5555555555555555555555555555555555555555",
		KYCStatus:     "approved",
		Tier:          "starter",
	})
	if err != nil {
		t.Fatalf("create lender: %v", err)
	}

	borrower, err := borrowerRepo.Create(ctx, borrowerdomain.CreateInput{
		BorrowerHash: []byte{0x66, 0x77},
		LenderID:     lender.ID,
		CountryCode:  "NG",
		Sector:       "retail",
	})
	if err != nil {
		t.Fatalf("create borrower: %v", err)
	}

	loanItem, err := loanRepo.Create(ctx, loandomain.CreateInput{
		LoanHash:        []byte{0xa1, 0xb2},
		LenderID:        lender.ID,
		BorrowerID:      borrower.ID,
		PrincipalMinor:  100000,
		CurrencyCode:    "NGN",
		InterestRateBPS: 2000,
		StartDate:       time.Now().UTC(),
		MaturityDate:    time.Now().UTC().Add(90 * 24 * time.Hour),
		RiskGrade:       "B",
		Metadata:        []byte(`{"source":"indexer-test"}`),
	})
	if err != nil {
		t.Fatalf("create loan: %v", err)
	}

	ins := `
INSERT INTO chain_events (contract_addr, event_name, tx_hash, block_number, log_index, raw_data, processed)
VALUES
  ($1, 'LoanRegistered', '0xabc1', 1, 1, $2::jsonb, FALSE),
  ($1, 'RepaymentRecorded', '0xabc2', 2, 2, $3::jsonb, FALSE),
  ($1, 'LoanDefaulted', '0xabc3', 3, 3, $4::jsonb, FALSE)
`
	if _, err := pool.Exec(ctx, ins,
		"0x3c20Fd0B57711a199776B53C2F24385563d1670F",
		`{"loan_id":"`+loanItem.ID+`"}`,
		`{"loan_id":"`+loanItem.ID+`","amount_minor":20000}`,
		`{"loan_id":"`+loanItem.ID+`"}`,
	); err != nil {
		t.Fatalf("insert chain events: %v", err)
	}

	svc := indexer.NewService(idxRepo, idxRepo)
	if err := svc.RunOnce(ctx, 10); err != nil {
		t.Fatalf("indexer run once: %v", err)
	}

	updatedLoan, err := loanRepo.GetByID(ctx, loanItem.ID)
	if err != nil {
		t.Fatalf("get updated loan: %v", err)
	}
	if updatedLoan.OnChainTX != "0xabc1" {
		t.Fatalf("expected on_chain_tx=0xabc1, got %s", updatedLoan.OnChainTX)
	}
	if updatedLoan.Status != "defaulted" {
		t.Fatalf("expected defaulted status, got %s", updatedLoan.Status)
	}
	if updatedLoan.AmountRepaid < 20000 {
		t.Fatalf("expected amount repaid >= 20000, got %d", updatedLoan.AmountRepaid)
	}

	var processedCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM chain_events WHERE processed = TRUE`).Scan(&processedCount); err != nil {
		t.Fatalf("count processed events: %v", err)
	}
	if processedCount != 3 {
		t.Fatalf("expected 3 processed events, got %d", processedCount)
	}

	var cacheCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM passport_cache WHERE borrower_id = $1`, borrower.ID).Scan(&cacheCount); err != nil {
		t.Fatalf("count passport cache: %v", err)
	}
	if cacheCount != 1 {
		t.Fatalf("expected passport cache row")
	}
}
