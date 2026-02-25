package handlers

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	loandomain "github.com/loangraph/backend/internal/domain/loan"
)

const maxUploadSizeBytes = 50 << 20

type LoanService interface {
	ProcessCSVUpload(ctx context.Context, lenderID string, csvReader io.Reader) (*loandomain.UploadResult, error)
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
