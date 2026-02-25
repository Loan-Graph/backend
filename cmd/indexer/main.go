package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/loangraph/backend/internal/config"
	"github.com/loangraph/backend/internal/db"
	"github.com/loangraph/backend/internal/indexer"
	"github.com/loangraph/backend/internal/observability"
	postgresrepo "github.com/loangraph/backend/internal/repository/postgres"
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

	idxRepo := postgresrepo.NewIndexerRepository(pool)
	svc := indexer.NewService(idxRepo, idxRepo)

	interval := cfg.IndexerPollInterval
	if interval <= 0 {
		interval = 2 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info("indexer started", "interval", interval.String(), "batch_size", cfg.IndexerBatchSize)
	for {
		select {
		case <-sigCtx.Done():
			logger.Info("indexer stopped")
			return
		case <-ticker.C:
			runCtx, runCancel := context.WithTimeout(context.Background(), 30*time.Second)
			err := svc.RunOnce(runCtx, cfg.IndexerBatchSize)
			runCancel()
			if err != nil && !errors.Is(err, context.Canceled) {
				logger.Error("indexer run failed", "err", err)
			}
		}
	}
}
