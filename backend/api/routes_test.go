package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestUnifiedAPIRoutes(t *testing.T) {
	app := fiber.New()
	NewServer(nil, nil).SetupRoutes(app)

	registered := make(map[string]bool)
	for _, route := range app.GetRoutes() {
		registered[route.Method+" "+route.Path] = true
	}
	apiRoutes := []string{
		"GET /api/health",
		"GET /api/videos/:id",
		"GET /api/comments/:videoId",
		"POST /api/videos/:id/view",
		"GET /api/users/:id",
		"POST /api/users/:identifier/subscribe",
		"DELETE /api/user/watch-later",
		"GET /api/comments/:id/replies",
		"GET /api/search/autocomplete",
		"GET /api/videos/recommendations",
		"GET /api/videos/suggestions",
		"POST /api/videos/remote-upload/start",
		"GET /api/videos/remote-upload/status",
		"POST /api/videos/:id/update-thumbnail",
		"GET /api/thumbnail/:seed",
		"PUT /api/playlists/",
		"PUT /api/videos/bulk",
		"PUT /api/videos/bulk-adult",
		"PUT /api/creator/videos/bulk-adult",
	}
	for _, route := range apiRoutes {
		if !registered[route] {
			t.Errorf("API route %q is not registered", route)
		}
	}

	tests := []struct {
		method string
		path   string
		status int
	}{
		{http.MethodGet, "/api/health", http.StatusOK},
		{http.MethodGet, "/api/search/autocomplete?q=a", http.StatusOK},
		{http.MethodGet, "/api/thumbnail/sample?title=Sample", http.StatusOK},
		{http.MethodGet, "/api/videos/remote-upload/status", http.StatusBadRequest},
		{http.MethodGet, "/api/user/watch-later", http.StatusUnauthorized},
		{http.MethodPost, "/api/videos/sample/like", http.StatusUnauthorized},
		{http.MethodPost, "/api/videos/remote-upload/start", http.StatusUnauthorized},
	}

	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, nil)
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()
			if response.StatusCode != test.status {
				t.Fatalf("got status %d, want %d", response.StatusCode, test.status)
			}
			if test.path == "/api/thumbnail/sample?title=Sample" {
				contentType := response.Header.Get("Content-Type")
				if !strings.Contains(contentType, "image/svg+xml") {
					t.Fatalf("unexpected thumbnail content type %q", contentType)
				}
			}
		})
	}
}
