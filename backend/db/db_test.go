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
