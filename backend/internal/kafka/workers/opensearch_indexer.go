package workers

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/divyanshmehta355/aurahub/backend/internal/kafka"
	"github.com/divyanshmehta355/aurahub/backend/internal/opensearch"
	"github.com/jackc/pgx/v5/pgtype"
)

// StartOpenSearchIndexer listens to the aurahub.video.lifecycle Kafka topic
// and automatically keeps OpenSearch documents in sync when videos are created, updated, or deleted.
func StartOpenSearchIndexer(ctx context.Context, brokers []string, auth kafka.AuthConfig, repo *db.Repository, osClient *opensearch.Client, wg ...*sync.WaitGroup) {
	if len(brokers) == 0 || osClient == nil || !osClient.Enabled() {
		log.Println("[OpenSearchIndexer] Kafka or OpenSearch not configured. Indexer operating in standby mode.")
		return
	}
	if len(wg) > 0 && wg[0] != nil {
		wg[0].Add(1)
		defer wg[0].Done()
	}

	reader := kafka.NewReader(brokers, kafka.GroupOpenSearchIndexer, kafka.TopicVideoLifecycle, auth)
	if reader == nil {
		return
	}
	defer reader.Close()

	log.Printf("[OpenSearchIndexer] Listening on Kafka topic: %s (group: %s)\n", kafka.TopicVideoLifecycle, kafka.GroupOpenSearchIndexer)

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			log.Printf("[OpenSearchIndexer] Error reading message: %v\n", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		var event kafka.VideoLifecycleEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("[OpenSearchIndexer] Failed to parse lifecycle event: %v\n", err)
			continue
		}

		if event.VideoID == "" {
			continue
		}

		switch event.Action {
		case "deleted":
			if err := osClient.DeleteVideo(ctx, event.VideoID); err != nil {
				log.Printf("[OpenSearchIndexer] Failed to delete video %s from index: %v\n", event.VideoID, err)
			} else {
				log.Printf("[OpenSearchIndexer] Deleted video %s from OpenSearch\n", event.VideoID)
			}

		case "created", "updated":
			detail, err := repo.GetVideoDetails(ctx, event.VideoID, pgtype.UUID{})
			if err != nil {
				log.Printf("[OpenSearchIndexer] Could not find video %s in DB: %v\n", event.VideoID, err)
				continue
			}

			// Do not index private videos in public search
			if detail.Visibility.VideoVisibility == db.VideoVisibilityPrivate {
				_ = osClient.DeleteVideo(ctx, event.VideoID)
				continue
			}

			doc := opensearch.VideoDocument{
				ID:               formatUUID(detail.ID),
				FileID:           detail.FileID,
				Title:            detail.Title,
				Description:      detail.Description.String,
				Category:         detail.Category.String,
				Tags:             detail.Tags,
				ThumbnailUrl:     detail.ThumbnailUrl.String,
				Views:            detail.Views.Int32,
				IsShort:          detail.IsShort.Bool,
				IsAdult:          detail.IsAdult.Bool,
				Visibility:       string(detail.Visibility.VideoVisibility),
				UploaderID:       formatUUID(detail.UploaderID),
				UploaderUsername: detail.UploaderUsername,
				CreatedAt:        detail.CreatedAt.Time,
			}

			if err := osClient.IndexVideo(ctx, doc); err != nil {
				log.Printf("[OpenSearchIndexer] Failed to index video %s: %v\n", event.VideoID, err)
			} else {
				log.Printf("[OpenSearchIndexer] Indexed video %s (%q) into OpenSearch\n", event.VideoID, detail.Title)
			}
		}
	}

	log.Println("[OpenSearchIndexer] Clean shutdown finished.")
}
