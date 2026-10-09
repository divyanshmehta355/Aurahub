package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const AuraApiBaseUrl = "https://aurahub-api-hono.ashwathama249.workers.dev"

// RequestAuraAPI performs requests to the upstream Aura/Streamtape integration worker.
func RequestAuraAPI(ctx context.Context, method, path string, query url.Values) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	requestURL := AuraApiBaseUrl + path
	if len(query) > 0 {
		requestURL += "?" + query.Encode()
	}
	request, err := http.NewRequestWithContext(ctx, method, requestURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "*/*")
	response, err := Default.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("upstream upload service returned error (status %d): %s", response.StatusCode, string(body))
	}
	return body, nil
}

// DeleteStreamtapeFile deletes a video file from Streamtape via the upstream Aura API worker.
// API Reference: DELETE https://aurahub-api-hono.ashwathama249.workers.dev/fs/files/delete/{fileID}
// Header: Accept: */*
// Response: {"success": true}
func DeleteStreamtapeFile(ctx context.Context, fileID string) error {
	cleanID := strings.TrimSpace(fileID)
	if cleanID == "" {
		return nil
	}

	path := "/fs/files/delete/" + url.PathEscape(cleanID)
	body, err := RequestAuraAPI(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to delete file %s from streamtape: %w", cleanID, err)
	}

	var res struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return fmt.Errorf("failed to parse streamtape delete response for %s: %w (body: %s)", cleanID, err, string(body))
	}

	if !res.Success {
		return fmt.Errorf("streamtape storage did not confirm deletion for file %s", cleanID)
	}

	return nil
}
