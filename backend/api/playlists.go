package api

import (
	"context"
	"strings"

	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgtype"
)

type CreatePlaylistRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	IsPublic    *bool  `json:"isPublic"`
}

func (s *Server) CreatePlaylistHandler(c *fiber.Ctx) error {
	var req CreatePlaylistRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid payload"})
	}

	userIdLocal := c.Locals("userId")
	if userIdLocal == nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userId, err := parseUUID(userIdLocal.(string))
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return c.Status(400).JSON(fiber.Map{"message": "Title is required"})
	}
	isPublic := true
	if req.IsPublic != nil {
		isPublic = *req.IsPublic
	}

	playlist, err := s.Repository.CreatePlaylist(context.Background(), db.CreatePlaylistParams{
		Title:       title,
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
		OwnerID:     userId,
		IsPublic:    pgtype.Bool{Bool: isPublic, Valid: true},
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to create playlist"})
	}

	return c.Status(201).JSON(fiber.Map{
		"id":          formatUUID(playlist.ID),
		"title":       playlist.Title,
		"description": playlist.Description.String,
		"isPublic":    playlist.IsPublic.Bool,
	})
}

func (s *Server) ListPlaylistsHandler(c *fiber.Ctx) error {
	userIdLocal := c.Locals("userId")
	if userIdLocal == nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userId, err := parseUUID(userIdLocal.(string))
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}

	playlists, err := s.Repository.ListUserPlaylists(context.Background(), userId)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch playlists"})
	}

	var res []fiber.Map
	for _, p := range playlists {
		res = append(res, fiber.Map{
			"id":          formatUUID(p.ID),
			"title":       p.Title,
			"description": p.Description.String,
			"isPublic":    p.IsPublic.Bool,
			"updatedAt":   p.UpdatedAt.Time,
		})
	}

	return c.JSON(fiber.Map{"playlists": res})
}

func (s *Server) GetPlaylistHandler(c *fiber.Ctx) error {
	playlistIdStr := c.Params("id")
	playlistId, err := parseUUID(playlistIdStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid playlist ID"})
	}

	playlist, err := s.Repository.GetPlaylist(context.Background(), playlistId)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"message": "Playlist not found"})
	}
	if !playlist.IsPublic.Bool {
		viewerID, _ := c.Locals("userId").(string)
		viewerUUID, parseErr := parseUUID(viewerID)
		if parseErr != nil || viewerUUID != playlist.OwnerID {
			return c.Status(403).JSON(fiber.Map{"message": "This playlist is private"})
		}
	}

	videos, err := s.Repository.ListVisiblePlaylistVideos(context.Background(), playlistId)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to fetch playlist videos"})
	}

	var vRes []fiber.Map
	for _, v := range videos {
		vRes = append(vRes, fiber.Map{
			"id":           formatUUID(v.ID),
			"title":        v.Title,
			"thumbnailUrl": v.ThumbnailUrl.String,
			"views":        v.Views.Int32,
			"isShort":      v.IsShort.Bool,
		})
	}
	return c.JSON(fiber.Map{
		"playlist": fiber.Map{
			"id":          formatUUID(playlist.ID),
			"title":       playlist.Title,
			"description": playlist.Description.String,
			"isPublic":    playlist.IsPublic.Bool,
			"ownerId":     formatUUID(playlist.OwnerID),
		},
		"videos": vRes,
	})
}

func (s *Server) UpdatePlaylistCollectionHandler(c *fiber.Ctx) error {
	var req struct {
		PlaylistID    string   `json:"playlistId"`
		VideoID       string   `json:"videoId"`
		Title         string   `json:"title"`
		Description   *string  `json:"description"`
		IsPublic      *bool    `json:"isPublic"`
		NewVideoOrder []string `json:"newVideoOrder"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid payload"})
	}
	playlistID, err := parseUUID(req.PlaylistID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid playlist ID"})
	}
	userID, ok := c.Locals("userId").(string)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userUUID, err := parseUUID(userID)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	playlist, err := s.Repository.GetPlaylist(context.Background(), playlistID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"message": "Playlist not found"})
	}
	if playlist.OwnerID != userUUID {
		return c.Status(403).JSON(fiber.Map{"message": "User not authorized to edit this playlist"})
	}

	var isPublic pgtype.Bool
	if req.IsPublic != nil {
		isPublic = pgtype.Bool{Bool: *req.IsPublic, Valid: true}
	}
	var description interface{}
	if req.Description != nil {
		description = *req.Description
	}
	if req.Title != "" || req.Description != nil || req.IsPublic != nil {
		playlist, err = s.Repository.UpdatePlaylist(context.Background(), db.UpdatePlaylistParams{
			ID: playlistID, Column2: req.Title, Column3: description, IsPublic: isPublic,
		})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "Failed to update playlist"})
		}
	}

	if req.NewVideoOrder != nil {
		videoIDs := make([]pgtype.UUID, 0, len(req.NewVideoOrder))
		for _, id := range req.NewVideoOrder {
			videoID, err := parseUUID(id)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"message": "Invalid video ID in order"})
			}
			videoIDs = append(videoIDs, videoID)
		}
		if err := s.Repository.ReplacePlaylistVideoOrder(context.Background(), playlistID, videoIDs); err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "Failed to reorder playlist videos"})
		}
	} else if req.VideoID != "" {
		videoID, err := parseUUID(req.VideoID)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "Invalid video ID"})
		}
		exists, err := s.Repository.IsVideoInPlaylist(context.Background(), playlistID, videoID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "Failed to check playlist"})
		}
		if exists {
			err = s.Repository.RemoveVideoFromPlaylist(context.Background(), db.RemoveVideoFromPlaylistParams{PlaylistID: playlistID, VideoID: videoID})
		} else {
			err = s.Repository.AddVideoToPlaylist(context.Background(), db.AddVideoToPlaylistParams{PlaylistID: playlistID, VideoID: videoID})
		}
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "Failed to update playlist videos"})
		}
	}
	return c.JSON(fiber.Map{
		"id": formatUUID(playlist.ID), "title": playlist.Title,
		"description": playlist.Description.String, "isPublic": playlist.IsPublic.Bool,
	})
}

func (s *Server) UpdatePlaylistHandler(c *fiber.Ctx) error {
	playlistIdStr := c.Params("id")
	playlistId, err := parseUUID(playlistIdStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid playlist ID"})
	}

	var req CreatePlaylistRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid payload"})
	}

	userIdLocal := c.Locals("userId")
	userID, ok := userIdLocal.(string)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userId, err := parseUUID(userID)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}

	playlist, err := s.Repository.GetPlaylist(context.Background(), playlistId)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"message": "Playlist not found"})
	}
	if playlist.OwnerID != userId {
		return c.Status(403).JSON(fiber.Map{"message": "Forbidden"})
	}

	isPublic := pgtype.Bool{}
	if req.IsPublic != nil {
		isPublic = pgtype.Bool{Bool: *req.IsPublic, Valid: true}
	}
	updated, err := s.Repository.UpdatePlaylist(context.Background(), db.UpdatePlaylistParams{
		ID:       playlistId,
		Column2:  req.Title,
		Column3:  req.Description,
		IsPublic: isPublic,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Server error"})
	}

	return c.JSON(fiber.Map{
		"id":          formatUUID(updated.ID),
		"title":       updated.Title,
		"description": updated.Description.String,
		"isPublic":    updated.IsPublic.Bool,
	})
}

func (s *Server) DeletePlaylistHandler(c *fiber.Ctx) error {
	playlistIdStr := c.Params("id")
	playlistId, err := parseUUID(playlistIdStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid playlist ID"})
	}

	userIdLocal := c.Locals("userId")
	userID, ok := userIdLocal.(string)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userId, err := parseUUID(userID)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	playlist, err := s.Repository.GetPlaylist(context.Background(), playlistId)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"message": "Playlist not found"})
	}
	if playlist.OwnerID != userId {
		return c.Status(403).JSON(fiber.Map{"message": "User not authorized to delete this playlist"})
	}

	err = s.Repository.DeletePlaylist(context.Background(), db.DeletePlaylistParams{
		ID:      playlistId,
		OwnerID: userId,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to delete"})
	}

	return c.JSON(fiber.Map{"message": "Deleted"})
}

func (s *Server) TogglePlaylistVideoHandler(c *fiber.Ctx) error {
	playlistIdStr := c.Params("id")
	playlistId, err := parseUUID(playlistIdStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid playlist ID"})
	}

	var req struct {
		VideoID string `json:"videoId"`
		Action  string `json:"action"` // add, remove
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid payload"})
	}

	userID, ok := c.Locals("userId").(string)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userUUID, err := parseUUID(userID)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	playlist, err := s.Repository.GetPlaylist(context.Background(), playlistId)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"message": "Playlist not found"})
	}
	if playlist.OwnerID != userUUID {
		return c.Status(403).JSON(fiber.Map{"message": "User not authorized to edit this playlist"})
	}
	videoId, err := parseUUID(req.VideoID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid video ID"})
	}

	if req.Action == "add" {
		err = s.Repository.AddVideoToPlaylist(context.Background(), db.AddVideoToPlaylistParams{
			PlaylistID: playlistId,
			VideoID:    videoId,
		})
	} else if req.Action == "remove" {
		err = s.Repository.RemoveVideoFromPlaylist(context.Background(), db.RemoveVideoFromPlaylistParams{
			PlaylistID: playlistId,
			VideoID:    videoId,
		})
	} else {
		return c.Status(400).JSON(fiber.Map{"message": "Action must be add or remove"})
	}

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Action failed"})
	}
	return c.JSON(fiber.Map{"message": "Success"})
}
