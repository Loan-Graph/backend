package blockchain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type LogFilter struct {
	FromBlock uint64
	ToBlock   uint64
	Address   string
	Topics    []string
}

type LogEntry struct {
	Address         string
	Topics          []string
	Data            string
	BlockNumber     uint64
	TransactionHash string
	LogIndex        uint64
	Removed         bool
}

type LogRPCClient interface {
	BlockNumber(ctx context.Context) (uint64, error)
	GetLogs(ctx context.Context, filter LogFilter) ([]LogEntry, error)
}

type JSONRPCLogClient struct {
	httpURL    string
	httpClient *http.Client
}

func NewJSONRPCLogClient(httpURL string) (*JSONRPCLogClient, error) {
	if strings.TrimSpace(httpURL) == "" {
		return nil, fmt.Errorf("missing CREDITCOIN_HTTP_RPC")
	}
	return &JSONRPCLogClient{
		httpURL:    strings.TrimSpace(httpURL),
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}, nil
}

func (c *JSONRPCLogClient) BlockNumber(ctx context.Context) (uint64, error) {
	var out string
	if err := c.rpc(ctx, "eth_blockNumber", []any{}, &out); err != nil {
		return 0, err
	}
	return parseHexUint64(out)
}

func (c *JSONRPCLogClient) GetLogs(ctx context.Context, filter LogFilter) ([]LogEntry, error) {
	reqFilter := map[string]any{
		"fromBlock": fmt.Sprintf("0x%x", filter.FromBlock),
		"toBlock":   fmt.Sprintf("0x%x", filter.ToBlock),
		"address":   filter.Address,
		"topics":    []any{filter.Topics},
	}
	var rawLogs []struct {
		Address         string   `json:"address"`
		Topics          []string `json:"topics"`
		Data            string   `json:"data"`
		BlockNumber     string   `json:"blockNumber"`
		TransactionHash string   `json:"transactionHash"`
		LogIndex        string   `json:"logIndex"`
		Removed         bool     `json:"removed"`
	}
	if err := c.rpc(ctx, "eth_getLogs", []any{reqFilter}, &rawLogs); err != nil {
		return nil, err
	}

	out := make([]LogEntry, 0, len(rawLogs))
	for _, item := range rawLogs {
		blockNum, err := parseHexUint64(item.BlockNumber)
		if err != nil {
			return nil, fmt.Errorf("invalid blockNumber in log: %w", err)
		}
		logIndex, err := parseHexUint64(item.LogIndex)
		if err != nil {
			return nil, fmt.Errorf("invalid logIndex in log: %w", err)
		}
		out = append(out, LogEntry{
			Address:         item.Address,
			Topics:          item.Topics,
			Data:            item.Data,
			BlockNumber:     blockNum,
			TransactionHash: item.TransactionHash,
			LogIndex:        logIndex,
			Removed:         item.Removed,
		})
	}
	return out, nil
}

func (c *JSONRPCLogClient) rpc(ctx context.Context, method string, params []any, out any) error {
	reqBody, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.httpURL, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
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

func parseHexUint64(v string) (uint64, error) {
	clean := strings.TrimSpace(strings.ToLower(v))
	clean = strings.TrimPrefix(clean, "0x")
	if clean == "" {
		return 0, fmt.Errorf("empty hex value")
	}
	return strconv.ParseUint(clean, 16, 64)
}
