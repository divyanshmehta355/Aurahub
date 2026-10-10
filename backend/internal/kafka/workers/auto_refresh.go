package workers

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/divyanshmehta355/aurahub/backend/internal/httpclient"
	"github.com/divyanshmehta355/aurahub/backend/internal/kafka"
	"github.com/redis/go-redis/v9"
)

// Default settings for video refresh
const (
	DefaultRefreshDays     = 28
	DefaultRefreshInterval = 12 * time.Hour
	DefaultBatchLimit      = 10
)

// StartAutoRefreshWorker launches a background worker that periodically inspects
// the PostgreSQL database for videos older than the expiration threshold (default: 28 days),
// re-uploads them directly on Streamtape to reset their inactivity timer, swaps the file_id,
// and deletes the old copy from Streamtape.
func StartAutoRefreshWorker(ctx context.Context, repo *db.Repository, cache *redis.Client, producer *kafka.Producer, wg ...*sync.WaitGroup) {
	if repo == nil {
		log.Println("[AutoRefresh] Repository is nil. Auto-refresh worker will not start.")
		return
	}

	if len(wg) > 0 && wg[0] != nil {
		wg[0].Add(1)
		defer wg[0].Done()
	}

	refreshDays := DefaultRefreshDays
	if envDays := os.Getenv("VIDEO_REFRESH_DAYS"); envDays != "" {
		if val, err := strconv.Atoi(envDays); err == nil && val > 0 {
			refreshDays = val
		}
	}

	interval := DefaultRefreshInterval
	if envInterval := os.Getenv("VIDEO_REFRESH_INTERVAL"); envInterval != "" {
		if d, err := time.ParseDuration(envInterval); err == nil && d > 0 {
			interval = d
		}
	}

	log.Printf("[AutoRefresh] Video auto-refresh worker started (threshold: %d days, check interval: %v)\n", refreshDays, interval)

	// Initial delay of 1 minute after boot to allow other startup routines to complete
	initialTimer := time.NewTimer(1 * time.Minute)
	select {
	case <-ctx.Done():
		initialTimer.Stop()
		return
	case <-initialTimer.C:
		runCycleWithLogging(ctx, repo, cache, producer, time.Duration(refreshDays)*24*time.Hour, DefaultBatchLimit)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[AutoRefresh] Clean shutdown finished.")
			return
		case <-ticker.C:
			runCycleWithLogging(ctx, repo, cache, producer, time.Duration(refreshDays)*24*time.Hour, DefaultBatchLimit)
		}
	}
}

func runCycleWithLogging(ctx context.Context, repo *db.Repository, cache *redis.Client, producer *kafka.Producer, olderThan time.Duration, limit int) {
	log.Printf("[AutoRefresh] Running scheduled 28-day expiration check (older than %v)...\n", olderThan)
	refreshed, err := RunAutoRefreshCycle(ctx, repo, cache, producer, olderThan, limit)
	if err != nil {
		log.Printf("[AutoRefresh] Cycle completed with error: %v (refreshed: %d)\n", err, refreshed)
	} else if refreshed > 0 {
		log.Printf("[AutoRefresh] Cycle completed successfully. Refreshed %d video(s).\n", refreshed)
	} else {
		log.Println("[AutoRefresh] Cycle completed. No videos reached the expiration threshold.")
	}
}

// RunAutoRefreshCycle performs a single pass over expired videos and re-uploads them.
func RunAutoRefreshCycle(ctx context.Context, repo *db.Repository, cache *redis.Client, producer *kafka.Producer, olderThan time.Duration, limit int) (int, error) {
	if repo == nil {
		return 0, fmt.Errorf("repository is nil")
	}

	cutoff := time.Now().Add(-olderThan)
	videos, err := repo.GetExpiredVideos(ctx, cutoff, limit)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch expired videos: %w", err)
	}

	if len(videos) == 0 {
		return 0, nil
	}

	refreshedCount := 0
	folderID := os.Getenv("UPLOAD_FOLDER_ID")

	for _, v := range videos {
		select {
		case <-ctx.Done():
			return refreshedCount, ctx.Err()
		default:
		}

		err := RefreshSingleVideo(ctx, repo, cache, producer, v, folderID)
		if err != nil {
			log.Printf("[AutoRefresh] Failed to refresh video %v (%s, file %s): %v\n", formatUUID(v.ID), v.Title, v.FileID, err)
			continue
		}
		refreshedCount++

		// Brief rest between videos to stay well within Streamtape concurrency limits
		time.Sleep(2 * time.Second)
	}

	return refreshedCount, nil
}

// RefreshSingleVideo executes the two-phase renewal for an individual video:
// 1. Resolve direct stream URL from Streamtape
// 2. Trigger remote upload to clone the video to a new Streamtape file ID
// 3. Atomically update PostgreSQL with the new file_id and reset last_refreshed_at to NOW()
// 4. Invalidate cache and emit Kafka lifecycle update
// 5. Delete the old expired file on Streamtape
func RefreshSingleVideo(ctx context.Context, repo *db.Repository, cache *redis.Client, producer *kafka.Producer, video db.Video, folderID string) error {
	videoUUID := video.ID
	oldFileID := strings.TrimSpace(video.FileID)
	videoIDStr := formatUUID(videoUUID)

	if oldFileID == "" {
		return fmt.Errorf("video has empty file_id")
	}

	log.Printf("[AutoRefresh] Resolving direct stream URL for video %s (%q, file %s)...\n", videoIDStr, video.Title, oldFileID)

	// Step 1: Resolve playable media stream URL
	streamURL, err := httpclient.ResolveStreamtapeDirectURL(ctx, oldFileID)
	if err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			log.Printf("[AutoRefresh] Video %s file %s was deleted upstream. Marking as dead.\n", videoIDStr, oldFileID)
			_ = repo.MarkVideoStreamtapeDead(ctx, videoUUID)
			return fmt.Errorf("video file missing upstream: %w", err)
		}
		_ = repo.IncrementCloneAttempts(ctx, videoUUID)
		return fmt.Errorf("failed to resolve stream URL: %w", err)
	}

	// Step 2: Trigger remote upload on Streamtape
	log.Printf("[AutoRefresh] Triggering Streamtape remote upload for video %s...\n", videoIDStr)
	remoteID, err := httpclient.TriggerRemoteUpload(ctx, streamURL, folderID)
	if err != nil {
		_ = repo.IncrementCloneAttempts(ctx, videoUUID)
		return fmt.Errorf("failed to trigger remote upload: %w", err)
	}

	// Step 3: Poll remote upload status until complete
	newFileID, err := httpclient.PollRemoteUpload(ctx, remoteID, 15*time.Minute)
	if err != nil {
		_ = repo.IncrementCloneAttempts(ctx, videoUUID)
		return fmt.Errorf("remote upload failed for video %s: %w", videoIDStr, err)
	}

	if newFileID == "" || newFileID == oldFileID {
		_ = repo.IncrementCloneAttempts(ctx, videoUUID)
		return fmt.Errorf("invalid new file ID received: %q", newFileID)
	}

	newStreamtapeURL := "https://streamtape.com/v/" + newFileID + "/"

	// Step 4: Atomically update database record
	if err := repo.RefreshVideoFile(ctx, videoUUID, newFileID, newStreamtapeURL); err != nil {
		// New file was uploaded but DB failed to update - DO NOT delete old file!
		return fmt.Errorf("failed to update video metadata in DB: %w", err)
	}

	log.Printf("[AutoRefresh] Updated database for video %s: %s -> %s\n", videoIDStr, oldFileID, newFileID)

	// Step 5: Invalidate Valkey cache
	if cache != nil {
		_ = cache.Del(ctx, "stream_url:"+oldFileID).Err()
		_ = cache.Del(ctx, "video:"+videoIDStr).Err()
	}

	// Step 6: Notify Kafka lifecycle topic so OpenSearch updates the document
	if producer != nil {
		_ = producer.Publish(ctx, kafka.TopicVideoLifecycle, videoIDStr, kafka.VideoLifecycleEvent{
			VideoID:   videoIDStr,
			Action:    "updated",
			Timestamp: time.Now(),
		})
	}

	// Step 7: Delete old file from Streamtape storage
	if err := httpclient.DeleteStreamtapeFile(ctx, oldFileID); err != nil {
		// Log warning but don't fail the renewal, since the new file is already live in Aurahub
		log.Printf("[AutoRefresh] Warning: Could not delete old Streamtape file %s: %v\n", oldFileID, err)
	} else {
		log.Printf("[AutoRefresh] Successfully purged old Streamtape file %s\n", oldFileID)
	}

	return nil
}
