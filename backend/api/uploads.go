package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	AuraApiBaseUrl  = "https://aurahub-api-hono.ashwathama249.workers.dev"
	FreeimageApiUrl = "https://freeimage.host/api/1/upload"
	uploadIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
)

func (s *Server) GetUploadUrlHandler(c *fiber.Ctx) error {
	query := url.Values{}
	query.Set("folder", os.Getenv("UPLOAD_FOLDER_ID"))
	body, err := requestAuraAPI(http.MethodGet, "/upload/url", query)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"message": "Failed to get upload URL"})
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return c.Status(502).JSON(fiber.Map{"message": "Invalid response from upload service"})
	}
	return c.JSON(data)
}

func (s *Server) UploadChunkHandler(c *fiber.Ctx) error {
	userID, ok := c.Locals("userId").(string)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	if _, err := parseUUID(userID); err != nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}

	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid form"})
	}

	uploadId := c.FormValue("uploadId")
	chunkIndex, chunkErr := strconv.Atoi(c.FormValue("chunkIndex"))
	totalChunks, totalErr := strconv.Atoi(c.FormValue("totalChunks"))
	fileName := c.FormValue("fileName")

	if chunkErr != nil || totalErr != nil || !uploadIDPattern.MatchString(uploadId) || fileName == "" ||
		totalChunks < 1 || totalChunks > 10000 || chunkIndex < 0 || chunkIndex >= totalChunks {
		return c.Status(400).JSON(fiber.Map{"message": "Missing parameters"})
	}

	files := form.File["chunk"]
	if len(files) == 0 {
		return c.Status(400).JSON(fiber.Map{"message": "Missing chunk"})
	}

	fileHeader := files[0]
	safeFileName := filepath.Base(strings.ReplaceAll(fileName, "\\", "/"))
	if safeFileName == "." || safeFileName == "" || len(safeFileName) > 255 {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid upload parameters"})
	}
	tempDir := os.TempDir()
	uploadKey := sha256.Sum256([]byte(userID + "\x00" + uploadId + "\x00" + safeFileName))
	uploadFilePath := filepath.Join(tempDir, "aurahub-upload-"+hex.EncodeToString(uploadKey[:]))

	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Error reading chunk"})
	}
	defer file.Close()

	destFile, err := os.OpenFile(uploadFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Error writing chunk"})
	}
	if _, err := io.Copy(destFile, file); err != nil {
		destFile.Close()
		return c.Status(500).JSON(fiber.Map{"message": "Error writing chunk"})
	}
	if err := destFile.Close(); err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Error writing chunk"})
	}

	if chunkIndex == totalChunks-1 {
		// Final chunk, upload to aura API
		query := url.Values{}
		query.Set("folder", os.Getenv("UPLOAD_FOLDER_ID"))
		urlBody, err := requestAuraAPI(http.MethodGet, "/upload/url", query)
		if err != nil {
			os.Remove(uploadFilePath)
			return c.Status(502).JSON(fiber.Map{"message": "Failed to get URL"})
		}

		var urlData struct {
			Url string `json:"url"`
		}
		if err := json.Unmarshal(urlBody, &urlData); err != nil {
			os.Remove(uploadFilePath)
			return c.Status(502).JSON(fiber.Map{"message": "Invalid upload URL response"})
		}
		if urlData.Url == "" {
			os.Remove(uploadFilePath)
			return c.Status(500).JSON(fiber.Map{"message": "Failed to get URL"})
		}

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", safeFileName)
		if err != nil {
			os.Remove(uploadFilePath)
			return c.Status(500).JSON(fiber.Map{"message": "Failed to prepare upload"})
		}

		finalFile, err := os.Open(uploadFilePath)
		if err != nil {
			os.Remove(uploadFilePath)
			return c.Status(500).JSON(fiber.Map{"message": "Failed to read completed upload"})
		}
		if _, err := io.Copy(part, finalFile); err != nil {
			finalFile.Close()
			os.Remove(uploadFilePath)
			return c.Status(500).JSON(fiber.Map{"message": "Failed to prepare upload"})
		}
		finalFile.Close()
		if err := writer.Close(); err != nil {
			os.Remove(uploadFilePath)
			return c.Status(500).JSON(fiber.Map{"message": "Failed to prepare upload"})
		}

		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, urlData.Url, body)
		if err != nil {
			os.Remove(uploadFilePath)
			return c.Status(502).JSON(fiber.Map{"message": "Invalid upload URL"})
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())

		uploadResp, err := (&http.Client{Timeout: 2 * time.Minute}).Do(req)
		os.Remove(uploadFilePath)

		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "Failed to upload"})
		}
		defer uploadResp.Body.Close()
		if uploadResp.StatusCode < 200 || uploadResp.StatusCode >= 300 {
			return c.Status(502).JSON(fiber.Map{"message": "Upload service rejected the file"})
		}

		var uploadData struct {
			Status int `json:"status"`
			Result struct {
				Id string `json:"id"`
			} `json:"result"`
		}
		if err := json.NewDecoder(io.LimitReader(uploadResp.Body, 1<<20)).Decode(&uploadData); err != nil || uploadData.Result.Id == "" {
			return c.Status(502).JSON(fiber.Map{"message": "Invalid upload service response"})
		}

		return c.JSON(fiber.Map{"completed": true, "videoId": uploadData.Result.Id})
	}

	return c.JSON(fiber.Map{"completed": false, "message": fmt.Sprintf("Chunk %d/%d received", chunkIndex+1, totalChunks)})
}

func (s *Server) CreateRecordHandler(c *fiber.Ctx) error {
	userIdLocal := c.Locals("userId")
	if userIdLocal == nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userId, err := parseUUID(userIdLocal.(string))
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}

	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid form"})
	}

	title := c.FormValue("title")
	description := c.FormValue("description")
	videoId := c.FormValue("videoId")
	category := c.FormValue("category")
	visibility := c.FormValue("visibility")
	isShort := c.FormValue("isShort") == "true"
	tagsString := c.FormValue("tags")

	if title == "" || videoId == "" {
		return c.Status(400).JSON(fiber.Map{"message": "Title and videoId required"})
	}
	playlistIDValue := c.FormValue("playlistId")
	var playlistID pgtype.UUID
	if playlistIDValue != "" {
		playlistID, err = parseUUID(playlistIDValue)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "Invalid playlist ID"})
		}
		playlist, err := s.Repository.GetPlaylist(context.Background(), playlistID)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"message": "Playlist not found"})
		}
		if playlist.OwnerID != userId {
			return c.Status(403).JSON(fiber.Map{"message": "Not authorized to add to this playlist"})
		}
	}

	var tags []string
	if tagsString != "" {
		if err := json.Unmarshal([]byte(tagsString), &tags); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "Invalid tags"})
		}
	}

	var thumbUrl string
	if len(form.File["thumbnailFile"]) > 0 {
		fileHeader := form.File["thumbnailFile"][0]
		file, err := fileHeader.Open()
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "Failed to read thumbnail"})
		}
		bytesData, err := io.ReadAll(io.LimitReader(file, 10<<20+1))
		file.Close()
		if err != nil || len(bytesData) > 10<<20 {
			return c.Status(400).JSON(fiber.Map{"message": "Invalid thumbnail file"})
		}
		thumbUrl, err = uploadImage(bytesData)
		if err != nil {
			return c.Status(502).JSON(fiber.Map{"message": "Failed to upload thumbnail"})
		}
	}

	videoVis := db.VideoVisibilityPublic
	if visibility != "" {
		switch db.VideoVisibility(visibility) {
		case db.VideoVisibilityPublic:
			videoVis = db.VideoVisibilityPublic
		case db.VideoVisibilityPrivate:
			videoVis = db.VideoVisibilityPrivate
		case db.VideoVisibilityUnlisted:
			videoVis = db.VideoVisibilityUnlisted
		default:
			return c.Status(400).JSON(fiber.Map{"message": "Invalid visibility"})
		}
	}

	video, err := s.Repository.CreateVideo(context.Background(), db.CreateVideoParams{
		Title:            title,
		Description:      pgtype.Text{String: description, Valid: description != ""},
		FileID:           videoId,
		ThumbnailUrl:     pgtype.Text{String: thumbUrl, Valid: thumbUrl != ""},
		Category:         pgtype.Text{String: category, Valid: category != ""},
		Tags:             tags,
		Visibility:       db.NullVideoVisibility{VideoVisibility: videoVis, Valid: true},
		UploaderID:       userId,
		IsShort:          pgtype.Bool{Bool: isShort, Valid: true},
		StreamtapeUrl:    pgtype.Text{String: "https://streamtape.com/v/" + videoId + "/", Valid: true},
		StreamtapeStatus: db.NullStreamtapeStatus{StreamtapeStatus: db.StreamtapeStatusActive, Valid: true},
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to save record"})
	}

	if playlistIDValue != "" {
		if err := s.Repository.AddVideoToPlaylist(context.Background(), db.AddVideoToPlaylistParams{
			PlaylistID: playlistID, VideoID: video.ID,
		}); err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "Video published, but failed to add it to the playlist"})
		}
	}

	return c.Status(201).JSON(fiber.Map{
		"message": "Video published successfully!",
		"video": fiber.Map{
			"id": formatUUID(video.ID),
		},
	})
}
