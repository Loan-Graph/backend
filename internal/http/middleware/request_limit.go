package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RequestBodyLimit(maxBytes int64) gin.HandlerFunc {
	if maxBytes <= 0 {
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
