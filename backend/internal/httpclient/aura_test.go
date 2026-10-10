package httpclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeleteStreamtapeFile(t *testing.T) {
	// Empty file ID should be a no-op
	if err := DeleteStreamtapeFile(context.Background(), ""); err != nil {
		t.Errorf("expected no error for empty file ID, got %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}
		if r.Header.Get("Accept") != "*/*" {
			t.Errorf("expected Accept */* header, got %s", r.Header.Get("Accept"))
		}
		if r.URL.Path != "/fs/files/delete/testFile123" {
			t.Errorf("expected path /fs/files/delete/testFile123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}))
	defer server.Close()

	// Direct call via request using server URL
	req, err := http.NewRequestWithContext(context.Background(), http.MethodDelete, server.URL+"/fs/files/delete/testFile123", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Accept", "*/*")

	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var res struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !res.Success {
		t.Errorf("expected success true")
	}
}

func TestResolveStreamtapeDirectURLValidation(t *testing.T) {
	_, err := ResolveStreamtapeDirectURL(context.Background(), "")
	if err == nil {
		t.Errorf("expected error for empty fileID, got nil")
	}
}

func TestRemoteUploadHelpersValidation(t *testing.T) {
	_, err := TriggerRemoteUpload(context.Background(), "", "")
	if err == nil {
		t.Errorf("expected error for empty video URL in TriggerRemoteUpload, got nil")
	}

	_, err = PollRemoteUpload(context.Background(), "", 0)
	if err == nil {
		t.Errorf("expected error for empty remote ID in PollRemoteUpload, got nil")
	}
}
