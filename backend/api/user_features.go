package api

import (
	"context"
	"strconv"
	"strings"

	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/gofiber/fiber/v2"
)

func (s *Server) GetWatchHistoryHandler(c *fiber.Ctx) error {
	userIdLocal := c.Locals("userId")
	if userIdLocal == nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userId, _ := parseUUID(userIdLocal.(string))

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset := int32((page - 1) * limit)

	videos, err := s.Repository.GetWatchHistory(context.Background(), db.GetWatchHistoryParams{
		UserID: userId,
		Limit:  int32(limit),
		Offset: offset,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch history"})
	}

	var res []fiber.Map
	for _, v := range videos {
		res = append(res, fiber.Map{
			"id":           formatUUID(v.ID),
			"title":        v.Title,
			"thumbnailUrl": v.ThumbnailUrl.String,
			"views":        v.Views.Int32,
			"isShort":      v.IsShort.Bool,
		})
	}

	return c.JSON(fiber.Map{"videos": res})
}

func (s *Server) DeleteWatchHistoryHandler(c *fiber.Ctx) error {
	userIdLocal := c.Locals("userId")
	if userIdLocal == nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userId, _ := parseUUID(userIdLocal.(string))

	videoIdStr := c.Query("videoId")
	if videoIdStr == "" {
		// Clear all
		err := s.Repository.ClearWatchHistory(context.Background(), userId)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "Failed to clear history"})
		}
	} else {
		videoId, err := parseUUID(videoIdStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "Invalid video ID"})
		}
		if err := s.Repository.DeleteWatchHistory(context.Background(), db.DeleteWatchHistoryParams{
			UserID: userId, VideoID: videoId,
		}); err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "Failed to remove video from history"})
		}
	}

	message := "History updated"
	if strings.HasPrefix(c.Path(), "/api/user/") {
		if videoIdStr == "" {
			message = "Watch history cleared"
		} else {
			message = "Video removed from history"
		}
	}
	return c.JSON(fiber.Map{"message": message})
}

func (s *Server) GetWatchLaterHandler(c *fiber.Ctx) error {
	userIdLocal := c.Locals("userId")
	if userIdLocal == nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userId, _ := parseUUID(userIdLocal.(string))

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset := int32((page - 1) * limit)

	videos, err := s.Repository.GetWatchLater(context.Background(), db.GetWatchLaterParams{
		UserID: userId,
		Limit:  int32(limit),
		Offset: offset,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch watch later"})
	}

	var res []fiber.Map
	for _, v := range videos {
		res = append(res, fiber.Map{
			"id":           formatUUID(v.ID),
			"title":        v.Title,
			"thumbnailUrl": v.ThumbnailUrl.String,
			"views":        v.Views.Int32,
			"isShort":      v.IsShort.Bool,
		})
	}

	return c.JSON(fiber.Map{"videos": res})
}

func (s *Server) ToggleWatchLaterHandler(c *fiber.Ctx) error {
	userIdLocal := c.Locals("userId")
	if userIdLocal == nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userId, err := parseUUID(userIdLocal.(string))
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}

	var req struct {
		VideoID string `json:"videoId"`
		Action  string `json:"action"` // add, remove
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid payload"})
	}
	videoId, err := parseUUID(req.VideoID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid video ID"})
	}

	action := req.Action
	if action == "" {
		action = "add"
	}
	if action == "add" {
		err = s.Repository.AddWatchLater(context.Background(), db.AddWatchLaterParams{
			UserID:  userId,
			VideoID: videoId,
		})
	} else if action == "remove" {
		err = s.Repository.RemoveWatchLater(context.Background(), db.RemoveWatchLaterParams{
			UserID:  userId,
			VideoID: videoId,
		})
	} else {
		return c.Status(400).JSON(fiber.Map{"message": "Action must be add or remove"})
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to update watch later"})
	}
	message := "Success"
	if strings.HasPrefix(c.Path(), "/api/user/") && action == "add" {
		message = "Added to Watch Later"
	}
	return c.JSON(fiber.Map{"message": message})
}

func (s *Server) DeleteWatchLaterHandler(c *fiber.Ctx) error {
	userIDValue := c.Locals("userId")
	userID, ok := userIDValue.(string)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userUUID, err := parseUUID(userID)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	videoID, err := parseUUID(c.Query("videoId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid video ID"})
	}
	if err := s.Repository.RemoveWatchLater(context.Background(), db.RemoveWatchLaterParams{
		UserID: userUUID, VideoID: videoID,
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to remove video"})
	}
	return c.JSON(fiber.Map{"message": "Removed from Watch Later"})
}
