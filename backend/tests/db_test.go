package tests

import (
	"sync"
	"testing"

	"github.com/divyanshmehta355/aurahub/backend/db"
	"gorm.io/gorm/schema"
)

func TestGormModelsHaveExplicitTableNames(t *testing.T) {
	models := []struct {
		value     interface{}
		tableName string
	}{
		{db.Comment{}, "comments"},
		{db.Notification{}, "notifications"},
		{db.Playlist{}, "playlists"},
		{db.PlaylistVideo{}, "playlist_videos"},
		{db.Subscription{}, "subscriptions"},
		{db.User{}, "users"},
		{db.UserActivity{}, "user_activities"},
		{db.Video{}, "videos"},
		{db.WatchHistory{}, "watch_history"},
		{db.WatchLater{}, "watch_later"},
	}
	for _, model := range models {
		parsed, err := schema.Parse(model.value, &sync.Map{}, schema.NamingStrategy{})
		if err != nil {
			t.Errorf("parse %T: %v", model.value, err)
			continue
		}
		if parsed.Table != model.tableName {
			t.Errorf("%T table name = %q, want %q", model.value, parsed.Table, model.tableName)
		}
	}
}

func TestSplitSQLStatements(t *testing.T) {
	testSQL := `-- Comment line
DO $$ BEGIN CREATE TYPE test_type AS ENUM ('a', 'b'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE TABLE test_table (
    id INT PRIMARY KEY,
    name TEXT
);
`
	stmts := db.SplitSQLStatements(testSQL)
	if len(stmts) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(stmts))
	}

	if stmts[0] != "DO $$ BEGIN CREATE TYPE test_type AS ENUM ('a', 'b'); EXCEPTION WHEN duplicate_object THEN NULL; END $$" {
		t.Errorf("unexpected statement 0: %q", stmts[0])
	}

	if len(db.SchemaSQL) == 0 {
		t.Fatalf("expected SchemaSQL to be embedded, got empty string")
	}

	parsedSchema := db.SplitSQLStatements(db.SchemaSQL)
	if len(parsedSchema) < 10 {
		t.Errorf("expected at least 10 statements from SchemaSQL, got %d", len(parsedSchema))
	}
}
