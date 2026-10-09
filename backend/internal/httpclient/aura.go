package httpclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
		return nil, fmt.Errorf("upstream upload service returned error (status %d)", response.StatusCode)
	}
	return body, nil
}
