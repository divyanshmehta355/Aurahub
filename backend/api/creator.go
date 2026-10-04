package api

import (
	"context"
	"strconv"
	"time"

	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Server) CreatorDashboardHandler(c *fiber.Ctx) error {
	userID, ok := c.Locals("userId").(string)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userId, err := parseUUID(userID)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	page, limit := pageLimit(c, 20, 100)
	offset := int32((page - 1) * limit)

	videos, err := s.Repository.ListCreatorVideos(context.Background(), userId, int32(limit), offset)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch videos"})
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
			"visibility":   v.Visibility.VideoVisibility,
			"createdAt":    v.CreatedAt.Time,
		})
	}

	return c.JSON(fiber.Map{"videos": res, "page": page})
}

func (s *Server) CreatorAnalyticsHandler(c *fiber.Ctx) error {
	userID, ok := c.Locals("userId").(string)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userId, err := parseUUID(userID)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}

	stats, err := s.Repository.GetCreatorLifetimeStats(context.Background(), userId)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch analytics"})
	}
	days, err := strconv.Atoi(c.Query("days", "30"))
	if err != nil || (days != 7 && days != 30 && days != 90) {
		days = 30
	}
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -(days - 1))
	activities, err := s.Repository.GetCreatorActivity(context.Background(), userId, start)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch analytics activity"})
	}
	topVideos, err := s.Repository.GetCreatorTopVideos(context.Background(), userId)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch top videos"})
	}
	categories, err := s.Repository.GetCreatorVideoCategories(context.Background(), userId)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch video categories"})
	}

	user, err := s.Repository.GetUser(context.Background(), userId)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch creator"})
	}
	subscribersCount, err := s.Repository.GetSubscriberCount(context.Background(), userId)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch subscriber count"})
	}

	activityByDate := make(map[string]fiber.Map, len(activities))
	periodActivity := fiber.Map{"viewInteractions": int64(0), "likeInteractions": int64(0)}
	for _, activity := range activities {
		dateKey := activity.Date.UTC().Format("2006-01-02")
		day := activityByDate[dateKey]
		if day == nil {
			day = fiber.Map{"viewInteractions": int64(0), "likeInteractions": int64(0)}
			activityByDate[dateKey] = day
		}
		switch activity.Type {
		case string(db.InteractionTypeView):
			day["viewInteractions"] = activity.Count
			periodActivity["viewInteractions"] = periodActivity["viewInteractions"].(int64) + activity.Count
		case string(db.InteractionTypeLike):
			day["likeInteractions"] = activity.Count
			periodActivity["likeInteractions"] = periodActivity["likeInteractions"].(int64) + activity.Count
		}
	}
	timeSeries := make([]fiber.Map, 0, days)
	for day := 0; day < days; day++ {
		date := start.AddDate(0, 0, day)
		counts := activityByDate[date.Format("2006-01-02")]
		if counts == nil {
			counts = fiber.Map{"viewInteractions": int64(0), "likeInteractions": int64(0)}
		}
		timeSeries = append(timeSeries, fiber.Map{
			"date": date.Format("Jan 2"), "viewInteractions": counts["viewInteractions"],
			"likeInteractions": counts["likeInteractions"],
		})
	}

	topVideoResults := make([]fiber.Map, 0, len(topVideos))
	for _, video := range topVideos {
		topVideoResults = append(topVideoResults, fiber.Map{
			"_id": formatUUID(video.ID), "id": formatUUID(video.ID), "title": video.Title,
			"category": video.Category, "views": video.Views, "likesCount": video.LikesCount,
			"commentCount": video.CommentCount,
		})
	}
	categoryResults := make([]fiber.Map, 0, len(categories))
	for _, category := range categories {
		categoryResults = append(categoryResults, fiber.Map{
			"category": category.Category, "videos": category.Videos, "views": category.Views,
		})
	}

	return c.JSON(fiber.Map{
		"lifetimeStats": fiber.Map{
			"views":       stats.TotalViews,
			"likes":       stats.TotalLikes,
			"comments":    stats.TotalComments,
			"videos":      stats.TotalVideos,
			"subscribers": subscribersCount,
		},
		"user": fiber.Map{
			"id":       formatUUID(user.ID),
			"username": user.Username,
			"avatar":   user.Avatar.String,
		},
		"period":         days,
		"periodActivity": periodActivity,
		"timeSeries":     timeSeries,
		"topVideos":      topVideoResults,
		"categories":     categoryResults,
	})
}

type BulkUpdateVisibilityRequest struct {
	VideoIds   []string `json:"videoIds"`
	Visibility string   `json:"visibility"`
}

func (s *Server) BulkUpdateVideoVisibilityHandler(c *fiber.Ctx) error {
	var req BulkUpdateVisibilityRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid payload"})
	}

	userID, ok := c.Locals("userId").(string)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userId, err := parseUUID(userID)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}

	var uuids []pgtype.UUID
	for _, idStr := range req.VideoIds {
		id, err := parseUUID(idStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "Invalid video ID"})
		}
		uuids = append(uuids, id)
	}

	var visibility db.VideoVisibility
	switch req.Visibility {
	case "public":
		visibility = db.VideoVisibilityPublic
	case "unlisted":
		visibility = db.VideoVisibilityUnlisted
	case "private":
		visibility = db.VideoVisibilityPrivate
	default:
		return c.Status(400).JSON(fiber.Map{"message": "Invalid visibility"})
	}

	err = s.Repository.BulkUpdateVideoVisibility(context.Background(), db.BulkUpdateVideoVisibilityParams{
		UploaderID: userId,
		Column2:    uuids,
		Visibility: db.NullVideoVisibility{VideoVisibility: visibility, Valid: true},
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to update videos"})
	}

	return c.JSON(fiber.Map{"message": "Videos updated successfully"})
}

type BulkDeleteVideosRequest struct {
	VideoIds []string `json:"videoIds"`
}

func (s *Server) BulkDeleteVideosHandler(c *fiber.Ctx) error {
	var req BulkDeleteVideosRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid payload"})
	}

	userID, ok := c.Locals("userId").(string)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userId, err := parseUUID(userID)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}

	var uuids []pgtype.UUID
	for _, idStr := range req.VideoIds {
		id, err := parseUUID(idStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "Invalid video ID"})
		}
		uuids = append(uuids, id)
	}

	err = s.Repository.BulkDeleteVideos(context.Background(), db.BulkDeleteVideosParams{
		UploaderID: userId,
		Column2:    uuids,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to delete videos"})
	}

	return c.JSON(fiber.Map{"message": "Videos deleted successfully"})
}
