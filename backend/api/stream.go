package api

import (
	"context"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/gofiber/fiber/v2"
)

var robotLinkRegex = regexp.MustCompile(`document\.getElementById\(['"]robotlink['"]\)\.innerHTML\s*=\s*(.+?);`)

func (s *Server) StreamVideoHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).SendString("Invalid file ID")
	}

	cacheKey := "stream_url:" + id

	// Check cache
	cachedUrl, err := s.Cache.Get(context.Background(), cacheKey).Result()
	if err == nil && cachedUrl != "" {
		return c.Redirect(cachedUrl, fiber.StatusFound)
	}

	embedUrl := "https://streamtape.com/e/" + id
	req, _ := http.NewRequest("GET", embedUrl, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return c.Status(502).SendString("Stream source unavailable")
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	html := string(bodyBytes)

	matches := robotLinkRegex.FindStringSubmatch(html)
	if len(matches) < 2 {
		return c.Status(502).SendString("Failed to resolve stream link")
	}

	rawExpr := matches[1]

	vm := goja.New()
	val, err := vm.RunString(rawExpr)
	if err != nil {
		return c.Status(500).SendString("Error parsing stream URL")
	}

	streamUrl := val.String()
	if strings.HasPrefix(streamUrl, "//") {
		streamUrl = "https:" + streamUrl
	}

	if !strings.HasPrefix(streamUrl, "http") {
		return c.Status(500).SendString("Invalid stream URL resolved")
	}

	// Cache for 3 hours
	s.Cache.Set(context.Background(), cacheKey, streamUrl, 3*time.Hour)

	return c.Redirect(streamUrl, fiber.StatusFound)
}
