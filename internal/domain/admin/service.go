package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	lenderdomain "github.com/loangraph/backend/internal/domain/lender"
)

type LenderRepository interface {
	Create(ctx context.Context, in lenderdomain.CreateInput) (*lenderdomain.Entity, error)
	GetByID(ctx context.Context, id string) (*lenderdomain.Entity, error)
	UpdateKYCStatus(ctx context.Context, lenderID, kycStatus string) error
}

type AuditRepository interface {
	Log(ctx context.Context, in AuditLogInput) error
}

type AuditLogInput struct {
	AdminUserID string
	Action      string
	TargetType  string
	TargetID    string
	Payload     []byte
}

type Service struct {
	lenderRepo LenderRepository
	auditRepo  AuditRepository
}

func NewService(lenderRepo LenderRepository, auditRepo AuditRepository) *Service {
	return &Service{lenderRepo: lenderRepo, auditRepo: auditRepo}
}

func (s *Service) OnboardLender(ctx context.Context, adminUserID string, in lenderdomain.CreateInput) (*lenderdomain.Entity, error) {
	if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.CountryCode) == "" || strings.TrimSpace(in.WalletAddress) == "" {
		return nil, fmt.Errorf("invalid_lender_input")
	}
	if strings.TrimSpace(in.KYCStatus) == "" {
		in.KYCStatus = "pending"
	}
	if strings.TrimSpace(in.Tier) == "" {
		in.Tier = "starter"
	}

	created, err := s.lenderRepo.Create(ctx, in)
	if err != nil {
		return nil, err
	}
	payload, _ := json.Marshal(map[string]any{"name": created.Name, "country_code": created.CountryCode, "wallet_address": created.WalletAddress, "kyc_status": created.KYCStatus, "tier": created.Tier})
	_ = s.auditRepo.Log(ctx, AuditLogInput{
		AdminUserID: adminUserID,
		Action:      "lender_onboarded",
		TargetType:  "lender",
		TargetID:    created.ID,
		Payload:     payload,
	})
	return created, nil
}

func (s *Service) UpdateLenderStatus(ctx context.Context, adminUserID, lenderID, status string) error {
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "pending" && status != "approved" && status != "suspended" {
		return fmt.Errorf("invalid_kyc_status")
	}
	if strings.TrimSpace(lenderID) == "" {
		return fmt.Errorf("missing_lender_id")
	}
	if _, err := s.lenderRepo.GetByID(ctx, lenderID); err != nil {
		return err
	}
	if err := s.lenderRepo.UpdateKYCStatus(ctx, lenderID, status); err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]any{"kyc_status": status})
	_ = s.auditRepo.Log(ctx, AuditLogInput{
		AdminUserID: adminUserID,
		Action:      "lender_status_updated",
		TargetType:  "lender",
		TargetID:    lenderID,
		Payload:     payload,
	})
	return nil
}
