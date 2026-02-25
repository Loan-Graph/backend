package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	investordomain "github.com/loangraph/backend/internal/domain/investor"
	loandomain "github.com/loangraph/backend/internal/domain/loan"
	pooldomain "github.com/loangraph/backend/internal/domain/pool"
)

type InvestorService interface {
	ListPools(ctx context.Context, f investordomain.PoolFilter) ([]pooldomain.Entity, error)
	GetPool(ctx context.Context, poolID string) (*pooldomain.Entity, error)
	GetPoolPerformance(ctx context.Context, poolID string, days int32) ([]loandomain.PerformancePoint, error)
	GetLenderProfile(ctx context.Context, lenderID string) (*investordomain.LenderProfile, error)
}

type InvestorHandler struct {
	investorService InvestorService
}

func NewInvestorHandler(investorService InvestorService) *InvestorHandler {
	return &InvestorHandler{investorService: investorService}
}

func (h *InvestorHandler) ListPools(c *gin.Context) {
	limit, _ := strconv.ParseInt(strings.TrimSpace(c.DefaultQuery("limit", "50")), 10, 32)
	offset, _ := strconv.ParseInt(strings.TrimSpace(c.DefaultQuery("offset", "0")), 10, 32)
	items, err := h.investorService.ListPools(c.Request.Context(), investordomain.PoolFilter{
		LenderID:     strings.TrimSpace(c.Query("lender_id")),
		CurrencyCode: strings.TrimSpace(c.Query("currency")),
		Status:       strings.TrimSpace(c.Query("status")),
		Limit:        int32(limit),
		Offset:       int32(offset),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list_pools_failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *InvestorHandler) GetPool(c *gin.Context) {
	poolID := strings.TrimSpace(c.Param("poolId"))
	item, err := h.investorService.GetPool(c.Request.Context(), poolID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pool_not_found"})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *InvestorHandler) GetPoolPerformance(c *gin.Context) {
	poolID := strings.TrimSpace(c.Param("poolId"))
	days64, _ := strconv.ParseInt(strings.TrimSpace(c.DefaultQuery("days", "30")), 10, 32)
	points, err := h.investorService.GetPoolPerformance(c.Request.Context(), poolID, int32(days64))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pool_performance_failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": points})
}

func (h *InvestorHandler) GetLenderProfile(c *gin.Context) {
	lenderID := strings.TrimSpace(c.Param("lenderId"))
	profile, err := h.investorService.GetLenderProfile(c.Request.Context(), lenderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "lender_profile_not_found"})
		return
	}
	c.JSON(http.StatusOK, profile)
}
