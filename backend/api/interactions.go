package api

import (
	"context"
	"strconv"

	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgtype"
)

type CreateCommentRequest struct {
	Text            string `json:"text"`
	VideoID         string `json:"videoId"`
	ParentCommentID string `json:"parentCommentId"`
}

func (s *Server) CreateCommentHandler(c *fiber.Ctx) error {
	var req CreateCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid payload"})
	}

	userIdLocal := c.Locals("userId")
	if userIdLocal == nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}

	authorId, _ := parseUUID(userIdLocal.(string))
	videoIDValue := req.VideoID
	if pathVideoID := c.Params("id"); pathVideoID != "" {
		videoIDValue = pathVideoID
	}
	videoId, err := parseUUID(videoIDValue)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid video ID"})
	}

	var parentId pgtype.UUID
	if req.ParentCommentID != "" {
		parentId, _ = parseUUID(req.ParentCommentID)
	}

	comment, err := s.Repository.CreateComment(context.Background(), db.CreateCommentParams{
		Text:            req.Text,
		AuthorID:        authorId,
		VideoID:         videoId,
		ParentCommentID: parentId,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to post comment"})
	}

	return c.JSON(fiber.Map{
		"id":        formatUUID(comment.ID),
		"text":      comment.Text,
		"authorId":  formatUUID(comment.AuthorID),
		"createdAt": comment.CreatedAt.Time,
	})
}

func (s *Server) ListCommentsHandler(c *fiber.Ctx) error {
	videoIdStr := c.Params("videoId")
	if videoIdStr == "" {
		videoIdStr = c.Params("id")
	}
	videoId, err := parseUUID(videoIdStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid video ID"})
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := int32((page - 1) * limit)

	comments, err := s.Repository.ListCommentsForVideo(context.Background(), db.ListCommentsForVideoParams{
		VideoID: videoId,
		Limit:   int32(limit),
		Offset:  offset,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch comments"})
	}

	var res []fiber.Map
	for _, comment := range comments {
		res = append(res, fiber.Map{
			"id":        formatUUID(comment.ID),
			"text":      comment.Text,
			"authorId":  formatUUID(comment.AuthorID),
			"createdAt": comment.CreatedAt.Time,
		})
	}

	return c.JSON(fiber.Map{
		"comments": res,
	})
}

func uuidOrNil(id pgtype.UUID) interface{} {
	if !id.Valid {
		return nil
	}
	return formatUUID(id)
}

func (s *Server) ToggleLikeHandler(c *fiber.Ctx) error {
	videoIdStr := c.Params("videoId")
	if videoIdStr == "" {
		videoIdStr = c.Params("id")
	}
	videoId, err := parseUUID(videoIdStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid video ID"})
	}

	userIdLocal := c.Locals("userId")
	if userIdLocal == nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userId, err := parseUUID(userIdLocal.(string))
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}

	action := c.Query("action") // "like" or "unlike"
	if action == "" {
		liked, err := s.Repository.HasUserLikedVideo(context.Background(), userId, videoId)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "Failed to check like status"})
		}
		if liked {
			action = "unlike"
		} else {
			action = "like"
		}
	}

	if action == "like" {
		err = s.Repository.UpsertUserActivity(context.Background(), db.UpsertUserActivityParams{
			UserID:          userId,
			VideoID:         videoId,
			InteractionType: db.InteractionTypeLike,
		})
	} else if action == "unlike" {
		err = s.Repository.DeleteUserActivity(context.Background(), db.DeleteUserActivityParams{
			UserID:          userId,
			VideoID:         videoId,
			InteractionType: db.InteractionTypeLike,
		})
	} else {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid action"})
	}

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to toggle like"})
	}

	likes, err := s.Repository.CountVideoLikes(context.Background(), videoId)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to count likes"})
	}
	return c.JSON(fiber.Map{"message": "Success", "likes": likes, "isLiked": action == "like"})
}

func (s *Server) ListCommentRepliesHandler(c *fiber.Ctx) error {
	parentID, err := parseUUID(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid comment ID"})
	}
	replies, err := s.Repository.ListCommentReplies(context.Background(), parentID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch replies"})
	}

	result := make([]fiber.Map, 0, len(replies))
	for _, reply := range replies {
		author, err := s.Repository.GetUser(context.Background(), reply.AuthorID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch reply author"})
		}
		result = append(result, fiber.Map{
			"id":        formatUUID(reply.ID),
			"text":      reply.Text,
			"authorId":  formatUUID(reply.AuthorID),
			"author":    fiber.Map{"id": formatUUID(author.ID), "username": author.Username, "avatar": author.Avatar.String},
			"createdAt": reply.CreatedAt.Time,
		})
	}
	return c.JSON(result)
}

type UpdateCommentRequest struct {
	Text string `json:"text"`
}

func (s *Server) UpdateCommentHandler(c *fiber.Ctx) error {
	commentIdStr := c.Params("id")
	commentId, err := parseUUID(commentIdStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid comment ID"})
	}

	var req UpdateCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid payload"})
	}

	if req.Text == "" {
		return c.Status(400).JSON(fiber.Map{"message": "Text is required"})
	}

	userIdLocal := c.Locals("userId")
	if userIdLocal == nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}

	userId, _ := parseUUID(userIdLocal.(string))

	// Get comment
	comment, err := s.Repository.GetComment(context.Background(), commentId)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"message": "Comment not found"})
	}

	// Verify ownership
	if comment.AuthorID != userId {
		return c.Status(403).JSON(fiber.Map{"message": "User not authorized"})
	}

	// Update
	updatedComment, err := s.Repository.UpdateComment(context.Background(), db.UpdateCommentParams{
		ID:   commentId,
		Text: req.Text,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Server error"})
	}

	return c.JSON(fiber.Map{
		"id":        formatUUID(updatedComment.ID),
		"text":      updatedComment.Text,
		"authorId":  formatUUID(updatedComment.AuthorID),
		"createdAt": updatedComment.CreatedAt.Time,
	})
}

func (s *Server) DeleteCommentHandler(c *fiber.Ctx) error {
	commentIdStr := c.Params("id")
	commentId, err := parseUUID(commentIdStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid comment ID"})
	}

	userIdLocal := c.Locals("userId")
	if userIdLocal == nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}

	userId, _ := parseUUID(userIdLocal.(string))

	// Verify ownership
	comment, err := s.Repository.GetComment(context.Background(), commentId)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"message": "Comment not found"})
	}

	if comment.AuthorID != userId {
		return c.Status(403).JSON(fiber.Map{"message": "User not authorized"})
	}

	// Delete
	err = s.Repository.DeleteComment(context.Background(), db.DeleteCommentParams{
		ID:       commentId,
		AuthorID: userId,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Server error"})
	}

	return c.JSON(fiber.Map{"message": "Comment deleted successfully"})
}
