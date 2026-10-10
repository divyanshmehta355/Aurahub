package api

import (
	"time"

	"github.com/divyanshmehta355/aurahub/backend/internal/httpclient"
	"github.com/gofiber/fiber/v2"
)

func (s *Server) StreamVideoHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).SendString("Invalid file ID")
	}

	cacheKey := "stream_url:" + id

	// Check cache if Valkey/Redis is connected
	if s.Cache != nil {
		cachedUrl, err := s.Cache.Get(c.UserContext(), cacheKey).Result()
		if err == nil && cachedUrl != "" {
			return c.Redirect(cachedUrl, fiber.StatusFound)
		}
	}

	streamUrl, err := httpclient.ResolveStreamtapeDirectURL(c.UserContext(), id)
	if err != nil {
		return c.Status(502).SendString("Failed to resolve stream link: " + err.Error())
	}

	// Cache for 3 hours
	if s.Cache != nil {
		_ = s.Cache.Set(c.UserContext(), cacheKey, streamUrl, 3*time.Hour).Err()
	}

	return c.Redirect(streamUrl, fiber.StatusFound)
}
