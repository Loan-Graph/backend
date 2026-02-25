package unit

import (
	"context"
	"testing"

	admindomain "github.com/loangraph/backend/internal/domain/admin"
	lenderdomain "github.com/loangraph/backend/internal/domain/lender"
)

type adminLenderRepoMock struct {
	items map[string]*lenderdomain.Entity
}

func (m *adminLenderRepoMock) Create(_ context.Context, in lenderdomain.CreateInput) (*lenderdomain.Entity, error) {
	if m.items == nil {
		m.items = map[string]*lenderdomain.Entity{}
	}
	id := "lender-1"
	e := &lenderdomain.Entity{ID: id, Name: in.Name, CountryCode: in.CountryCode, WalletAddress: in.WalletAddress, KYCStatus: in.KYCStatus, Tier: in.Tier}
	m.items[id] = e
	return e, nil
}

func (m *adminLenderRepoMock) GetByID(_ context.Context, id string) (*lenderdomain.Entity, error) {
	if e, ok := m.items[id]; ok {
		return e, nil
	}
	return nil, context.Canceled
}

func (m *adminLenderRepoMock) UpdateKYCStatus(_ context.Context, lenderID, status string) error {
	if e, ok := m.items[lenderID]; ok {
		e.KYCStatus = status
		return nil
	}
	return context.Canceled
}

type adminAuditRepoMock struct {
	logs []admindomain.AuditLogInput
}

func (m *adminAuditRepoMock) Log(_ context.Context, in admindomain.AuditLogInput) error {
	m.logs = append(m.logs, in)
	return nil
}

func TestAdminServiceOnboardAndUpdateStatus(t *testing.T) {
	lenderRepo := &adminLenderRepoMock{items: map[string]*lenderdomain.Entity{}}
	auditRepo := &adminAuditRepoMock{}
	svc := admindomain.NewService(lenderRepo, auditRepo)

	created, err := svc.OnboardLender(context.Background(), "admin-1", lenderdomain.CreateInput{
		Name:          "New Lender",
		CountryCode:   "NG",
		WalletAddress: "0x7777777777777777777777777777777777777777",
	})
	if err != nil {
		t.Fatalf("onboard lender error: %v", err)
	}
	if created.KYCStatus != "pending" {
		t.Fatalf("expected default status pending")
	}

	if err := svc.UpdateLenderStatus(context.Background(), "admin-1", created.ID, "approved"); err != nil {
		t.Fatalf("update lender status error: %v", err)
	}
	if lenderRepo.items[created.ID].KYCStatus != "approved" {
		t.Fatalf("expected approved status")
	}
	if len(auditRepo.logs) != 2 {
		t.Fatalf("expected 2 audit logs, got %d", len(auditRepo.logs))
	}
}
