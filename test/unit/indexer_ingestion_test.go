package unit

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/loangraph/backend/internal/blockchain"
	"github.com/loangraph/backend/internal/indexer"
	"golang.org/x/crypto/sha3"
)

type fakeLogRPC struct {
	blockNumber uint64
	logs        []blockchain.LogEntry
	filter      blockchain.LogFilter
}

func (f *fakeLogRPC) BlockNumber(_ context.Context) (uint64, error) {
	return f.blockNumber, nil
}

func (f *fakeLogRPC) GetLogs(_ context.Context, filter blockchain.LogFilter) ([]blockchain.LogEntry, error) {
	f.filter = filter
	return f.logs, nil
}

type fakeIngestionRepo struct {
	hasCursor bool
	cursor    uint64
	setCursor uint64
	events    []indexer.IngestedEvent
}

func (r *fakeIngestionRepo) GetIngestionCursor(_ context.Context, _ string) (uint64, bool, error) {
	return r.cursor, r.hasCursor, nil
}

func (r *fakeIngestionRepo) SetIngestionCursor(_ context.Context, _ string, blockNumber uint64) error {
	r.setCursor = blockNumber
	r.hasCursor = true
	r.cursor = blockNumber
	return nil
}

func (r *fakeIngestionRepo) InsertChainEvent(_ context.Context, ev indexer.IngestedEvent) error {
	r.events = append(r.events, ev)
	return nil
}

func TestIngestionRunOnceIngestsAndAdvancesCursor(t *testing.T) {
	loanUUID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	repo := &fakeIngestionRepo{}
	rpc := &fakeLogRPC{
		blockNumber: 105,
		logs: []blockchain.LogEntry{
			{
				Address: "0x3c20Fd0B57711a199776B53C2F24385563d1670F",
				Topics: []string{
					eventTopic("RepaymentRecorded(bytes32,bytes32,uint256,uint256,uint256)"),
					bytes32TopicFromUUID(loanUUID),
					"0x" + zeroPaddedHex("ab", 64),
				},
				Data:            "0x" + zeroPaddedHex("1f4", 64) + zeroPaddedHex("3e8", 64) + zeroPaddedHex("7b", 64),
				BlockNumber:     102,
				TransactionHash: "0xabc123",
				LogIndex:        1,
			},
		},
	}
	svc := indexer.NewIngestionService(repo, rpc, "0x3c20Fd0B57711a199776B53C2F24385563d1670F", 100, 10, 2)

	if err := svc.RunOnce(context.Background()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if rpc.filter.FromBlock != 100 || rpc.filter.ToBlock != 103 {
		t.Fatalf("unexpected filter range: %d-%d", rpc.filter.FromBlock, rpc.filter.ToBlock)
	}
	if repo.setCursor != 103 {
		t.Fatalf("expected cursor=103, got %d", repo.setCursor)
	}
	if len(repo.events) != 1 {
		t.Fatalf("expected 1 ingested event, got %d", len(repo.events))
	}
	if repo.events[0].EventName != "RepaymentRecorded" {
		t.Fatalf("unexpected event name: %s", repo.events[0].EventName)
	}
	var raw map[string]any
	if err := json.Unmarshal(repo.events[0].RawData, &raw); err != nil {
		t.Fatalf("unmarshal raw data: %v", err)
	}
	if raw["loan_id"] != loanUUID.String() {
		t.Fatalf("expected loan_id uuid, got %#v", raw["loan_id"])
	}
	if raw["amount_minor"] != float64(500) {
		t.Fatalf("expected amount_minor=500, got %#v", raw["amount_minor"])
	}
}

func TestIngestionRunOnceNoopWhenCursorAheadOfSafeHead(t *testing.T) {
	repo := &fakeIngestionRepo{hasCursor: true, cursor: 200}
	rpc := &fakeLogRPC{blockNumber: 201}
	svc := indexer.NewIngestionService(repo, rpc, "0x3c20Fd0B57711a199776B53C2F24385563d1670F", 100, 10, 2)

	if err := svc.RunOnce(context.Background()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if len(repo.events) != 0 {
		t.Fatalf("expected no ingested events")
	}
	if repo.setCursor != 0 {
		t.Fatalf("expected cursor unchanged")
	}
}

func eventTopic(signature string) string {
	hash := sha3.NewLegacyKeccak256()
	_, _ = hash.Write([]byte(signature))
	return "0x" + hex.EncodeToString(hash.Sum(nil))
}

func bytes32TopicFromUUID(id uuid.UUID) string {
	raw := make([]byte, 32)
	copy(raw[:16], id[:])
	return "0x" + hex.EncodeToString(raw)
}

func zeroPaddedHex(v string, width int) string {
	for len(v) < width {
		v = "0" + v
	}
	return v
}
