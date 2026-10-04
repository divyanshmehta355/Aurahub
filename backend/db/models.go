package db

import (
	"database/sql/driver"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
)

type InteractionType string

const (
	InteractionTypeView InteractionType = "view"
	InteractionTypeLike InteractionType = "like"
)

func (e *InteractionType) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = InteractionType(s)
	case string:
		*e = InteractionType(s)
	default:
		return fmt.Errorf("unsupported scan type for InteractionType: %T", src)
	}
	return nil
}

type NullInteractionType struct {
	InteractionType InteractionType
	Valid           bool // Valid is true if InteractionType is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullInteractionType) Scan(value interface{}) error {
	if value == nil {
		ns.InteractionType, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.InteractionType.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullInteractionType) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.InteractionType), nil
}

type NotificationType string

const (
	NotificationTypeLike     NotificationType = "like"
	NotificationTypeComment  NotificationType = "comment"
	NotificationTypeReply    NotificationType = "reply"
	NotificationTypeNewVideo NotificationType = "new_video"
)

func (e *NotificationType) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = NotificationType(s)
	case string:
		*e = NotificationType(s)
	default:
		return fmt.Errorf("unsupported scan type for NotificationType: %T", src)
	}
	return nil
}

type NullNotificationType struct {
	NotificationType NotificationType
	Valid            bool // Valid is true if NotificationType is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullNotificationType) Scan(value interface{}) error {
	if value == nil {
		ns.NotificationType, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.NotificationType.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullNotificationType) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.NotificationType), nil
}

type StreamtapeStatus string

const (
	StreamtapeStatusActive  StreamtapeStatus = "active"
	StreamtapeStatusDead    StreamtapeStatus = "dead"
	StreamtapeStatusPending StreamtapeStatus = "pending"
)

func (e *StreamtapeStatus) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = StreamtapeStatus(s)
	case string:
		*e = StreamtapeStatus(s)
	default:
		return fmt.Errorf("unsupported scan type for StreamtapeStatus: %T", src)
	}
	return nil
}

type NullStreamtapeStatus struct {
	StreamtapeStatus StreamtapeStatus
	Valid            bool // Valid is true if StreamtapeStatus is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullStreamtapeStatus) Scan(value interface{}) error {
	if value == nil {
		ns.StreamtapeStatus, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.StreamtapeStatus.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullStreamtapeStatus) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.StreamtapeStatus), nil
}

type VideoVisibility string

const (
	VideoVisibilityPublic   VideoVisibility = "public"
	VideoVisibilityUnlisted VideoVisibility = "unlisted"
	VideoVisibilityPrivate  VideoVisibility = "private"
)

func (e *VideoVisibility) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = VideoVisibility(s)
	case string:
		*e = VideoVisibility(s)
	default:
		return fmt.Errorf("unsupported scan type for VideoVisibility: %T", src)
	}
	return nil
}

type NullVideoVisibility struct {
	VideoVisibility VideoVisibility
	Valid           bool // Valid is true if VideoVisibility is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullVideoVisibility) Scan(value interface{}) error {
	if value == nil {
		ns.VideoVisibility, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.VideoVisibility.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullVideoVisibility) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.VideoVisibility), nil
}

type Comment struct {
	ID              pgtype.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Text            string             `gorm:"type:text;not null"`
	AuthorID        pgtype.UUID        `gorm:"type:uuid;not null"`
	VideoID         pgtype.UUID        `gorm:"type:uuid;not null"`
	ParentCommentID pgtype.UUID        `gorm:"type:uuid"`
	CreatedAt       pgtype.Timestamptz `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP"`
	UpdatedAt       pgtype.Timestamptz `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP"`
}

type Notification struct {
	ID          pgtype.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RecipientID pgtype.UUID        `gorm:"type:uuid;not null"`
	SenderID    pgtype.UUID        `gorm:"type:uuid;not null"`
	Type        NotificationType   `gorm:"type:notification_type;not null"`
	VideoID     pgtype.UUID        `gorm:"type:uuid"`
	CommentID   pgtype.UUID        `gorm:"type:uuid"`
	IsRead      pgtype.Bool        `gorm:"default:false"`
	CreatedAt   pgtype.Timestamptz `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP"`
	UpdatedAt   pgtype.Timestamptz `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP"`
}

type Playlist struct {
	ID          pgtype.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Title       string             `gorm:"type:varchar(255);not null"`
	Description pgtype.Text        `gorm:"type:text"`
	OwnerID     pgtype.UUID        `gorm:"type:uuid;not null"`
	IsPublic    pgtype.Bool        `gorm:"default:true"`
	CreatedAt   pgtype.Timestamptz `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP"`
	UpdatedAt   pgtype.Timestamptz `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP"`
}

type PlaylistVideo struct {
	PlaylistID pgtype.UUID        `gorm:"type:uuid;primaryKey"`
	VideoID    pgtype.UUID        `gorm:"type:uuid;primaryKey"`
	AddedAt    pgtype.Timestamptz `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP"`
}

type Subscription struct {
	SubscriberID   pgtype.UUID        `gorm:"type:uuid;primaryKey"`
	SubscribedToID pgtype.UUID        `gorm:"type:uuid;primaryKey"`
	CreatedAt      pgtype.Timestamptz `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP"`
}

type User struct {
	ID        pgtype.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email     string             `gorm:"type:varchar(255);uniqueIndex;not null"`
	Password  string             `gorm:"type:varchar(255);not null"`
	Username  string             `gorm:"type:varchar(100);uniqueIndex;not null"`
	Avatar    pgtype.Text        `gorm:"type:text"`
	Banner    pgtype.Text        `gorm:"type:text"`
	Bio       pgtype.Text        `gorm:"type:text"`
	CreatedAt pgtype.Timestamptz `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP"`
	UpdatedAt pgtype.Timestamptz `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP"`
}

type UserActivity struct {
	ID              pgtype.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID          pgtype.UUID        `gorm:"type:uuid;not null;uniqueIndex:idx_user_video_interaction"`
	VideoID         pgtype.UUID        `gorm:"type:uuid;not null;uniqueIndex:idx_user_video_interaction"`
	InteractionType InteractionType    `gorm:"type:interaction_type;not null;uniqueIndex:idx_user_video_interaction"`
	CreatedAt       pgtype.Timestamptz `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP"`
	UpdatedAt       pgtype.Timestamptz `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP"`
}

type Video struct {
	ID                    pgtype.UUID          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Title                 string               `gorm:"type:varchar(255);not null"`
	Description           pgtype.Text          `gorm:"type:text;default:''"`
	FileID                string               `gorm:"column:file_id;type:varchar(255);uniqueIndex;not null"`
	ThumbnailUrl          pgtype.Text          `gorm:"column:thumbnail_url;type:text"`
	Category              pgtype.Text          `gorm:"type:varchar(100);default:'Other'"`
	Tags                  TextArray            `gorm:"type:text[]"`
	Visibility            NullVideoVisibility  `gorm:"type:video_visibility;default:public"`
	UploaderID            pgtype.UUID          `gorm:"type:uuid;not null"`
	IsShort               pgtype.Bool          `gorm:"default:false"`
	Views                 pgtype.Int4          `gorm:"default:0"`
	StreamtapeUrl         pgtype.Text          `gorm:"column:streamtape_url;type:text"`
	LastRefreshedAt       pgtype.Timestamptz   `gorm:"column:last_refreshed_at;type:timestamptz;default:CURRENT_TIMESTAMP"`
	PendingRemoteUploadID pgtype.Text          `gorm:"column:pending_remote_upload_id;type:varchar(255)"`
	StreamtapeStatus      NullStreamtapeStatus `gorm:"type:streamtape_status;default:active"`
	CloneAttempts         pgtype.Int4          `gorm:"default:0"`
	CreatedAt             pgtype.Timestamptz   `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP"`
	UpdatedAt             pgtype.Timestamptz   `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP"`
}

type WatchHistory struct {
	UserID    pgtype.UUID        `gorm:"type:uuid;primaryKey"`
	VideoID   pgtype.UUID        `gorm:"type:uuid;primaryKey"`
	CreatedAt pgtype.Timestamptz `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP"`
	UpdatedAt pgtype.Timestamptz `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP"`
}

type WatchLater struct {
	UserID    pgtype.UUID        `gorm:"type:uuid;primaryKey"`
	VideoID   pgtype.UUID        `gorm:"type:uuid;primaryKey"`
	CreatedAt pgtype.Timestamptz `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP"`
	UpdatedAt pgtype.Timestamptz `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP"`
}

func (Comment) TableName() string       { return "comments" }
func (Notification) TableName() string  { return "notifications" }
func (Playlist) TableName() string      { return "playlists" }
func (PlaylistVideo) TableName() string { return "playlist_videos" }
func (Subscription) TableName() string  { return "subscriptions" }
func (User) TableName() string          { return "users" }
func (UserActivity) TableName() string  { return "user_activities" }
func (Video) TableName() string         { return "videos" }
func (WatchHistory) TableName() string  { return "watch_history" }
func (WatchLater) TableName() string    { return "watch_later" }
