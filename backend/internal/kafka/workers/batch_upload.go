package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/divyanshmehta355/aurahub/backend/internal/httpclient"
	"github.com/divyanshmehta355/aurahub/backend/internal/kafka"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
)

// StartBatchUploadWorker listens on the batch-upload-jobs Kafka topic and performs
// asynchronous remote video downloads on Streamtape, detached from the client browser session.
func StartBatchUploadWorker(ctx context.Context, brokers []string, auth kafka.AuthConfig, repo *db.Repository, cache *redis.Client, producer *kafka.Producer, wg ...*sync.WaitGroup) {
	if len(brokers) == 0 {
		log.Println("[BatchUploadWorker] Kafka not configured, worker will not start.")
		return
	}
	if len(wg) > 0 && wg[0] != nil {
		wg[0].Add(1)
		defer wg[0].Done()
	}

	reader := kafka.NewReader(brokers, kafka.GroupBatchUploadJobs, kafka.TopicBatchUploadJobs, auth)
	if reader == nil {
		return
	}
	defer reader.Close()

	log.Printf("[BatchUploadWorker] Listening on Kafka topic: %s (group: %s)\n", kafka.TopicBatchUploadJobs, kafka.GroupBatchUploadJobs)

	// Bounded worker pool: limit concurrent upstream uploads to 5
	const maxConcurrentUploads = 5
	sem := make(chan struct{}, maxConcurrentUploads)
	var inFlightWg sync.WaitGroup

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			log.Printf("[BatchUploadWorker] Error reading message: %v\n", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		var job kafka.BatchUploadJobEvent
		if err := json.Unmarshal(msg.Value, &job); err != nil {
			log.Printf("[BatchUploadWorker] Invalid job payload: %v\n", err)
			continue
		}

		inFlightWg.Add(1)
		go func(j kafka.BatchUploadJobEvent) {
			defer inFlightWg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
				processUploadJob(ctx, j, repo, cache, producer)
			case <-ctx.Done():
				return
			}
		}(job)
	}

	// Drain in-flight jobs gracefully before exiting
	inFlightWg.Wait()
	log.Println("[BatchUploadWorker] Clean shutdown finished, all pending upload jobs completed.")
}

func processUploadJob(ctx context.Context, job kafka.BatchUploadJobEvent, repo *db.Repository, cache *redis.Client, producer *kafka.Producer) {
	publishJobProgress(ctx, cache, job.UserID, map[string]interface{}{
		"event":    "batch_upload_progress",
		"jobId":    job.JobID,
		"title":    job.Title,
		"status":   "queuing",
		"progress": 0,
	})

	// 1. Trigger remote upload on Streamtape / Aura integration
	query := url.Values{}
	query.Set("url", job.VideoURL)
	query.Set("folder", os.Getenv("UPLOAD_FOLDER_ID"))

	resBytes, err := httpclient.RequestAuraAPI(ctx, http.MethodGet, "/remote/add", query)
	if err != nil {
		publishJobProgress(ctx, cache, job.UserID, map[string]interface{}{
			"event":    "batch_upload_progress",
			"jobId":    job.JobID,
			"title":    job.Title,
			"status":   "error",
			"error":    "Failed to start upstream download: " + err.Error(),
			"progress": 0,
		})
		return
	}

	var startRes struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(resBytes, &startRes); err != nil || startRes.ID == "" {
		publishJobProgress(ctx, cache, job.UserID, map[string]interface{}{
			"event":    "batch_upload_progress",
			"jobId":    job.JobID,
			"title":    job.Title,
			"status":   "error",
			"error":    "Invalid upstream response from download service",
			"progress": 0,
		})
		return
	}

	publishJobProgress(ctx, cache, job.UserID, map[string]interface{}{
		"event":    "batch_upload_progress",
		"jobId":    job.JobID,
		"title":    job.Title,
		"status":   "downloading",
		"progress": 5,
	})

	// 2. Poll remote upload status
	finalVideoID, err := pollRemoteUpload(ctx, startRes.ID, job, cache)
	if err != nil {
		publishJobProgress(ctx, cache, job.UserID, map[string]interface{}{
			"event":    "batch_upload_progress",
			"jobId":    job.JobID,
			"title":    job.Title,
			"status":   "error",
			"error":    err.Error(),
			"progress": 0,
		})
		return
	}

	// 3. Persist completed video in PostgreSQL
	var userUUID pgtype.UUID
	_ = userUUID.Scan(job.UserID)

	videoVis := db.VideoVisibilityPublic
	if job.Visibility != "" {
		videoVis = db.VideoVisibility(job.Visibility)
	}

	createdVideo, err := repo.CreateVideo(ctx, db.CreateVideoParams{
		Title:            job.Title,
		FileID:           finalVideoID,
		Category:         pgtype.Text{String: job.Category, Valid: job.Category != ""},
		Visibility:       db.NullVideoVisibility{VideoVisibility: videoVis, Valid: true},
		UploaderID:       userUUID,
		IsShort:          pgtype.Bool{Bool: job.IsShort, Valid: true},
		IsAdult:          pgtype.Bool{Bool: job.IsAdult, Valid: true},
		StreamtapeUrl:    pgtype.Text{String: "https://streamtape.com/v/" + finalVideoID + "/", Valid: true},
		StreamtapeStatus: db.NullStreamtapeStatus{StreamtapeStatus: db.StreamtapeStatusActive, Valid: true},
	})
	if err != nil {
		publishJobProgress(ctx, cache, job.UserID, map[string]interface{}{
			"event":    "batch_upload_progress",
			"jobId":    job.JobID,
			"title":    job.Title,
			"status":   "error",
			"error":    "Failed to save video to database: " + err.Error(),
			"progress": 0,
		})
		return
	}

	// 4. Invalidate public feed cache so new video appears
	if cache != nil {
		iter := cache.Scan(ctx, 0, "feed:videos:*", 100).Iterator()
		var feedKeys []string
		for iter.Next(ctx) {
			feedKeys = append(feedKeys, iter.Val())
		}
		if len(feedKeys) > 0 {
			_ = cache.Del(ctx, feedKeys...).Err()
		}
	}

	// 5. Attach to playlist if requested
	if job.PlaylistID != "" {
		var playlistUUID pgtype.UUID
		if err := playlistUUID.Scan(job.PlaylistID); err == nil && playlistUUID.Valid {
			_ = repo.AddVideoToPlaylist(ctx, db.AddVideoToPlaylistParams{
				PlaylistID: playlistUUID,
				VideoID:    createdVideo.ID,
			})
		}
	}

	videoIDStr := formatUUID(createdVideo.ID)

	// 5. Emit video lifecycle event for OpenSearch indexing (Phase 3)
	if producer != nil {
		_ = producer.Publish(ctx, kafka.TopicVideoLifecycle, videoIDStr, kafka.VideoLifecycleEvent{
			Action:    "created",
			VideoID:   videoIDStr,
			Timestamp: time.Now(),
		})
	}

	// 6. Notify user of completion over SSE
	publishJobProgress(ctx, cache, job.UserID, map[string]interface{}{
		"event":    "batch_upload_progress",
		"jobId":    job.JobID,
		"title":    job.Title,
		"status":   "completed",
		"progress": 100,
		"videoId":  videoIDStr,
	})
}

func pollRemoteUpload(ctx context.Context, remoteID string, job kafka.BatchUploadJobEvent, cache *redis.Client) (string, error) {
	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()

	failedAttempts := 0

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			statusQuery := url.Values{}
			statusQuery.Set("id", remoteID)

			body, err := httpclient.RequestAuraAPI(ctx, http.MethodGet, "/remote/status", statusQuery)
			if err != nil {
				failedAttempts++
				if failedAttempts > 6 {
					return "", fmt.Errorf("lost connection to upstream download status")
				}
				continue
			}
			failedAttempts = 0

			var statusMap map[string]struct {
				Status       string `json:"status"`
				LinkID       string `json:"linkid"`
				BytesLoaded  int64  `json:"bytes_loaded"`
				BytesTotal   int64  `json:"bytes_total"`
				ErrorMessage string `json:"error_message"`
			}
			if err := json.Unmarshal(body, &statusMap); err != nil {
				continue
			}

			statusData, exists := statusMap[remoteID]
			if !exists {
				continue
			}

			switch statusData.Status {
			case "finished":
				return statusData.LinkID, nil
			case "error":
				errMsg := statusData.ErrorMessage
				if errMsg == "" {
					errMsg = "Remote download failed on upstream server"
				}
				return "", fmt.Errorf("%s", errMsg)
			default:
				if statusData.BytesTotal > 0 {
					pct := int((statusData.BytesLoaded * 100) / statusData.BytesTotal)
					if pct > 95 {
						pct = 95
					}
					publishJobProgress(ctx, cache, job.UserID, map[string]interface{}{
						"event":    "batch_upload_progress",
						"jobId":    job.JobID,
						"title":    job.Title,
						"status":   "downloading",
						"progress": pct,
					})
				}
			}
		}
	}
}

func publishJobProgress(ctx context.Context, cache *redis.Client, userID string, data map[string]interface{}) {
	if cache == nil || userID == "" {
		return
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	channel := fmt.Sprintf("notifications:user:%s", userID)
	_ = cache.Publish(ctx, channel, payload).Err()
}

func formatUUID(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	src := id.Bytes
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		src[0:4], src[4:6], src[6:8], src[8:10], src[10:16])
}
