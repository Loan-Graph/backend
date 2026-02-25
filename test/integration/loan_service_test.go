package integration

import (
	"context"
	"strings"
	"testing"

	lenderdomain "github.com/loangraph/backend/internal/domain/lender"
	loandomain "github.com/loangraph/backend/internal/domain/loan"
	postgresrepo "github.com/loangraph/backend/internal/repository/postgres"
	"github.com/loangraph/backend/test/integration/testutil"
)

func TestLoanServiceProcessCSVUploadWithPostgres(t *testing.T) {
	pool := testutil.NewTestPool(t)
	defer pool.Close()
	testutil.ApplyMigrations(t, pool)
	testutil.ResetTables(t, pool)

	ctx := context.Background()
	lenderRepo := postgresrepo.NewLenderRepository(pool)
	borrowerRepo := postgresrepo.NewBorrowerRepository(pool)
	loanRepo := postgresrepo.NewLoanRepository(pool)
	outboxRepo := postgresrepo.NewOutboxRepository(pool)
	loanSvc := loandomain.NewService(borrowerRepo, loanRepo, outboxRepo)

	lender, err := lenderRepo.Create(ctx, lenderdomain.CreateInput{
		Name:          "CSV Lender",
		CountryCode:   "NG",
		WalletAddress: "0x2222222222222222222222222222222222222222",
		KYCStatus:     "approved",
		Tier:          "starter",
	})
	if err != nil {
		t.Fatalf("create lender: %v", err)
	}

	csvInput := strings.NewReader("borrower_kyc_id,gov_id_hash,principal_minor,currency,interest_rate_bps,maturity_date,loan_reference\nsmile:NG-BVN:1,abc123,500000,NGN,2200,2030-12-31T00:00:00Z,LOAN-001\n")
	res, err := loanSvc.ProcessCSVUpload(ctx, lender.ID, csvInput)
	if err != nil {
		t.Fatalf("process upload: %v", err)
	}
	if res.Processed != 1 || len(res.LoanIDs) != 1 || len(res.Errors) != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}

	var outboxCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_jobs WHERE topic = 'register_loan'`).Scan(&outboxCount); err != nil {
		t.Fatalf("count outbox jobs: %v", err)
	}
	if outboxCount != 1 {
		t.Fatalf("expected 1 outbox job, got %d", outboxCount)
	}
}

func TestLoanServiceLifecycleWithPostgres(t *testing.T) {
	pool := testutil.NewTestPool(t)
	defer pool.Close()
	testutil.ApplyMigrations(t, pool)
	testutil.ResetTables(t, pool)

	ctx := context.Background()
	lenderRepo := postgresrepo.NewLenderRepository(pool)
	borrowerRepo := postgresrepo.NewBorrowerRepository(pool)
	loanRepo := postgresrepo.NewLoanRepository(pool)
	outboxRepo := postgresrepo.NewOutboxRepository(pool)
	loanSvc := loandomain.NewService(borrowerRepo, loanRepo, outboxRepo)

	lender, err := lenderRepo.Create(ctx, lenderdomain.CreateInput{
		Name:          "Lifecycle Lender",
		CountryCode:   "NG",
		WalletAddress: "0x4444444444444444444444444444444444444444",
		KYCStatus:     "approved",
		Tier:          "starter",
	})
	if err != nil {
		t.Fatalf("create lender: %v", err)
	}

	csvInput := strings.NewReader("borrower_kyc_id,gov_id_hash,principal_minor,currency,interest_rate_bps,maturity_date,loan_reference\nsmile:NG-BVN:2,def456,200000,NGN,1800,2030-12-31T00:00:00Z,LOAN-002\n")
	res, err := loanSvc.ProcessCSVUpload(ctx, lender.ID, csvInput)
	if err != nil || len(res.LoanIDs) != 1 {
		t.Fatalf("process upload failed: %+v err=%v", res, err)
	}
	loanID := res.LoanIDs[0]

	if err := loanSvc.RecordRepayment(ctx, loandomain.RepaymentInput{LoanID: loanID, AmountMinor: 50000, Currency: "NGN"}); err != nil {
		t.Fatalf("record repayment: %v", err)
	}
	if err := loanSvc.MarkDefault(ctx, loandomain.DefaultInput{LoanID: loanID, Reason: "test", LenderID: lender.ID}); err != nil {
		t.Fatalf("mark default: %v", err)
	}

	analytics, err := loanSvc.PortfolioAnalytics(ctx, lender.ID)
	if err != nil {
		t.Fatalf("analytics: %v", err)
	}
	if analytics.TotalLoans != 1 {
		t.Fatalf("expected 1 total loan, got %d", analytics.TotalLoans)
	}

	var repayJobs int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_jobs WHERE topic = 'record_repayment'`).Scan(&repayJobs); err != nil {
		t.Fatalf("count repayment jobs: %v", err)
	}
	if repayJobs != 1 {
		t.Fatalf("expected 1 repayment outbox job, got %d", repayJobs)
	}

	var defaultJobs int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox_jobs WHERE topic = 'mark_default'`).Scan(&defaultJobs); err != nil {
		t.Fatalf("count default jobs: %v", err)
	}
	if defaultJobs != 1 {
		t.Fatalf("expected 1 default outbox job, got %d", defaultJobs)
	}
}
