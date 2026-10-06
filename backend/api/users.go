package api

import (
	"context"
	"strconv"
	"strings"

	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

type UpdateUserRequest struct {
	Avatar string `json:"avatar"`
	Banner string `json:"banner"`
	Bio    string `json:"bio"`
}

func (s *Server) GetUserHandler(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		idStr = c.Params("identifier")
	}
	userId, err := parseUUID(idStr)
	var user db.User
	if err == nil {
		user, err = s.Repository.GetUser(context.Background(), userId)
	} else {
		user, err = s.Repository.GetUserByUsername(context.Background(), idStr)
	}
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"message": "User not found"})
	}
	return c.JSON(fiber.Map{
		"id":       formatUUID(user.ID),
		"username": user.Username,
		"avatar":   user.Avatar.String,
		"banner":   user.Banner.String,
		"bio":      user.Bio.String,
	})
}

func (s *Server) ToggleSubscriptionHandler(c *fiber.Ctx) error {
	viewerValue := c.Locals("userId")
	viewerID, ok := viewerValue.(string)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	viewerUUID, err := parseUUID(viewerID)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}

	identifier := c.Params("identifier")
	targetUUID, parseErr := parseUUID(identifier)
	var target db.User
	if parseErr == nil {
		target, err = s.Repository.GetUser(context.Background(), targetUUID)
	} else {
		target, err = s.Repository.GetUserByUsername(context.Background(), identifier)
	}
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"message": "User not found"})
	}
	if target.ID == viewerUUID {
		return c.Status(400).JSON(fiber.Map{"message": "You cannot subscribe to yourself"})
	}

	alreadySubscribed, err := s.Repository.IsSubscribed(context.Background(), db.IsSubscribedParams{
		SubscriberID: viewerUUID, SubscribedToID: target.ID,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to check subscription"})
	}
	if alreadySubscribed {
		err = s.Repository.Unsubscribe(context.Background(), db.UnsubscribeParams{
			SubscriberID: viewerUUID, SubscribedToID: target.ID,
		})
	} else {
		err = s.Repository.Subscribe(context.Background(), db.SubscribeParams{
			SubscriberID: viewerUUID, SubscribedToID: target.ID,
		})
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to update subscription"})
	}

	count, err := s.Repository.GetSubscriberCount(context.Background(), target.ID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to count subscribers"})
	}
	return c.JSON(fiber.Map{"isSubscribed": !alreadySubscribed, "subscriberCount": count})
}

func (s *Server) UpdateUserHandler(c *fiber.Ctx) error {
	var req UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid payload"})
	}

	userIdLocal := c.Locals("userId")
	if userIdLocal == nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}

	userId, err := parseUUID(userIdLocal.(string))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid session data"})
	}

	user, err := s.Repository.UpdateUser(context.Background(), db.UpdateUserParams{
		ID:      userId,
		Column2: req.Avatar,
		Column3: req.Banner,
		Column4: req.Bio,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to update profile"})
	}

	return c.JSON(fiber.Map{
		"id":       formatUUID(user.ID),
		"username": user.Username,
		"avatar":   user.Avatar.String,
		"banner":   user.Banner.String,
		"bio":      user.Bio.String,
	})
}

func (s *Server) GetProfileHandler(c *fiber.Ctx) error {
	identifier := c.Params("identifier")

	user, err := s.Repository.GetUserByUsername(context.Background(), identifier)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"message": "User not found"})
	}

	subscriberCount, err := s.Repository.GetSubscriberCount(context.Background(), user.ID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch subscriber count"})
	}

	var isSubscribed bool
	userIdLocal := c.Locals("userId")
	if userIdLocal != nil {
		if viewerId, err := parseUUID(userIdLocal.(string)); err == nil {
			isSubscribed, err = s.Repository.IsSubscribed(context.Background(), db.IsSubscribedParams{
				SubscriberID:   viewerId,
				SubscribedToID: user.ID,
			})
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"message": "Failed to check subscription"})
			}
		}
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	showAdult := c.Query("adult") == "true"

	videos, err := s.Repository.ListUserVideos(context.Background(), db.ListUserVideosParams{
		UploaderID: user.ID,
		Limit:      int32(limit),
		Offset:     int32(offset),
		ShowAdult:  showAdult,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch user videos"})
	}
	videoResults := make([]fiber.Map, 0, len(videos))
	for _, video := range videos {
		likesCount, _ := s.Repository.CountVideoLikes(context.Background(), video.ID)
		commentCount, _ := s.Repository.CountVideoComments(context.Background(), video.ID)
		videoResults = append(videoResults, fiber.Map{
			"_id": formatUUID(video.ID), "id": formatUUID(video.ID), "fileId": video.FileID,
			"title": video.Title, "description": video.Description.String,
			"thumbnailUrl": video.ThumbnailUrl.String, "category": video.Category.String,
			"views": video.Views.Int32, "likesCount": likesCount, "commentCount": commentCount,
			"isShort": video.IsShort.Bool, "createdAt": video.CreatedAt.Time,
			"uploader": fiber.Map{"_id": formatUUID(user.ID), "username": user.Username, "avatar": user.Avatar.String},
		})
	}

	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"id":              formatUUID(user.ID),
			"username":        user.Username,
			"avatar":          user.Avatar.String,
			"bio":             user.Bio.String,
			"banner":          user.Banner.String,
			"joined":          user.CreatedAt,
			"subscriberCount": subscriberCount,
			"isSubscribed":    isSubscribed,
		},
		"videos": videoResults,
		"currentPage": page,
		"hasMore": len(videos) == limit,
	})
}

type FullProfileUpdateRequest struct {
	Username         string  `json:"username"`
	Email            string  `json:"email"`
	Password         string  `json:"password"`
	Avatar           string  `json:"avatar"`
	Bio              *string `json:"bio"`
	Banner           string  `json:"banner"`
	ShowAdultContent *bool   `json:"showAdultContent"`
}

func (s *Server) UpdateProfileHandler(c *fiber.Ctx) error {
	var req FullProfileUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid payload"})
	}

	userIdLocal := c.Locals("userId")
	if userIdLocal == nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}

	userId, err := parseUUID(userIdLocal.(string))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid session data"})
	}

	var update db.UpdateUserProfileParams
	update.ID = userId
	if req.Username != "" {
		usernameUsed, err := s.Repository.IsUsernameUsedByAnotherUser(context.Background(), req.Username, userId)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "Failed to check username"})
		}
		if usernameUsed {
			return c.Status(409).JSON(fiber.Map{"message": "An account with this username already exists."})
		}
		update.Username = pgtype.Text{String: req.Username, Valid: true}
	}
	if req.Email != "" {
		email := strings.ToLower(strings.TrimSpace(req.Email))
		emailUsed, err := s.Repository.IsEmailUsedByAnotherUser(context.Background(), email, userId)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "Failed to check email"})
		}
		if emailUsed {
			return c.Status(409).JSON(fiber.Map{"message": "An account with this email already exists."})
		}
		update.Email = pgtype.Text{String: email, Valid: true}
	}
	if req.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "Failed to update password"})
		}
		update.Password = pgtype.Text{String: string(hashedPassword), Valid: true}
	}
	if req.Avatar != "" {
		update.Avatar = pgtype.Text{String: req.Avatar, Valid: true}
	}
	if req.Bio != nil {
		update.Bio = pgtype.Text{String: *req.Bio, Valid: true}
	}
	if req.Banner != "" {
		update.Banner = pgtype.Text{String: req.Banner, Valid: true}
	}
	if req.ShowAdultContent != nil {
		update.ShowAdultContent = pgtype.Bool{Bool: *req.ShowAdultContent, Valid: true}
	}
	user, err := s.Repository.UpdateUserProfile(context.Background(), update)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to update profile"})
	}

	return c.JSON(fiber.Map{
		"id":               formatUUID(user.ID),
		"username":         user.Username,
		"email":            user.Email,
		"avatar":           user.Avatar.String,
		"banner":           user.Banner.String,
		"bio":              user.Bio.String,
		"showAdultContent": user.ShowAdultContent.Bool,
	})
}
