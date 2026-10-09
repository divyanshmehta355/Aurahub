package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/divyanshmehta355/aurahub/backend/internal/kafka"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Server) AutocompleteHandler(c *fiber.Ctx) error {
	query := strings.TrimSpace(c.Query("q"))
	showAdult := c.Query("adult") == "true"
	if len([]rune(query)) < 2 {
		return c.JSON(fiber.Map{"videos": []fiber.Map{}, "users": []fiber.Map{}})
	}

	cacheKey := fmt.Sprintf("autocomplete:%s:adult:%t", strings.ToLower(query), showAdult)
	if s.Cache != nil {
		if cached, err := s.Cache.Get(c.UserContext(), cacheKey).Result(); err == nil && cached != "" {
			c.Set("Content-Type", "application/json")
			c.Set("X-Cache", "HIT")
			return c.SendString(cached)
		}
	}

	videos, err := s.Repository.SearchAutocompleteVideos(context.Background(), query, showAdult)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to search videos"})
	}
	users, err := s.Repository.SearchAutocompleteUsers(context.Background(), query)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to search users"})
	}

	videoResults := make([]fiber.Map, 0, len(videos))
	for _, video := range videos {
		videoResults = append(videoResults, fiber.Map{
			"id":           formatUUID(video.ID),
			"title":        video.Title,
			"thumbnailUrl": video.ThumbnailUrl.String,
			"category":     video.Category.String,
		})
	}
	userResults := make([]fiber.Map, 0, len(users))
	for _, user := range users {
		userResults = append(userResults, fiber.Map{
			"id":       formatUUID(user.ID),
			"username": user.Username,
			"avatar":   user.Avatar.String,
		})
	}

	resMap := fiber.Map{"videos": videoResults, "users": userResults}
	if s.Cache != nil {
		if data, err := json.Marshal(resMap); err == nil {
			_ = s.Cache.Set(c.UserContext(), cacheKey, string(data), 5*time.Minute).Err()
		}
	}

	c.Set("X-Cache", "MISS")
	return c.JSON(resMap)
}

func parseUUID(idStr string) (pgtype.UUID, error) {
	var u pgtype.UUID
	err := u.Scan(idStr)
	return u, err
}

func (s *Server) resolveVideoUUID(ctx context.Context, idStr string) (pgtype.UUID, error) {
	if id, err := parseUUID(idStr); err == nil {
		return id, nil
	}
	video, err := s.Repository.GetVideoByFileId(ctx, idStr)
	if err == nil {
		return video.ID, nil
	}
	return pgtype.UUID{}, err
}

func (s *Server) ListVideosHandler(c *fiber.Ctx) error {
	page, limit := pageLimit(c, 12, 100)
	showAdult := c.Query("adult") == "true"
	category := c.Query("category")
	if category == "All" {
		category = ""
	}
	videoType := c.Query("type", "video")
	sortType := c.Query("sort", "trending")

	cacheKey := fmt.Sprintf("feed:videos:cat=%s:type=%s:adult=%t:sort=%s:p=%d:l=%d",
		category, videoType, showAdult, sortType, page, limit)

	// Microcache first 2 pages for 60 seconds
	if s.Cache != nil && page <= 2 {
		if cached, err := s.Cache.Get(c.UserContext(), cacheKey).Result(); err == nil && cached != "" {
			c.Set("Content-Type", "application/json")
			c.Set("X-Cache", "HIT")
			return c.SendString(cached)
		}
	}

	var shortFilter pgtype.Bool
	if videoType == "short" {
		shortFilter = pgtype.Bool{Bool: true, Valid: true}
	} else {
		// Default to standard videos only; shorts are only served to the shorts section
		shortFilter = pgtype.Bool{Bool: false, Valid: true}
	}
	videos, err := s.Repository.ListPublicVideos(context.Background(), category, shortFilter, showAdult, sortType, int32(limit), int32((page-1)*limit))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch videos"})
	}
	total, err := s.Repository.CountPublicVideos(context.Background(), category, shortFilter, showAdult)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to count videos"})
	}
	res := make([]fiber.Map, 0, len(videos))
	for _, v := range videos {
		res = append(res, fiber.Map{
			"id":           formatUUID(v.ID),
			"fileId":       v.FileID,
			"title":        v.Title,
			"description":  v.Description.String,
			"thumbnailUrl": v.ThumbnailUrl.String,
			"views":        v.Views.Int32,
			"isShort":      v.IsShort.Bool,
			"visibility":   v.Visibility.VideoVisibility,
			"category":     v.Category.String,
			"createdAt":    v.CreatedAt.Time,
		})
	}

	respMap := fiber.Map{
		"videos":      res,
		"page":        page,
		"currentPage": page,
		"limit":       limit,
		"totalVideos": total,
		"totalPages":  (total + int64(limit) - 1) / int64(limit),
	}

	if s.Cache != nil && page <= 2 {
		if data, err := json.Marshal(respMap); err == nil {
			_ = s.Cache.Set(c.UserContext(), cacheKey, string(data), 60*time.Second).Err()
		}
	}

	c.Set("X-Cache", "MISS")
	return c.JSON(respMap)
}

func (s *Server) GetVideoHandler(c *fiber.Ctx) error {
	idStr := strings.TrimSpace(c.Params("id"))
	if idStr == "" {
		return c.Status(400).JSON(fiber.Map{"message": "Video ID is required"})
	}

	var viewerUUID pgtype.UUID
	userIdLocal := c.Locals("userId")
	if userID, ok := userIdLocal.(string); ok {
		viewerUUID, _ = parseUUID(userID)
	}

	detail, err := s.Repository.GetVideoDetails(c.UserContext(), idStr, viewerUUID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"message": "Video not found"})
	}

	isUploader := viewerUUID.Valid && viewerUUID == detail.UploaderID
	if detail.Visibility.VideoVisibility == db.VideoVisibilityPrivate && !isUploader {
		return c.Status(403).JSON(fiber.Map{"message": "This video is private"})
	}

	return c.JSON(fiber.Map{
		"id":            formatUUID(detail.ID),
		"title":         detail.Title,
		"description":   detail.Description.String,
		"thumbnailUrl":  detail.ThumbnailUrl.String,
		"streamtapeUrl": detail.StreamtapeUrl.String,
		"fileId":        detail.FileID,
		"views":         detail.Views.Int32,
		"createdAt":     detail.CreatedAt.Time,
		"likesCount":    detail.LikesCount,
		"commentCount":  detail.CommentCount,
		"isLiked":       detail.IsLiked,
		"isAdult":       detail.IsAdult.Bool,
		"uploader": fiber.Map{
			"id":               formatUUID(detail.UploaderID),
			"username":         detail.UploaderUsername,
			"avatar":           detail.UploaderAvatar.String,
			"subscribersCount": detail.SubscribersCount,
			"isSubscribed":     detail.IsSubscribed,
		},
	})
}

func (s *Server) DeleteVideoHandler(c *fiber.Ctx) error {
	idStr := c.Params("id")
	videoId, err := s.resolveVideoUUID(context.Background(), idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid video ID."})
	}

	userIdLocal := c.Locals("userId")
	if userIdLocal == nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}

	video, err := s.Repository.GetVideo(context.Background(), videoId)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"message": "Video not found"})
	}

	if userIdLocal.(string) != formatUUID(video.UploaderID) {
		return c.Status(403).JSON(fiber.Map{"message": "Not authorized to delete this video"})
	}

	response, err := requestAuraAPI(c.UserContext(), http.MethodDelete, "/fs/files/delete/"+url.PathEscape(video.FileID), nil)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"message": "Failed to delete video from storage. The video was not removed."})
	}
	var deletion struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(response, &deletion); err != nil || !deletion.Success {
		return c.Status(502).JSON(fiber.Map{"message": "Storage did not confirm deletion. The video was not removed."})
	}

	err = s.Repository.DeleteVideo(context.Background(), videoId)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to delete video"})
	}
	if s.Cache != nil {
		if err := s.Cache.Del(context.Background(), "thumbnail:"+idStr, "stream:"+idStr).Err(); err != nil {
			log.Printf("Cache invalidation failed after deleting video %s: %v", idStr, err)
		}
		s.invalidateFeedCache(context.Background())
	}

	return c.JSON(fiber.Map{"message": "Video deleted successfully"})
}

// invalidateFeedCache clears cached public feeds and search entries on upload or deletion
func (s *Server) invalidateFeedCache(ctx context.Context) {
	if s.Cache == nil {
		return
	}
	iter := s.Cache.Scan(ctx, 0, "feed:videos:*", 100).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
		if len(keys) >= 100 {
			_ = s.Cache.Del(ctx, keys...).Err()
			keys = keys[:0]
		}
	}
	if len(keys) > 0 {
		_ = s.Cache.Del(ctx, keys...).Err()
	}
}

func (s *Server) SearchVideosHandler(c *fiber.Ctx) error {
	q := strings.TrimSpace(c.Query("q", ""))
	showAdult := c.Query("adult") == "true"
	if q == "" {
		return c.JSON(fiber.Map{"videos": []fiber.Map{}, "query": q, "total": 0})
	}
	sortType := c.Query("sort", "relevance")
	cacheKey := fmt.Sprintf("search:q=%s:adult=%t:sort=%s", strings.ToLower(q), showAdult, sortType)

	if s.Cache != nil {
		if cached, err := s.Cache.Get(c.UserContext(), cacheKey).Result(); err == nil && cached != "" {
			c.Set("Content-Type", "application/json")
			c.Set("X-Cache", "HIT")
			return c.SendString(cached)
		}
	}

	videos, err := s.Repository.SearchPublicVideos(context.Background(), q, showAdult, sortType, 30)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to search videos"})
	}

	res := make([]fiber.Map, 0, len(videos))
	for _, v := range videos {
		res = append(res, fiber.Map{
			"id":           formatUUID(v.ID),
			"fileId":       v.FileID,
			"title":        v.Title,
			"description":  v.Description.String,
			"thumbnailUrl": v.ThumbnailUrl.String,
			"views":        v.Views.Int32,
			"isShort":      v.IsShort.Bool,
			"category":     v.Category.String,
			"createdAt":    v.CreatedAt.Time,
		})
	}

	respMap := fiber.Map{"videos": res, "query": q, "total": len(res)}
	if s.Cache != nil {
		if data, err := json.Marshal(respMap); err == nil {
			_ = s.Cache.Set(c.UserContext(), cacheKey, string(data), 2*time.Minute).Err()
		}
	}

	c.Set("X-Cache", "MISS")
	return c.JSON(respMap)
}

func pageLimit(c *fiber.Ctx, defaultLimit, maxLimit int) (int, int) {
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(c.Query("limit", strconv.Itoa(defaultLimit)))
	if err != nil || limit < 1 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return page, limit
}

func (s *Server) RecordVideoViewHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	var video db.Video
	var err error

	// If the ID is a valid UUID, look up directly to prevent slow fallback queries
	if videoUUID, parseErr := parseUUID(id); parseErr == nil {
		video, err = s.Repository.GetVideo(c.UserContext(), videoUUID)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"message": "Video not found"})
		}
	} else {
		video, err = s.Repository.GetVideoByFileId(c.UserContext(), id)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"message": "Video not found"})
		}
	}

	var userIDStr string
	if userIDValue := c.Locals("userId"); userIDValue != nil {
		if uid, ok := userIDValue.(string); ok {
			userIDStr = uid
		}
	}

	// High-throughput Kafka Stream: emit view event non-blocking and return immediately (< 2ms)
	if s.Kafka != nil {
		videoIDStr := formatUUID(video.ID)
		go func(vID, uID string) {
			pubCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.Kafka.Publish(pubCtx, kafka.TopicVideoViews, vID, kafka.VideoViewEvent{
				VideoID:   vID,
				UserID:    uID,
				Timestamp: time.Now(),
			})
		}(videoIDStr, userIDStr)
		return c.JSON(fiber.Map{"success": true})
	}

	// Standby / Fallback path if Kafka is not running
	ctx := c.UserContext()
	if err := s.Repository.IncrementVideoViews(ctx, video.ID); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Could not count view."})
	}

	if userIDStr != "" {
		if userUUID, err := parseUUID(userIDStr); err == nil {
			_ = s.Repository.UpsertVideoViewActivity(ctx, userUUID, video.ID)
			_ = s.Repository.UpsertWatchHistory(ctx, db.UpsertWatchHistoryParams{
				UserID:  userUUID,
				VideoID: video.ID,
			})
		}
	}

	return c.JSON(fiber.Map{"success": true})
}

func (s *Server) SubscriptionFeedHandler(c *fiber.Ctx) error {
	userID, ok := c.Locals("userId").(string)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userIDUUID, err := parseUUID(userID)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	page, limit := pageLimit(c, 10, 100)
	ctx := context.Background()
	total, err := s.Repository.CountSubscriptionFeed(ctx, userIDUUID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to count subscription feed"})
	}
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	if totalPages == 0 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	videos, err := s.Repository.GetSubscriptionFeed(ctx, db.GetSubscriptionFeedParams{
		SubscriberID: userIDUUID,
		Limit:        int32(limit),
		Offset:       int32((page - 1) * limit),
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch feed"})
	}

	var res []fiber.Map
	for _, v := range videos {
		res = append(res, fiber.Map{
			"id":           formatUUID(v.ID),
			"fileId":       v.FileID,
			"title":        v.Title,
			"thumbnailUrl": v.ThumbnailUrl.String,
			"views":        v.Views.Int32,
			"isShort":      v.IsShort.Bool,
			"createdAt":    v.CreatedAt.Time,
		})
	}

	return c.JSON(fiber.Map{
		"videos":      res,
		"currentPage": page,
		"totalPages":  totalPages,
		"totalVideos": total,
	})
}

type UpdateVideoRequest struct {
	Title       string          `json:"title"`
	Description *string         `json:"description"`
	Visibility  *string         `json:"visibility"`
	Category    *string         `json:"category"`
	Tags        json.RawMessage `json:"tags"`
	IsAdult     *bool           `json:"isAdult"`
}

func (s *Server) UpdateVideoHandler(c *fiber.Ctx) error {
	videoIdStr := c.Params("id")
	videoId, err := s.resolveVideoUUID(context.Background(), videoIdStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid video ID"})
	}

	var req UpdateVideoRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid payload"})
	}

	userIdLocal := c.Locals("userId")
	userId, _ := parseUUID(userIdLocal.(string))

	video, err := s.Repository.GetVideo(context.Background(), videoId)
	if err != nil || video.UploaderID != userId {
		return c.Status(403).JSON(fiber.Map{"message": "Forbidden"})
	}

	update := db.UpdateVideoMetadataParams{ID: videoId}
	if req.Title != "" {
		update.Title = pgtype.Text{String: req.Title, Valid: true}
	}
	if req.Description != nil {
		update.Description = pgtype.Text{String: *req.Description, Valid: true}
	}
	if req.Visibility != nil && *req.Visibility != "" {
		visibility := db.VideoVisibility(*req.Visibility)
		if visibility != db.VideoVisibilityPublic && visibility != db.VideoVisibilityUnlisted && visibility != db.VideoVisibilityPrivate {
			return c.Status(400).JSON(fiber.Map{"message": "Invalid visibility"})
		}
		update.Visibility = db.NullVideoVisibility{VideoVisibility: visibility, Valid: true}
	}
	if req.Category != nil {
		update.Category = pgtype.Text{String: *req.Category, Valid: true}
	}
	if len(req.Tags) > 0 && string(req.Tags) != "null" {
		var tags []string
		if err := json.Unmarshal(req.Tags, &tags); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "Tags must be an array of strings"})
		}
		seen := make(map[string]struct{}, len(tags))
		update.Tags = make([]string, 0, len(tags))
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}
			if _, exists := seen[tag]; !exists {
				seen[tag] = struct{}{}
				update.Tags = append(update.Tags, tag)
			}
		}
		update.ReplaceTags = true
	}
	if req.IsAdult != nil {
		update.IsAdult = pgtype.Bool{Bool: *req.IsAdult, Valid: true}
	}

	updated, err := s.Repository.UpdateVideoMetadata(context.Background(), update)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Update failed"})
	}

	return c.JSON(fiber.Map{
		"id":          formatUUID(updated.ID),
		"title":       updated.Title,
		"description": updated.Description.String,
		"visibility":  updated.Visibility.VideoVisibility,
		"category":    updated.Category.String,
		"tags":        updated.Tags,
	})
}
