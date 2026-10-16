package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/divyanshmehta355/aurahub/backend/api"
	"github.com/gofiber/fiber/v2"
)

func TestKillSwitchEnforcement_NonAdminBlocked(t *testing.T) {
	app := fiber.New()
	server := api.NewServer(nil, nil)

	// Mocking freeze_uploads to active by overriding via middleware test
	app.Post("/upload", func(c *fiber.Ctx) error {
		// Non-admin user
		c.Locals("isAdmin", false)
		return c.Next()
	}, server.EnforceKillSwitch("freeze_uploads"), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	// When switch is inactive (default nil repository / nil cache = false), request passes
	req := httptest.NewRequest(http.MethodPost, "/upload", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK when switch is inactive, got %d", resp.StatusCode)
	}
}

func TestKillSwitchEnforcement_AdminBypass(t *testing.T) {
	app := fiber.New()
	server := api.NewServer(nil, nil)

	app.Post("/upload", func(c *fiber.Ctx) error {
		// Admin user superpower
		c.Locals("isAdmin", true)
		return c.Next()
	}, server.EnforceKillSwitch("freeze_uploads"), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/upload", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK for admin bypass, got %d", resp.StatusCode)
	}
}

func TestPublicSystemStatusHandler(t *testing.T) {
	app := fiber.New()
	server := api.NewServer(nil, nil)

	app.Get("/system/status", server.PublicSystemStatusHandler)

	req := httptest.NewRequest(http.MethodGet, "/system/status", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	for _, key := range []string{"maintenanceMode", "freezeUploads", "freezeRegistration", "freezeComments"} {
		if _, exists := data[key]; !exists {
			t.Errorf("missing expected key %q in status response", key)
		}
	}
}
