package indexer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/google/uuid"
	"github.com/loangraph/backend/internal/blockchain"
	"golang.org/x/crypto/sha3"
)

const (
	ingestionCursorKey = "indexer.loan_registry.last_block"
)

type IngestedEvent struct {
	ContractAddr string
	EventName    string
	TXHash       string
	BlockNumber  uint64
	LogIndex     uint64
	RawData      json.RawMessage
}

type IngestionRepository interface {
	GetIngestionCursor(ctx context.Context, key string) (uint64, bool, error)
	SetIngestionCursor(ctx context.Context, key string, blockNumber uint64) error
	InsertChainEvent(ctx context.Context, ev IngestedEvent) error
}

type IngestionService struct {
	repo          IngestionRepository
	rpc           blockchain.LogRPCClient
	contractAddr  string
	startBlock    uint64
	blockBatch    uint64
	confirmations uint64
}

func NewIngestionService(repo IngestionRepository, rpc blockchain.LogRPCClient, contractAddr string, startBlock, blockBatch, confirmations uint64) *IngestionService {
	if blockBatch == 0 {
		blockBatch = 500
	}
	return &IngestionService{
		repo:          repo,
		rpc:           rpc,
		contractAddr:  strings.TrimSpace(contractAddr),
		startBlock:    startBlock,
		blockBatch:    blockBatch,
		confirmations: confirmations,
	}
}

func (s *IngestionService) RunOnce(ctx context.Context) error {
	latest, err := s.rpc.BlockNumber(ctx)
	if err != nil {
		return err
	}

	if latest < s.confirmations {
		return nil
	}
	safeHead := latest - s.confirmations

	last, ok, err := s.repo.GetIngestionCursor(ctx, ingestionCursorKey)
	if err != nil {
		return err
	}
	var fromBlock uint64
	if ok {
		fromBlock = last + 1
	} else {
		fromBlock = s.startBlock
	}
	if fromBlock > safeHead {
		return nil
	}

	toBlock := minUint64(safeHead, fromBlock+s.blockBatch-1)
	logs, err := s.rpc.GetLogs(ctx, blockchain.LogFilter{
		FromBlock: fromBlock,
		ToBlock:   toBlock,
		Address:   s.contractAddr,
		Topics:    []string{topicLoanRegistered, topicRepaymentRecorded, topicLoanDefaulted},
	})
	if err != nil {
		return err
	}

	for _, lg := range logs {
		if lg.Removed {
			continue
		}
		ev, ok, err := decodeLogToEvent(lg)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if err := s.repo.InsertChainEvent(ctx, ev); err != nil {
			return err
		}
	}

	return s.repo.SetIngestionCursor(ctx, ingestionCursorKey, toBlock)
}

var (
	topicLoanRegistered    = eventTopic("LoanRegistered(bytes32,bytes32,address,uint256,uint256,string)")
	topicRepaymentRecorded = eventTopic("RepaymentRecorded(bytes32,bytes32,uint256,uint256,uint256)")
	topicLoanDefaulted     = eventTopic("LoanDefaulted(bytes32,bytes32,uint256)")
)

func decodeLogToEvent(log blockchain.LogEntry) (IngestedEvent, bool, error) {
	if len(log.Topics) == 0 {
		return IngestedEvent{}, false, nil
	}
	name := ""
	raw := map[string]any{}
	switch strings.ToLower(log.Topics[0]) {
	case strings.ToLower(topicLoanRegistered):
		if len(log.Topics) < 3 {
			return IngestedEvent{}, false, fmt.Errorf("LoanRegistered missing indexed topics")
		}
		loanIDBytes32 := normalizeBytes32Hex(log.Topics[1])
		borrowerIDBytes32 := normalizeBytes32Hex(log.Topics[2])
		loanID := projectedLoanID(loanIDBytes32)
		principal, maturity, currency := parseLoanRegisteredData(log.Data)
		name = "LoanRegistered"
		raw = map[string]any{
			"loan_id":             loanID,
			"loan_id_bytes32":     loanIDBytes32,
			"borrower_id_bytes32": borrowerIDBytes32,
			"principal_minor":     principal,
			"maturity_ts":         maturity,
			"currency_code":       currency,
		}

	case strings.ToLower(topicRepaymentRecorded):
		if len(log.Topics) < 3 {
			return IngestedEvent{}, false, fmt.Errorf("RepaymentRecorded missing indexed topics")
		}
		loanIDBytes32 := normalizeBytes32Hex(log.Topics[1])
		borrowerIDBytes32 := normalizeBytes32Hex(log.Topics[2])
		loanID := projectedLoanID(loanIDBytes32)
		amountMinor, totalRepaid, ts := parseRepaymentData(log.Data)
		name = "RepaymentRecorded"
		raw = map[string]any{
			"loan_id":             loanID,
			"loan_id_bytes32":     loanIDBytes32,
			"borrower_id_bytes32": borrowerIDBytes32,
			"amount_minor":        amountMinor,
			"total_repaid_minor":  totalRepaid,
			"timestamp":           ts,
		}

	case strings.ToLower(topicLoanDefaulted):
		if len(log.Topics) < 3 {
			return IngestedEvent{}, false, fmt.Errorf("LoanDefaulted missing indexed topics")
		}
		loanIDBytes32 := normalizeBytes32Hex(log.Topics[1])
		borrowerIDBytes32 := normalizeBytes32Hex(log.Topics[2])
		loanID := projectedLoanID(loanIDBytes32)
		ts := parseDefaultData(log.Data)
		name = "LoanDefaulted"
		raw = map[string]any{
			"loan_id":             loanID,
			"loan_id_bytes32":     loanIDBytes32,
			"borrower_id_bytes32": borrowerIDBytes32,
			"timestamp":           ts,
		}
	default:
		return IngestedEvent{}, false, nil
	}

	rawJSON, err := json.Marshal(raw)
	if err != nil {
		return IngestedEvent{}, false, err
	}
	return IngestedEvent{
		ContractAddr: strings.ToLower(log.Address),
		EventName:    name,
		TXHash:       strings.ToLower(log.TransactionHash),
		BlockNumber:  log.BlockNumber,
		LogIndex:     log.LogIndex,
		RawData:      rawJSON,
	}, true, nil
}

func parseLoanRegisteredData(dataHex string) (int64, int64, string) {
	words := abiWords(dataHex)
	if len(words) < 2 {
		return 0, 0, ""
	}
	return toInt64(words[0]), toInt64(words[1]), ""
}

func parseRepaymentData(dataHex string) (int64, int64, int64) {
	words := abiWords(dataHex)
	if len(words) < 3 {
		return 0, 0, 0
	}
	return toInt64(words[0]), toInt64(words[1]), toInt64(words[2])
}

func parseDefaultData(dataHex string) int64 {
	words := abiWords(dataHex)
	if len(words) < 1 {
		return 0
	}
	return toInt64(words[0])
}

func abiWords(dataHex string) []string {
	clean := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(dataHex)), "0x")
	if len(clean)%64 != 0 {
		return nil
	}
	words := make([]string, 0, len(clean)/64)
	for i := 0; i+64 <= len(clean); i += 64 {
		words = append(words, clean[i:i+64])
	}
	return words
}

func toInt64(word string) int64 {
	n, ok := new(big.Int).SetString(word, 16)
	if !ok || !n.IsInt64() {
		return 0
	}
	return n.Int64()
}

func normalizeBytes32Hex(topic string) string {
	clean := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(topic)), "0x")
	if len(clean) < 64 {
		clean = strings.Repeat("0", 64-len(clean)) + clean
	}
	if len(clean) > 64 {
		clean = clean[len(clean)-64:]
	}
	return "0x" + clean
}

func projectedLoanID(bytes32Hex string) string {
	if out, ok := bytes32ToUUID(bytes32Hex); ok {
		return out
	}
	return bytes32Hex
}

func bytes32ToUUID(bytes32Hex string) (string, bool) {
	clean := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(bytes32Hex)), "0x")
	if len(clean) != 64 {
		return "", false
	}
	raw, err := hex.DecodeString(clean)
	if err != nil || len(raw) != 32 {
		return "", false
	}

	if allZero(raw[16:]) {
		id, err := uuid.FromBytes(raw[:16])
		if err == nil {
			return id.String(), true
		}
	}
	if allZero(raw[:16]) {
		id, err := uuid.FromBytes(raw[16:])
		if err == nil {
			return id.String(), true
		}
	}
	return "", false
}

func allZero(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}

func minUint64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

func eventTopic(signature string) string {
	hash := sha3.NewLegacyKeccak256()
	_, _ = hash.Write([]byte(signature))
	return "0x" + hex.EncodeToString(hash.Sum(nil))
}
