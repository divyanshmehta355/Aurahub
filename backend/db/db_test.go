package db

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

func TestGormModelsHaveExplicitTableNames(t *testing.T) {
	models := []struct {
		value     interface{}
		tableName string
	}{
		{Comment{}, "comments"},
		{Notification{}, "notifications"},
		{Playlist{}, "playlists"},
		{PlaylistVideo{}, "playlist_videos"},
		{Subscription{}, "subscriptions"},
		{User{}, "users"},
		{UserActivity{}, "user_activities"},
		{Video{}, "videos"},
		{WatchHistory{}, "watch_history"},
		{WatchLater{}, "watch_later"},
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
	stmts := SplitSQLStatements(testSQL)
	if len(stmts) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(stmts))
	}

	if stmts[0] != "DO $$ BEGIN CREATE TYPE test_type AS ENUM ('a', 'b'); EXCEPTION WHEN duplicate_object THEN NULL; END $$" {
		t.Errorf("unexpected statement 0: %q", stmts[0])
	}

	if len(SchemaSQL) == 0 {
		t.Fatalf("expected SchemaSQL to be embedded, got empty string")
	}

	parsedSchema := SplitSQLStatements(SchemaSQL)
	if len(parsedSchema) < 10 {
		t.Errorf("expected at least 10 statements from SchemaSQL, got %d", len(parsedSchema))
	}
}
