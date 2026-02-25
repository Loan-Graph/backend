package blockchain

import (
	"fmt"
	"strings"

	"github.com/loangraph/backend/internal/config"
)

func NewWriterFromConfig(cfg config.Config) (LoanRegistryWriter, error) {
	mode := strings.ToLower(strings.TrimSpace(cfg.ChainWriterMode))
	if mode == "" || mode == "stub" {
		return NewStubWriter(), nil
	}
	if mode != "real" {
		return nil, fmt.Errorf("invalid CHAIN_WRITER_MODE: %s", cfg.ChainWriterMode)
	}
	return NewRPCWriter(cfg.CreditcoinHTTPRPC, cfg.ChainWriterFromAddress, cfg.LoanRegistryProxy, cfg.ChainTxGasLimit)
}
