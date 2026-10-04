package mongomigrate

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

const (
	batchSize = 500
)

var errOrphanReference = errors.New("orphaned reference")

type Report map[string]int64

type importer struct {
	ctx       context.Context
	source    *mongo.Database
	target    *gorm.DB
	write     bool
	report    Report
	sourceIDs map[string]map[primitive.ObjectID]struct{}
	skipped   map[string]map[primitive.ObjectID]struct{}
}

func Run(ctx context.Context, source *mongo.Database, target *gorm.DB, write bool) (Report, error) {
	if source == nil {
		return nil, fmt.Errorf("MongoDB source is required")
	}
	if write && target == nil {
		return nil, fmt.Errorf("PostgreSQL target is required when applying the import")
	}
	i := importer{
		ctx:       ctx,
		source:    source,
		target:    target,
		write:     write,
		report:    make(Report),
		sourceIDs: make(map[string]map[primitive.ObjectID]struct{}),
		skipped:   make(map[string]map[primitive.ObjectID]struct{}),
	}
	for _, collection := range []string{"users", "videos", "comments", "playlists"} {
		ids, err := i.loadIDs(collection)
		if err != nil {
			return nil, err
		}
		i.sourceIDs[collection] = ids
	}

	run := func(tx *gorm.DB) error {
		if tx != nil {
			i.target = tx
		}
		for _, step := range []func() error{
			i.importUsers,
			i.importVideos,
			i.importComments,
			i.importNotifications,
			i.importPlaylists,
			i.importSubscriptions,
			i.importActivities,
			i.importEmbeddedLikes,
			i.importWatchHistory,
			i.importWatchLater,
			i.importPlaylistVideos,
		} {
			if err := step(); err != nil {
				return err
			}
		}
		return nil
	}

	if !write {
		return i.report, run(nil)
	}
	silentTarget := target.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
	err := silentTarget.WithContext(ctx).Transaction(run)
	return i.report, err
}

func (i *importer) importUsers() error {
	return processCollection(i, "users", func(doc bson.M) (db.User, error) {
		id, err := requiredID(doc, "_id")
		if err != nil {
			return db.User{}, err
		}
		email := stringValue(doc, "email")
		username := stringValue(doc, "username")
		password := stringValue(doc, "password")
		if email == "" || username == "" || password == "" {
			return db.User{}, fmt.Errorf("email, username, and password are required")
		}
		return db.User{
			ID:        pgUUID("users", id),
			Email:     email,
			Username:  username,
			Password:  password,
			Avatar:    nullableText(doc, "avatar"),
			Banner:    nullableText(doc, "banner"),
			Bio:       nullableText(doc, "bio"),
			CreatedAt: createdTimestamp(doc),
			UpdatedAt: timestampOrCreated(doc),
		}, nil
	}, func(rows []db.User) error {
		return i.insert(rows, "id")
	})
}

func (i *importer) importVideos() error {
	return processCollection(i, "videos", func(doc bson.M) (db.Video, error) {
		id, err := requiredID(doc, "_id")
		if err != nil {
			return db.Video{}, err
		}
		uploader, err := requiredID(doc, "uploader", "uploaderId")
		if err != nil {
			return db.Video{}, err
		}
		if err := i.requireReference("users", uploader); err != nil {
			return db.Video{}, err
		}
		title, fileID := stringValue(doc, "title"), stringValue(doc, "fileId", "file_id")
		if title == "" || fileID == "" {
			return db.Video{}, fmt.Errorf("title and fileId are required")
		}
		views, err := int32Value(doc, "views")
		if err != nil {
			return db.Video{}, fmt.Errorf("views: %w", err)
		}
		cloneAttempts, err := int32Value(doc, "cloneAttempts", "clone_attempts")
		if err != nil {
			return db.Video{}, fmt.Errorf("cloneAttempts: %w", err)
		}
		visibility := stringValue(doc, "visibility")
		if visibility == "" {
			visibility = string(db.VideoVisibilityPublic)
		}
		switch db.VideoVisibility(visibility) {
		case db.VideoVisibilityPublic, db.VideoVisibilityUnlisted, db.VideoVisibilityPrivate:
		default:
			return db.Video{}, fmt.Errorf("unsupported visibility %q", visibility)
		}
		status := stringValue(doc, "streamtapeStatus", "streamtape_status")
		if status == "" {
			status = string(db.StreamtapeStatusActive)
		}
		switch db.StreamtapeStatus(status) {
		case db.StreamtapeStatusActive, db.StreamtapeStatusDead, db.StreamtapeStatusPending:
		default:
			return db.Video{}, fmt.Errorf("unsupported streamtapeStatus %q", status)
		}
		category := stringValue(doc, "category")
		if category == "" {
			category = "Other"
		}
		tags, err := stringArray(doc, "tags")
		if err != nil {
			return db.Video{}, fmt.Errorf("tags: %w", err)
		}
		lastRefreshed := timestamp(doc, "lastRefreshedAt", "last_refreshed_at")
		if !lastRefreshed.Valid {
			lastRefreshed = timestampOrCreated(doc)
		}
		return db.Video{
			ID:                    pgUUID("videos", id),
			Title:                 title,
			Description:           nullableText(doc, "description"),
			FileID:                fileID,
			ThumbnailUrl:          nullableText(doc, "thumbnailUrl", "thumbnail_url"),
			Category:              pgtype.Text{String: category, Valid: true},
			Tags:                  db.TextArray(tags),
			Visibility:            db.NullVideoVisibility{VideoVisibility: db.VideoVisibility(visibility), Valid: true},
			UploaderID:            pgUUID("users", uploader),
			IsShort:               pgtype.Bool{Bool: boolValue(doc, "isShort", "is_short"), Valid: true},
			Views:                 pgtype.Int4{Int32: views, Valid: true},
			StreamtapeUrl:         nullableText(doc, "streamtapeUrl", "streamtape_url"),
			LastRefreshedAt:       lastRefreshed,
			PendingRemoteUploadID: nullableText(doc, "pendingRemoteUploadId", "pending_remote_upload_id"),
			StreamtapeStatus:      db.NullStreamtapeStatus{StreamtapeStatus: db.StreamtapeStatus(status), Valid: true},
			CloneAttempts:         pgtype.Int4{Int32: cloneAttempts, Valid: true},
			CreatedAt:             createdTimestamp(doc),
			UpdatedAt:             timestampOrCreated(doc),
		}, nil
	}, func(rows []db.Video) error {
		return i.insert(rows, "id")
	})
}

func (i *importer) importComments() error {
	cursor, err := i.source.Collection("comments").Find(i.ctx, bson.D{}, options.Find().SetBatchSize(batchSize))
	if err != nil {
		return fmt.Errorf("read MongoDB comments: %w", err)
	}
	defer cursor.Close(i.ctx)

	comments := make([]db.Comment, 0)
	parents := make(map[primitive.ObjectID]primitive.ObjectID)
	order := make([]primitive.ObjectID, 0)
	initiallySkipped := make(map[primitive.ObjectID]struct{})
	for cursor.Next(i.ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return fmt.Errorf("decode MongoDB comments document: %w", err)
		}
		id, err := requiredID(doc, "_id")
		if err != nil {
			return documentError("comments", doc, err)
		}
		author, err := requiredID(doc, "author", "authorId")
		if err != nil {
			return documentError("comments", doc, err)
		}
		video, err := requiredID(doc, "video", "videoId")
		if err != nil {
			return documentError("comments", doc, err)
		}
		if err := i.requireReference("users", author); errors.Is(err, errOrphanReference) {
			initiallySkipped[id] = struct{}{}
		} else if err != nil {
			return documentError("comments", doc, err)
		}
		if err := i.requireReference("videos", video); errors.Is(err, errOrphanReference) {
			initiallySkipped[id] = struct{}{}
		} else if err != nil {
			return documentError("comments", doc, err)
		}
		text := stringValue(doc, "text")
		if text == "" {
			return documentError("comments", doc, fmt.Errorf("text is required"))
		}
		row := db.Comment{
			ID:        pgUUID("comments", id),
			Text:      text,
			AuthorID:  pgUUID("users", author),
			VideoID:   pgUUID("videos", video),
			CreatedAt: createdTimestamp(doc),
			UpdatedAt: timestampOrCreated(doc),
		}
		if parent, ok, err := optionalID(doc, "parentComment", "parentCommentId", "parent_comment_id"); err != nil {
			return documentError("comments", doc, err)
		} else if ok {
			parents[id] = parent
		}
		comments = append(comments, row)
		order = append(order, id)
	}
	if err := cursor.Err(); err != nil {
		return fmt.Errorf("read MongoDB comments: %w", err)
	}
	sorted, skippedIDs, err := sortComments(comments, parents, order, initiallySkipped)
	if err != nil {
		return err
	}
	for _, id := range skippedIDs {
		i.markSkippedID("comments", id)
	}
	if err := i.insert(sorted, "id"); err != nil {
		return fmt.Errorf("import comments: %w", err)
	}
	i.report["comments"] = int64(len(sorted))
	return nil
}

func (i *importer) importNotifications() error {
	return processCollection(i, "notifications", func(doc bson.M) (db.Notification, error) {
		id, err := requiredID(doc, "_id")
		if err != nil {
			return db.Notification{}, err
		}
		recipient, err := requiredID(doc, "recipient", "recipientId")
		if err != nil {
			return db.Notification{}, err
		}
		sender, err := requiredID(doc, "sender", "senderId")
		if err != nil {
			return db.Notification{}, err
		}
		if err := i.requireReference("users", recipient); err != nil {
			return db.Notification{}, err
		}
		if err := i.requireReference("users", sender); err != nil {
			return db.Notification{}, err
		}
		notificationType := db.NotificationType(stringValue(doc, "type"))
		switch notificationType {
		case db.NotificationTypeLike, db.NotificationTypeComment, db.NotificationTypeReply, db.NotificationTypeNewVideo:
		default:
			return db.Notification{}, fmt.Errorf("unsupported notification type %q", notificationType)
		}
		row := db.Notification{
			ID:          pgUUID("notifications", id),
			RecipientID: pgUUID("users", recipient),
			SenderID:    pgUUID("users", sender),
			Type:        notificationType,
			IsRead:      pgtype.Bool{Bool: boolValue(doc, "isRead", "is_read"), Valid: true},
			CreatedAt:   createdTimestamp(doc),
			UpdatedAt:   timestampOrCreated(doc),
		}
		if video, ok, err := optionalID(doc, "video", "videoId"); err != nil {
			return db.Notification{}, err
		} else if ok {
			if err := i.requireReference("videos", video); err != nil {
				return db.Notification{}, err
			}
			row.VideoID = pgUUID("videos", video)
		}
		if comment, ok, err := optionalID(doc, "comment", "commentId"); err != nil {
			return db.Notification{}, err
		} else if ok {
			if err := i.requireReference("comments", comment); err != nil {
				return db.Notification{}, err
			}
			row.CommentID = pgUUID("comments", comment)
		}
		return row, nil
	}, func(rows []db.Notification) error {
		return i.insert(rows, "id")
	})
}

func (i *importer) importPlaylists() error {
	return processCollection(i, "playlists", func(doc bson.M) (db.Playlist, error) {
		id, err := requiredID(doc, "_id")
		if err != nil {
			return db.Playlist{}, err
		}
		owner, err := requiredID(doc, "owner", "ownerId")
		if err != nil {
			return db.Playlist{}, err
		}
		if err := i.requireReference("users", owner); err != nil {
			return db.Playlist{}, err
		}
		title := stringValue(doc, "title")
		if title == "" {
			return db.Playlist{}, fmt.Errorf("title is required")
		}
		return db.Playlist{
			ID:          pgUUID("playlists", id),
			Title:       title,
			Description: nullableText(doc, "description"),
			OwnerID:     pgUUID("users", owner),
			IsPublic:    pgtype.Bool{Bool: boolDefault(doc, true, "isPublic", "is_public"), Valid: true},
			CreatedAt:   createdTimestamp(doc),
			UpdatedAt:   timestampOrCreated(doc),
		}, nil
	}, func(rows []db.Playlist) error {
		return i.insert(rows, "id")
	})
}

func (i *importer) importSubscriptions() error {
	batch := make([]db.Subscription, 0, batchSize)
	flush := func() error {
		if err := i.insert(batch, "subscriber_id", "subscribed_to_id"); err != nil {
			return err
		}
		batch = batch[:0]
		return nil
	}
	cursor, err := i.source.Collection("users").Find(i.ctx, bson.D{}, options.Find().SetBatchSize(batchSize))
	if err != nil {
		return fmt.Errorf("read MongoDB subscriptions from users: %w", err)
	}
	defer cursor.Close(i.ctx)
	for cursor.Next(i.ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return fmt.Errorf("decode MongoDB subscription document: %w", err)
		}
		target, err := requiredID(doc, "_id")
		if err != nil {
			return documentError("users", doc, err)
		}
		createdAt := createdTimestamp(doc)
		for _, field := range []string{"subscriptions", "subscribers"} {
			ids, err := idArray(doc, field)
			if err != nil {
				return documentError("users", doc, fmt.Errorf("%s: %w", field, err))
			}
			for _, related := range ids {
				if err := i.requireReference("users", related); errors.Is(err, errOrphanReference) {
					i.markSkipped("subscriptions")
					continue
				} else if err != nil {
					return documentError("users", doc, err)
				}
				row := db.Subscription{CreatedAt: createdAt}
				if field == "subscriptions" {
					row.SubscriberID = pgUUID("users", target)
					row.SubscribedToID = pgUUID("users", related)
				} else {
					row.SubscriberID = pgUUID("users", related)
					row.SubscribedToID = pgUUID("users", target)
				}
				batch = append(batch, row)
				i.report["subscriptions"]++
				if len(batch) == cap(batch) {
					if err := flush(); err != nil {
						return fmt.Errorf("import subscriptions: %w", err)
					}
				}
			}
		}
	}
	if err := cursor.Err(); err != nil {
		return fmt.Errorf("read MongoDB subscriptions from users: %w", err)
	}
	if len(batch) > 0 {
		if err := flush(); err != nil {
			return fmt.Errorf("import subscriptions: %w", err)
		}
	}
	return nil
}

func (i *importer) importActivities() error {
	return processCollection(i, "useractivities", func(doc bson.M) (db.UserActivity, error) {
		userID, err := requiredID(doc, "userId", "user_id")
		if err != nil {
			return db.UserActivity{}, err
		}
		videoID, err := requiredID(doc, "videoId", "video_id")
		if err != nil {
			return db.UserActivity{}, err
		}
		if err := i.requireReference("users", userID); err != nil {
			return db.UserActivity{}, err
		}
		if err := i.requireReference("videos", videoID); err != nil {
			return db.UserActivity{}, err
		}
		interaction := db.InteractionType(stringValue(doc, "interactionType", "interaction_type"))
		if interaction != db.InteractionTypeLike && interaction != db.InteractionTypeView {
			return db.UserActivity{}, fmt.Errorf("unsupported interaction type %q", interaction)
		}
		return activityRow(userID, videoID, interaction, createdTimestamp(doc)), nil
	}, func(rows []db.UserActivity) error {
		return i.insertWithConflict(rows, "user_id", "video_id", "interaction_type")
	})
}

func (i *importer) importEmbeddedLikes() error {
	batch := make([]db.UserActivity, 0, batchSize)
	flush := func() error {
		if err := i.insertWithConflict(batch, "user_id", "video_id", "interaction_type"); err != nil {
			return err
		}
		batch = batch[:0]
		return nil
	}
	cursor, err := i.source.Collection("videos").Find(i.ctx, bson.D{}, options.Find().SetBatchSize(batchSize))
	if err != nil {
		return fmt.Errorf("read MongoDB video likes: %w", err)
	}
	defer cursor.Close(i.ctx)
	for cursor.Next(i.ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return fmt.Errorf("decode MongoDB video likes document: %w", err)
		}
		videoID, err := requiredID(doc, "_id")
		if err != nil {
			return documentError("videos", doc, err)
		}
		likes, err := idArray(doc, "likes")
		if err != nil {
			return documentError("videos", doc, fmt.Errorf("likes: %w", err))
		}
		createdAt := createdTimestamp(doc)
		for _, userID := range likes {
			if err := i.requireReference("videos", videoID); errors.Is(err, errOrphanReference) {
				i.markSkipped("embedded_likes")
				continue
			} else if err != nil {
				return documentError("videos", doc, err)
			}
			if err := i.requireReference("users", userID); errors.Is(err, errOrphanReference) {
				i.markSkipped("embedded_likes")
				continue
			} else if err != nil {
				return documentError("videos", doc, err)
			}
			batch = append(batch, activityRow(userID, videoID, db.InteractionTypeLike, createdAt))
			i.report["embedded_likes"]++
			if len(batch) == cap(batch) {
				if err := flush(); err != nil {
					return fmt.Errorf("import embedded video likes: %w", err)
				}
			}
		}
	}
	if err := cursor.Err(); err != nil {
		return fmt.Errorf("read MongoDB video likes: %w", err)
	}
	if len(batch) > 0 {
		if err := flush(); err != nil {
			return fmt.Errorf("import embedded video likes: %w", err)
		}
	}
	return nil
}

func (i *importer) importWatchHistory() error {
	return i.importUserVideoCollection("watchhistories", "watch_history")
}

func (i *importer) importWatchLater() error {
	return i.importUserVideoCollection("watchlaters", "watch_later")
}

func (i *importer) importUserVideoCollection(collection, table string) error {
	transform := func(doc bson.M) (any, error) {
		userID, err := requiredID(doc, "userId", "user_id")
		if err != nil {
			return nil, err
		}
		videoID, err := requiredID(doc, "videoId", "video_id")
		if err != nil {
			return nil, err
		}
		if err := i.requireReference("users", userID); err != nil {
			return nil, err
		}
		if err := i.requireReference("videos", videoID); err != nil {
			return nil, err
		}
		createdAt := createdTimestamp(doc)
		updatedAt := timestampOrCreated(doc)
		if table == "watch_history" {
			return db.WatchHistory{
				UserID: pgUUID("users", userID), VideoID: pgUUID("videos", videoID),
				CreatedAt: createdAt, UpdatedAt: updatedAt,
			}, nil
		}
		return db.WatchLater{
			UserID: pgUUID("users", userID), VideoID: pgUUID("videos", videoID),
			CreatedAt: createdAt, UpdatedAt: updatedAt,
		}, nil
	}
	return processCollection(i, collection, transform, func(rows []any) error {
		if table == "watch_history" {
			typed := make([]db.WatchHistory, 0, len(rows))
			for _, row := range rows {
				typed = append(typed, row.(db.WatchHistory))
			}
			return i.insert(typed, "user_id", "video_id")
		}
		typed := make([]db.WatchLater, 0, len(rows))
		for _, row := range rows {
			typed = append(typed, row.(db.WatchLater))
		}
		return i.insert(typed, "user_id", "video_id")
	})
}

func (i *importer) importPlaylistVideos() error {
	batch := make([]db.PlaylistVideo, 0, batchSize)
	flush := func() error {
		if err := i.insert(batch, "playlist_id", "video_id"); err != nil {
			return err
		}
		batch = batch[:0]
		return nil
	}
	cursor, err := i.source.Collection("playlists").Find(i.ctx, bson.D{}, options.Find().SetBatchSize(batchSize))
	if err != nil {
		return fmt.Errorf("read MongoDB playlist videos: %w", err)
	}
	defer cursor.Close(i.ctx)
	for cursor.Next(i.ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return fmt.Errorf("decode MongoDB playlist videos document: %w", err)
		}
		playlistID, err := requiredID(doc, "_id")
		if err != nil {
			return documentError("playlists", doc, err)
		}
		videos, err := idArray(doc, "videos")
		if err != nil {
			return documentError("playlists", doc, fmt.Errorf("videos: %w", err))
		}
		seen := make(map[primitive.ObjectID]struct{}, len(videos))
		baseTime := createdTimestamp(doc).Time
		for index, videoID := range videos {
			if err := i.requireReference("playlists", playlistID); errors.Is(err, errOrphanReference) {
				i.markSkipped("playlist_videos")
				continue
			} else if err != nil {
				return documentError("playlists", doc, err)
			}
			if err := i.requireReference("videos", videoID); errors.Is(err, errOrphanReference) {
				i.markSkipped("playlist_videos")
				continue
			} else if err != nil {
				return documentError("playlists", doc, err)
			}
			if _, exists := seen[videoID]; exists {
				return documentError("playlists", doc, fmt.Errorf("duplicate video in playlist"))
			}
			seen[videoID] = struct{}{}
			batch = append(batch, db.PlaylistVideo{
				PlaylistID: pgUUID("playlists", playlistID),
				VideoID:    pgUUID("videos", videoID),
				AddedAt:    pgtype.Timestamptz{Time: baseTime.Add(time.Duration(index) * time.Microsecond), Valid: true},
			})
			i.report["playlist_videos"]++
			if len(batch) == cap(batch) {
				if err := flush(); err != nil {
					return fmt.Errorf("import playlist videos: %w", err)
				}
			}
		}
	}
	if err := cursor.Err(); err != nil {
		return fmt.Errorf("read MongoDB playlist videos: %w", err)
	}
	if len(batch) > 0 {
		if err := flush(); err != nil {
			return fmt.Errorf("import playlist videos: %w", err)
		}
	}
	return nil
}

func processCollection[T any](i *importer, collection string, transform func(bson.M) (T, error), write func([]T) error) error {
	cursor, err := i.source.Collection(collection).Find(i.ctx, bson.D{}, options.Find().SetBatchSize(batchSize))
	if err != nil {
		return fmt.Errorf("read MongoDB %s: %w", collection, err)
	}
	defer cursor.Close(i.ctx)
	batch := make([]T, 0, batchSize)
	for cursor.Next(i.ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return fmt.Errorf("decode MongoDB %s document: %w", collection, err)
		}
		row, err := transform(doc)
		if err != nil {
			if errors.Is(err, errOrphanReference) {
				i.markSkippedDocument(collection, doc)
				continue
			}
			return documentError(collection, doc, err)
		}
		batch = append(batch, row)
		i.report[collection]++
		if len(batch) == cap(batch) {
			if err := write(batch); err != nil {
				return fmt.Errorf("import %s: %w", collection, err)
			}
			batch = batch[:0]
		}
	}
	if err := cursor.Err(); err != nil {
		return fmt.Errorf("read MongoDB %s: %w", collection, err)
	}
	if len(batch) > 0 {
		if err := write(batch); err != nil {
			return fmt.Errorf("import %s: %w", collection, err)
		}
	}
	return nil
}

func (i *importer) insert(rows any, conflictColumns ...string) error {
	return i.insertWithConflict(rows, conflictColumns...)
}

func (i *importer) loadIDs(collection string) (map[primitive.ObjectID]struct{}, error) {
	cursor, err := i.source.Collection(collection).Find(i.ctx, bson.D{}, options.Find().SetBatchSize(batchSize))
	if err != nil {
		return nil, fmt.Errorf("read MongoDB %s identifiers: %w", collection, err)
	}
	defer cursor.Close(i.ctx)
	ids := make(map[primitive.ObjectID]struct{})
	for cursor.Next(i.ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("decode MongoDB %s identifier: %w", collection, err)
		}
		id, err := requiredID(doc, "_id")
		if err != nil {
			return nil, documentError(collection, doc, err)
		}
		ids[id] = struct{}{}
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("read MongoDB %s identifiers: %w", collection, err)
	}
	return ids, nil
}

func (i *importer) requireReference(collection string, id primitive.ObjectID) error {
	if _, exists := i.sourceIDs[collection][id]; !exists {
		return fmt.Errorf("%w: references a missing %s document", errOrphanReference, collection)
	}
	if _, skipped := i.skipped[collection][id]; skipped {
		return fmt.Errorf("%w: references a skipped %s document", errOrphanReference, collection)
	}
	return nil
}

func (i *importer) markSkipped(collection string) {
	i.report["skipped_orphans."+collection]++
}

func (i *importer) markSkippedID(collection string, id primitive.ObjectID) {
	if i.skipped[collection] == nil {
		i.skipped[collection] = make(map[primitive.ObjectID]struct{})
	}
	if _, exists := i.skipped[collection][id]; exists {
		return
	}
	i.skipped[collection][id] = struct{}{}
	i.markSkipped(collection)
}

func (i *importer) markSkippedDocument(collection string, doc bson.M) {
	if id, ok, err := optionalID(doc, "_id"); err == nil && ok {
		i.markSkippedID(collection, id)
		return
	}
	i.markSkipped(collection)
}

func (i *importer) insertWithConflict(rows any, conflictColumns ...string) error {
	if !i.write {
		return nil
	}
	columns := make([]clause.Column, 0, len(conflictColumns))
	for _, name := range conflictColumns {
		columns = append(columns, clause.Column{Name: name})
	}
	result := i.target.Clauses(clause.OnConflict{Columns: columns, DoNothing: true}).CreateInBatches(rows, batchSize)
	if result.Error != nil {
		return result.Error
	}
	i.report["inserted."+modelTable(rows)] += result.RowsAffected
	return nil
}

func modelTable(rows any) string {
	switch values := rows.(type) {
	case []db.User:
		return "users"
	case []db.Video:
		return "videos"
	case []db.Comment:
		return "comments"
	case []db.Notification:
		return "notifications"
	case []db.Playlist:
		return "playlists"
	case []db.PlaylistVideo:
		return "playlist_videos"
	case []db.Subscription:
		return "subscriptions"
	case []db.UserActivity:
		return "user_activities"
	case []db.WatchHistory:
		return "watch_history"
	case []db.WatchLater:
		return "watch_later"
	case []any:
		if len(values) > 0 {
			switch values[0].(type) {
			case db.WatchHistory:
				return "watch_history"
			case db.WatchLater:
				return "watch_later"
			}
		}
	}
	return "unknown"
}

func activityRow(userID, videoID primitive.ObjectID, kind db.InteractionType, createdAt pgtype.Timestamptz) db.UserActivity {
	userUUID, videoUUID := mappedUUID("users", userID), mappedUUID("videos", videoID)
	key := fmt.Sprintf("aurahub-mongo-v1:activity:%x:%x:%s", userUUID, videoUUID, kind)
	return db.UserActivity{
		ID:              pgtype.UUID{Bytes: uuidBytes(uuid.NewSHA1(uuid.NameSpaceOID, []byte(key))), Valid: true},
		UserID:          pgtype.UUID{Bytes: uuidBytes(userUUID), Valid: true},
		VideoID:         pgtype.UUID{Bytes: uuidBytes(videoUUID), Valid: true},
		InteractionType: kind,
		CreatedAt:       createdAt,
		UpdatedAt:       createdAt,
	}
}

func sortComments(rows []db.Comment, parents map[primitive.ObjectID]primitive.ObjectID, order []primitive.ObjectID, initiallySkipped map[primitive.ObjectID]struct{}) ([]db.Comment, []primitive.ObjectID, error) {
	byID := make(map[primitive.ObjectID]db.Comment, len(rows))
	for index, id := range order {
		byID[id] = rows[index]
	}
	state := make(map[primitive.ObjectID]uint8, len(rows))
	skipped := make(map[primitive.ObjectID]struct{}, len(initiallySkipped))
	for id := range initiallySkipped {
		skipped[id] = struct{}{}
	}
	skippedIDs := make([]primitive.ObjectID, 0, len(initiallySkipped))
	for id := range initiallySkipped {
		skippedIDs = append(skippedIDs, id)
	}
	sorted := make([]db.Comment, 0, len(rows))
	var visit func(primitive.ObjectID) error
	visit = func(id primitive.ObjectID) error {
		switch state[id] {
		case 1:
			return fmt.Errorf("comment parent cycle detected at MongoDB id %s", id.Hex())
		case 2:
			return nil
		}
		state[id] = 1
		if _, shouldSkip := skipped[id]; shouldSkip {
			state[id] = 2
			return nil
		}
		row := byID[id]
		if parent, ok := parents[id]; ok {
			if _, exists := byID[parent]; !exists {
				skipped[id] = struct{}{}
				skippedIDs = append(skippedIDs, id)
				state[id] = 2
				return nil
			}
			if err := visit(parent); err != nil {
				return err
			}
			if _, parentSkipped := skipped[parent]; parentSkipped {
				skipped[id] = struct{}{}
				skippedIDs = append(skippedIDs, id)
				state[id] = 2
				return nil
			}
			row.ParentCommentID = pgUUID("comments", parent)
			byID[id] = row
		}
		state[id] = 2
		sorted = append(sorted, row)
		return nil
	}
	for _, id := range order {
		if err := visit(id); err != nil {
			return nil, nil, err
		}
	}
	return sorted, skippedIDs, nil
}

func requiredID(doc bson.M, fields ...string) (primitive.ObjectID, error) {
	id, ok, err := optionalID(doc, fields...)
	if err != nil {
		return primitive.NilObjectID, err
	}
	if !ok {
		return primitive.NilObjectID, fmt.Errorf("%s is required", fields[0])
	}
	return id, nil
}

func optionalID(doc bson.M, fields ...string) (primitive.ObjectID, bool, error) {
	for _, field := range fields {
		value, exists := doc[field]
		if !exists || value == nil {
			continue
		}
		switch id := value.(type) {
		case primitive.ObjectID:
			return id, true, nil
		case string:
			parsed, err := primitive.ObjectIDFromHex(id)
			if err != nil {
				return primitive.NilObjectID, false, fmt.Errorf("%s is not a valid ObjectId", field)
			}
			return parsed, true, nil
		default:
			return primitive.NilObjectID, false, fmt.Errorf("%s has unsupported id type %T", field, value)
		}
	}
	return primitive.NilObjectID, false, nil
}

func idArray(doc bson.M, field string) ([]primitive.ObjectID, error) {
	value, exists := doc[field]
	if !exists || value == nil {
		return nil, nil
	}
	var values []any
	switch items := value.(type) {
	case bson.A:
		values = items
	case []any:
		values = items
	case []primitive.ObjectID:
		values = make([]any, len(items))
		for index, item := range items {
			values[index] = item
		}
	default:
		return nil, fmt.Errorf("expected an array, got %T", value)
	}
	result := make([]primitive.ObjectID, 0, len(values))
	for index, item := range values {
		switch id := item.(type) {
		case primitive.ObjectID:
			result = append(result, id)
		case string:
			parsed, err := primitive.ObjectIDFromHex(id)
			if err != nil {
				return nil, fmt.Errorf("item %d is not a valid ObjectId", index)
			}
			result = append(result, parsed)
		default:
			return nil, fmt.Errorf("item %d has unsupported id type %T", index, item)
		}
	}
	return result, nil
}

func stringArray(doc bson.M, field string) ([]string, error) {
	value, exists := doc[field]
	if !exists || value == nil {
		return []string{}, nil
	}
	var values []any
	switch items := value.(type) {
	case bson.A:
		values = items
	case []any:
		values = items
	case []string:
		return items, nil
	default:
		return nil, fmt.Errorf("expected an array, got %T", value)
	}
	result := make([]string, 0, len(values))
	for index, item := range values {
		text, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("item %d is not a string", index)
		}
		result = append(result, text)
	}
	return result, nil
}

func stringValue(doc bson.M, fields ...string) string {
	for _, field := range fields {
		if value, ok := doc[field].(string); ok {
			return value
		}
	}
	return ""
}

func nullableText(doc bson.M, fields ...string) pgtype.Text {
	for _, field := range fields {
		if value, exists := doc[field]; exists {
			if value == nil {
				return pgtype.Text{}
			}
			if text, ok := value.(string); ok {
				return pgtype.Text{String: text, Valid: true}
			}
		}
	}
	return pgtype.Text{}
}

func boolValue(doc bson.M, fields ...string) bool {
	for _, field := range fields {
		if value, ok := doc[field].(bool); ok {
			return value
		}
	}
	return false
}

func boolDefault(doc bson.M, fallback bool, fields ...string) bool {
	for _, field := range fields {
		if value, ok := doc[field].(bool); ok {
			return value
		}
	}
	return fallback
}

func int32Value(doc bson.M, fields ...string) (int32, error) {
	var value any
	for _, field := range fields {
		if candidate, exists := doc[field]; exists {
			value = candidate
			break
		}
	}
	if value == nil {
		return 0, nil
	}
	var integer int64
	switch number := value.(type) {
	case int32:
		integer = int64(number)
	case int64:
		integer = number
	case int:
		integer = int64(number)
	case float64:
		if math.Trunc(number) != number || number > math.MaxInt32 || number < math.MinInt32 {
			return 0, fmt.Errorf("value is outside the PostgreSQL integer range")
		}
		integer = int64(number)
	default:
		return 0, fmt.Errorf("expected a number, got %T", value)
	}
	if integer > math.MaxInt32 || integer < math.MinInt32 {
		return 0, fmt.Errorf("value is outside the PostgreSQL integer range")
	}
	return int32(integer), nil
}

func timestamp(doc bson.M, fields ...string) pgtype.Timestamptz {
	for _, field := range fields {
		switch value := doc[field].(type) {
		case time.Time:
			return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
		case primitive.DateTime:
			return pgtype.Timestamptz{Time: value.Time().UTC(), Valid: true}
		}
	}
	return pgtype.Timestamptz{}
}

func timestampOrCreated(doc bson.M) pgtype.Timestamptz {
	if value := timestamp(doc, "updatedAt", "updated_at"); value.Valid {
		return value
	}
	return createdTimestamp(doc)
}

func createdTimestamp(doc bson.M) pgtype.Timestamptz {
	if value := timestamp(doc, "createdAt", "created_at"); value.Valid {
		return value
	}
	return pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
}

func mappedUUID(collection string, id primitive.ObjectID) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte("aurahub-mongo-v1:"+collection+":"+id.Hex()))
}

func pgUUID(collection string, id primitive.ObjectID) pgtype.UUID {
	return pgtype.UUID{Bytes: uuidBytes(mappedUUID(collection, id)), Valid: true}
}

func uuidBytes(id uuid.UUID) [16]byte {
	return [16]byte(id)
}

func documentError(collection string, doc bson.M, err error) error {
	id := stringValue(doc, "_id")
	if raw, ok := doc["_id"].(primitive.ObjectID); ok {
		id = raw.Hex()
	}
	if id == "" {
		id = "unknown"
	}
	return fmt.Errorf("MongoDB %s document %s: %w", collection, id, err)
}
