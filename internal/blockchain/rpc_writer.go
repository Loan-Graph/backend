package blockchain

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var addressPattern = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

type RPCWriter struct {
	httpURL      string
	fromAddress  string
	contractAddr string
	gasLimit     uint64
	httpClient   *http.Client
}

func NewRPCWriter(httpURL, fromAddress, contractAddr string, gasLimit uint64) (*RPCWriter, error) {
	if strings.TrimSpace(httpURL) == "" {
		return nil, fmt.Errorf("missing CREDITCOIN_HTTP_RPC")
	}
	if !addressPattern.MatchString(strings.TrimSpace(fromAddress)) {
		return nil, fmt.Errorf("invalid CHAIN_WRITER_FROM_ADDRESS")
	}
	if !addressPattern.MatchString(strings.TrimSpace(contractAddr)) {
		return nil, fmt.Errorf("invalid LOAN_REGISTRY_PROXY")
	}
	if gasLimit == 0 {
		gasLimit = 300000
	}
	return &RPCWriter{
		httpURL:      strings.TrimSpace(httpURL),
		fromAddress:  strings.TrimSpace(fromAddress),
		contractAddr: strings.TrimSpace(contractAddr),
		gasLimit:     gasLimit,
		httpClient:   &http.Client{Timeout: 20 * time.Second},
	}, nil
}

func (w *RPCWriter) RegisterLoan(ctx context.Context, loanID string) (string, error) {
	if strings.TrimSpace(loanID) == "" {
		return "", fmt.Errorf("missing loan id")
	}
	return w.sendMarker(ctx, "register_loan", map[string]any{"loan_id": strings.TrimSpace(loanID)})
}

func (w *RPCWriter) RecordRepayment(ctx context.Context, loanID string, amountMinor int64, currency string) (string, error) {
	if strings.TrimSpace(loanID) == "" || amountMinor <= 0 || len(strings.TrimSpace(currency)) != 3 {
		return "", fmt.Errorf("invalid repayment args")
	}
	return w.sendMarker(ctx, "record_repayment", map[string]any{
		"loan_id":      strings.TrimSpace(loanID),
		"amount_minor": amountMinor,
		"currency":     strings.ToUpper(strings.TrimSpace(currency)),
	})
}

func (w *RPCWriter) MarkDefault(ctx context.Context, loanID string, reason string) (string, error) {
	if strings.TrimSpace(loanID) == "" {
		return "", fmt.Errorf("invalid default args")
	}
	return w.sendMarker(ctx, "mark_default", map[string]any{"loan_id": strings.TrimSpace(loanID), "reason": strings.TrimSpace(reason)})
}

func (w *RPCWriter) sendMarker(ctx context.Context, action string, payload map[string]any) (string, error) {
	dataBytes, _ := json.Marshal(map[string]any{
		"action":  action,
		"payload": payload,
	})
	txObj := map[string]string{
		"from":  w.fromAddress,
		"to":    w.contractAddr,
		"gas":   fmt.Sprintf("0x%x", w.gasLimit),
		"data":  "0x" + hex.EncodeToString(dataBytes),
		"value": "0x0",
	}

	var txHash string
	if err := w.rpc(ctx, "eth_sendTransaction", []any{txObj}, &txHash); err != nil {
		return "", err
	}
	if !strings.HasPrefix(txHash, "0x") {
		return "", fmt.Errorf("invalid tx hash response")
	}
	return txHash, nil
}

func (w *RPCWriter) rpc(ctx context.Context, method string, params []any, out any) error {
	reqBody, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.httpURL, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var payload struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return err
	}
	if payload.Error != nil {
		return fmt.Errorf("rpc error %d: %s", payload.Error.Code, payload.Error.Message)
	}
	if len(payload.Result) == 0 {
		return fmt.Errorf("rpc empty result")
	}
	if err := json.Unmarshal(payload.Result, out); err != nil {
		return err
	}
	return nil
}
