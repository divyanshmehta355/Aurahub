package api

import (
	"time"

	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Server) RecommendationsHandler(c *fiber.Ctx) error {
	page, limit := pageLimit(c, 12, 100)
	category := c.Query("category")
	if category == "All" {
		category = ""
	}
	var shortFilter pgtype.Bool
	if c.Query("type") == "short" {
		shortFilter = pgtype.Bool{Bool: true, Valid: true}
	} else {
		shortFilter = pgtype.Bool{Bool: false, Valid: true}
	}

	ctx := c.UserContext()
	showAdult := c.Query("adult") == "true"
	viewerUUID, _ := s.getOptionalUserID(c)
	seed := c.Query("seed")
	if seed == "" {
		seed = time.Now().Format("2006-01-02-15")
	}

	if s.Repository == nil {
		return c.JSON(fiber.Map{
			"videos":           []fiber.Map{},
			"currentPage":      page,
			"totalPages":       0,
			"totalVideos":      0,
			"seed":             seed,
			"isAiVectorSearch": false,
		})
	}

	videos, err := s.Repository.ListRecommendedVideos(ctx, viewerUUID, category, shortFilter, showAdult, seed, int32(limit), int32((page-1)*limit))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch recommendations"})
	}
	total, err := s.Repository.CountPublicVideos(ctx, category, shortFilter, showAdult)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to count videos"})
	}
	return c.JSON(fiber.Map{
		"videos":           toVideoSummaries(videos),
		"currentPage":      page,
		"totalPages":       (total + int64(limit) - 1) / int64(limit),
		"totalVideos":      total,
		"seed":             seed,
		"isAiVectorSearch": false,
	})
}

func (s *Server) SuggestionsHandler(c *fiber.Ctx) error {
	page, limit := pageLimit(c, 10, 100)
	offset := int32((page - 1) * limit)
	ctx := c.UserContext()
	showAdult := c.Query("adult") == "true"
	viewerUUID, _ := s.getOptionalUserID(c)
	seed := c.Query("seed")
	if seed == "" {
		seed = time.Now().Format("2006-01-02-15")
	}
	excludeID := pgtype.UUID{}
	var videos []db.Video
	var total int64

	if s.Repository == nil {
		return c.JSON(fiber.Map{
			"videos":           []fiber.Map{},
			"currentPage":      page,
			"totalPages":       0,
			"seed":             seed,
			"isAiVectorSearch": false,
		})
	}

	if exclude := c.Query("exclude"); exclude != "" {
		current, err := s.Repository.GetVideoByFileId(ctx, exclude)
		if err != nil {
			if id, parseErr := parseUUID(exclude); parseErr == nil {
				current, err = s.Repository.GetVideo(ctx, id)
			}
		}
		if err == nil {
			excludeID = current.ID
			var queryErr error
			videos, queryErr = s.Repository.ListSuggestedVideos(ctx, current.ID, current.UploaderID, viewerUUID, current.Category.String, []string(current.Tags), showAdult, seed, int32(limit), offset)
			if queryErr != nil {
				return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch suggestions"})
			}
			total, queryErr = s.Repository.CountSuggestedVideos(ctx, current.ID, showAdult)
			if queryErr != nil {
				return c.Status(500).JSON(fiber.Map{"message": "Failed to count suggestions"})
			}
		}
	}

	if !excludeID.Valid {
		var queryErr error
		videos, queryErr = s.Repository.ListRecommendedVideos(ctx, viewerUUID, "", pgtype.Bool{Bool: false, Valid: true}, showAdult, seed, int32(limit), offset)
		if queryErr != nil {
			return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch suggestions"})
		}
		total, queryErr = s.Repository.CountPublicVideos(ctx, "", pgtype.Bool{Bool: false, Valid: true}, showAdult)
		if queryErr != nil {
			return c.Status(500).JSON(fiber.Map{"message": "Failed to count suggestions"})
		}
	}

	return c.JSON(fiber.Map{
		"videos":           toVideoSummaries(videos),
		"currentPage":      page,
		"totalPages":       (total + int64(limit) - 1) / int64(limit),
		"seed":             seed,
		"isAiVectorSearch": false,
	})
}

func toVideoSummaries(videos []db.Video) []fiber.Map {
	result := make([]fiber.Map, 0, len(videos))
	for _, video := range videos {
		result = append(result, fiber.Map{
			"id":           formatUUID(video.ID),
			"fileId":       video.FileID,
			"title":        video.Title,
			"description":  video.Description.String,
			"thumbnailUrl": video.ThumbnailUrl.String,
			"category":     video.Category.String,
			"views":        video.Views.Int32,
			"isShort":      video.IsShort.Bool,
			"isAdult":      video.IsAdult.Bool,
			"createdAt":    video.CreatedAt.Time,
		})
	}
	return result
}
