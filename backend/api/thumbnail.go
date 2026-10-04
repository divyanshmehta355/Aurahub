package api

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

const AURA_API_BASE_URL = "https://aurahub-api-hono.ashwathama249.workers.dev"

func (s *Server) DynamicThumbnailHandler(c *fiber.Ctx) error {
	seed := c.Params("seed")
	title := c.Query("title")
	if title == "" {
		title = seed
	}
	if title == "" || title == "default" {
		title = "Aurahub Video"
	}
	category := c.Query("category")
	if category == "" {
		category = "Video"
	}

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1280 720">
<defs><linearGradient id="bg" x1="0" y1="0" x2="1" y2="1"><stop stop-color="#101827"/><stop offset="1" stop-color="#4f46e5"/></linearGradient></defs>
<rect width="1280" height="720" fill="url(#bg)"/>
<circle cx="1080" cy="130" r="220" fill="#fff" opacity=".06"/>
<text x="80" y="510" fill="#fff" font-family="Arial,sans-serif" font-size="26" opacity=".8">%s</text>
<text x="80" y="590" fill="#fff" font-family="Arial,sans-serif" font-size="54" font-weight="700">%s</text>
</svg>`, html.EscapeString(strings.ToUpper(category)), html.EscapeString(title))
	c.Set("Content-Type", "image/svg+xml; charset=utf-8")
	c.Set("Cache-Control", "public, max-age=31536000, immutable")
	return c.SendString(svg)
}

func (s *Server) GetVideoThumbnailHandler(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid video ID"})
	}

	cacheKey := "thumbnail:" + idStr

	// Check cache
	var cachedURL string
	var err error
	if s.Cache != nil {
		cachedURL, err = s.Cache.Get(context.Background(), cacheKey).Result()
		if err != nil && err != redis.Nil {
			log.Printf("Thumbnail cache lookup failed for %s: %v", idStr, err)
		}
	}
	if err == nil && cachedURL != "" {
		return c.JSON(fiber.Map{"thumbnailUrl": cachedURL})
	}

	var video db.Video
	parsedID, parseErr := parseUUID(idStr)

	if parseErr == nil {
		video, err = s.Repository.GetVideo(context.Background(), parsedID)
	} else {
		video, err = s.Repository.GetVideoByFileId(context.Background(), idStr)
	}

	if err != nil {
		return c.Status(404).JSON(fiber.Map{"message": "Video not found"})
	}

	finalUrl := ""

	if video.ThumbnailUrl.String != "" {
		finalUrl = video.ThumbnailUrl.String
	} else if video.FileID != "" {
		// Try fetching from Aura Worker
		reqUrl := AURA_API_BASE_URL + "/fs/files/thumbnail/" + video.FileID
		client := &http.Client{Timeout: 5 * time.Second}
		resp, apiErr := client.Get(reqUrl)
		if apiErr == nil && resp.StatusCode == 200 {
			var respData struct {
				ThumbnailUrl string `json:"thumbnail_url"`
			}
			json.NewDecoder(resp.Body).Decode(&respData)
			if respData.ThumbnailUrl != "" {
				finalUrl = respData.ThumbnailUrl
			}
			resp.Body.Close()
		}
	}

	// Fallback to dynamic SVG
	if finalUrl == "" {
		seed := url.QueryEscape(video.FileID)
		if seed == "" {
			seed = url.QueryEscape(idStr)
		}

		q := url.Values{}
		if video.Title != "" {
			q.Set("title", video.Title)
		}
		if video.Category.String != "" {
			q.Set("category", video.Category.String)
		}

		queryString := q.Encode()
		if queryString != "" {
			finalUrl = "/api/thumbnail/" + seed + "?" + queryString
		} else {
			finalUrl = "/api/thumbnail/" + seed
		}
	}

	// Cache for 24 hours
	if s.Cache != nil {
		if err := s.Cache.Set(context.Background(), cacheKey, finalUrl, 24*time.Hour).Err(); err != nil {
			log.Printf("Thumbnail cache write failed for %s: %v", idStr, err)
		}
	}

	return c.JSON(fiber.Map{"thumbnailUrl": finalUrl})
}
