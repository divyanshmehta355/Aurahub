package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func TestIsPrivateOrLocalIP(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"127.0.0.1", true},
		{"127.0.0.1:8080", true},
		{"::1", true},
		{"[::1]:9000", true},
		{"10.0.0.1", true},
		{"172.18.0.1", true}, // Docker bridge
		{"192.168.1.100", true},
		{"203.0.113.1", false},
		{"8.8.8.8", false},
		{"invalid-ip", false},
	}

	for _, tt := range tests {
		got := isPrivateOrLocalIP(tt.ip)
		if got != tt.want {
			t.Errorf("isPrivateOrLocalIP(%q) = %v, want %v", tt.ip, got, tt.want)
		}
	}
}

func TestClientIPExtraction(t *testing.T) {
	app := fiber.New()
	var extractedIP string
	app.Get("/test-ip", func(c *fiber.Ctx) error {
		extractedIP = clientIP(c)
		return c.SendString(extractedIP)
	})

	tests := []struct {
		name     string
		headers  map[string]string
		remoteIP string
		wantIP   string
	}{
		{
			name:     "Direct connection default",
			headers:  map[string]string{},
			remoteIP: "192.0.2.1:1234",
			wantIP:   "0.0.0.0", // Fiber app.Test defaults to 0.0.0.0 for direct connection
		},
		{
			name: "Cloudflare CF-Connecting-IP header priority when peer is local/private",
			headers: map[string]string{
				"CF-Connecting-IP": "203.0.113.195",
				"X-Forwarded-For":  "198.51.100.1, 10.0.0.1",
				"X-Real-IP":        "198.51.100.2",
			},
			remoteIP: "172.18.0.1:5678",
			wantIP:   "203.0.113.195",
		},
		{
			name: "X-Forwarded-For multiple proxies takes first client IP when peer is local/private",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.50, 198.51.100.1, 10.0.0.5",
			},
			remoteIP: "10.0.0.5:8080",
			wantIP:   "203.0.113.50",
		},
		{
			name: "X-Real-IP fallback when no CF or XFF and peer is local/private",
			headers: map[string]string{
				"X-Real-IP": "198.51.100.77",
			},
			remoteIP: "10.0.0.1:443",
			wantIP:   "198.51.100.77",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test-ip", nil)
			req.RemoteAddr = tt.remoteIP
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if extractedIP != tt.wantIP {
				t.Errorf("clientIP() = %q, want %q", extractedIP, tt.wantIP)
			}
		})
	}
}

func TestRegisterLimiterEnforcement(t *testing.T) {
	app := fiber.New()
	app.Post("/register", RegisterLimiter(), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	clientIPAddr := "198.51.100.99"

	// 5 requests allowed
	for i := 1; i <= 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/register", nil)
		req.Header.Set("X-Real-IP", clientIPAddr)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("req %d failed: %v", i, err)
		}
		resp.Body.Close()
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("request %d got status %d, want 200", i, resp.StatusCode)
		}
	}

	// 6th request should hit rate limit
	req := httptest.NewRequest(http.MethodPost, "/register", nil)
	req.Header.Set("X-Real-IP", clientIPAddr)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("6th request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusTooManyRequests {
		t.Fatalf("expected 429 Too Many Requests, got %d", resp.StatusCode)
	}

	retryAfter := resp.Header.Get(fiber.HeaderRetryAfter)
	if retryAfter == "" {
		t.Errorf("expected Retry-After header on 429 response")
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode 429 response body: %v", err)
	}
	if body["error"] != "rate_limit_exceeded" {
		t.Errorf("got error %v, want 'rate_limit_exceeded'", body["error"])
	}
}

func TestSearchLimiter(t *testing.T) {
	app := fiber.New()
	app.Get("/search", SearchLimiter(), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	req.Header.Set("CF-Connecting-IP", "192.0.2.88")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
}

func TestRateLimitResponseFormat(t *testing.T) {
	app := fiber.New()
	app.Get("/limited", func(c *fiber.Ctx) error {
		return rateLimitResponse(c, "Action limit reached.", 30*time.Second)
	})

	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", resp.StatusCode)
	}

	if resp.Header.Get(fiber.HeaderRetryAfter) == "" {
		t.Errorf("expected Retry-After header to be set")
	}

	var res struct {
		Error      string `json:"error"`
		Message    string `json:"message"`
		RetryAfter int    `json:"retryAfter"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if res.Error != "rate_limit_exceeded" {
		t.Errorf("error = %q, want 'rate_limit_exceeded'", res.Error)
	}
	if res.RetryAfter <= 0 {
		t.Errorf("retryAfter = %d, want > 0", res.RetryAfter)
	}
}
