package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Server) AutocompleteHandler(c *fiber.Ctx) error {
	query := strings.TrimSpace(c.Query("q"))
	showAdult := c.Query("adult") == "true"
	if len([]rune(query)) < 2 {
		return c.JSON(fiber.Map{"videos": []fiber.Map{}, "users": []fiber.Map{}})
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
	return c.JSON(fiber.Map{"videos": videoResults, "users": userResults})
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
	var shortFilter pgtype.Bool
	if c.Query("type") == "short" {
		shortFilter = pgtype.Bool{Bool: true, Valid: true}
	} else {
		// Default to standard videos only; shorts are only served to the shorts section
		shortFilter = pgtype.Bool{Bool: false, Valid: true}
	}
	videos, err := s.Repository.ListPublicVideos(context.Background(), category, shortFilter, showAdult, c.Query("sort", "trending"), int32(limit), int32((page-1)*limit))
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

	return c.JSON(fiber.Map{
		"videos":      res,
		"page":        page,
		"currentPage": page,
		"limit":       limit,
		"totalVideos": total,
		"totalPages":  (total + int64(limit) - 1) / int64(limit),
	})
}

func (s *Server) GetVideoHandler(c *fiber.Ctx) error {
	idStr := c.Params("id")

	// Try getting by FileID first (short ID)
	video, err := s.Repository.GetVideoByFileId(context.Background(), idStr)
	if err != nil {
		// Fallback to UUID in case old URLs are hit
		videoId, parseErr := parseUUID(idStr)
		if parseErr != nil {
			return c.Status(400).JSON(fiber.Map{"message": "Invalid video ID format."})
		}
		video, err = s.Repository.GetVideo(context.Background(), videoId)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"message": "Video not found"})
		}
	}
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"message": "Video not found"})
	}

	uploader, err := s.Repository.GetUser(context.Background(), video.UploaderID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch video owner"})
	}

	userIdLocal := c.Locals("userId")
	isUploader := false
	if userID, ok := userIdLocal.(string); ok && userID == formatUUID(video.UploaderID) {
		isUploader = true
	}

	if video.Visibility.VideoVisibility == db.VideoVisibilityPrivate && !isUploader {
		return c.Status(403).JSON(fiber.Map{"message": "This video is private"})
	}
	likesCount, err := s.Repository.CountVideoLikes(context.Background(), video.ID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch likes"})
	}
	commentCount, err := s.Repository.CountVideoComments(context.Background(), video.ID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch comments"})
	}
	isLiked := false
	if userID, ok := userIdLocal.(string); ok {
		viewerID, parseErr := parseUUID(userID)
		if parseErr == nil {
			isLiked, err = s.Repository.HasUserLikedVideo(context.Background(), viewerID, video.ID)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch like status"})
			}
		}
	}

	return c.JSON(fiber.Map{
		"id":            formatUUID(video.ID),
		"title":         video.Title,
		"description":   video.Description.String,
		"thumbnailUrl":  video.ThumbnailUrl.String,
		"streamtapeUrl": video.StreamtapeUrl.String,
		"fileId":        video.FileID,
		"views":         video.Views.Int32,
		"createdAt":     video.CreatedAt.Time,
		"likesCount":    likesCount,
		"commentCount":  commentCount,
		"isLiked":       isLiked,
		"isAdult":       video.IsAdult.Bool,
		"uploader": fiber.Map{
			"id":       formatUUID(uploader.ID),
			"username": uploader.Username,
			"avatar":   uploader.Avatar.String,
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

	response, err := requestAuraAPI(http.MethodDelete, "/fs/files/delete/"+url.PathEscape(video.FileID), nil)
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
		if err := s.Cache.Del(context.Background(), "video:"+idStr, "thumbnail:"+idStr).Err(); err != nil {
			log.Printf("Cache invalidation failed after deleting video %s: %v", idStr, err)
		}
	}

	return c.JSON(fiber.Map{"message": "Video deleted successfully"})
}

func (s *Server) SearchVideosHandler(c *fiber.Ctx) error {
	q := strings.TrimSpace(c.Query("q", ""))
	showAdult := c.Query("adult") == "true"
	if q == "" {
		return c.JSON(fiber.Map{"videos": []fiber.Map{}, "query": q, "total": 0})
	}
	videos, err := s.Repository.SearchPublicVideos(context.Background(), q, showAdult, c.Query("sort", "relevance"), 30)
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
	return c.JSON(fiber.Map{"videos": res, "query": q, "total": len(res)})
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
	video, err := s.Repository.GetVideoByFileId(context.Background(), id)
	if err != nil {
		videoID, parseErr := parseUUID(id)
		if parseErr != nil {
			return c.Status(404).JSON(fiber.Map{"message": "Video not found"})
		}
		video, err = s.Repository.GetVideo(context.Background(), videoID)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"message": "Video not found"})
		}
	}

	ctx := context.Background()
	if err := s.Repository.IncrementVideoViews(ctx, video.ID); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Could not count view."})
	}

	if userIDValue := c.Locals("userId"); userIDValue != nil {
		userID, ok := userIDValue.(string)
		if !ok {
			return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
		}
		userUUID, err := parseUUID(userID)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
		}
		if err := s.Repository.UpsertVideoViewActivity(ctx, userUUID, video.ID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "Could not record view activity."})
		}
		if err := s.Repository.UpsertWatchHistory(ctx, db.UpsertWatchHistoryParams{
			UserID: userUUID, VideoID: video.ID,
		}); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "Could not update watch history."})
		}
	}

	if s.Cache != nil {
		if err := s.Cache.Del(ctx, "video:"+id).Err(); err != nil {
			log.Printf("Cache invalidation failed after recording view for %s: %v", id, err)
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
