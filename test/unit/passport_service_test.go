package unit

import (
	"context"
	"testing"

	borrowerdomain "github.com/loangraph/backend/internal/domain/borrower"
	loandomain "github.com/loangraph/backend/internal/domain/loan"
	passportdomain "github.com/loangraph/backend/internal/domain/passport"
)

type passportBorrowerRepoMock struct {
	entity *borrowerdomain.Entity
	err    error
}

func (m *passportBorrowerRepoMock) GetByHash(_ context.Context, _ []byte) (*borrowerdomain.Entity, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.entity, nil
}

type passportRepoMock struct {
	cache *passportdomain.Cache
	err   error
}

func (m *passportRepoMock) Upsert(_ context.Context, _ passportdomain.UpsertInput) (*passportdomain.Cache, error) {
	return m.cache, nil
}

func (m *passportRepoMock) GetByBorrowerID(_ context.Context, _ string) (*passportdomain.Cache, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.cache, nil
}

type passportLoanRepoMock struct {
	history []loandomain.Entity
	health  *loandomain.PortfolioHealth
}

func (m *passportLoanRepoMock) ListByBorrower(_ context.Context, _ string, _ int32, _ int32) ([]loandomain.Entity, error) {
	return m.history, nil
}

func (m *passportLoanRepoMock) GetPortfolioHealth(_ context.Context, _ string) (*loandomain.PortfolioHealth, error) {
	return m.health, nil
}

func TestPassportServiceReadsByBorrowerHash(t *testing.T) {
	svc := passportdomain.NewService(
		&passportBorrowerRepoMock{entity: &borrowerdomain.Entity{ID: "b-1"}},
		&passportRepoMock{cache: &passportdomain.Cache{BorrowerID: "b-1", CreditScore: 700}},
		&passportLoanRepoMock{},
	)

	cache, err := svc.GetPassportByBorrowerHash(context.Background(), "0x0102")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cache.CreditScore != 700 {
		t.Fatalf("expected score 700")
	}
}

func TestPassportServiceRejectsInvalidHash(t *testing.T) {
	svc := passportdomain.NewService(&passportBorrowerRepoMock{}, &passportRepoMock{}, &passportLoanRepoMock{})
	if _, err := svc.GetPassportByBorrowerHash(context.Background(), "zz-not-hex"); err == nil {
		t.Fatalf("expected invalid hash error")
	}
}

func TestPassportServiceNFTShape(t *testing.T) {
	tokenID := int64(11)
	svc := passportdomain.NewService(
		&passportBorrowerRepoMock{entity: &borrowerdomain.Entity{ID: "b-1"}},
		&passportRepoMock{cache: &passportdomain.Cache{BorrowerID: "b-1", CreditScore: 680, TokenID: &tokenID}},
		&passportLoanRepoMock{},
	)

	nft, err := svc.GetNFTByBorrowerHash(context.Background(), "0a0b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := nft["token_uri"]; !ok {
		t.Fatalf("expected token_uri in nft response")
	}
}
