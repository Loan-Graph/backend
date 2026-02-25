package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/loangraph/backend/internal/blockchain"
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
	var ingestSvc *indexer.IngestionService
	if cfg.IndexerIngestEnabled {
		if strings.TrimSpace(cfg.CreditcoinHTTPRPC) == "" || strings.TrimSpace(cfg.LoanRegistryProxy) == "" {
			logger.Error("indexer ingestion enabled but missing chain config", "CREDITCOIN_HTTP_RPC", cfg.CreditcoinHTTPRPC != "", "LOAN_REGISTRY_PROXY", cfg.LoanRegistryProxy != "")
			os.Exit(1)
		}
		rpcClient, err := blockchain.NewJSONRPCLogClient(cfg.CreditcoinHTTPRPC)
		if err != nil {
			logger.Error("failed to initialize indexer rpc client", "err", err)
			os.Exit(1)
		}
		ingestSvc = indexer.NewIngestionService(
			idxRepo,
			rpcClient,
			cfg.LoanRegistryProxy,
			cfg.IndexerStartBlock,
			cfg.IndexerBlockBatchSize,
			cfg.IndexerConfirmations,
		)
	}

	interval := cfg.IndexerPollInterval
	if interval <= 0 {
		interval = 2 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info("indexer started", "interval", interval.String(), "batch_size", cfg.IndexerBatchSize, "ingestion_enabled", cfg.IndexerIngestEnabled)
	for {
		select {
		case <-sigCtx.Done():
			logger.Info("indexer stopped")
			return
		case <-ticker.C:
			runCtx, runCancel := context.WithTimeout(context.Background(), 30*time.Second)
			if ingestSvc != nil {
				err := ingestSvc.RunOnce(runCtx)
				if err != nil && !errors.Is(err, context.Canceled) {
					logger.Error("indexer ingestion failed", "err", err)
				}
			}
			err := svc.RunOnce(runCtx, cfg.IndexerBatchSize)
			runCancel()
			if err != nil && !errors.Is(err, context.Canceled) {
				logger.Error("indexer run failed", "err", err)
			}
		}
	}
}
