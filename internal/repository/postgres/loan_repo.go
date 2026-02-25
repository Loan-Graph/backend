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
