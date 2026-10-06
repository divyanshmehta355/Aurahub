package db

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AddVideoToPlaylistParams struct {
	PlaylistID pgtype.UUID
	VideoID    pgtype.UUID
}

func (q *Repository) AddVideoToPlaylist(ctx context.Context, arg AddVideoToPlaylistParams) error {
	row := PlaylistVideo{PlaylistID: arg.PlaylistID, VideoID: arg.VideoID}
	return q.orm.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
}

type AddWatchLaterParams struct {
	UserID  pgtype.UUID
	VideoID pgtype.UUID
}

func (q *Repository) AddWatchLater(ctx context.Context, arg AddWatchLaterParams) error {
	row := WatchLater{UserID: arg.UserID, VideoID: arg.VideoID}
	return q.orm.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
}

type BulkDeleteVideosParams struct {
	UploaderID pgtype.UUID
	Column2    []pgtype.UUID
}

func (q *Repository) BulkDeleteVideos(ctx context.Context, arg BulkDeleteVideosParams) error {
	return q.orm.WithContext(ctx).Where("uploader_id = ? AND id IN ?", arg.UploaderID, arg.Column2).
		Delete(&Video{}).Error
}

type BulkUpdateVideoVisibilityParams struct {
	UploaderID pgtype.UUID
	Column2    []pgtype.UUID
	Visibility NullVideoVisibility
}

func (q *Repository) BulkUpdateVideoVisibility(ctx context.Context, arg BulkUpdateVideoVisibilityParams) error {
	if !arg.Visibility.Valid {
		return gorm.ErrInvalidData
	}
	return q.orm.WithContext(ctx).Model(&Video{}).
		Where("uploader_id = ? AND id IN ?", arg.UploaderID, arg.Column2).
		Update("visibility", arg.Visibility.VideoVisibility).Error
}

type BulkUpdateVideoAdultParams struct {
	UploaderID pgtype.UUID
	Column2    []pgtype.UUID
	IsAdult    bool
}

func (q *Repository) BulkUpdateVideoAdult(ctx context.Context, arg BulkUpdateVideoAdultParams) error {
	return q.orm.WithContext(ctx).Model(&Video{}).
		Where("uploader_id = ? AND id IN ?", arg.UploaderID, arg.Column2).
		Update("is_adult", arg.IsAdult).Error
}

func (q *Repository) CheckEmailExists(ctx context.Context, email string) (bool, error) {
	var count int64
	err := q.orm.WithContext(ctx).Model(&User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

func (q *Repository) CheckUsernameExists(ctx context.Context, username string) (bool, error) {
	var count int64
	err := q.orm.WithContext(ctx).Model(&User{}).Where("username = ?", username).Count(&count).Error
	return count > 0, err
}

func (q *Repository) ClearNotifications(ctx context.Context, recipientID pgtype.UUID) error {
	return q.orm.WithContext(ctx).Where("recipient_id = ?", recipientID).Delete(&Notification{}).Error
}

func (q *Repository) ClearWatchHistory(ctx context.Context, userID pgtype.UUID) error {
	return q.orm.WithContext(ctx).Where("user_id = ?", userID).Delete(&WatchHistory{}).Error
}

func (q *Repository) CountUnreadNotifications(ctx context.Context, recipientID pgtype.UUID) (int64, error) {
	var count int64
	err := q.orm.WithContext(ctx).Model(&Notification{}).
		Where("recipient_id = ? AND is_read = ?", recipientID, false).Count(&count).Error
	return count, err
}

func (q *Repository) CountVideoComments(ctx context.Context, videoID pgtype.UUID) (int64, error) {
	var count int64
	err := q.orm.WithContext(ctx).Model(&Comment{}).Where("video_id = ?", videoID).Count(&count).Error
	return count, err
}

type CreateCommentParams struct {
	Text            string
	AuthorID        pgtype.UUID
	VideoID         pgtype.UUID
	ParentCommentID pgtype.UUID
}

func (q *Repository) CreateComment(ctx context.Context, arg CreateCommentParams) (Comment, error) {
	comment := Comment{
		Text: arg.Text, AuthorID: arg.AuthorID, VideoID: arg.VideoID,
		ParentCommentID: arg.ParentCommentID,
	}
	err := q.orm.WithContext(ctx).Create(&comment).Error
	return comment, err
}

type CreateNotificationParams struct {
	RecipientID pgtype.UUID
	SenderID    pgtype.UUID
	Type        NotificationType
	VideoID     pgtype.UUID
	CommentID   pgtype.UUID
}

func (q *Repository) CreateNotificationWithResult(ctx context.Context, arg CreateNotificationParams) (*Notification, error) {
	if arg.RecipientID == arg.SenderID {
		return nil, nil
	}
	notification := Notification{
		RecipientID: arg.RecipientID, SenderID: arg.SenderID, Type: arg.Type,
		VideoID: arg.VideoID, CommentID: arg.CommentID,
	}
	err := q.orm.WithContext(ctx).Create(&notification).Error
	if err != nil {
		return nil, err
	}
	return &notification, nil
}

func (q *Repository) CreateNotification(ctx context.Context, arg CreateNotificationParams) error {
	_, err := q.CreateNotificationWithResult(ctx, arg)
	return err
}

func (q *Repository) NotifySubscribersNewVideoWithResult(ctx context.Context, uploaderID pgtype.UUID, videoID pgtype.UUID) ([]Notification, error) {
	var subscriberIDs []pgtype.UUID
	err := q.orm.WithContext(ctx).Model(&Subscription{}).
		Where("subscribed_to_id = ?", uploaderID).
		Pluck("subscriber_id", &subscriberIDs).Error
	if err != nil || len(subscriberIDs) == 0 {
		return nil, err
	}

	notifications := make([]Notification, 0, len(subscriberIDs))
	for _, subID := range subscriberIDs {
		if subID == uploaderID {
			continue
		}
		notifications = append(notifications, Notification{
			RecipientID: subID,
			SenderID:    uploaderID,
			Type:        NotificationTypeNewVideo,
			VideoID:     videoID,
		})
	}
	if len(notifications) == 0 {
		return nil, nil
	}
	err = q.orm.WithContext(ctx).Create(&notifications).Error
	if err != nil {
		return nil, err
	}
	return notifications, nil
}

func (q *Repository) NotifySubscribersNewVideo(ctx context.Context, uploaderID pgtype.UUID, videoID pgtype.UUID) error {
	_, err := q.NotifySubscribersNewVideoWithResult(ctx, uploaderID, videoID)
	return err
}

type CreatePlaylistParams struct {
	Title       string
	Description pgtype.Text
	OwnerID     pgtype.UUID
	IsPublic    pgtype.Bool
}

func (q *Repository) CreatePlaylist(ctx context.Context, arg CreatePlaylistParams) (Playlist, error) {
	playlist := Playlist{
		Title: arg.Title, Description: arg.Description, OwnerID: arg.OwnerID, IsPublic: arg.IsPublic,
	}
	err := q.orm.WithContext(ctx).Create(&playlist).Error
	return playlist, err
}

type CreateUserParams struct {
	Email    string
	Password string
	Username string
	Avatar   pgtype.Text
	Banner   pgtype.Text
	Bio      pgtype.Text
}

func (q *Repository) CreateUser(ctx context.Context, arg CreateUserParams) (User, error) {
	user := User{
		Email: arg.Email, Password: arg.Password, Username: arg.Username,
		Avatar: arg.Avatar, Banner: arg.Banner, Bio: arg.Bio,
	}
	err := q.orm.WithContext(ctx).Create(&user).Error
	return user, err
}

type CreateVideoParams struct {
	Title            string
	Description      pgtype.Text
	FileID           string
	ThumbnailUrl     pgtype.Text
	Category         pgtype.Text
	Tags             []string
	Visibility       NullVideoVisibility
	UploaderID       pgtype.UUID
	IsShort          pgtype.Bool
	IsAdult          pgtype.Bool
	StreamtapeUrl    pgtype.Text
	StreamtapeStatus NullStreamtapeStatus
}

func (q *Repository) CreateVideo(ctx context.Context, arg CreateVideoParams) (Video, error) {
	video := Video{
		Title: arg.Title, Description: arg.Description, FileID: arg.FileID,
		ThumbnailUrl: arg.ThumbnailUrl, Category: arg.Category, Tags: TextArray(arg.Tags),
		Visibility: arg.Visibility, UploaderID: arg.UploaderID, IsShort: arg.IsShort,
		IsAdult: arg.IsAdult, StreamtapeUrl: arg.StreamtapeUrl, StreamtapeStatus: arg.StreamtapeStatus,
	}
	err := q.orm.WithContext(ctx).Create(&video).Error
	return video, err
}

type DeleteCommentParams struct {
	ID       pgtype.UUID
	AuthorID pgtype.UUID
}

func (q *Repository) DeleteComment(ctx context.Context, arg DeleteCommentParams) error {
	_ = q.orm.WithContext(ctx).Where("comment_id = ?", arg.ID).Delete(&Notification{}).Error
	_ = q.orm.WithContext(ctx).Where("parent_comment_id = ?", arg.ID).Delete(&Comment{}).Error
	return q.orm.WithContext(ctx).Where("id = ? AND author_id = ?", arg.ID, arg.AuthorID).
		Delete(&Comment{}).Error
}

func (q *Repository) DeleteCommentByID(ctx context.Context, id pgtype.UUID) error {
	_ = q.orm.WithContext(ctx).Where("comment_id = ?", id).Delete(&Notification{}).Error
	_ = q.orm.WithContext(ctx).Where("parent_comment_id = ?", id).Delete(&Comment{}).Error
	return q.orm.WithContext(ctx).Where("id = ?", id).Delete(&Comment{}).Error
}

type DeletePlaylistParams struct {
	ID      pgtype.UUID
	OwnerID pgtype.UUID
}

func (q *Repository) DeletePlaylist(ctx context.Context, arg DeletePlaylistParams) error {
	return q.orm.WithContext(ctx).Where("id = ? AND owner_id = ?", arg.ID, arg.OwnerID).
		Delete(&Playlist{}).Error
}

type DeleteUserActivityParams struct {
	UserID          pgtype.UUID
	VideoID         pgtype.UUID
	InteractionType InteractionType
}

func (q *Repository) DeleteUserActivity(ctx context.Context, arg DeleteUserActivityParams) error {
	return q.orm.WithContext(ctx).Where(
		"user_id = ? AND video_id = ? AND interaction_type = ?",
		arg.UserID, arg.VideoID, arg.InteractionType,
	).Delete(&UserActivity{}).Error
}

func (q *Repository) DeleteVideo(ctx context.Context, id pgtype.UUID) error {
	return q.orm.WithContext(ctx).Delete(&Video{}, "id = ?", id).Error
}

type DeleteWatchHistoryParams struct {
	UserID  pgtype.UUID
	VideoID pgtype.UUID
}

func (q *Repository) DeleteWatchHistory(ctx context.Context, arg DeleteWatchHistoryParams) error {
	return q.orm.WithContext(ctx).Where("user_id = ? AND video_id = ?", arg.UserID, arg.VideoID).
		Delete(&WatchHistory{}).Error
}

func (q *Repository) GetComment(ctx context.Context, id pgtype.UUID) (Comment, error) {
	var comment Comment
	err := q.orm.WithContext(ctx).First(&comment, "id = ?", id).Error
	return comment, err
}

type GetCreatorLifetimeStatsRow struct {
	TotalVideos   int64
	TotalViews    int64
	TotalLikes    int64
	TotalComments int64
}

func (q *Repository) GetCreatorLifetimeStats(ctx context.Context, uploaderID pgtype.UUID) (GetCreatorLifetimeStatsRow, error) {
	var stats GetCreatorLifetimeStatsRow
	videoQuery := q.orm.WithContext(ctx).Model(&Video{}).Where("uploader_id = ?", uploaderID)
	if err := videoQuery.Count(&stats.TotalVideos).Error; err != nil {
		return stats, err
	}
	if err := q.orm.WithContext(ctx).Model(&Video{}).Where("uploader_id = ?", uploaderID).
		Select("COALESCE(SUM(views), 0)").Scan(&stats.TotalViews).Error; err != nil {
		return stats, err
	}
	if err := q.orm.WithContext(ctx).Model(&UserActivity{}).
		Joins("JOIN videos ON videos.id = user_activities.video_id").
		Where("videos.uploader_id = ? AND user_activities.interaction_type = ?", uploaderID, InteractionTypeLike).
		Count(&stats.TotalLikes).Error; err != nil {
		return stats, err
	}
	err := q.orm.WithContext(ctx).Model(&Comment{}).
		Joins("JOIN videos ON videos.id = comments.video_id").
		Where("videos.uploader_id = ?", uploaderID).Count(&stats.TotalComments).Error
	return stats, err
}

func (q *Repository) GetPlaylist(ctx context.Context, id pgtype.UUID) (Playlist, error) {
	var playlist Playlist
	err := q.orm.WithContext(ctx).First(&playlist, "id = ?", id).Error
	return playlist, err
}

func (q *Repository) GetSubscriberCount(ctx context.Context, subscribedToID pgtype.UUID) (int64, error) {
	var count int64
	err := q.orm.WithContext(ctx).Model(&Subscription{}).
		Where("subscribed_to_id = ?", subscribedToID).Count(&count).Error
	return count, err
}

type GetSubscriptionFeedParams struct {
	SubscriberID pgtype.UUID
	Limit        int32
	Offset       int32
}

func (q *Repository) GetSubscriptionFeed(ctx context.Context, arg GetSubscriptionFeedParams) ([]Video, error) {
	var videos []Video
	err := q.orm.WithContext(ctx).Model(&Video{}).
		Joins("JOIN subscriptions ON subscriptions.subscribed_to_id = videos.uploader_id").
		Where("subscriptions.subscriber_id = ? AND videos.visibility = ?", arg.SubscriberID, VideoVisibilityPublic).
		Order("videos.created_at DESC").Limit(int(arg.Limit)).Offset(int(arg.Offset)).Find(&videos).Error
	return videos, err
}

func (q *Repository) CountSubscriptionFeed(ctx context.Context, subscriberID pgtype.UUID) (int64, error) {
	var count int64
	err := q.orm.WithContext(ctx).Model(&Video{}).
		Joins("JOIN subscriptions ON subscriptions.subscribed_to_id = videos.uploader_id").
		Where("subscriptions.subscriber_id = ? AND videos.visibility = ?", subscriberID, VideoVisibilityPublic).
		Count(&count).Error
	return count, err
}

func (q *Repository) GetUser(ctx context.Context, id pgtype.UUID) (User, error) {
	var user User
	err := q.orm.WithContext(ctx).First(&user, "id = ?", id).Error
	return user, err
}

func (q *Repository) GetUserByEmail(ctx context.Context, email string) (User, error) {
	var user User
	err := q.orm.WithContext(ctx).First(&user, "email = ?", email).Error
	return user, err
}

func (q *Repository) GetUserByUsername(ctx context.Context, username string) (User, error) {
	var user User
	err := q.orm.WithContext(ctx).First(&user, "username = ?", username).Error
	return user, err
}

func (q *Repository) GetVideo(ctx context.Context, id pgtype.UUID) (Video, error) {
	var video Video
	err := q.orm.WithContext(ctx).First(&video, "id = ?", id).Error
	return video, err
}

func (q *Repository) GetVideoByFileId(ctx context.Context, fileID string) (Video, error) {
	var video Video
	err := q.orm.WithContext(ctx).First(&video, "file_id = ?", fileID).Error
	return video, err
}

type GetWatchHistoryParams struct {
	UserID pgtype.UUID
	Limit  int32
	Offset int32
}

func (q *Repository) GetWatchHistory(ctx context.Context, arg GetWatchHistoryParams) ([]Video, error) {
	var videos []Video
	err := q.orm.WithContext(ctx).Model(&Video{}).
		Joins("JOIN watch_history ON watch_history.video_id = videos.id").
		Where("watch_history.user_id = ?", arg.UserID).
		Order("watch_history.updated_at DESC").Limit(int(arg.Limit)).Offset(int(arg.Offset)).Find(&videos).Error
	return videos, err
}

type GetWatchLaterParams struct {
	UserID pgtype.UUID
	Limit  int32
	Offset int32
}

func (q *Repository) GetWatchLater(ctx context.Context, arg GetWatchLaterParams) ([]Video, error) {
	var videos []Video
	err := q.orm.WithContext(ctx).Model(&Video{}).
		Joins("JOIN watch_later ON watch_later.video_id = videos.id").
		Where("watch_later.user_id = ?", arg.UserID).
		Order("watch_later.created_at DESC").Limit(int(arg.Limit)).Offset(int(arg.Offset)).Find(&videos).Error
	return videos, err
}

func (q *Repository) IncrementVideoViews(ctx context.Context, id pgtype.UUID) error {
	return q.orm.WithContext(ctx).Model(&Video{}).Where("id = ?", id).
		UpdateColumn("views", gorm.Expr("views + ?", 1)).Error
}

type IsSubscribedParams struct {
	SubscriberID   pgtype.UUID
	SubscribedToID pgtype.UUID
}

func (q *Repository) IsSubscribed(ctx context.Context, arg IsSubscribedParams) (bool, error) {
	var count int64
	err := q.orm.WithContext(ctx).Model(&Subscription{}).
		Where("subscriber_id = ? AND subscribed_to_id = ?", arg.SubscriberID, arg.SubscribedToID).
		Count(&count).Error
	return count > 0, err
}

type ListCommentsForVideoParams struct {
	VideoID pgtype.UUID
	Limit   int32
	Offset  int32
}

func (q *Repository) ListCommentsForVideo(ctx context.Context, arg ListCommentsForVideoParams) ([]Comment, error) {
	var comments []Comment
	err := q.orm.WithContext(ctx).Where("video_id = ? AND parent_comment_id IS NULL", arg.VideoID).
		Order("created_at DESC").Limit(int(arg.Limit)).Offset(int(arg.Offset)).Find(&comments).Error
	return comments, err
}

type ListNotificationsParams struct {
	RecipientID pgtype.UUID
	Limit       int32
	Offset      int32
}

func (q *Repository) ListNotifications(ctx context.Context, arg ListNotificationsParams) ([]Notification, error) {
	var notifications []Notification
	err := q.orm.WithContext(ctx).Where("recipient_id = ?", arg.RecipientID).
		Order("created_at DESC").Limit(int(arg.Limit)).Offset(int(arg.Offset)).Find(&notifications).Error
	return notifications, err
}

func (q *Repository) ListPlaylistVideos(ctx context.Context, playlistID pgtype.UUID) ([]Video, error) {
	var videos []Video
	err := q.orm.WithContext(ctx).Model(&Video{}).
		Joins("JOIN playlist_videos ON playlist_videos.video_id = videos.id").
		Where("playlist_videos.playlist_id = ?", playlistID).
		Order("playlist_videos.added_at ASC").Find(&videos).Error
	return videos, err
}

func (q *Repository) ListUserPlaylists(ctx context.Context, ownerID pgtype.UUID) ([]Playlist, error) {
	var playlists []Playlist
	err := q.orm.WithContext(ctx).Where("owner_id = ?", ownerID).
		Order("updated_at DESC").Find(&playlists).Error
	return playlists, err
}

type ListUserVideosParams struct {
	UploaderID pgtype.UUID
	Limit      int32
	Offset     int32
	ShowAdult  bool
}

func (q *Repository) ListUserVideos(ctx context.Context, arg ListUserVideosParams) ([]Video, error) {
	var videos []Video
	query := q.orm.WithContext(ctx).Where("uploader_id = ? AND visibility = ?", arg.UploaderID, VideoVisibilityPublic)
	if !arg.ShowAdult {
		query = query.Where("is_adult = ?", false)
	}
	err := query.Order("created_at DESC").Limit(int(arg.Limit)).Offset(int(arg.Offset)).Find(&videos).Error
	return videos, err
}

type ListVideosParams struct {
	Limit  int32
	Offset int32
}

func (q *Repository) ListVideos(ctx context.Context, arg ListVideosParams) ([]Video, error) {
	var videos []Video
	err := q.orm.WithContext(ctx).Order("created_at DESC").
		Limit(int(arg.Limit)).Offset(int(arg.Offset)).Find(&videos).Error
	return videos, err
}

func (q *Repository) MarkNotificationsRead(ctx context.Context, recipientID pgtype.UUID) error {
	return q.orm.WithContext(ctx).Model(&Notification{}).
		Where("recipient_id = ? AND is_read = ?", recipientID, false).
		Update("is_read", true).Error
}

func (q *Repository) MarkNotificationRead(ctx context.Context, recipientID, notificationID pgtype.UUID) error {
	return q.orm.WithContext(ctx).Model(&Notification{}).
		Where("recipient_id = ? AND id = ?", recipientID, notificationID).
		Update("is_read", true).Error
}

func (q *Repository) DeleteNotification(ctx context.Context, recipientID, notificationID pgtype.UUID) error {
	return q.orm.WithContext(ctx).
		Where("recipient_id = ? AND id = ?", recipientID, notificationID).
		Delete(&Notification{}).Error
}

type RemoveVideoFromPlaylistParams struct {
	PlaylistID pgtype.UUID
	VideoID    pgtype.UUID
}

func (q *Repository) RemoveVideoFromPlaylist(ctx context.Context, arg RemoveVideoFromPlaylistParams) error {
	return q.orm.WithContext(ctx).Where("playlist_id = ? AND video_id = ?", arg.PlaylistID, arg.VideoID).
		Delete(&PlaylistVideo{}).Error
}

type RemoveWatchLaterParams struct {
	UserID  pgtype.UUID
	VideoID pgtype.UUID
}

func (q *Repository) RemoveWatchLater(ctx context.Context, arg RemoveWatchLaterParams) error {
	return q.orm.WithContext(ctx).Where("user_id = ? AND video_id = ?", arg.UserID, arg.VideoID).
		Delete(&WatchLater{}).Error
}

type SearchVideosParams struct {
	Column1 pgtype.Text
	Limit   int32
	Offset  int32
}

func (q *Repository) SearchVideos(ctx context.Context, arg SearchVideosParams) ([]Video, error) {
	query := q.orm.WithContext(ctx).Where("visibility = ?", VideoVisibilityPublic)
	if arg.Column1.Valid && strings.TrimSpace(arg.Column1.String) != "" {
		term := strings.TrimSpace(arg.Column1.String)
		query = query.Where("search_vector @@ websearch_to_tsquery('english', ?) OR title % ? OR description % ?", term, term, term)
		query = query.Order(clause.Expr{
			SQL:  "ts_rank(search_vector, websearch_to_tsquery('english', ?)) + GREATEST(SIMILARITY(title, ?), SIMILARITY(description, ?)) DESC",
			Vars: []interface{}{term, term, term},
		}).Order("created_at DESC")
	} else {
		query = query.Order("created_at DESC")
	}
	var videos []Video
	err := query.Limit(int(arg.Limit)).Offset(int(arg.Offset)).Find(&videos).Error
	return videos, err
}

type SubscribeParams struct {
	SubscriberID   pgtype.UUID
	SubscribedToID pgtype.UUID
}

func (q *Repository) Subscribe(ctx context.Context, arg SubscribeParams) error {
	row := Subscription{SubscriberID: arg.SubscriberID, SubscribedToID: arg.SubscribedToID}
	return q.orm.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
}

type UnsubscribeParams struct {
	SubscriberID   pgtype.UUID
	SubscribedToID pgtype.UUID
}

func (q *Repository) Unsubscribe(ctx context.Context, arg UnsubscribeParams) error {
	return q.orm.WithContext(ctx).
		Where("subscriber_id = ? AND subscribed_to_id = ?", arg.SubscriberID, arg.SubscribedToID).
		Delete(&Subscription{}).Error
}

type UpdateCommentParams struct {
	ID   pgtype.UUID
	Text string
}

func (q *Repository) UpdateComment(ctx context.Context, arg UpdateCommentParams) (Comment, error) {
	if err := q.orm.WithContext(ctx).Model(&Comment{}).Where("id = ?", arg.ID).
		Updates(map[string]interface{}{"text": arg.Text, "updated_at": gorm.Expr("CURRENT_TIMESTAMP")}).Error; err != nil {
		return Comment{}, err
	}
	var comment Comment
	err := q.orm.WithContext(ctx).First(&comment, "id = ?", arg.ID).Error
	return comment, err
}

type UpdatePlaylistParams struct {
	ID       pgtype.UUID
	Column2  interface{}
	Column3  interface{}
	IsPublic pgtype.Bool
}

func (q *Repository) UpdatePlaylist(ctx context.Context, arg UpdatePlaylistParams) (Playlist, error) {
	updates := map[string]interface{}{"updated_at": gorm.Expr("CURRENT_TIMESTAMP")}
	if title, ok := arg.Column2.(string); ok && title != "" {
		updates["title"] = title
	}
	switch description := arg.Column3.(type) {
	case string:
		if description != "" {
			updates["description"] = description
		}
	case pgtype.Text:
		if description.Valid {
			updates["description"] = description.String
		}
	}
	if arg.IsPublic.Valid {
		updates["is_public"] = arg.IsPublic.Bool
	}
	if err := q.orm.WithContext(ctx).Model(&Playlist{}).Where("id = ?", arg.ID).Updates(updates).Error; err != nil {
		return Playlist{}, err
	}
	var playlist Playlist
	err := q.orm.WithContext(ctx).First(&playlist, "id = ?", arg.ID).Error
	return playlist, err
}

type UpdateUserParams struct {
	ID      pgtype.UUID
	Column2 interface{}
	Column3 interface{}
	Column4 interface{}
}

func (q *Repository) UpdateUser(ctx context.Context, arg UpdateUserParams) (User, error) {
	updates := make(map[string]interface{})
	if value, ok := arg.Column2.(string); ok {
		updates["avatar"] = value
	}
	if value, ok := arg.Column3.(string); ok {
		updates["banner"] = value
	}
	if value, ok := arg.Column4.(string); ok {
		updates["bio"] = value
	}
	if len(updates) > 0 {
		if err := q.orm.WithContext(ctx).Model(&User{}).Where("id = ?", arg.ID).Updates(updates).Error; err != nil {
			return User{}, err
		}
	}
	var user User
	err := q.orm.WithContext(ctx).First(&user, "id = ?", arg.ID).Error
	return user, err
}

type UpdateVideoParams struct {
	ID         pgtype.UUID
	Column2    interface{}
	Column3    interface{}
	Visibility NullVideoVisibility
	Column5    interface{}
}

func (q *Repository) UpdateVideo(ctx context.Context, arg UpdateVideoParams) (Video, error) {
	updates := make(map[string]interface{})
	if value, ok := arg.Column2.(string); ok && value != "" {
		updates["title"] = value
	}
	if value, ok := arg.Column3.(string); ok && value != "" {
		updates["description"] = value
	}
	if arg.Visibility.Valid {
		updates["visibility"] = arg.Visibility.VideoVisibility
	}
	if value, ok := arg.Column5.(string); ok && value != "" {
		updates["category"] = value
	}
	if len(updates) > 0 {
		if err := q.orm.WithContext(ctx).Model(&Video{}).Where("id = ?", arg.ID).Updates(updates).Error; err != nil {
			return Video{}, err
		}
	}
	var video Video
	err := q.orm.WithContext(ctx).First(&video, "id = ?", arg.ID).Error
	return video, err
}

type UpsertUserActivityParams struct {
	UserID          pgtype.UUID
	VideoID         pgtype.UUID
	InteractionType InteractionType
}

func (q *Repository) UpsertUserActivity(ctx context.Context, arg UpsertUserActivityParams) error {
	row := UserActivity{UserID: arg.UserID, VideoID: arg.VideoID, InteractionType: arg.InteractionType}
	return q.orm.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "video_id"}, {Name: "interaction_type"}},
		DoNothing: true,
	}).Create(&row).Error
}

type UpsertWatchHistoryParams struct {
	UserID  pgtype.UUID
	VideoID pgtype.UUID
}

func (q *Repository) UpsertWatchHistory(ctx context.Context, arg UpsertWatchHistoryParams) error {
	row := WatchHistory{UserID: arg.UserID, VideoID: arg.VideoID}
	return q.orm.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "video_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"updated_at": gorm.Expr("CURRENT_TIMESTAMP")}),
	}).Create(&row).Error
}
