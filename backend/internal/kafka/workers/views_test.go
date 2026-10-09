package workers

import (
	"encoding/json"
	"testing"

	"github.com/divyanshmehta355/aurahub/backend/internal/kafka"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestUUIDScan(t *testing.T) {
	rawUUID := "aa5ad577-dc3a-4906-87fd-0278729341e0"
	var u pgtype.UUID
	err := u.Scan(rawUUID)
	if err != nil {
		t.Fatalf("u.Scan failed: %v", err)
	}
	if !u.Valid {
		t.Fatalf("u is not valid")
	}

	event := kafka.VideoViewEvent{
		VideoID: rawUUID,
	}
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var parsed kafka.VideoViewEvent
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if parsed.VideoID != rawUUID {
		t.Fatalf("expected %s, got %s", rawUUID, parsed.VideoID)
	}
}
