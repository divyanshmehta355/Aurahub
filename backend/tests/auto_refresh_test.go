package tests

import (
	"context"
	"testing"
	"time"

	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/divyanshmehta355/aurahub/backend/internal/kafka/workers"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestRunAutoRefreshCycle_NilRepo(t *testing.T) {
	_, err := workers.RunAutoRefreshCycle(context.Background(), nil, nil, nil, 28*24*time.Hour, 10)
	if err == nil {
		t.Fatalf("expected error when repo is nil, got nil")
	}
}

func TestRefreshSingleVideo_EmptyFileID(t *testing.T) {
	video := db.Video{
		ID:     pgtype.UUID{Valid: true},
		FileID: "",
		Title:  "Test Video",
	}

	err := workers.RefreshSingleVideo(context.Background(), nil, nil, nil, video, "")
	if err == nil {
		t.Fatalf("expected error for empty file_id, got nil")
	}
}
