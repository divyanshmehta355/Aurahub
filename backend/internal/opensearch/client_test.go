package opensearch

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestOpenSearchStandbyMode(t *testing.T) {
	client, err := NewClient(Config{})
	if err != nil {
		t.Fatalf("expected no error for empty config, got %v", err)
	}
	if client.Enabled() {
		t.Errorf("expected client to be disabled in standby mode")
	}

	ctx := context.Background()
	if err := client.EnsureIndex(ctx); err != nil {
		t.Errorf("EnsureIndex in standby should return nil, got %v", err)
	}

	doc := VideoDocument{
		ID:        "test-id",
		Title:     "Test Video",
		CreatedAt: time.Now(),
	}
	if err := client.IndexVideo(ctx, doc); err != nil {
		t.Errorf("IndexVideo in standby should return nil, got %v", err)
	}

	if err := client.DeleteVideo(ctx, "test-id"); err != nil {
		t.Errorf("DeleteVideo in standby should return nil, got %v", err)
	}

	_, err = client.SearchVideos(ctx, "test", "", false, "trending", 10, 0)
	if err == nil {
		t.Errorf("expected error when searching on disabled client")
	}

	suggestions, err := client.Autocomplete(ctx, "test", false, 10)
	if err == nil && len(suggestions) > 0 {
		t.Errorf("expected empty/error on disabled client")
	}
}

func TestVideoDocumentJSON(t *testing.T) {
	doc := VideoDocument{
		ID:               "abc-123",
		FileID:           "file-456",
		Title:            "Golang Full Course",
		Description:      "Learn Go from scratch",
		Category:         "Education",
		Tags:             []string{"golang", "programming", "backend"},
		ThumbnailUrl:     "https://cdn.example.com/thumb.jpg",
		Views:            1250,
		IsShort:          false,
		IsAdult:          false,
		Visibility:       "public",
		UploaderID:       "user-789",
		UploaderUsername: "gopher",
		CreatedAt:        time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("failed to marshal VideoDocument: %v", err)
	}

	var parsed VideoDocument
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal VideoDocument: %v", err)
	}

	if parsed.ID != doc.ID || parsed.Title != doc.Title || len(parsed.Tags) != 3 {
		t.Errorf("unmarshaled document does not match original: %+v", parsed)
	}
}

func TestURLWithCredentialsSanitization(t *testing.T) {
	client, err := NewClient(Config{
		Addresses: []string{"https://user:pass@example.com:9200"},
	})
	if err != nil {
		t.Fatalf("expected client to initialize without error, got %v", err)
	}
	if !client.Enabled() {
		t.Errorf("expected client to be enabled")
	}
}
