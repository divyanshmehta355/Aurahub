package api

import (
	"strconv"
	"strings"

	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/gofiber/fiber/v2"
)

func (s *Server) GetWatchHistoryHandler(c *fiber.Ctx) error {
	userId, ok := s.requireAuth(c)
	if !ok {
		return nil
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset := int32((page - 1) * limit)

	videos, err := s.Repository.GetWatchHistory(c.UserContext(), db.GetWatchHistoryParams{
		UserID: userId,
		Limit:  int32(limit),
		Offset: offset,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to fetch history"})
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
	userId, ok := s.requireAuth(c)
	if !ok {
		return nil
	}

	videoIdStr := c.Query("videoId")
	if videoIdStr == "" {
		// Clear all
		err := s.Repository.ClearWatchHistory(c.UserContext(), userId)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to clear history"})
		}
	} else {
		videoId, err := parseUUID(videoIdStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid video ID"})
		}
		if err := s.Repository.DeleteWatchHistory(c.UserContext(), db.DeleteWatchHistoryParams{
			UserID: userId, VideoID: videoId,
		}); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to remove video from history"})
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
	userId, ok := s.requireAuth(c)
	if !ok {
		return nil
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset := int32((page - 1) * limit)

	videos, err := s.Repository.GetWatchLater(c.UserContext(), db.GetWatchLaterParams{
		UserID: userId,
		Limit:  int32(limit),
		Offset: offset,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to fetch watch later"})
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
	userId, ok := s.requireAuth(c)
	if !ok {
		return nil
	}

	var req struct {
		VideoID string `json:"videoId"`
		Action  string `json:"action"` // add, remove
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid payload"})
	}
	videoId, err := parseUUID(req.VideoID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid video ID"})
	}

	action := req.Action
	if action == "" {
		action = "add"
	}
	if action == "add" {
		err = s.Repository.AddWatchLater(c.UserContext(), db.AddWatchLaterParams{
			UserID:  userId,
			VideoID: videoId,
		})
	} else if action == "remove" {
		err = s.Repository.RemoveWatchLater(c.UserContext(), db.RemoveWatchLaterParams{
			UserID:  userId,
			VideoID: videoId,
		})
	} else {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Action must be add or remove"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to update watch later"})
	}
	message := "Success"
	if strings.HasPrefix(c.Path(), "/api/user/") && action == "add" {
		message = "Added to Watch Later"
	}
	return c.JSON(fiber.Map{"message": message})
}

func (s *Server) DeleteWatchLaterHandler(c *fiber.Ctx) error {
	userUUID, ok := s.requireAuth(c)
	if !ok {
		return nil
	}
	videoID, err := parseUUID(c.Query("videoId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid video ID"})
	}
	if err := s.Repository.RemoveWatchLater(c.UserContext(), db.RemoveWatchLaterParams{
		UserID: userUUID, VideoID: videoID,
	}); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to remove video"})
	}
	return c.JSON(fiber.Map{"message": "Removed from Watch Later"})
}
