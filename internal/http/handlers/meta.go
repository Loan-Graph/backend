package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type MetaHandler struct {
	env     string
	version string
}

func NewMetaHandler(env, version string) *MetaHandler {
	return &MetaHandler{env: env, version: version}
}

func (h *MetaHandler) GetMeta(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"name":    "LoanGraph Backend",
		"version": h.version,
		"env":     h.env,
	})
}
