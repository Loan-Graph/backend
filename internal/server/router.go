package server

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/loangraph/backend/internal/config"
	"github.com/loangraph/backend/internal/http/handlers"
	"github.com/loangraph/backend/internal/version"
)

func NewRouter(cfg config.Config, logger *slog.Logger, pinger handlers.Pinger) *gin.Engine {
	if cfg.Env == "prod" || cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		logger.Info("request", "method", c.Request.Method, "path", c.Request.URL.Path)
		c.Next()
	})

	health := handlers.NewHealthHandler(pinger)
	meta := handlers.NewMetaHandler(cfg.Env, version.Version)

	r.GET("/health", health.Health)
	r.GET("/ready", health.Ready)
	r.GET("/v1/meta", meta.GetMeta)

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
	})

	return r
}
