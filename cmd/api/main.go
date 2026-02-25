package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/loangraph/backend/internal/auth"
	"github.com/loangraph/backend/internal/config"
	"github.com/loangraph/backend/internal/db"
	loandomain "github.com/loangraph/backend/internal/domain/loan"
	"github.com/loangraph/backend/internal/http/handlers"
	"github.com/loangraph/backend/internal/observability"
	postgresrepo "github.com/loangraph/backend/internal/repository/postgres"
	"github.com/loangraph/backend/internal/server"
)

func main() {
	cfg := config.Load()
	logger := observability.NewLogger(cfg.Env)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.NewPostgresPool(ctx, cfg)
	if err != nil {
		logger.Error("failed to connect postgres", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	authRepo := db.NewAuthRepository(pool)
	jwtManager := auth.NewJWTManager(cfg.JWTIssuer, cfg.JWTAudience, cfg.JWTSigningKey)
	privyVerifier := auth.NewPrivyTokenVerifier(cfg.PrivyIssuer, cfg.PrivyAudience, cfg.PrivyVerificationKey, cfg.PrivyJWKSURL)
	authService := auth.NewService(authRepo, jwtManager, privyVerifier, cfg.JWTAccessTTL, cfg.JWTRefreshTTL, cfg.AuthBootstrapAdminSubject)
	authHandler := handlers.NewAuthHandler(authService, auth.CookieConfig{Domain: cfg.CookieDomain, Secure: cfg.CookieSecure}, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	loanService := loandomain.NewService(
		postgresrepo.NewBorrowerRepository(pool),
		postgresrepo.NewLoanRepository(pool),
		postgresrepo.NewOutboxRepository(pool),
	)
	loanHandler := handlers.NewLoanHandler(loanService)

	r := server.NewRouter(cfg, logger, server.Dependencies{
		Pinger:      pool,
		AuthHandler: authHandler,
		LoanHandler: loanHandler,
		JWTManager:  jwtManager,
	})
	httpServer := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("api server starting", "addr", cfg.Addr())
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-sigCtx.Done()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = httpServer.Shutdown(shutdownCtx)
	logger.Info("api server stopped")
}
