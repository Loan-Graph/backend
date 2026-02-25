package passport

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	borrowerdomain "github.com/loangraph/backend/internal/domain/borrower"
	loandomain "github.com/loangraph/backend/internal/domain/loan"
)

type BorrowerRepository interface {
	GetByHash(ctx context.Context, borrowerHash []byte) (*borrowerdomain.Entity, error)
}

type LoanRepository interface {
	ListByBorrower(ctx context.Context, borrowerID string, limit, offset int32) ([]loandomain.Entity, error)
	GetPortfolioHealth(ctx context.Context, lenderID string) (*loandomain.PortfolioHealth, error)
}

type Service struct {
	borrowerRepo BorrowerRepository
	passportRepo Repository
	loanRepo     LoanRepository
}

func NewService(borrowerRepo BorrowerRepository, passportRepo Repository, loanRepo LoanRepository) *Service {
	return &Service{
		borrowerRepo: borrowerRepo,
		passportRepo: passportRepo,
		loanRepo:     loanRepo,
	}
}

func (s *Service) GetPassportByBorrowerHash(ctx context.Context, borrowerHashHex string) (*Cache, error) {
	borrowerHash, err := decodeBorrowerHash(borrowerHashHex)
	if err != nil {
		return nil, err
	}
	borrower, err := s.borrowerRepo.GetByHash(ctx, borrowerHash)
	if err != nil {
		return nil, err
	}
	return s.passportRepo.GetByBorrowerID(ctx, borrower.ID)
}

func (s *Service) GetHistoryByBorrowerHash(ctx context.Context, borrowerHashHex string, limit, offset int32) ([]loandomain.Entity, error) {
	borrowerHash, err := decodeBorrowerHash(borrowerHashHex)
	if err != nil {
		return nil, err
	}
	borrower, err := s.borrowerRepo.GetByHash(ctx, borrowerHash)
	if err != nil {
		return nil, err
	}
	return s.loanRepo.ListByBorrower(ctx, borrower.ID, limit, offset)
}

func (s *Service) GetNFTByBorrowerHash(ctx context.Context, borrowerHashHex string) (map[string]any, error) {
	cache, err := s.GetPassportByBorrowerHash(ctx, borrowerHashHex)
	if err != nil {
		return nil, err
	}
	var tokenID any = nil
	if cache.TokenID != nil {
		tokenID = *cache.TokenID
	}
	return map[string]any{
		"token_id": tokenID,
		"token_uri": map[string]any{
			"credit_score":    cache.CreditScore,
			"total_loans":     cache.TotalLoans,
			"total_repaid":    cache.TotalRepaid,
			"total_defaulted": cache.TotalDefaulted,
		},
		"on_chain_data": map[string]any{
			"synced": cache.TokenID != nil,
		},
	}, nil
}

func (s *Service) GetPortfolioHealth(ctx context.Context, lenderID string) (*loandomain.PortfolioHealth, error) {
	return s.loanRepo.GetPortfolioHealth(ctx, lenderID)
}

func decodeBorrowerHash(input string) ([]byte, error) {
	raw := strings.TrimSpace(strings.ToLower(input))
	raw = strings.TrimPrefix(raw, "0x")
	if len(raw) == 0 {
		return nil, fmt.Errorf("missing_borrower_hash")
	}
	out, err := hex.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid_borrower_hash")
	}
	return out, nil
}
