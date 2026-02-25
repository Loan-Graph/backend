package postgres

import (
	"context"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/loangraph/backend/internal/domain/loan"
)

type LoanRepository struct {
	pool *pgxpool.Pool
}

func NewLoanRepository(pool *pgxpool.Pool) *LoanRepository {
	return &LoanRepository{pool: pool}
}

func (r *LoanRepository) Create(ctx context.Context, in loan.CreateInput) (*loan.Entity, error) {
	q := `
INSERT INTO loans (
  loan_hash, lender_id, borrower_id, principal_minor, currency_code,
  interest_rate_bps, start_date, maturity_date, risk_grade, metadata
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
RETURNING id, loan_hash, lender_id, borrower_id, principal_minor, currency_code,
          interest_rate_bps, start_date, maturity_date, amount_repaid_minor,
          status, on_chain_tx, on_chain_confirmed, risk_grade, metadata, created_at, updated_at
`
	out := &loan.Entity{}
	err := r.pool.QueryRow(ctx, q,
		in.LoanHash, in.LenderID, in.BorrowerID, in.PrincipalMinor, in.CurrencyCode,
		in.InterestRateBPS, in.StartDate, in.MaturityDate, in.RiskGrade, in.Metadata,
	).Scan(
		&out.ID, &out.LoanHash, &out.LenderID, &out.BorrowerID, &out.PrincipalMinor, &out.CurrencyCode,
		&out.InterestRateBPS, &out.StartDate, &out.MaturityDate, &out.AmountRepaid,
		&out.Status, &out.OnChainTX, &out.OnChainConfirmed, &out.RiskGrade, &out.Metadata, &out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *LoanRepository) GetByID(ctx context.Context, id string) (*loan.Entity, error) {
	q := `
SELECT id, loan_hash, lender_id, borrower_id, principal_minor, currency_code,
       interest_rate_bps, start_date, maturity_date, amount_repaid_minor,
       status, on_chain_tx, on_chain_confirmed, risk_grade, metadata, created_at, updated_at
FROM loans WHERE id = $1
`
	out := &loan.Entity{}
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&out.ID, &out.LoanHash, &out.LenderID, &out.BorrowerID, &out.PrincipalMinor, &out.CurrencyCode,
		&out.InterestRateBPS, &out.StartDate, &out.MaturityDate, &out.AmountRepaid,
		&out.Status, &out.OnChainTX, &out.OnChainConfirmed, &out.RiskGrade, &out.Metadata, &out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *LoanRepository) GetByHash(ctx context.Context, loanHash []byte) (*loan.Entity, error) {
	q := `
SELECT id, loan_hash, lender_id, borrower_id, principal_minor, currency_code,
       interest_rate_bps, start_date, maturity_date, amount_repaid_minor,
       status, on_chain_tx, on_chain_confirmed, risk_grade, metadata, created_at, updated_at
FROM loans WHERE loan_hash = $1
`
	out := &loan.Entity{}
	err := r.pool.QueryRow(ctx, q, loanHash).Scan(
		&out.ID, &out.LoanHash, &out.LenderID, &out.BorrowerID, &out.PrincipalMinor, &out.CurrencyCode,
		&out.InterestRateBPS, &out.StartDate, &out.MaturityDate, &out.AmountRepaid,
		&out.Status, &out.OnChainTX, &out.OnChainConfirmed, &out.RiskGrade, &out.Metadata, &out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *LoanRepository) List(ctx context.Context, f loan.ListFilter) ([]loan.Entity, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Offset < 0 {
		f.Offset = 0
	}

	builder := strings.Builder{}
	builder.WriteString(`
SELECT id, loan_hash, lender_id, borrower_id, principal_minor, currency_code,
       interest_rate_bps, start_date, maturity_date, amount_repaid_minor,
       status, on_chain_tx, on_chain_confirmed, risk_grade, metadata, created_at, updated_at
FROM loans
WHERE 1=1`)

	args := []any{}
	argPos := 1
	if strings.TrimSpace(f.LenderID) != "" {
		builder.WriteString(" AND lender_id = $")
		builder.WriteString(strconv.Itoa(argPos))
		args = append(args, f.LenderID)
		argPos++
	}
	if strings.TrimSpace(f.Status) != "" {
		builder.WriteString(" AND status = $")
		builder.WriteString(strconv.Itoa(argPos))
		args = append(args, f.Status)
		argPos++
	}
	if strings.TrimSpace(f.RiskGrade) != "" {
		builder.WriteString(" AND risk_grade = $")
		builder.WriteString(strconv.Itoa(argPos))
		args = append(args, f.RiskGrade)
		argPos++
	}
	builder.WriteString(" ORDER BY created_at DESC")
	builder.WriteString(" LIMIT $")
	builder.WriteString(strconv.Itoa(argPos))
	args = append(args, f.Limit)
	argPos++
	builder.WriteString(" OFFSET $")
	builder.WriteString(strconv.Itoa(argPos))
	args = append(args, f.Offset)

	rows, err := r.pool.Query(ctx, builder.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]loan.Entity, 0)
	for rows.Next() {
		var item loan.Entity
		if err := rows.Scan(
			&item.ID, &item.LoanHash, &item.LenderID, &item.BorrowerID, &item.PrincipalMinor, &item.CurrencyCode,
			&item.InterestRateBPS, &item.StartDate, &item.MaturityDate, &item.AmountRepaid,
			&item.Status, &item.OnChainTX, &item.OnChainConfirmed, &item.RiskGrade, &item.Metadata, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *LoanRepository) SetOnChainSubmission(ctx context.Context, loanID, txHash string, confirmed bool) error {
	q := `UPDATE loans SET on_chain_tx = $2, on_chain_confirmed = $3, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, q, loanID, txHash, confirmed)
	return err
}

func (r *LoanRepository) RecordRepayment(ctx context.Context, loanID string, amountMinor int64) error {
	q := `
UPDATE loans
SET amount_repaid_minor = amount_repaid_minor + $2,
    status = CASE WHEN (amount_repaid_minor + $2) >= principal_minor THEN 'repaid' ELSE status END,
    updated_at = NOW()
WHERE id = $1 AND status != 'defaulted'
`
	_, err := r.pool.Exec(ctx, q, loanID, amountMinor)
	return err
}

func (r *LoanRepository) MarkDefault(ctx context.Context, loanID string) error {
	q := `UPDATE loans SET status = 'defaulted', updated_at = NOW() WHERE id = $1 AND status = 'active'`
	_, err := r.pool.Exec(ctx, q, loanID)
	return err
}

func (r *LoanRepository) GetPortfolioAnalytics(ctx context.Context, lenderID string) (*loan.PortfolioAnalytics, error) {
	q := `
SELECT
  COUNT(*)::bigint AS total_loans,
  COUNT(*) FILTER (WHERE status = 'active')::bigint AS active_loans,
  COUNT(*) FILTER (WHERE status = 'repaid')::bigint AS repaid_loans,
  COUNT(*) FILTER (WHERE status = 'defaulted')::bigint AS defaulted_loans,
  COALESCE(SUM(principal_minor), 0)::bigint AS total_principal_minor,
  COALESCE(SUM(amount_repaid_minor), 0)::bigint AS total_repaid_minor
FROM loans
WHERE lender_id = $1
`
	out := &loan.PortfolioAnalytics{LenderID: lenderID}
	err := r.pool.QueryRow(ctx, q, lenderID).Scan(
		&out.TotalLoans,
		&out.ActiveLoans,
		&out.RepaidLoans,
		&out.DefaultedLoans,
		&out.TotalPrincipalMinor,
		&out.TotalRepaidMinor,
	)
	if err != nil {
		return nil, err
	}
	if out.TotalPrincipalMinor > 0 {
		out.RepaymentRatePercent = (float64(out.TotalRepaidMinor) / float64(out.TotalPrincipalMinor)) * 100
	}
	return out, nil
}

func (r *LoanRepository) ListByBorrower(ctx context.Context, borrowerID string, limit, offset int32) ([]loan.Entity, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	q := `
SELECT id, loan_hash, lender_id, borrower_id, principal_minor, currency_code,
       interest_rate_bps, start_date, maturity_date, amount_repaid_minor,
       status, on_chain_tx, on_chain_confirmed, risk_grade, metadata, created_at, updated_at
FROM loans
WHERE borrower_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3
`
	rows, err := r.pool.Query(ctx, q, borrowerID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]loan.Entity, 0)
	for rows.Next() {
		var item loan.Entity
		if err := rows.Scan(
			&item.ID, &item.LoanHash, &item.LenderID, &item.BorrowerID, &item.PrincipalMinor, &item.CurrencyCode,
			&item.InterestRateBPS, &item.StartDate, &item.MaturityDate, &item.AmountRepaid,
			&item.Status, &item.OnChainTX, &item.OnChainConfirmed, &item.RiskGrade, &item.Metadata, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *LoanRepository) GetPortfolioHealth(ctx context.Context, lenderID string) (*loan.PortfolioHealth, error) {
	out := &loan.PortfolioHealth{
		LenderID: lenderID,
		ScoreBands: []loan.ScoreBand{
			{Label: "300-549", Count: 0},
			{Label: "550-699", Count: 0},
			{Label: "700-850", Count: 0},
		},
	}

	qSummary := `
SELECT
  COUNT(DISTINCT b.id)::bigint AS unique_borrowers,
  COALESCE(AVG(pc.credit_score), 0)::float8 AS average_score
FROM borrowers b
LEFT JOIN passport_cache pc ON pc.borrower_id = b.id
WHERE b.lender_id = $1
`
	if err := r.pool.QueryRow(ctx, qSummary, lenderID).Scan(&out.UniqueBorrowers, &out.AverageScore); err != nil {
		return nil, err
	}

	qBands := `
SELECT
  SUM(CASE WHEN pc.credit_score BETWEEN 300 AND 549 THEN 1 ELSE 0 END)::bigint AS band1,
  SUM(CASE WHEN pc.credit_score BETWEEN 550 AND 699 THEN 1 ELSE 0 END)::bigint AS band2,
  SUM(CASE WHEN pc.credit_score BETWEEN 700 AND 850 THEN 1 ELSE 0 END)::bigint AS band3
FROM borrowers b
LEFT JOIN passport_cache pc ON pc.borrower_id = b.id
WHERE b.lender_id = $1
`
	var b1, b2, b3 int64
	if err := r.pool.QueryRow(ctx, qBands, lenderID).Scan(&b1, &b2, &b3); err != nil {
		return nil, err
	}
	out.ScoreBands[0].Count = b1
	out.ScoreBands[1].Count = b2
	out.ScoreBands[2].Count = b3
	return out, nil
}
