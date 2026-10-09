package kafka

import "time"

const (
	TopicVideoViews       = "aurahub.video.views"
	TopicBatchUploadJobs  = "aurahub.batch-upload-jobs"
	TopicNotifications    = "aurahub.notifications"
	TopicVideoLifecycle   = "aurahub.video.lifecycle"

	GroupViewsFlusher     = "aurahub-views-flusher"
	GroupBatchUploadJobs  = "aurahub-batch-upload-worker"
)

// VideoViewEvent is emitted every time a user or guest views a video.
type VideoViewEvent struct {
	VideoID   string    `json:"videoId"`
	UserID    string    `json:"userId,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// BatchUploadJobEvent represents a single video in an asynchronous batch remote-upload job.
type BatchUploadJobEvent struct {
	JobID       string    `json:"jobId"`
	UserID      string    `json:"userId"`
	PlaylistID  string    `json:"playlistId,omitempty"`
	Title       string    `json:"title"`
	VideoURL    string    `json:"videoUrl"`
	Category    string    `json:"category"`
	Visibility  string    `json:"visibility"`
	IsShort     bool      `json:"isShort"`
	IsAdult     bool      `json:"isAdult"`
	Timestamp   time.Time `json:"timestamp"`
}

// NotificationEvent is emitted when a user receives a new video alert, comment, or reply.
type NotificationEvent struct {
	RecipientID string    `json:"recipientId"`
	SenderID    string    `json:"senderId"`
	Type        string    `json:"type"`
	VideoID     string    `json:"videoId,omitempty"`
	CommentID   string    `json:"commentId,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// VideoLifecycleEvent notifies downstream consumers (e.g. OpenSearch) of video changes.
type VideoLifecycleEvent struct {
	Action    string    `json:"action"` // "created", "updated", "deleted"
	VideoID   string    `json:"videoId"`
	Timestamp time.Time `json:"timestamp"`
}
