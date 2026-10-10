package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UpdateVideoMetadataParams struct {
	ID          pgtype.UUID
	Title       pgtype.Text
	Description pgtype.Text
	Visibility  NullVideoVisibility
	Category    pgtype.Text
	Tags        []string
	ReplaceTags bool
	IsAdult     pgtype.Bool
}

func (q *Repository) UpdateVideoMetadata(ctx context.Context, arg UpdateVideoMetadataParams) (Video, error) {
	updates := map[string]interface{}{"updated_at": gorm.Expr("CURRENT_TIMESTAMP")}
	if arg.Title.Valid {
		updates["title"] = arg.Title.String
	}
	if arg.Description.Valid {
		updates["description"] = arg.Description.String
	}
	if arg.Visibility.Valid {
		updates["visibility"] = arg.Visibility.VideoVisibility
	}
	if arg.Category.Valid {
		updates["category"] = arg.Category.String
	}
	if arg.ReplaceTags {
		updates["tags"] = TextArray(arg.Tags)
	}
	if arg.IsAdult.Valid {
		updates["is_adult"] = arg.IsAdult.Bool
	}
	if err := q.orm.WithContext(ctx).Model(&Video{}).Where("id = ?", arg.ID).Updates(updates).Error; err != nil {
		return Video{}, err
	}
	var video Video
	err := q.orm.WithContext(ctx).Where("id = ?", arg.ID).Take(&video).Error
	return video, err
}

type CreatorActivityDay struct {
	Date  time.Time
	Type  string
	Count int64
}

type CreatorTopVideo struct {
	ID           pgtype.UUID
	Title        string
	Category     string
	Views        int32
	LikesCount   int64
	CommentCount int64
}

func (q *Repository) GetCreatorActivity(ctx context.Context, uploaderID pgtype.UUID, start time.Time) ([]CreatorActivityDay, error) {
	var activities []CreatorActivityDay
	err := q.orm.WithContext(ctx).Table("user_activities AS ua").
		Select("(ua.created_at AT TIME ZONE 'UTC')::date AS date, ua.interaction_type::text AS type, COUNT(*) AS count").
		Joins("JOIN videos v ON v.id = ua.video_id").
		Where("v.uploader_id = ? AND ua.created_at >= ?", uploaderID, start).
		Group("date, type").Order("date, type").Scan(&activities).Error
	return activities, err
}

func (q *Repository) GetCreatorTopVideos(ctx context.Context, uploaderID pgtype.UUID) ([]CreatorTopVideo, error) {
	var videos []CreatorTopVideo
	err := q.orm.WithContext(ctx).Model(&Video{}).
		Select(`videos.id, videos.title, COALESCE(videos.category, 'Other') AS category, videos.views,
			(SELECT COUNT(*) FROM user_activities ua WHERE ua.video_id = videos.id AND ua.interaction_type = ?) AS likes_count,
			(SELECT COUNT(*) FROM comments c WHERE c.video_id = videos.id) AS comment_count`, InteractionTypeLike).
		Where("videos.uploader_id = ?", uploaderID).
		Order("videos.views DESC, videos.id").Limit(10).Scan(&videos).Error
	return videos, err
}

func (q *Repository) GetCreatorVideoCategories(ctx context.Context, uploaderID pgtype.UUID) ([]struct {
	Category string
	Videos   int64
	Views    int64
}, error) {
	var categories []struct {
		Category string
		Videos   int64
		Views    int64
	}
	err := q.orm.WithContext(ctx).Model(&Video{}).
		Select("COALESCE(category, 'Other') AS category, COUNT(*) AS videos, COALESCE(SUM(views), 0)::bigint AS views").
		Where("uploader_id = ?", uploaderID).
		Group("COALESCE(category, 'Other')").
		Order("COALESCE(SUM(views), 0) DESC, COALESCE(category, 'Other')").
		Limit(6).Scan(&categories).Error
	return categories, err
}

func (q *Repository) ListPublicVideos(ctx context.Context, category string, shortFilter pgtype.Bool, showAdult bool, sort string, limit, offset int32) ([]Video, error) {
	query := q.orm.WithContext(ctx).Model(&Video{}).
		Where("visibility = ?", VideoVisibilityPublic)
	if !showAdult {
		query = query.Where("is_adult = ?", false)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if shortFilter.Valid {
		query = query.Where("is_short = ?", shortFilter.Bool)
	}
	switch sort {
	case "newest":
		query = query.Order("created_at DESC")
	case "views":
		query = query.Order("views DESC, created_at DESC")
	case "likes":
		query = query.Order("(SELECT COUNT(*) FROM user_activities ua WHERE ua.video_id = videos.id AND ua.interaction_type = 'like') DESC, created_at DESC")
	case "comments":
		query = query.Order("(SELECT COUNT(*) FROM comments c WHERE c.video_id = videos.id) DESC, created_at DESC")
	case "random":
		query = query.Order("random()")
	default:
		query = query.Order("created_at DESC, views DESC")
	}
	var videos []Video
	err := query.Limit(int(limit)).Offset(int(offset)).Find(&videos).Error
	return videos, err
}

func (q *Repository) ListCreatorVideos(ctx context.Context, uploaderID pgtype.UUID, limit, offset int32) ([]Video, error) {
	var videos []Video
	err := q.orm.WithContext(ctx).Model(&Video{}).
		Where("uploader_id = ?", uploaderID).
		Order("created_at DESC").
		Limit(int(limit)).
		Offset(int(offset)).
		Find(&videos).Error
	return videos, err
}

func (q *Repository) CountPublicVideos(ctx context.Context, category string, shortFilter pgtype.Bool, showAdult bool) (int64, error) {
	query := q.orm.WithContext(ctx).Model(&Video{}).Where("visibility = ?", VideoVisibilityPublic)
	if !showAdult {
		query = query.Where("is_adult = ?", false)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if shortFilter.Valid {
		query = query.Where("is_short = ?", shortFilter.Bool)
	}
	var count int64
	err := query.Count(&count).Error
	return count, err
}

func (q *Repository) ListSuggestedVideos(ctx context.Context, excludeID pgtype.UUID, category string, tags []string, showAdult bool, limit, offset int32) ([]Video, error) {
	score := "(CASE WHEN videos.category = ? AND ? <> '' THEN 20 ELSE 0 END + 15 * (SELECT COUNT(*) FROM unnest(videos.tags) AS video_tag WHERE video_tag = ANY(?::text[]))) DESC"
	var videos []Video
	query := q.orm.WithContext(ctx).Model(&Video{}).
		Where("videos.visibility = ? AND videos.id <> ? AND videos.is_short = ?", VideoVisibilityPublic, excludeID, false)
	if !showAdult {
		query = query.Where("videos.is_adult = ?", false)
	}
	err := query.Order(clause.Expr{SQL: score, Vars: []interface{}{category, category, tags}}).
		Order("videos.views DESC, videos.created_at DESC").
		Limit(int(limit)).Offset(int(offset)).Find(&videos).Error
	return videos, err
}

func (q *Repository) CountSuggestedVideos(ctx context.Context, excludeID pgtype.UUID, showAdult bool) (int64, error) {
	var count int64
	query := q.orm.WithContext(ctx).Model(&Video{}).
		Where("visibility = ? AND id <> ? AND is_short = ?", VideoVisibilityPublic, excludeID, false)
	if !showAdult {
		query = query.Where("is_adult = ?", false)
	}
	err := query.Count(&count).Error
	return count, err
}

func (q *Repository) SearchPublicVideos(ctx context.Context, query string, showAdult bool, sort string, limit int32) ([]Video, error) {
	queryBuilder := q.orm.WithContext(ctx).Model(&Video{}).
		Where("videos.visibility = ?", VideoVisibilityPublic).
		Where("videos.is_short = ?", false).
		Where("videos.streamtape_status IS DISTINCT FROM ?", StreamtapeStatusDead)
	
	if !showAdult {
		queryBuilder = queryBuilder.Where("videos.is_adult = ?", false)
	}

	queryBuilder = queryBuilder.Where(`(videos.search_vector @@ plainto_tsquery('english', ?) 
			OR videos.title ILIKE ? OR videos.category ILIKE ?
			OR EXISTS (SELECT 1 FROM unnest(videos.tags) AS tag WHERE tag ILIKE ?))`,
			query, "%"+query+"%", "%"+query+"%", "%"+query+"%")
	switch sort {
	case "date_desc":
		queryBuilder = queryBuilder.Order("videos.created_at DESC")
	case "views_desc":
		queryBuilder = queryBuilder.Order("videos.views DESC, videos.created_at DESC")
	case "likes_desc":
		queryBuilder = queryBuilder.Order("(SELECT COUNT(*) FROM user_activities ua WHERE ua.video_id = videos.id AND ua.interaction_type = 'like') DESC, videos.created_at DESC")
	default:
		queryBuilder = queryBuilder.Order("videos.created_at DESC, videos.views DESC")
	}
	var videos []Video
	err := queryBuilder.Limit(int(limit)).Find(&videos).Error
	return videos, err
}

func (q *Repository) SearchAutocompleteVideos(ctx context.Context, query string, showAdult bool) ([]Video, error) {
	var videos []Video
	queryBuilder := q.orm.WithContext(ctx).Model(&Video{}).
		Select("id, title, thumbnail_url, category").
		Where("visibility = ? AND is_short = ? AND (title ILIKE ? OR EXISTS (SELECT 1 FROM unnest(tags) AS tag WHERE tag ILIKE ?))",
			VideoVisibilityPublic, false, "%"+query+"%", "%"+query+"%")
	
	if !showAdult {
		queryBuilder = queryBuilder.Where("is_adult = ?", false)
	}
	
	err := queryBuilder.Order("created_at DESC").Limit(4).Find(&videos).Error
	return videos, err
}

func (q *Repository) SearchAutocompleteUsers(ctx context.Context, query string) ([]User, error) {
	var users []User
	err := q.orm.WithContext(ctx).Select("id, username, avatar").
		Where("username ILIKE ?", "%"+query+"%").
		Order("username").Limit(2).Find(&users).Error
	return users, err
}

func (q *Repository) ListCommentReplies(ctx context.Context, parentCommentID pgtype.UUID) ([]Comment, error) {
	var comments []Comment
	err := q.orm.WithContext(ctx).Model(&Comment{}).
		Where("parent_comment_id = ?", parentCommentID).
		Order("created_at ASC").
		Find(&comments).Error
	return comments, err
}

func (q *Repository) CountVideoLikes(ctx context.Context, videoID pgtype.UUID) (int64, error) {
	var count int64
	err := q.orm.WithContext(ctx).Model(&UserActivity{}).
		Where("video_id = ? AND interaction_type = ?", videoID, InteractionTypeLike).
		Count(&count).Error
	return count, err
}

func (q *Repository) HasUserLikedVideo(ctx context.Context, userID, videoID pgtype.UUID) (bool, error) {
	var count int64
	err := q.orm.WithContext(ctx).Model(&UserActivity{}).
		Where("user_id = ? AND video_id = ? AND interaction_type = ?", userID, videoID, InteractionTypeLike).
		Count(&count).Error
	return count > 0, err
}

func (q *Repository) UpsertVideoViewActivity(ctx context.Context, userID, videoID pgtype.UUID) error {
	activity := UserActivity{UserID: userID, VideoID: videoID, InteractionType: InteractionTypeView}
	return q.orm.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "video_id"}, {Name: "interaction_type"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"updated_at": gorm.Expr("CURRENT_TIMESTAMP")}),
	}).Create(&activity).Error
}

func (q *Repository) UpdateVideoThumbnail(ctx context.Context, videoID pgtype.UUID, thumbnailURL string) error {
	return q.orm.WithContext(ctx).Model(&Video{}).
		Where("id = ?", videoID).
		Updates(map[string]interface{}{"thumbnail_url": thumbnailURL, "updated_at": gorm.Expr("CURRENT_TIMESTAMP")}).
		Error
}

func (q *Repository) ReplacePlaylistVideoOrder(ctx context.Context, playlistID pgtype.UUID, videoIDs []pgtype.UUID) error {
	return q.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		remove := tx.Where("playlist_id = ?", playlistID)
		if len(videoIDs) > 0 {
			remove = remove.Where("video_id NOT IN ?", videoIDs)
		}
		if err := remove.Delete(&PlaylistVideo{}).Error; err != nil {
			return err
		}
		baseTime := time.Now().UTC()
		for index, videoID := range videoIDs {
			row := PlaylistVideo{PlaylistID: playlistID, VideoID: videoID, AddedAt: pgtype.Timestamptz{Time: baseTime.Add(time.Duration(index) * time.Microsecond), Valid: true}}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "playlist_id"}, {Name: "video_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"added_at"}),
			}).Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (q *Repository) IsVideoInPlaylist(ctx context.Context, playlistID, videoID pgtype.UUID) (bool, error) {
	var count int64
	err := q.orm.WithContext(ctx).Model(&PlaylistVideo{}).
		Where("playlist_id = ? AND video_id = ?", playlistID, videoID).
		Count(&count).Error
	return count > 0, err
}

func (q *Repository) ListVisiblePlaylistVideos(ctx context.Context, playlistID pgtype.UUID) ([]Video, error) {
	var videos []Video
	err := q.orm.WithContext(ctx).Model(&Video{}).
		Joins("JOIN playlist_videos ON playlist_videos.video_id = videos.id").
		Where("playlist_videos.playlist_id = ? AND videos.visibility IN ?", playlistID,
			[]VideoVisibility{VideoVisibilityPublic, VideoVisibilityUnlisted}).
		Order("playlist_videos.added_at ASC").Find(&videos).Error
	return videos, err
}

func (q *Repository) IsEmailUsedByAnotherUser(ctx context.Context, email string, userID pgtype.UUID) (bool, error) {
	var count int64
	err := q.orm.WithContext(ctx).Model(&User{}).
		Where("email = ? AND id <> ?", email, userID).
		Count(&count).Error
	return count > 0, err
}

func (q *Repository) IsUsernameUsedByAnotherUser(ctx context.Context, username string, userID pgtype.UUID) (bool, error) {
	var count int64
	err := q.orm.WithContext(ctx).Model(&User{}).
		Where("username = ? AND id <> ?", username, userID).
		Count(&count).Error
	return count > 0, err
}

type UpdateUserProfileParams struct {
	ID       pgtype.UUID
	Username pgtype.Text
	Email    pgtype.Text
	Password pgtype.Text
	Avatar           pgtype.Text
	Bio              pgtype.Text
	Banner           pgtype.Text
	ShowAdultContent pgtype.Bool
}

func (q *Repository) UpdateUserProfile(ctx context.Context, arg UpdateUserProfileParams) (User, error) {
	updates := map[string]interface{}{"updated_at": gorm.Expr("CURRENT_TIMESTAMP")}
	if arg.Username.Valid {
		updates["username"] = arg.Username.String
	}
	if arg.Email.Valid {
		updates["email"] = arg.Email.String
	}
	if arg.Password.Valid {
		updates["password"] = arg.Password.String
	}
	if arg.Avatar.Valid {
		updates["avatar"] = arg.Avatar.String
	}
	if arg.Bio.Valid {
		updates["bio"] = arg.Bio.String
	}
	if arg.Banner.Valid {
		updates["banner"] = arg.Banner.String
	}
	if arg.ShowAdultContent.Valid {
		updates["show_adult_content"] = arg.ShowAdultContent.Bool
	}
	if err := q.orm.WithContext(ctx).Model(&User{}).Where("id = ?", arg.ID).Updates(updates).Error; err != nil {
		return User{}, err
	}
	var user User
	err := q.orm.WithContext(ctx).Where("id = ?", arg.ID).Take(&user).Error
	return user, err
}

type VideoDetailsResult struct {
	Video
	UploaderUsername string      `gorm:"column:uploader_username"`
	UploaderAvatar   pgtype.Text `gorm:"column:uploader_avatar"`
	LikesCount       int64       `gorm:"column:likes_count"`
	CommentCount     int64       `gorm:"column:comment_count"`
	SubscribersCount int64       `gorm:"column:subscribers_count"`
	IsLiked          bool        `gorm:"column:is_liked"`
	IsSubscribed     bool        `gorm:"column:is_subscribed"`
}

// GetVideoDetails retrieves a video with uploader info, counters, and viewer interactions in a single query.
func (q *Repository) GetVideoDetails(ctx context.Context, identifier string, viewerID pgtype.UUID) (*VideoDetailsResult, error) {
	var result VideoDetailsResult
	var parsedUUID pgtype.UUID
	_ = parsedUUID.Scan(identifier)

	query := q.orm.WithContext(ctx).Table("videos").
		Joins("JOIN users ON users.id = videos.uploader_id")

	if viewerID.Valid {
		query = query.Select(`videos.*,
			users.username AS uploader_username,
			users.avatar AS uploader_avatar,
			COALESCE((SELECT COUNT(*) FROM user_activities ua WHERE ua.video_id = videos.id AND ua.interaction_type = 'like'), 0) AS likes_count,
			COALESCE((SELECT COUNT(*) FROM comments c WHERE c.video_id = videos.id), 0) AS comment_count,
			COALESCE((SELECT COUNT(*) FROM subscriptions s WHERE s.subscribed_to_id = videos.uploader_id), 0) AS subscribers_count,
			EXISTS(SELECT 1 FROM user_activities ua WHERE ua.video_id = videos.id AND ua.user_id = ? AND ua.interaction_type = 'like') AS is_liked,
			EXISTS(SELECT 1 FROM subscriptions s WHERE s.subscribed_to_id = videos.uploader_id AND s.subscriber_id = ?) AS is_subscribed`,
			viewerID, viewerID)
	} else {
		query = query.Select(`videos.*,
			users.username AS uploader_username,
			users.avatar AS uploader_avatar,
			COALESCE((SELECT COUNT(*) FROM user_activities ua WHERE ua.video_id = videos.id AND ua.interaction_type = 'like'), 0) AS likes_count,
			COALESCE((SELECT COUNT(*) FROM comments c WHERE c.video_id = videos.id), 0) AS comment_count,
			COALESCE((SELECT COUNT(*) FROM subscriptions s WHERE s.subscribed_to_id = videos.uploader_id), 0) AS subscribers_count,
			false AS is_liked,
			false AS is_subscribed`)
	}

	if parsedUUID.Valid {
		query = query.Where("videos.id = ? OR videos.file_id = ?", parsedUUID, identifier)
	} else {
		query = query.Where("videos.file_id = ?", identifier)
	}

	err := query.First(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// IncrementVideoViewsBy atomically increments view counts by a batch delta.
func (q *Repository) IncrementVideoViewsBy(ctx context.Context, id pgtype.UUID, delta int) error {
	if delta <= 0 {
		return nil
	}
	res := q.orm.WithContext(ctx).Model(&Video{}).Where("id = ?", id).
		UpdateColumn("views", gorm.Expr("views + ?", delta))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 && id.Valid {
		b := id.Bytes
		uuidStr := fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
		return q.orm.WithContext(ctx).Model(&Video{}).Where("id = ?", uuidStr).
			UpdateColumn("views", gorm.Expr("views + ?", delta)).Error
	}
	return nil
}

// GetExpiredVideos returns videos whose last_refreshed_at is older than the given cutoff (or NULL),
// with active streamtape_status and fewer than 5 failed clone attempts.
func (q *Repository) GetExpiredVideos(ctx context.Context, cutoff time.Time, limit int) ([]Video, error) {
	if limit <= 0 {
		limit = 10
	}
	var videos []Video
	err := q.orm.WithContext(ctx).
		Where("(last_refreshed_at < ? OR last_refreshed_at IS NULL)", cutoff).
		Where("(streamtape_status = ? OR streamtape_status IS NULL)", StreamtapeStatusActive).
		Where("clone_attempts < ?", 5).
		Order("last_refreshed_at ASC NULLS FIRST").
		Limit(limit).
		Find(&videos).Error
	return videos, err
}

// RefreshVideoFile atomically updates the video's file_id, streamtape_url, resets last_refreshed_at to CURRENT_TIMESTAMP,
// and resets clone_attempts to 0.
func (q *Repository) RefreshVideoFile(ctx context.Context, videoID pgtype.UUID, newFileID, newStreamtapeURL string) error {
	return q.orm.WithContext(ctx).Model(&Video{}).Where("id = ?", videoID).Updates(map[string]interface{}{
		"file_id":           newFileID,
		"streamtape_url":    newStreamtapeURL,
		"last_refreshed_at": gorm.Expr("CURRENT_TIMESTAMP"),
		"streamtape_status": StreamtapeStatusActive,
		"clone_attempts":    0,
		"updated_at":        gorm.Expr("CURRENT_TIMESTAMP"),
	}).Error
}

// IncrementCloneAttempts increments the failed refresh attempt counter.
func (q *Repository) IncrementCloneAttempts(ctx context.Context, videoID pgtype.UUID) error {
	return q.orm.WithContext(ctx).Model(&Video{}).Where("id = ?", videoID).
		Update("clone_attempts", gorm.Expr("clone_attempts + 1")).Error
}

// MarkVideoStreamtapeDead marks a video as dead if the file is confirmed deleted upstream.
func (q *Repository) MarkVideoStreamtapeDead(ctx context.Context, videoID pgtype.UUID) error {
	return q.orm.WithContext(ctx).Model(&Video{}).Where("id = ?", videoID).
		Update("streamtape_status", StreamtapeStatusDead).Error
}
