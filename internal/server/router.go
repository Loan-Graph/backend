package server

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/loangraph/backend/internal/auth"
	"github.com/loangraph/backend/internal/config"
	"github.com/loangraph/backend/internal/http/handlers"
	"github.com/loangraph/backend/internal/http/middleware"
	"github.com/loangraph/backend/internal/version"
)

type Dependencies struct {
	Pinger      handlers.Pinger
	AuthHandler *handlers.AuthHandler
	JWTManager  *auth.JWTManager
}

func NewRouter(cfg config.Config, logger *slog.Logger, deps Dependencies) *gin.Engine {
	if cfg.Env == "prod" || cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		logger.Info("request", "method", c.Request.Method, "path", c.Request.URL.Path)
		c.Next()
	})

	health := handlers.NewHealthHandler(deps.Pinger)
	meta := handlers.NewMetaHandler(cfg.Env, version.Version)

	r.GET("/health", health.Health)
	r.GET("/ready", health.Ready)
	r.GET("/v1/meta", meta.GetMeta)

	if deps.AuthHandler != nil && deps.JWTManager != nil {
		authGroup := r.Group("/v1/auth")
		authGroup.POST("/privy/login", deps.AuthHandler.LoginWithPrivy)
		authGroup.POST("/refresh", deps.AuthHandler.Refresh)
		authGroup.POST("/logout", deps.AuthHandler.Logout)

		protected := authGroup.Group("")
		protected.Use(middleware.RequireAuth(deps.JWTManager))
		protected.GET("/me", deps.AuthHandler.Me)

		adminHandler := handlers.NewAdminHandler()
		adminGroup := r.Group("/admin")
		adminGroup.Use(middleware.RequireAuth(deps.JWTManager), middleware.RequireRole(auth.RoleAdmin))
		adminGroup.GET("/system/health", adminHandler.SystemHealth)
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
	})

	return r
}
