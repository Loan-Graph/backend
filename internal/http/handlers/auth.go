package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/loangraph/backend/internal/auth"
)

type AuthHandler struct {
	authService *auth.Service
	cookieCfg   auth.CookieConfig
	accessTTL   time.Duration
	refreshTTL  time.Duration
}

type privyLoginRequest struct {
	PrivyAccessToken string `json:"privy_access_token" binding:"required"`
}

func NewAuthHandler(authService *auth.Service, cookieCfg auth.CookieConfig, accessTTL, refreshTTL time.Duration) *AuthHandler {
	return &AuthHandler{authService: authService, cookieCfg: cookieCfg, accessTTL: accessTTL, refreshTTL: refreshTTL}
}

func (h *AuthHandler) LoginWithPrivy(c *gin.Context) {
	var req privyLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	userAgent := c.GetHeader("User-Agent")
	ipAddress := auth.ClientIP(c.Request)
	tokens, err := h.authService.LoginWithPrivy(c.Request.Context(), req.PrivyAccessToken, userAgent, ipAddress)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication_failed"})
		return
	}

	auth.SetAuthCookies(c.Writer, h.cookieCfg, tokens.AccessToken, tokens.RefreshToken, h.accessTTL, h.refreshTTL)
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":             tokens.User.ID,
			"email":          tokens.User.Email,
			"email_verified": tokens.User.EmailVerified,
			"wallet_address": tokens.User.WalletAddress,
			"privy_subject":  tokens.User.PrivySubject,
		},
		"session": gin.H{"authenticated": true},
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	cookie, err := c.Request.Cookie(auth.RefreshCookieName)
	if err != nil || cookie.Value == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing_refresh_cookie"})
		return
	}

	userAgent := c.GetHeader("User-Agent")
	ipAddress := auth.ClientIP(c.Request)
	tokens, err := h.authService.Refresh(c.Request.Context(), cookie.Value, userAgent, ipAddress)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh_failed"})
		return
	}

	auth.SetAuthCookies(c.Writer, h.cookieCfg, tokens.AccessToken, tokens.RefreshToken, h.accessTTL, h.refreshTTL)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	cookie, err := c.Request.Cookie(auth.RefreshCookieName)
	if err == nil && cookie.Value != "" {
		_ = h.authService.Logout(c.Request.Context(), cookie.Value)
	}
	auth.ClearAuthCookies(c.Writer, h.cookieCfg)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AuthHandler) Me(c *gin.Context) {
	uid, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, err := h.authService.Me(c.Request.Context(), uid.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":             user.ID,
			"email":          user.Email,
			"email_verified": user.EmailVerified,
			"wallet_address": user.WalletAddress,
			"privy_subject":  user.PrivySubject,
		},
	})
}
