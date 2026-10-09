package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/divyanshmehta355/aurahub/backend/config"
	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/divyanshmehta355/aurahub/backend/internal/opensearch"
	"github.com/jackc/pgx/v5/pgtype"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	cfg := config.LoadConfig()

	if cfg.OpenSearchURL == "" {
		log.Fatal("OPENSEARCH_URL must be configured in environment or .env")
	}

	// 1. Initialize OpenSearch
	osClient, err := opensearch.NewClient(opensearch.Config{
		Addresses:          []string{cfg.OpenSearchURL},
		Username:           cfg.OpenSearchUser,
		Password:           cfg.OpenSearchPassword,
		InsecureSkipVerify: cfg.OpenSearchInsecure,
	})
	if err != nil {
		log.Fatalf("Failed to initialize OpenSearch client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := osClient.Ping(ctx); err != nil {
		log.Fatalf("Cannot connect to OpenSearch cluster: %v", err)
	}

	log.Println("[OpenSearch Backfill] Connected to OpenSearch cluster. Ensuring index schema...")
	if err := osClient.EnsureIndex(ctx); err != nil {
		log.Fatalf("Failed to ensure index schema: %v", err)
	}

	// 2. Connect to PostgreSQL
	gormDB, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}

	type videoRow struct {
		db.Video
		UploaderUsername string `gorm:"column:uploader_username"`
	}

	var rows []videoRow
	err = gormDB.WithContext(ctx).Table("videos").
		Select("videos.*, users.username AS uploader_username").
		Joins("LEFT JOIN users ON users.id = videos.uploader_id").
		Where("videos.visibility = ? OR videos.visibility IS NULL", "public").
		Scan(&rows).Error
	if err != nil {
		log.Fatalf("Failed to query videos from database: %v", err)
	}

	log.Printf("[OpenSearch Backfill] Found %d public video(s) in PostgreSQL. Indexing...\n", len(rows))

	if len(rows) == 0 {
		log.Println("No videos to backfill.")
		return
	}

	// 3. Batch bulk index
	batchSize := 100
	totalIndexed := 0

	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}

		batch := rows[i:end]
		docs := make([]opensearch.VideoDocument, 0, len(batch))

		for _, r := range batch {
			docs = append(docs, opensearch.VideoDocument{
				ID:               formatUUID(r.ID),
				FileID:           r.FileID,
				Title:            r.Title,
				Description:      r.Description.String,
				Category:         r.Category.String,
				Tags:             r.Tags,
				ThumbnailUrl:     r.ThumbnailUrl.String,
				Views:            r.Views.Int32,
				IsShort:          r.IsShort.Bool,
				IsAdult:          r.IsAdult.Bool,
				Visibility:       string(r.Visibility.VideoVisibility),
				UploaderID:       formatUUID(r.UploaderID),
				UploaderUsername: r.UploaderUsername,
				CreatedAt:        r.CreatedAt.Time,
			})
		}

		count, err := osClient.BulkIndexVideos(ctx, docs)
		if err != nil {
			log.Printf("Warning: batch %d..%d failed: %v\n", i, end, err)
		} else {
			totalIndexed += count
		}
	}

	log.Printf("[OpenSearch Backfill] Successfully indexed %d / %d video(s) into %q!\n",
		totalIndexed, len(rows), opensearch.IndexVideos)
}

func formatUUID(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	src := id.Bytes
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		src[0:4], src[4:6], src[6:8], src[8:10], src[10:16])
}
