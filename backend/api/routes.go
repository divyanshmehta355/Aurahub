package api

import (
	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/divyanshmehta355/aurahub/backend/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	Repository *db.Repository
	Cache      *redis.Client
}

func NewServer(repository *db.Repository, cache *redis.Client) *Server {
	return &Server{
		Repository: repository,
		Cache:      cache,
	}
}

func (s *Server) SetupRoutes(app *fiber.App) {
	api := app.Group("/api")

	// Health check
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Media routes
	app.Get("/api/avatar/:seed", s.AvatarHandler)
	app.Post("/api/upload/avatar", s.AvatarUploadHandler)
	api.Get("/thumbnail/:seed", s.DynamicThumbnailHandler)

	// Auth routes
	auth := api.Group("/auth")
	auth.Post("/register", s.RegisterHandler)
	auth.Post("/login", s.LoginHandler)
	auth.Post("/logout", s.LogoutHandler)
	auth.Post("/check-email", s.CheckEmailHandler)
	auth.Post("/check-username", s.CheckUsernameHandler)
	auth.Get("/me", middleware.AuthRequired(), s.GetMeHandler)

	videos := api.Group("/videos")
	videos.Get("/", s.ListVideosHandler)
	videos.Get("/search", s.SearchVideosHandler)
	videos.Get("/recommendations", s.RecommendationsHandler)
	videos.Get("/suggestions", s.SuggestionsHandler)
	videos.Get("/stream/:id", s.StreamVideoHandler)
	videos.Get("/remote-upload/status", s.RemoteUploadStatusHandler)
	videos.Post("/remote-upload/start", middleware.AuthRequired(), s.RemoteUploadStartHandler)
	videos.Put("/bulk", middleware.AuthRequired(), s.BulkUpdateVideoVisibilityHandler)
	videos.Put("/bulk-adult", middleware.AuthRequired(), s.BulkUpdateVideoAdultHandler)
	videos.Delete("/bulk", middleware.AuthRequired(), s.BulkDeleteVideosHandler)

	// Protected video routes
	videos.Get("/feed/subscriptions", middleware.AuthRequired(), s.SubscriptionFeedHandler)
	videos.Get("/get-upload-url", middleware.AuthRequired(), s.GetUploadUrlHandler)
	videos.Post("/upload-chunk", middleware.AuthRequired(), s.UploadChunkHandler)
	videos.Post("/create-record", middleware.AuthRequired(), s.CreateRecordHandler)

	// Dynamic ID routes should come last
	videos.Get("/:id/thumbnail", s.GetVideoThumbnailHandler)
	videos.Post("/:id/update-thumbnail", middleware.AuthRequired(), s.UpdateVideoThumbnailHandler)
	videos.Get("/:id", middleware.OptionalAuth(), s.GetVideoHandler)
	videos.Delete("/:id", middleware.AuthRequired(), s.DeleteVideoHandler)
	videos.Put("/:id", middleware.AuthRequired(), s.UpdateVideoHandler)
	videos.Post("/:videoId/like", middleware.AuthRequired(), s.ToggleLikeHandler)
	videos.Post("/:id/view", middleware.OptionalAuth(), s.RecordVideoViewHandler)

	// User routes
	users := api.Group("/users")
	users.Get("/profile/:identifier", middleware.OptionalAuth(), s.GetProfileHandler)
	users.Get("/:id", s.GetUserHandler)
	users.Post("/:identifier/subscribe", middleware.AuthRequired(), s.ToggleSubscriptionHandler)
	users.Put("/me", middleware.AuthRequired(), s.UpdateUserHandler)
	users.Put("/profile", middleware.AuthRequired(), s.UpdateProfileHandler)
	users.Put("/security", middleware.AuthRequired(), s.UpdateProfileHandler)
	users.Post("/profile/image", middleware.AuthRequired(), s.AvatarUploadHandler)

	// Comments routes
	comments := api.Group("/comments")
	comments.Get("/:videoId", s.ListCommentsHandler)
	comments.Post("/", middleware.AuthRequired(), s.CreateCommentHandler)
	comments.Get("/:id/replies", s.ListCommentRepliesHandler)
	api.Get("/search/autocomplete", s.AutocompleteHandler)
	comments.Put("/:id", middleware.AuthRequired(), s.UpdateCommentHandler)
	comments.Delete("/:id", middleware.AuthRequired(), s.DeleteCommentHandler)

	// Notifications routes
	notifications := api.Group("/notifications", middleware.AuthRequired())
	notifications.Get("/stream", s.NotificationStreamHandler)
	notifications.Get("/", s.ListNotificationsHandler)
	notifications.Post("/", s.MarkNotificationsReadHandler)
	notifications.Post("/read-all", s.MarkNotificationsReadHandler)
	notifications.Post("/:id/read", s.MarkNotificationReadHandler)
	notifications.Patch("/:id/read", s.MarkNotificationReadHandler)
	notifications.Delete("/:id", s.DeleteNotificationHandler)
	notifications.Delete("/", s.ClearNotificationsHandler)

	// User History, Watch Later, and Profile settings routes
	userGroup := api.Group("/user", middleware.AuthRequired())
	userGroup.Get("/history", s.GetWatchHistoryHandler)
	userGroup.Delete("/history", s.DeleteWatchHistoryHandler)
	userGroup.Get("/watch-later", s.GetWatchLaterHandler)
	userGroup.Post("/watch-later", s.ToggleWatchLaterHandler)
	userGroup.Delete("/watch-later", s.DeleteWatchLaterHandler)
	userGroup.Put("/profile", s.UpdateProfileHandler)
	userGroup.Put("/security", s.UpdateProfileHandler)
	userGroup.Post("/profile/image", s.AvatarUploadHandler)

	// Playlists routes
	playlists := api.Group("/playlists")
	playlists.Get("/", middleware.AuthRequired(), s.ListPlaylistsHandler)
	playlists.Post("/", middleware.AuthRequired(), s.CreatePlaylistHandler)
	playlists.Put("/", middleware.AuthRequired(), s.UpdatePlaylistCollectionHandler)
	playlists.Get("/:id", middleware.OptionalAuth(), s.GetPlaylistHandler)
	playlists.Put("/:id", middleware.AuthRequired(), s.UpdatePlaylistHandler)
	playlists.Delete("/:id", middleware.AuthRequired(), s.DeletePlaylistHandler)
	playlists.Post("/:id/videos", middleware.AuthRequired(), s.TogglePlaylistVideoHandler)

	// Creator routes
	creator := api.Group("/creator", middleware.AuthRequired())
	creator.Get("/dashboard", s.CreatorDashboardHandler)
	creator.Get("/analytics", s.CreatorAnalyticsHandler)
	creator.Put("/videos/bulk", s.BulkUpdateVideoVisibilityHandler)
	creator.Put("/videos/bulk-adult", s.BulkUpdateVideoAdultHandler)
	creator.Put("/videos/bulk/adult", s.BulkUpdateVideoAdultHandler)
	creator.Delete("/videos/bulk", s.BulkDeleteVideosHandler)
}
