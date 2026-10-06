package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/valyala/fasthttp"
)

// PublishNotificationEvent sends a JSON payload to a user's Valkey pub/sub channel.
func (s *Server) PublishNotificationEvent(ctx context.Context, recipientID pgtype.UUID, eventType string, data fiber.Map) {
	if s.Cache == nil {
		return
	}
	data["event"] = eventType
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	channel := fmt.Sprintf("notifications:user:%s", formatUUID(recipientID))
	_ = s.Cache.Publish(ctx, channel, payload).Err()
}

// CreateAndPublishNotification inserts a notification in DB and broadcasts it via Valkey Pub/Sub in real time.
func (s *Server) CreateAndPublishNotification(ctx context.Context, arg db.CreateNotificationParams) error {
	if arg.RecipientID == arg.SenderID {
		return nil
	}

	notification, err := s.Repository.CreateNotificationWithResult(ctx, arg)
	if err != nil {
		return err
	}
	if notification == nil {
		return nil
	}

	sender, sErr := s.Repository.GetUser(ctx, arg.SenderID)
	senderMap := fiber.Map{
		"_id":      formatUUID(arg.SenderID),
		"id":       formatUUID(arg.SenderID),
		"username": "Someone",
		"avatar":   "",
	}
	if sErr == nil {
		senderMap["username"] = sender.Username
		senderMap["avatar"] = sender.Avatar.String
	}

	var video interface{}
	if arg.VideoID.Valid {
		videoRecord, vErr := s.Repository.GetVideo(ctx, arg.VideoID)
		if vErr == nil {
			video = fiber.Map{
				"_id":          formatUUID(videoRecord.ID),
				"id":           formatUUID(videoRecord.ID),
				"title":        videoRecord.Title,
				"thumbnailUrl": videoRecord.ThumbnailUrl.String,
			}
		} else {
			video = fiber.Map{
				"_id":   formatUUID(arg.VideoID),
				"id":    formatUUID(arg.VideoID),
				"title": "Video",
			}
		}
	}

	var comment interface{}
	if arg.CommentID.Valid {
		commentRecord, cErr := s.Repository.GetComment(ctx, arg.CommentID)
		if cErr == nil {
			comment = fiber.Map{
				"id":   formatUUID(commentRecord.ID),
				"text": commentRecord.Text,
			}
		}
	}

	unreadCount, countErr := s.Repository.CountUnreadNotifications(ctx, arg.RecipientID)
	if countErr != nil {
		unreadCount = 1
	}

	notifMap := fiber.Map{
		"id":        formatUUID(notification.ID),
		"senderId":  formatUUID(arg.SenderID),
		"sender":    senderMap,
		"type":      arg.Type,
		"videoId":   uuidOrNil(arg.VideoID),
		"video":     video,
		"commentId": uuidOrNil(arg.CommentID),
		"comment":   comment,
		"isRead":    false,
		"createdAt": notification.CreatedAt.Time,
	}

	s.PublishNotificationEvent(ctx, arg.RecipientID, "new_notification", fiber.Map{
		"notification": notifMap,
		"unreadCount":  unreadCount,
	})

	return nil
}

// PublishSubscribersNewVideo notifies all subscribers in DB and pushes real-time events via Valkey.
func (s *Server) PublishSubscribersNewVideo(ctx context.Context, uploaderID pgtype.UUID, videoID pgtype.UUID) {
	createdNotifications, err := s.Repository.NotifySubscribersNewVideoWithResult(ctx, uploaderID, videoID)
	if err != nil || len(createdNotifications) == 0 {
		return
	}

	sender, sErr := s.Repository.GetUser(ctx, uploaderID)
	senderMap := fiber.Map{
		"_id":      formatUUID(uploaderID),
		"id":       formatUUID(uploaderID),
		"username": "Someone",
		"avatar":   "",
	}
	if sErr == nil {
		senderMap["username"] = sender.Username
		senderMap["avatar"] = sender.Avatar.String
	}

	var video interface{}
	videoRecord, vErr := s.Repository.GetVideo(ctx, videoID)
	if vErr == nil {
		video = fiber.Map{
			"_id":          formatUUID(videoRecord.ID),
			"id":           formatUUID(videoRecord.ID),
			"title":        videoRecord.Title,
			"thumbnailUrl": videoRecord.ThumbnailUrl.String,
		}
	} else {
		video = fiber.Map{
			"_id":   formatUUID(videoID),
			"id":    formatUUID(videoID),
			"title": "New Video",
		}
	}

	for _, n := range createdNotifications {
		unreadCount, _ := s.Repository.CountUnreadNotifications(ctx, n.RecipientID)
		notifMap := fiber.Map{
			"id":        formatUUID(n.ID),
			"senderId":  formatUUID(n.SenderID),
			"sender":    senderMap,
			"type":      n.Type,
			"videoId":   uuidOrNil(n.VideoID),
			"video":     video,
			"commentId": nil,
			"comment":   nil,
			"isRead":    false,
			"createdAt": n.CreatedAt.Time,
		}
		s.PublishNotificationEvent(ctx, n.RecipientID, "new_notification", fiber.Map{
			"notification": notifMap,
			"unreadCount":  unreadCount,
		})
	}
}

// NotificationStreamHandler establishes a Server-Sent Events (SSE) connection backed by Valkey Pub/Sub.
func (s *Server) NotificationStreamHandler(c *fiber.Ctx) error {
	userIdLocal := c.Locals("userId")
	if userIdLocal == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userIdStr, ok := userIdLocal.(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userId, err := parseUUID(userIdStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid user ID"})
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")
	c.Set("X-Accel-Buffering", "no")

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		if s.Cache == nil {
			fmt.Fprintf(w, ": valkey not connected\n\n")
			_ = w.Flush()
			return
		}

		channelName := fmt.Sprintf("notifications:user:%s", userIdStr)
		pubsub := s.Cache.Subscribe(context.Background(), channelName)
		defer pubsub.Close()

		// Send initial state with current unread count
		unreadCount, _ := s.Repository.CountUnreadNotifications(context.Background(), userId)
		initPayload, _ := json.Marshal(fiber.Map{
			"event":       "init",
			"unreadCount": unreadCount,
		})
		fmt.Fprintf(w, "event: init\ndata: %s\n\n", initPayload)
		if err := w.Flush(); err != nil {
			return
		}

		ch := pubsub.Channel()
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return
				}
				fmt.Fprintf(w, "event: notification\ndata: %s\n\n", msg.Payload)
				if err := w.Flush(); err != nil {
					return
				}
			case <-ticker.C:
				fmt.Fprintf(w, ": keepalive\n\n")
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	}))

	return nil
}

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
		senderMap := fiber.Map{
			"_id":      formatUUID(n.SenderID),
			"id":       formatUUID(n.SenderID),
			"username": "Someone",
			"avatar":   "",
		}
		if err == nil {
			senderMap["username"] = sender.Username
			senderMap["avatar"] = sender.Avatar.String
		}

		var video interface{}
		if n.VideoID.Valid {
			videoRecord, vErr := s.Repository.GetVideo(context.Background(), n.VideoID)
			if vErr == nil {
				video = fiber.Map{
					"_id":          formatUUID(videoRecord.ID),
					"id":           formatUUID(videoRecord.ID),
					"title":        videoRecord.Title,
					"thumbnailUrl": videoRecord.ThumbnailUrl.String,
				}
			} else {
				video = fiber.Map{
					"_id":   formatUUID(n.VideoID),
					"id":    formatUUID(n.VideoID),
					"title": "Video",
				}
			}
		}

		var comment interface{}
		if n.CommentID.Valid {
			commentRecord, cErr := s.Repository.GetComment(context.Background(), n.CommentID)
			if cErr == nil {
				comment = fiber.Map{
					"id":   formatUUID(commentRecord.ID),
					"text": commentRecord.Text,
				}
			}
		}

		res = append(res, fiber.Map{
			"id":        formatUUID(n.ID),
			"senderId":  formatUUID(n.SenderID),
			"sender":    senderMap,
			"type":      n.Type,
			"videoId":   uuidOrNil(n.VideoID),
			"video":     video,
			"commentId": uuidOrNil(n.CommentID),
			"comment":   comment,
			"isRead":    n.IsRead.Bool,
			"createdAt": n.CreatedAt.Time,
		})
	}

	unreadCount, err := s.Repository.CountUnreadNotifications(context.Background(), userId)
	if err != nil {
		unreadCount = 0
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

	// Real-time broadcast to all client tabs via Valkey
	s.PublishNotificationEvent(context.Background(), userId, "all_read", fiber.Map{
		"unreadCount": 0,
	})

	return c.JSON(fiber.Map{"message": "Notifications marked as read"})
}

func (s *Server) MarkNotificationReadHandler(c *fiber.Ctx) error {
	userIdLocal := c.Locals("userId")
	if userIdLocal == nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userId, _ := parseUUID(userIdLocal.(string))

	notificationId, err := parseUUID(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid notification ID"})
	}

	err = s.Repository.MarkNotificationRead(context.Background(), userId, notificationId)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to mark notification as read"})
	}

	unreadCount, _ := s.Repository.CountUnreadNotifications(context.Background(), userId)
	// Real-time broadcast to all client tabs via Valkey
	s.PublishNotificationEvent(context.Background(), userId, "notification_read", fiber.Map{
		"id":          c.Params("id"),
		"unreadCount": unreadCount,
	})

	return c.JSON(fiber.Map{"message": "Notification marked as read"})
}

func (s *Server) DeleteNotificationHandler(c *fiber.Ctx) error {
	userIdLocal := c.Locals("userId")
	if userIdLocal == nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userId, _ := parseUUID(userIdLocal.(string))

	notificationId, err := parseUUID(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid notification ID"})
	}

	err = s.Repository.DeleteNotification(context.Background(), userId, notificationId)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to delete notification"})
	}

	unreadCount, _ := s.Repository.CountUnreadNotifications(context.Background(), userId)
	// Real-time broadcast to all client tabs via Valkey
	s.PublishNotificationEvent(context.Background(), userId, "notification_deleted", fiber.Map{
		"id":          c.Params("id"),
		"unreadCount": unreadCount,
	})

	return c.JSON(fiber.Map{"message": "Notification deleted"})
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

	// Real-time broadcast to all client tabs via Valkey
	s.PublishNotificationEvent(context.Background(), userId, "all_cleared", fiber.Map{
		"unreadCount": 0,
	})

	return c.JSON(fiber.Map{"message": "Notifications cleared"})
}
