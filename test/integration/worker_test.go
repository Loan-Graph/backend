package integration

import (
	"context"
	"testing"
	"time"

	"github.com/loangraph/backend/internal/blockchain"
	borrowerdomain "github.com/loangraph/backend/internal/domain/borrower"
	lenderdomain "github.com/loangraph/backend/internal/domain/lender"
	loandomain "github.com/loangraph/backend/internal/domain/loan"
	"github.com/loangraph/backend/internal/jobs"
	postgresrepo "github.com/loangraph/backend/internal/repository/postgres"
	"github.com/loangraph/backend/test/integration/testutil"
)

func TestWorkerProcessesRegisterLoanOutboxJob(t *testing.T) {
	pool := testutil.NewTestPool(t)
	defer pool.Close()
	testutil.ApplyMigrations(t, pool)
	testutil.ResetTables(t, pool)

	ctx := context.Background()
	lenderRepo := postgresrepo.NewLenderRepository(pool)
	borrowerRepo := postgresrepo.NewBorrowerRepository(pool)
	loanRepo := postgresrepo.NewLoanRepository(pool)
	outboxRepo := postgresrepo.NewOutboxRepository(pool)

	lender, err := lenderRepo.Create(ctx, lenderdomain.CreateInput{
		Name:          "Worker Lender",
		CountryCode:   "NG",
		WalletAddress: "0x3333333333333333333333333333333333333333",
		KYCStatus:     "approved",
		Tier:          "starter",
	})
	if err != nil {
		t.Fatalf("create lender: %v", err)
	}

	borrower, err := borrowerRepo.Create(ctx, borrowerdomain.CreateInput{
		BorrowerHash: []byte{0x44, 0x55},
		LenderID:     lender.ID,
		CountryCode:  "NG",
		Sector:       "retail",
	})
	if err != nil {
		t.Fatalf("create borrower: %v", err)
	}

	loanItem, err := loanRepo.Create(ctx, loandomain.CreateInput{
		LoanHash:        []byte{0xaa, 0xbb},
		LenderID:        lender.ID,
		BorrowerID:      borrower.ID,
		PrincipalMinor:  100000,
		CurrencyCode:    "NGN",
		InterestRateBPS: 2200,
		StartDate:       time.Now().UTC(),
		MaturityDate:    time.Now().UTC().Add(180 * 24 * time.Hour),
		RiskGrade:       "A",
		Metadata:        []byte(`{"source":"worker-test"}`),
	})
	if err != nil {
		t.Fatalf("create loan: %v", err)
	}

	if err := outboxRepo.Enqueue(ctx, "register_loan", []byte(`{"loan_id":"`+loanItem.ID+`"}`)); err != nil {
		t.Fatalf("enqueue outbox: %v", err)
	}

	worker := jobs.NewWorker(outboxRepo, loanRepo, blockchain.NewStubWriter())
	if err := worker.RunOnce(ctx, 10); err != nil {
		t.Fatalf("run worker: %v", err)
	}

	updatedLoan, err := loanRepo.GetByID(ctx, loanItem.ID)
	if err != nil {
		t.Fatalf("get loan: %v", err)
	}
	if updatedLoan.OnChainTX == "" {
		t.Fatalf("expected on_chain_tx to be set")
	}

	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM outbox_jobs ORDER BY id DESC LIMIT 1`).Scan(&status); err != nil {
		t.Fatalf("query outbox status: %v", err)
	}
	if status != "done" {
		t.Fatalf("expected outbox status done, got %s", status)
	}
}
