package handlers

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	lenderdomain "github.com/loangraph/backend/internal/domain/lender"
)

type AdminService interface {
	OnboardLender(ctx context.Context, adminUserID string, in lenderdomain.CreateInput) (*lenderdomain.Entity, error)
	UpdateLenderStatus(ctx context.Context, adminUserID, lenderID, status string) error
}

type AdminHandler struct {
	adminService AdminService
}

var evmAddressPattern = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

func NewAdminHandler(adminService AdminService) *AdminHandler {
	return &AdminHandler{adminService: adminService}
}

func (h *AdminHandler) SystemHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *AdminHandler) OnboardLender(c *gin.Context) {
	if h.adminService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "admin_service_unavailable"})
		return
	}
	var req struct {
		Name          string `json:"name"`
		CountryCode   string `json:"country_code"`
		WalletAddress string `json:"wallet_address"`
		KYCStatus     string `json:"kyc_status"`
		Tier          string `json:"tier"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	if strings.TrimSpace(req.Name) == "" || len(strings.TrimSpace(req.CountryCode)) != 2 || !evmAddressPattern.MatchString(strings.TrimSpace(req.WalletAddress)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	adminUserID, _ := c.Get("user_id")
	created, err := h.adminService.OnboardLender(c.Request.Context(), toString(adminUserID), lenderdomain.CreateInput{
		Name:          strings.TrimSpace(req.Name),
		CountryCode:   strings.ToUpper(strings.TrimSpace(req.CountryCode)),
		WalletAddress: strings.TrimSpace(req.WalletAddress),
		KYCStatus:     req.KYCStatus,
		Tier:          req.Tier,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "onboard_lender_failed"})
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *AdminHandler) UpdateLenderStatus(c *gin.Context) {
	if h.adminService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "admin_service_unavailable"})
		return
	}
	lenderID := c.Param("lenderId")
	var req struct {
		KYCStatus string `json:"kyc_status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	if strings.TrimSpace(req.KYCStatus) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	adminUserID, _ := c.Get("user_id")
	if err := h.adminService.UpdateLenderStatus(c.Request.Context(), toString(adminUserID), lenderID, req.KYCStatus); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "update_lender_status_failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
