package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/divyanshmehta355/aurahub/backend/internal/httpclient"
	"github.com/divyanshmehta355/aurahub/backend/internal/kafka"
	"github.com/gofiber/fiber/v2"
)

func (s *Server) RemoteUploadStartHandler(c *fiber.Ctx) error {
	var request struct {
		VideoURL string `json:"videoUrl"`
	}
	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid request payload"})
	}
	if strings.TrimSpace(request.VideoURL) == "" {
		return c.Status(400).JSON(fiber.Map{"message": "Video URL is required"})
	}
	query := url.Values{}
	query.Set("url", request.VideoURL)
	query.Set("folder", os.Getenv("UPLOAD_FOLDER_ID"))
	response, err := requestAuraAPI(c.UserContext(), http.MethodGet, "/remote/add", query)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"message": "Failed to start remote upload"})
	}
	return c.Status(fiber.StatusAccepted).Type("json").Send(response)
}

func (s *Server) RemoteUploadStatusHandler(c *fiber.Ctx) error {
	remoteID := c.Query("id")
	if remoteID == "" {
		remoteID = c.Query("ids")
	}
	if remoteID == "" {
		return c.Status(400).JSON(fiber.Map{"message": "Remote ID is required."})
	}
	query := url.Values{}
	query.Set("id", remoteID)
	response, err := requestAuraAPI(c.UserContext(), http.MethodGet, "/remote/status", query)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"message": "Failed to check status"})
	}
	return c.Type("json").Send(response)
}

func requestAuraAPI(ctx context.Context, method, path string, query url.Values) ([]byte, error) {
	return httpclient.RequestAuraAPI(ctx, method, path, query)
}

func (s *Server) UpdateVideoThumbnailHandler(c *fiber.Ctx) error {
	userID, ok := c.Locals("userId").(string)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userUUID, err := parseUUID(userID)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	videoID, err := parseUUID(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid video ID"})
	}
	video, err := s.Repository.GetVideo(context.Background(), videoID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"message": "Video not found"})
	}
	if video.UploaderID != userUUID {
		return c.Status(403).JSON(fiber.Map{"message": "User not authorized"})
	}

	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid form"})
	}
	files := form.File["thumbnailFile"]
	if len(files) == 0 {
		return c.Status(400).JSON(fiber.Map{"message": "Thumbnail file is required."})
	}
	file, err := files[0].Open()
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Failed to read thumbnail"})
	}
	defer file.Close()
	imageBytes, err := io.ReadAll(io.LimitReader(file, 10<<20+1))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Failed to read thumbnail"})
	}
	if len(imageBytes) > 10<<20 {
		return c.Status(413).JSON(fiber.Map{"message": "Thumbnail exceeds the 10 MB limit"})
	}
	thumbnailURL, err := uploadImage(imageBytes)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"message": "Failed to upload thumbnail"})
	}
	if err := s.Repository.UpdateVideoThumbnail(context.Background(), video.ID, thumbnailURL); err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to update thumbnail"})
	}
	if s.Cache != nil {
		if err := s.Cache.Del(context.Background(), "thumbnail:"+c.Params("id")).Err(); err != nil {
			log.Printf("Thumbnail cache invalidation failed for video %s: %v", c.Params("id"), err)
		}
	}
	return c.JSON(fiber.Map{"thumbnailUrl": thumbnailURL})
}

func uploadImage(image []byte) (string, error) {
	apiKey := os.Getenv("FREEIMAGE_API_KEY")
	if apiKey == "" {
		return "", fiber.NewError(fiber.StatusServiceUnavailable, "FREEIMAGE_API_KEY is not configured")
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("key", apiKey); err != nil {
		return "", err
	}
	if err := writer.WriteField("source", base64.StdEncoding.EncodeToString(image)); err != nil {
		return "", err
	}
	if err := writer.WriteField("action", "upload"); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, FreeimageApiUrl, body)
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := httpclient.Default.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fiber.NewError(fiber.StatusBadGateway, "image host returned an error")
	}
	var result struct {
		StatusCode int `json:"status_code"`
		Image      struct {
			URL string `json:"url"`
		} `json:"image"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil {
		return "", err
	}
	if result.StatusCode != 200 || result.Image.URL == "" {
		return "", fiber.NewError(fiber.StatusBadGateway, "image host did not return an image URL")
	}
	return result.Image.URL, nil
}

type BatchRemoteUploadItem struct {
	Title    string `json:"title"`
	VideoURL string `json:"videoUrl"`
}

type BatchRemoteUploadRequest struct {
	PlaylistID string                  `json:"playlistId"`
	Category   string                  `json:"category"`
	Visibility string                  `json:"visibility"`
	IsShort    bool                    `json:"isShort"`
	IsAdult    bool                    `json:"isAdult"`
	Items      []BatchRemoteUploadItem `json:"items"`
}

// BatchRemoteUploadHandler queues a playlist batch of remote video downloads to Kafka,
// freeing the client browser from having to keep the tab open for polling.
func (s *Server) BatchRemoteUploadHandler(c *fiber.Ctx) error {
	userIdLocal := c.Locals("userId")
	if userIdLocal == nil {
		return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
	}
	userID := userIdLocal.(string)

	var req BatchRemoteUploadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid batch upload payload"})
	}

	if len(req.Items) == 0 {
		return c.Status(400).JSON(fiber.Map{"message": "At least one video item is required"})
	}

	batchID := fmt.Sprintf("batch-%d", time.Now().UnixNano())
	enqueuedCount := 0

	for _, item := range req.Items {
		title := strings.TrimSpace(item.Title)
		videoURL := strings.TrimSpace(item.VideoURL)
		if title == "" || videoURL == "" {
			continue
		}

		jobID := fmt.Sprintf("%s-%d", batchID, enqueuedCount)
		event := kafka.BatchUploadJobEvent{
			JobID:       jobID,
			UserID:      userID,
			PlaylistID:  req.PlaylistID,
			Title:       title,
			VideoURL:    videoURL,
			Category:    req.Category,
			Visibility:  req.Visibility,
			IsShort:     req.IsShort,
			IsAdult:     req.IsAdult,
			Timestamp:   time.Now(),
		}

		if s.Kafka != nil {
			_ = s.Kafka.Publish(c.UserContext(), kafka.TopicBatchUploadJobs, jobID, event)
		}
		enqueuedCount++
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"message":       "Batch upload queued successfully",
		"batchId":       batchID,
		"enqueuedCount": enqueuedCount,
	})
}

