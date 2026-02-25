package loan

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	borrowerdomain "github.com/loangraph/backend/internal/domain/borrower"
	"golang.org/x/crypto/sha3"
)

const (
	outboxTopicRegisterLoan = "register_loan"
	outboxTopicRepayment    = "record_repayment"
	outboxTopicDefault      = "mark_default"
)

var expectedHeaders = []string{
	"borrower_kyc_id",
	"gov_id_hash",
	"principal_minor",
	"currency",
	"interest_rate_bps",
	"maturity_date",
	"loan_reference",
}

type ValidationError struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

type UploadResult struct {
	LoanIDs   []string          `json:"loan_ids"`
	Processed int               `json:"processed"`
	Errors    []ValidationError `json:"errors"`
}

type RepaymentInput struct {
	LoanID      string `json:"loan_id"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

type DefaultInput struct {
	LoanID   string `json:"loan_id"`
	Reason   string `json:"reason"`
	LenderID string `json:"lender_id"`
}

type BorrowerRepository interface {
	Create(ctx context.Context, in borrowerdomain.CreateInput) (*borrowerdomain.Entity, error)
	GetByHash(ctx context.Context, borrowerHash []byte) (*borrowerdomain.Entity, error)
}

type OutboxRepository interface {
	Enqueue(ctx context.Context, topic string, payload []byte) error
}

type Service struct {
	borrowerRepo BorrowerRepository
	loanRepo     Repository
	outboxRepo   OutboxRepository
	now          func() time.Time
}

func NewService(borrowerRepo BorrowerRepository, loanRepo Repository, outboxRepo OutboxRepository) *Service {
	return &Service{
		borrowerRepo: borrowerRepo,
		loanRepo:     loanRepo,
		outboxRepo:   outboxRepo,
		now:          func() time.Time { return time.Now().UTC() },
	}
}

func HashBorrowerID(kycProviderID, govIDHash string) []byte {
	input := fmt.Sprintf("%s:%s", strings.TrimSpace(kycProviderID), strings.TrimSpace(govIDHash))
	h := sha3.NewLegacyKeccak256()
	_, _ = h.Write([]byte(input))
	out := h.Sum(nil)
	return out
}

func (s *Service) ProcessCSVUpload(ctx context.Context, lenderID string, csvReader io.Reader) (*UploadResult, error) {
	reader := csv.NewReader(csvReader)
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("invalid_csv")
	}
	if len(rows) < 2 {
		return &UploadResult{LoanIDs: []string{}, Processed: 0, Errors: []ValidationError{{Row: 1, Field: "file", Message: "csv must include header and at least one data row"}}}, nil
	}

	if err := validateHeader(rows[0]); err != nil {
		return &UploadResult{LoanIDs: []string{}, Processed: 0, Errors: []ValidationError{{Row: 1, Field: "header", Message: err.Error()}}}, nil
	}

	result := &UploadResult{LoanIDs: []string{}, Errors: []ValidationError{}}
	for i := 1; i < len(rows); i++ {
		rowNum := i + 1
		record := rows[i]

		parsed, validationErr := parseRow(record)
		if validationErr != nil {
			result.Errors = append(result.Errors, ValidationError{Row: rowNum, Field: validationErr.Field, Message: validationErr.Message})
			continue
		}

		borrowerHash := HashBorrowerID(parsed.BorrowerKYCID, parsed.GovIDHash)
		borrowerEntity, err := s.borrowerRepo.GetByHash(ctx, borrowerHash)
		if err != nil {
			borrowerEntity, err = s.borrowerRepo.Create(ctx, borrowerdomain.CreateInput{
				BorrowerHash: borrowerHash,
				LenderID:     lenderID,
				CountryCode:  "NG",
				Sector:       "",
			})
			if err != nil {
				return nil, err
			}
		}

		loanHash := hashLoanID(lenderID, parsed.LoanReference)
		meta, _ := json.Marshal(map[string]any{"loan_reference": parsed.LoanReference, "borrower_kyc_id": parsed.BorrowerKYCID})

		created, err := s.loanRepo.Create(ctx, CreateInput{
			LoanHash:        loanHash,
			LenderID:        lenderID,
			BorrowerID:      borrowerEntity.ID,
			PrincipalMinor:  parsed.PrincipalMinor,
			CurrencyCode:    parsed.Currency,
			InterestRateBPS: parsed.InterestRateBPS,
			StartDate:       s.now(),
			MaturityDate:    parsed.MaturityDate,
			RiskGrade:       "",
			Metadata:        meta,
		})
		if err != nil {
			return nil, err
		}

		payload, _ := json.Marshal(map[string]any{"loan_id": created.ID})
		if err := s.outboxRepo.Enqueue(ctx, outboxTopicRegisterLoan, payload); err != nil {
			return nil, err
		}

		result.LoanIDs = append(result.LoanIDs, created.ID)
		result.Processed++
	}

	return result, nil
}

func (s *Service) ListLoans(ctx context.Context, filter ListFilter) ([]Entity, error) {
	return s.loanRepo.List(ctx, filter)
}

func (s *Service) GetLoan(ctx context.Context, loanID string) (*Entity, error) {
	return s.loanRepo.GetByID(ctx, loanID)
}

func (s *Service) RecordRepayment(ctx context.Context, in RepaymentInput) error {
	if strings.TrimSpace(in.LoanID) == "" || in.AmountMinor <= 0 || len(strings.TrimSpace(in.Currency)) != 3 {
		return fmt.Errorf("invalid_repayment_input")
	}
	if err := s.loanRepo.RecordRepayment(ctx, in.LoanID, in.AmountMinor); err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]any{
		"loan_id":      in.LoanID,
		"amount_minor": in.AmountMinor,
		"currency":     strings.ToUpper(strings.TrimSpace(in.Currency)),
	})
	return s.outboxRepo.Enqueue(ctx, outboxTopicRepayment, payload)
}

func (s *Service) MarkDefault(ctx context.Context, in DefaultInput) error {
	if strings.TrimSpace(in.LoanID) == "" {
		return fmt.Errorf("invalid_default_input")
	}
	if err := s.loanRepo.MarkDefault(ctx, in.LoanID); err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]any{
		"loan_id":   in.LoanID,
		"reason":    strings.TrimSpace(in.Reason),
		"lender_id": strings.TrimSpace(in.LenderID),
	})
	return s.outboxRepo.Enqueue(ctx, outboxTopicDefault, payload)
}

func (s *Service) PortfolioAnalytics(ctx context.Context, lenderID string) (*PortfolioAnalytics, error) {
	if strings.TrimSpace(lenderID) == "" {
		return nil, fmt.Errorf("missing_lender_id")
	}
	return s.loanRepo.GetPortfolioAnalytics(ctx, lenderID)
}

type rowValidationError struct {
	Field   string
	Message string
}

type parsedRow struct {
	BorrowerKYCID   string
	GovIDHash       string
	PrincipalMinor  int64
	Currency        string
	InterestRateBPS int32
	MaturityDate    time.Time
	LoanReference   string
}

func validateHeader(header []string) error {
	if len(header) < len(expectedHeaders) {
		return fmt.Errorf("invalid column count")
	}
	for i, expected := range expectedHeaders {
		if strings.TrimSpace(strings.ToLower(header[i])) != expected {
			return fmt.Errorf("expected header %q at position %d", expected, i+1)
		}
	}
	return nil
}

func parseRow(row []string) (*parsedRow, *rowValidationError) {
	if len(row) < len(expectedHeaders) {
		return nil, &rowValidationError{Field: "row", Message: "invalid column count"}
	}

	borrowerKYCID := strings.TrimSpace(row[0])
	if borrowerKYCID == "" {
		return nil, &rowValidationError{Field: "borrower_kyc_id", Message: "required"}
	}

	govIDHash := strings.TrimSpace(row[1])
	if govIDHash == "" {
		return nil, &rowValidationError{Field: "gov_id_hash", Message: "required"}
	}

	principalMinor, err := strconv.ParseInt(strings.TrimSpace(row[2]), 10, 64)
	if err != nil || principalMinor <= 0 {
		return nil, &rowValidationError{Field: "principal_minor", Message: "must be a positive integer"}
	}

	currency := strings.ToUpper(strings.TrimSpace(row[3]))
	if len(currency) != 3 {
		return nil, &rowValidationError{Field: "currency", Message: "must be 3-letter code"}
	}

	interestRateBPS64, err := strconv.ParseInt(strings.TrimSpace(row[4]), 10, 32)
	if err != nil || interestRateBPS64 < 0 {
		return nil, &rowValidationError{Field: "interest_rate_bps", Message: "must be a non-negative integer"}
	}

	maturityDate, err := time.Parse(time.RFC3339, strings.TrimSpace(row[5]))
	if err != nil {
		return nil, &rowValidationError{Field: "maturity_date", Message: "must be RFC3339"}
	}

	loanReference := strings.TrimSpace(row[6])
	if loanReference == "" {
		return nil, &rowValidationError{Field: "loan_reference", Message: "required"}
	}

	return &parsedRow{
		BorrowerKYCID:   borrowerKYCID,
		GovIDHash:       govIDHash,
		PrincipalMinor:  principalMinor,
		Currency:        currency,
		InterestRateBPS: int32(interestRateBPS64),
		MaturityDate:    maturityDate,
		LoanReference:   loanReference,
	}, nil
}

func hashLoanID(lenderID, loanReference string) []byte {
	input := fmt.Sprintf("%s:%s", strings.TrimSpace(lenderID), strings.TrimSpace(loanReference))
	h := sha3.NewLegacyKeccak256()
	_, _ = h.Write([]byte(input))
	return h.Sum(nil)
}
