package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	loandomain "github.com/loangraph/backend/internal/domain/loan"
	passportdomain "github.com/loangraph/backend/internal/domain/passport"
)

type PassportService interface {
	GetPassportByBorrowerHash(ctx context.Context, borrowerHashHex string) (*passportdomain.Cache, error)
	GetHistoryByBorrowerHash(ctx context.Context, borrowerHashHex string, limit, offset int32) ([]loandomain.Entity, error)
	GetNFTByBorrowerHash(ctx context.Context, borrowerHashHex string) (map[string]any, error)
	GetPortfolioHealth(ctx context.Context, lenderID string) (*loandomain.PortfolioHealth, error)
}

type PassportHandler struct {
	passportService PassportService
}

func NewPassportHandler(passportService PassportService) *PassportHandler {
	return &PassportHandler{passportService: passportService}
}

func (h *PassportHandler) GetPassport(c *gin.Context) {
	borrowerHash := strings.TrimSpace(c.Param("borrowerHash"))
	cache, err := h.passportService.GetPassportByBorrowerHash(c.Request.Context(), borrowerHash)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "passport_not_found"})
		return
	}
	c.JSON(http.StatusOK, cache)
}

func (h *PassportHandler) GetPassportHistory(c *gin.Context) {
	borrowerHash := strings.TrimSpace(c.Param("borrowerHash"))
	limit, _ := strconv.ParseInt(strings.TrimSpace(c.DefaultQuery("limit", "50")), 10, 32)
	offset, _ := strconv.ParseInt(strings.TrimSpace(c.DefaultQuery("offset", "0")), 10, 32)
	items, err := h.passportService.GetHistoryByBorrowerHash(c.Request.Context(), borrowerHash, int32(limit), int32(offset))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "passport_history_not_found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *PassportHandler) GetPassportNFT(c *gin.Context) {
	borrowerHash := strings.TrimSpace(c.Param("borrowerHash"))
	out, err := h.passportService.GetNFTByBorrowerHash(c.Request.Context(), borrowerHash)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "passport_nft_not_found"})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *PassportHandler) GetPortfolioHealth(c *gin.Context) {
	lenderID := strings.TrimSpace(c.Query("lender_id"))
	health, err := h.passportService.GetPortfolioHealth(c.Request.Context(), lenderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "portfolio_health_failed"})
		return
	}
	c.JSON(http.StatusOK, health)
}
