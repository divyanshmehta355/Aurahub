package tests

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/divyanshmehta355/aurahub/backend/api"
	"github.com/gofiber/fiber/v2"
)

func TestRecommendationsEndpoints(t *testing.T) {
	app := fiber.New()
	api.NewServer(nil, nil).SetupRoutes(app)

	t.Run("GET /api/videos returns recommended feed with seed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/videos?seed=test-session-123", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if result["seed"] != "test-session-123" {
			t.Errorf("expected seed 'test-session-123', got %v", result["seed"])
		}
	})

	t.Run("GET /api/videos/recommendations returns 200 OK with seed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/videos/recommendations?seed=rotate-seed-456", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if result["seed"] != "rotate-seed-456" {
			t.Errorf("expected seed 'rotate-seed-456', got %v", result["seed"])
		}
		if result["isAiVectorSearch"] != false {
			t.Errorf("expected isAiVectorSearch to be false (no AI), got %v", result["isAiVectorSearch"])
		}
	})

	t.Run("GET /api/videos/suggestions returns 200 OK with seed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/videos/suggestions?seed=sidebar-seed-789", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if result["seed"] != "sidebar-seed-789" {
			t.Errorf("expected seed 'sidebar-seed-789', got %v", result["seed"])
		}
		if result["isAiVectorSearch"] != false {
			t.Errorf("expected isAiVectorSearch to be false (no AI), got %v", result["isAiVectorSearch"])
		}
	})
}
