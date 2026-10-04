package api

import (
	"context"
	"strconv"

	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/gofiber/fiber/v2"
)

func (s *Server) ListNotificationsHandler(c *fiber.Ctx) error {
	userIdLocal := c.Locals("userId")
	if userIdLocal == nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userId, _ := parseUUID(userIdLocal.(string))

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := int32((page - 1) * limit)

	notifications, err := s.Repository.ListNotifications(context.Background(), db.ListNotificationsParams{
		RecipientID: userId,
		Limit:       int32(limit),
		Offset:      offset,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch notifications"})
	}

	var res []fiber.Map
	for _, n := range notifications {
		sender, err := s.Repository.GetUser(context.Background(), n.SenderID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch notification sender"})
		}
		var video interface{}
		if n.VideoID.Valid {
			videoRecord, err := s.Repository.GetVideo(context.Background(), n.VideoID)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch notification video"})
			}
			video = fiber.Map{"_id": formatUUID(videoRecord.ID), "id": formatUUID(videoRecord.ID), "title": videoRecord.Title}
		}
		res = append(res, fiber.Map{
			"id":        formatUUID(n.ID),
			"senderId":  formatUUID(n.SenderID),
			"sender":    fiber.Map{"_id": formatUUID(sender.ID), "id": formatUUID(sender.ID), "username": sender.Username, "avatar": sender.Avatar.String},
			"type":      n.Type,
			"videoId":   uuidOrNil(n.VideoID),
			"video":     video,
			"commentId": uuidOrNil(n.CommentID),
			"isRead":    n.IsRead.Bool,
			"createdAt": n.CreatedAt.Time,
		})
	}

	unreadCount, err := s.Repository.CountUnreadNotifications(context.Background(), userId)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to count unread notifications"})
	}

	return c.JSON(fiber.Map{
		"notifications": res,
		"unreadCount":   unreadCount,
	})
}

func (s *Server) MarkNotificationsReadHandler(c *fiber.Ctx) error {
	userIdLocal := c.Locals("userId")
	if userIdLocal == nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userId, _ := parseUUID(userIdLocal.(string))

	err := s.Repository.MarkNotificationsRead(context.Background(), userId)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to mark as read"})
	}

	return c.JSON(fiber.Map{"message": "Notifications marked as read"})
}

func (s *Server) ClearNotificationsHandler(c *fiber.Ctx) error {
	userIdLocal := c.Locals("userId")
	if userIdLocal == nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userId, _ := parseUUID(userIdLocal.(string))

	err := s.Repository.ClearNotifications(context.Background(), userId)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to clear notifications"})
	}

	return c.JSON(fiber.Map{"message": "Notifications cleared"})
}
