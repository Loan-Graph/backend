package unit

import (
	"context"
	"strings"
	"testing"
	"time"

	borrowerdomain "github.com/loangraph/backend/internal/domain/borrower"
	loandomain "github.com/loangraph/backend/internal/domain/loan"
)

type borrowerRepoMock struct {
	byHash map[string]*borrowerdomain.Entity
	nextID int
}

func (m *borrowerRepoMock) GetByHash(_ context.Context, borrowerHash []byte) (*borrowerdomain.Entity, error) {
	if e, ok := m.byHash[string(borrowerHash)]; ok {
		return e, nil
	}
	return nil, context.Canceled
}

func (m *borrowerRepoMock) Create(_ context.Context, in borrowerdomain.CreateInput) (*borrowerdomain.Entity, error) {
	m.nextID++
	e := &borrowerdomain.Entity{ID: "b-" + string(rune('0'+m.nextID)), BorrowerHash: in.BorrowerHash, LenderID: in.LenderID}
	m.byHash[string(in.BorrowerHash)] = e
	return e, nil
}

type loanRepoMock struct {
	items             []loandomain.Entity
	recordRepaymentID string
	recordAmount      int64
	defaultLoanID     string
}

func (m *loanRepoMock) Create(_ context.Context, in loandomain.CreateInput) (*loandomain.Entity, error) {
	id := "l-" + time.Now().UTC().Format("150405.000000")
	e := loandomain.Entity{ID: id, LoanHash: in.LoanHash, LenderID: in.LenderID, BorrowerID: in.BorrowerID, PrincipalMinor: in.PrincipalMinor, CurrencyCode: in.CurrencyCode, InterestRateBPS: in.InterestRateBPS, MaturityDate: in.MaturityDate}
	m.items = append(m.items, e)
	return &e, nil
}

func (m *loanRepoMock) GetByID(_ context.Context, id string) (*loandomain.Entity, error) {
	for _, item := range m.items {
		if item.ID == id {
			cp := item
			return &cp, nil
		}
	}
	return nil, context.Canceled
}

func (m *loanRepoMock) GetByHash(_ context.Context, loanHash []byte) (*loandomain.Entity, error) {
	for _, item := range m.items {
		if string(item.LoanHash) == string(loanHash) {
			cp := item
			return &cp, nil
		}
	}
	return nil, context.Canceled
}

func (m *loanRepoMock) List(_ context.Context, _ loandomain.ListFilter) ([]loandomain.Entity, error) {
	return m.items, nil
}

func (m *loanRepoMock) SetOnChainSubmission(_ context.Context, _ string, _ string, _ bool) error {
	return nil
}

func (m *loanRepoMock) RecordRepayment(_ context.Context, loanID string, amount int64) error {
	m.recordRepaymentID = loanID
	m.recordAmount = amount
	return nil
}

func (m *loanRepoMock) MarkDefault(_ context.Context, loanID string) error {
	m.defaultLoanID = loanID
	return nil
}

func (m *loanRepoMock) GetPortfolioAnalytics(_ context.Context, lenderID string) (*loandomain.PortfolioAnalytics, error) {
	return &loandomain.PortfolioAnalytics{LenderID: lenderID}, nil
}

func (m *loanRepoMock) ListByBorrower(_ context.Context, _ string, _ int32, _ int32) ([]loandomain.Entity, error) {
	return []loandomain.Entity{}, nil
}

func (m *loanRepoMock) GetPortfolioHealth(_ context.Context, lenderID string) (*loandomain.PortfolioHealth, error) {
	return &loandomain.PortfolioHealth{LenderID: lenderID}, nil
}

func (m *loanRepoMock) GetRepaymentTimeSeriesByLender(_ context.Context, _ string, _ int32) ([]loandomain.PerformancePoint, error) {
	return []loandomain.PerformancePoint{}, nil
}

type outboxRepoMock struct {
	topics []string
}

func (m *outboxRepoMock) Enqueue(_ context.Context, topic string, _ []byte) error {
	m.topics = append(m.topics, topic)
	return nil
}

func TestHashBorrowerIDDeterministic(t *testing.T) {
	h1 := loandomain.HashBorrowerID("smile:NG-BVN:12345", "abc123hash")
	h2 := loandomain.HashBorrowerID("smile:NG-BVN:12345", "abc123hash")
	if string(h1) != string(h2) {
		t.Fatalf("expected deterministic hash")
	}
}

func TestProcessCSVUploadSuccess(t *testing.T) {
	borrowerRepo := &borrowerRepoMock{byHash: map[string]*borrowerdomain.Entity{}}
	loanRepo := &loanRepoMock{}
	outboxRepo := &outboxRepoMock{}

	svc := loandomain.NewService(borrowerRepo, loanRepo, outboxRepo)
	csvInput := strings.NewReader("borrower_kyc_id,gov_id_hash,principal_minor,currency,interest_rate_bps,maturity_date,loan_reference\nsmile:NG-BVN:1,abc123,500000,NGN,2200,2030-12-31T00:00:00Z,LOAN-001\n")

	result, err := svc.ProcessCSVUpload(context.Background(), "lender-1", csvInput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Processed != 1 || len(result.LoanIDs) != 1 || len(result.Errors) != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(outboxRepo.topics) != 1 || outboxRepo.topics[0] != "register_loan" {
		t.Fatalf("expected one register_loan outbox message")
	}
}

func TestProcessCSVUploadValidationError(t *testing.T) {
	borrowerRepo := &borrowerRepoMock{byHash: map[string]*borrowerdomain.Entity{}}
	loanRepo := &loanRepoMock{}
	outboxRepo := &outboxRepoMock{}

	svc := loandomain.NewService(borrowerRepo, loanRepo, outboxRepo)
	csvInput := strings.NewReader("borrower_kyc_id,gov_id_hash,principal_minor,currency,interest_rate_bps,maturity_date,loan_reference\nsmile:NG-BVN:1,abc123,-5,NGN,2200,2030-12-31T00:00:00Z,LOAN-001\n")

	result, err := svc.ProcessCSVUpload(context.Background(), "lender-1", csvInput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Processed != 0 {
		t.Fatalf("expected processed=0, got %d", result.Processed)
	}
	if len(result.Errors) != 1 || result.Errors[0].Field != "principal_minor" {
		t.Fatalf("unexpected errors: %+v", result.Errors)
	}
}

func TestRecordRepaymentQueuesOutbox(t *testing.T) {
	borrowerRepo := &borrowerRepoMock{byHash: map[string]*borrowerdomain.Entity{}}
	loanRepo := &loanRepoMock{}
	outboxRepo := &outboxRepoMock{}
	svc := loandomain.NewService(borrowerRepo, loanRepo, outboxRepo)

	err := svc.RecordRepayment(context.Background(), loandomain.RepaymentInput{
		LoanID:      "loan-1",
		AmountMinor: 1000,
		Currency:    "NGN",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(outboxRepo.topics) != 1 || outboxRepo.topics[0] != "record_repayment" {
		t.Fatalf("expected record_repayment outbox topic")
	}
}

func TestMarkDefaultQueuesOutbox(t *testing.T) {
	borrowerRepo := &borrowerRepoMock{byHash: map[string]*borrowerdomain.Entity{}}
	loanRepo := &loanRepoMock{}
	outboxRepo := &outboxRepoMock{}
	svc := loandomain.NewService(borrowerRepo, loanRepo, outboxRepo)

	err := svc.MarkDefault(context.Background(), loandomain.DefaultInput{
		LoanID:   "loan-1",
		Reason:   "late payment",
		LenderID: "lender-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(outboxRepo.topics) != 1 || outboxRepo.topics[0] != "mark_default" {
		t.Fatalf("expected mark_default outbox topic")
	}
}
