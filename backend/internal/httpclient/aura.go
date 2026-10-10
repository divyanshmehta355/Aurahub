package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/dop251/goja"
)

const AuraApiBaseUrl = "https://aurahub-api-hono.ashwathama249.workers.dev"

var robotLinkRegex = regexp.MustCompile(`document\.getElementById\(['"]robotlink['"]\)\.innerHTML\s*=\s*(.+?);`)

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

// ResolveStreamtapeDirectURL resolves the direct downloadable media stream URL from a Streamtape file ID.
// It first attempts the official ticket+link endpoint through the Aura Cloudflare worker (which avoids ISP-level blocks),
// falling back to direct embed player scraping.
func ResolveStreamtapeDirectURL(ctx context.Context, fileID string) (string, error) {
	cleanID := strings.TrimSpace(fileID)
	if cleanID == "" {
		return "", fmt.Errorf("empty file ID")
	}

	// 1. Try official ticket flow through upstream Cloudflare worker
	ticketBody, err := RequestAuraAPI(ctx, http.MethodGet, "/stream/ticket/"+cleanID, nil)
	if err == nil {
		var ticketRes struct {
			Ticket   string `json:"ticket"`
			WaitTime int    `json:"wait_time"`
		}
		if json.Unmarshal(ticketBody, &ticketRes) == nil && ticketRes.Ticket != "" {
			if ticketRes.WaitTime > 0 {
				select {
				case <-ctx.Done():
					return "", ctx.Err()
				case <-time.After(time.Duration(ticketRes.WaitTime) * time.Second):
				}
			}

			q := url.Values{}
			q.Set("file_id", cleanID)
			q.Set("ticket", ticketRes.Ticket)

			linkBody, err := RequestAuraAPI(ctx, http.MethodGet, "/stream/link", q)
			if err == nil {
				var linkRes struct {
					URL string `json:"url"`
				}
				if json.Unmarshal(linkBody, &linkRes) == nil && strings.HasPrefix(linkRes.URL, "http") {
					return linkRes.URL, nil
				}
			}
		}
	}

	// 2. Fallback to direct embed player resolution
	embedURL := "https://streamtape.com/e/" + cleanID
	req, err := http.NewRequestWithContext(ctx, "GET", embedURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to build stream request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := Default.Do(req)
	if err != nil {
		return "", fmt.Errorf("stream source unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("file not found on streamtape (404)")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("stream source returned status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("failed to read stream response: %w", err)
	}
	html := string(bodyBytes)

	matches := robotLinkRegex.FindStringSubmatch(html)
	if len(matches) < 2 {
		return "", fmt.Errorf("robotlink expression not found in embed HTML (file may be deleted or blocked)")
	}

	vm := goja.New()
	val, err := vm.RunString(matches[1])
	if err != nil {
		return "", fmt.Errorf("error evaluating robotlink expression: %w", err)
	}

	streamURL := val.String()
	if strings.HasPrefix(streamURL, "//") {
		streamURL = "https:" + streamURL
	}
	if !strings.HasPrefix(streamURL, "http") {
		return "", fmt.Errorf("invalid stream URL resolved: %s", streamURL)
	}

	return streamURL, nil
}

// TriggerRemoteUpload initiates a remote upload on Streamtape via the Aura API worker.
// Returns the remote task ID for polling.
func TriggerRemoteUpload(ctx context.Context, videoURL, folderID string) (string, error) {
	cleanURL := strings.TrimSpace(videoURL)
	if cleanURL == "" {
		return "", fmt.Errorf("video URL is required")
	}

	query := url.Values{}
	query.Set("url", cleanURL)
	if folderID != "" {
		query.Set("folder", folderID)
	}

	body, err := RequestAuraAPI(ctx, http.MethodGet, "/remote/add", query)
	if err != nil {
		return "", fmt.Errorf("failed to start remote upload: %w", err)
	}

	var res struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &res); err != nil || res.ID == "" {
		return "", fmt.Errorf("invalid response from remote upload: %s", string(body))
	}

	return res.ID, nil
}

// PollRemoteUpload polls the remote upload status until completion or failure.
// Returns the resulting new Streamtape file ID (linkid).
func PollRemoteUpload(ctx context.Context, remoteID string, maxWait time.Duration) (string, error) {
	if strings.TrimSpace(remoteID) == "" {
		return "", fmt.Errorf("remote ID is required")
	}

	if maxWait <= 0 {
		maxWait = 10 * time.Minute
	}

	pollCtx, cancel := context.WithTimeout(ctx, maxWait)
	defer cancel()

	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()

	failedAttempts := 0

	for {
		select {
		case <-pollCtx.Done():
			return "", pollCtx.Err()
		case <-ticker.C:
			query := url.Values{}
			query.Set("id", remoteID)

			body, err := RequestAuraAPI(pollCtx, http.MethodGet, "/remote/status", query)
			if err != nil {
				failedAttempts++
				if failedAttempts > 6 {
					return "", fmt.Errorf("lost connection to remote upload status: %w", err)
				}
				continue
			}
			failedAttempts = 0

			var statusMap map[string]struct {
				Status       string `json:"status"`
				LinkID       string `json:"linkid"`
				BytesLoaded  int64  `json:"bytes_loaded"`
				BytesTotal   int64  `json:"bytes_total"`
				ErrorMessage string `json:"error_message"`
			}
			if err := json.Unmarshal(body, &statusMap); err != nil {
				continue
			}

			statusData, exists := statusMap[remoteID]
			if !exists {
				continue
			}

			switch statusData.Status {
			case "finished":
				if statusData.LinkID == "" {
					return "", fmt.Errorf("remote upload finished but linkid was empty")
				}
				return statusData.LinkID, nil
			case "error":
				errMsg := statusData.ErrorMessage
				if errMsg == "" {
					errMsg = "remote download failed on upstream server"
				}
				return "", fmt.Errorf("%s", errMsg)
			}
		}
	}
}
