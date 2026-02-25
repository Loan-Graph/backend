package handlers

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	loandomain "github.com/loangraph/backend/internal/domain/loan"
)

const maxUploadSizeBytes = 50 << 20

type LoanService interface {
	ProcessCSVUpload(ctx context.Context, lenderID string, csvReader io.Reader) (*loandomain.UploadResult, error)
	ListLoans(ctx context.Context, filter loandomain.ListFilter) ([]loandomain.Entity, error)
	GetLoan(ctx context.Context, loanID string) (*loandomain.Entity, error)
	RecordRepayment(ctx context.Context, in loandomain.RepaymentInput) error
	MarkDefault(ctx context.Context, in loandomain.DefaultInput) error
	PortfolioAnalytics(ctx context.Context, lenderID string) (*loandomain.PortfolioAnalytics, error)
}

type LoanHandler struct {
	loanService LoanService
}

func NewLoanHandler(loanService LoanService) *LoanHandler {
	return &LoanHandler{loanService: loanService}
}

func (h *LoanHandler) UploadLoanBook(c *gin.Context) {
	lenderID := strings.TrimSpace(c.PostForm("lender_id"))
	if lenderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing_lender_id"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing_file"})
		return
	}
	if file.Size > maxUploadSizeBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file_too_large"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_file"})
		return
	}
	defer src.Close()

	result, err := h.loanService.ProcessCSVUpload(c.Request.Context(), lenderID, src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "upload_failed"})
		return
	}

	if len(result.Errors) > 0 {
		c.JSON(http.StatusBadRequest, result)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *LoanHandler) ListLoans(c *gin.Context) {
	limit, _ := strconv.ParseInt(strings.TrimSpace(c.DefaultQuery("limit", "50")), 10, 32)
	offset, _ := strconv.ParseInt(strings.TrimSpace(c.DefaultQuery("offset", "0")), 10, 32)
	items, err := h.loanService.ListLoans(c.Request.Context(), loandomain.ListFilter{
		LenderID:  strings.TrimSpace(c.Query("lender_id")),
		Status:    strings.TrimSpace(c.Query("status")),
		RiskGrade: strings.TrimSpace(c.Query("risk_grade")),
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list_loans_failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *LoanHandler) GetLoan(c *gin.Context) {
	loanID := strings.TrimSpace(c.Param("loanId"))
	if loanID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing_loan_id"})
		return
	}
	item, err := h.loanService.GetLoan(c.Request.Context(), loanID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "loan_not_found"})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *LoanHandler) RecordRepayment(c *gin.Context) {
	loanID := strings.TrimSpace(c.Param("loanId"))
	if loanID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	var req struct {
		AmountMinor int64  `json:"amount_minor"`
		Currency    string `json:"currency"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	if err := h.loanService.RecordRepayment(c.Request.Context(), loandomain.RepaymentInput{
		LoanID:      loanID,
		AmountMinor: req.AmountMinor,
		Currency:    req.Currency,
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repayment_failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"updated_status": "processing"})
}

func (h *LoanHandler) MarkDefault(c *gin.Context) {
	loanID := strings.TrimSpace(c.Param("loanId"))
	if loanID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	if err := h.loanService.MarkDefault(c.Request.Context(), loandomain.DefaultInput{
		LoanID:   loanID,
		Reason:   req.Reason,
		LenderID: strings.TrimSpace(c.Query("lender_id")),
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "default_failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"updated_status": "processing"})
}

func (h *LoanHandler) GetPortfolioAnalytics(c *gin.Context) {
	lenderID := strings.TrimSpace(c.Query("lender_id"))
	analytics, err := h.loanService.PortfolioAnalytics(c.Request.Context(), lenderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "analytics_failed"})
		return
	}
	c.JSON(http.StatusOK, analytics)
}
