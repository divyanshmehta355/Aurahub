package workers

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/divyanshmehta355/aurahub/backend/internal/kafka"
	"github.com/jackc/pgx/v5/pgtype"
)

type userActivityRecord struct {
	userID  pgtype.UUID
	videoID pgtype.UUID
}

// StartViewsFlusher launches a background consumer that aggregates video views from Kafka
// and flushes them to PostgreSQL in 5-second batches to eliminate DB write-lock contention.
func StartViewsFlusher(ctx context.Context, brokers []string, auth kafka.AuthConfig, repo *db.Repository, wg ...*sync.WaitGroup) {
	if len(brokers) == 0 {
		log.Println("[ViewsFlusher] Kafka not configured, worker will not start.")
		return
	}
	if len(wg) > 0 && wg[0] != nil {
		wg[0].Add(1)
		defer wg[0].Done()
	}

	reader := kafka.NewReader(brokers, kafka.GroupViewsFlusher, kafka.TopicVideoViews, auth)
	if reader == nil {
		return
	}
	defer reader.Close()

	var (
		mu         sync.Mutex
		viewCounts = make(map[pgtype.UUID]int)
		activities []userActivityRecord
	)

	// Background ticker to flush batched counts every 5 seconds
	tickerDone := make(chan struct{})
	go func() {
		defer close(tickerDone)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				flushViews(repo, &mu, viewCounts, &activities)
			}
		}
	}()

	log.Printf("[ViewsFlusher] Listening on Kafka topic: %s (group: %s)\n", kafka.TopicVideoViews, kafka.GroupViewsFlusher)

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			log.Printf("[ViewsFlusher] Error reading message: %v\n", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		var event kafka.VideoViewEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("[ViewsFlusher] Failed to parse view event: %v\n", err)
			continue
		}

		var videoUUID pgtype.UUID
		if err := videoUUID.Scan(event.VideoID); err != nil || !videoUUID.Valid {
			log.Printf("[ViewsFlusher] Warning: Invalid video UUID in view event: %s\n", event.VideoID)
			continue
		}

		mu.Lock()
		viewCounts[videoUUID]++

		if event.UserID != "" {
			var userUUID pgtype.UUID
			if err := userUUID.Scan(event.UserID); err == nil && userUUID.Valid {
				activities = append(activities, userActivityRecord{
					userID:  userUUID,
					videoID: videoUUID,
				})
			}
		}
		mu.Unlock()
	}

	// Wait for ticker goroutine to finish
	<-tickerDone

	// Flush any pending views remaining in memory
	flushViews(repo, &mu, viewCounts, &activities)
	log.Println("[ViewsFlusher] Clean shutdown finished, all pending views flushed.")
}

func flushViews(repo *db.Repository, mu *sync.Mutex, viewCounts map[pgtype.UUID]int, activities *[]userActivityRecord) {
	mu.Lock()
	if len(viewCounts) == 0 && len(*activities) == 0 {
		mu.Unlock()
		return
	}

	countsToFlush := make(map[pgtype.UUID]int, len(viewCounts))
	for k, v := range viewCounts {
		countsToFlush[k] = v
		delete(viewCounts, k)
	}

	activitiesToFlush := make([]userActivityRecord, len(*activities))
	copy(activitiesToFlush, *activities)
	*activities = (*activities)[:0]
	mu.Unlock()

	flushCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Batch increment video view counts in DB
	for videoID, count := range countsToFlush {
		if err := repo.IncrementVideoViewsBy(flushCtx, videoID, count); err != nil {
			log.Printf("[ViewsFlusher] Failed to increment views for video %v: %v\n", formatUUID(videoID), err)
		} else {
			log.Printf("[ViewsFlusher] Updated PostgreSQL: video %v views += %d\n", formatUUID(videoID), count)
		}
	}
	log.Printf("[ViewsFlusher] Successfully flushed batched views for %d video(s) to PostgreSQL\n", len(countsToFlush))

	// 2. Persist watch history & activities asynchronously
	for _, act := range activitiesToFlush {
		_ = repo.UpsertVideoViewActivity(flushCtx, act.userID, act.videoID)
		_ = repo.UpsertWatchHistory(flushCtx, db.UpsertWatchHistoryParams{
			UserID:  act.userID,
			VideoID: act.videoID,
		})
	}
}
