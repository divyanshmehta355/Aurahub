package api

import (
	"os"

	"github.com/divyanshmehta355/aurahub/backend/config"
	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/divyanshmehta355/aurahub/backend/internal/kafka"
	"github.com/divyanshmehta355/aurahub/backend/internal/opensearch"
	"github.com/divyanshmehta355/aurahub/backend/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	Repository      *db.Repository
	Cache           *redis.Client
	Kafka           *kafka.Producer
	OpenSearch      *opensearch.Client
	UploadFolderID  string
	FreeImageAPIKey string
	JWTSecret       string
}

func NewServer(repository *db.Repository, cache *redis.Client, options ...any) *Server {
	s := &Server{
		Repository: repository,
		Cache:      cache,
	}
	for _, opt := range options {
		switch v := opt.(type) {
		case *kafka.Producer:
			s.Kafka = v
		case *opensearch.Client:
			s.OpenSearch = v
		case *config.Config:
			s.UploadFolderID = v.UploadFolderID
			s.FreeImageAPIKey = v.FreeImageAPIKey
			s.JWTSecret = v.JWTSecret
		}
	}
	return s
}

func (s *Server) getUploadFolderID() string {
	if s != nil && s.UploadFolderID != "" {
		return s.UploadFolderID
	}
	return os.Getenv("UPLOAD_FOLDER_ID")
}

func (s *Server) getFreeImageAPIKey() string {
	if s != nil && s.FreeImageAPIKey != "" {
		return s.FreeImageAPIKey
	}
	return os.Getenv("FREEIMAGE_API_KEY")
}

func (s *Server) getJWTSecret() string {
	if s != nil && s.JWTSecret != "" {
		return s.JWTSecret
	}
	return os.Getenv("JWT_SECRET")
}

func (s *Server) SetupRoutes(app *fiber.App) {
	api := app.Group("/api", s.MaintenanceModeMiddleware())

	// Public System Status & Feature Flags
	api.Get("/system/status", s.PublicSystemStatusHandler)

	// Health check
	api.Get("/health", func(c *fiber.Ctx) error {
		status := "ok"
		dbStatus := "disabled"
		if s.Repository != nil {
			if sqlDB, err := s.Repository.GetORM().DB(); err == nil {
				if err := sqlDB.PingContext(c.UserContext()); err != nil {
					return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
						"status": "degraded",
						"error":  "database ping failed: " + err.Error(),
					})
				}
				dbStatus = "connected"
			}
		}
		cacheStatus := "disabled"
		if s.Cache != nil {
			if err := s.Cache.Ping(c.UserContext()).Err(); err != nil {
				return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
					"status": "degraded",
					"error":  "cache ping failed: " + err.Error(),
				})
			}
			cacheStatus = "connected"
		}
		return c.JSON(fiber.Map{
			"status":   status,
			"database": dbStatus,
			"cache":    cacheStatus,
		})
	})

	// Media routes
	app.Get("/api/avatar/:seed", s.AvatarHandler)
	app.Post("/api/upload/avatar", middleware.AuthRequired(), s.ResolveAdminStatus(), s.EnforceKillSwitch(SwitchFreezeUploads), s.AvatarUploadHandler)
	api.Get("/thumbnail/:seed", s.DynamicThumbnailHandler)

	// Auth routes
	auth := api.Group("/auth")
	auth.Post("/register", s.EnforceKillSwitch(SwitchFreezeRegistration), middleware.RegisterLimiter(), s.RegisterHandler)
	auth.Post("/login", middleware.LoginLimiter(), s.LoginHandler)
	auth.Post("/logout", s.LogoutHandler)
	auth.Post("/check-email", middleware.CheckAvailabilityLimiter(), s.CheckEmailHandler)
	auth.Post("/check-username", middleware.CheckAvailabilityLimiter(), s.CheckUsernameHandler)
	auth.Get("/me", middleware.AuthRequired(), s.GetMeHandler)

	videos := api.Group("/videos")
	videos.Get("/", middleware.OptionalAuth(), s.ListVideosHandler)
	videos.Get("/search", middleware.SearchLimiter(), s.SearchVideosHandler)
	videos.Get("/recommendations", middleware.OptionalAuth(), s.RecommendationsHandler)
	videos.Get("/suggestions", middleware.OptionalAuth(), s.SuggestionsHandler)
	videos.Get("/stream/:id", middleware.OptionalAuth(), s.ResolveAdminStatus(), s.EnforceKillSwitch(SwitchFreezeStreaming), s.StreamVideoHandler)
	videos.Get("/remote-upload/status", s.RemoteUploadStatusHandler)
	videos.Post("/remote-upload/start", middleware.AuthRequired(), s.ResolveAdminStatus(), s.EnforceKillSwitch(SwitchFreezeUploads), middleware.RemoteUploadLimiter(), s.RemoteUploadStartHandler)
	videos.Post("/remote-upload/batch", middleware.AuthRequired(), s.ResolveAdminStatus(), s.EnforceKillSwitch(SwitchFreezeUploads), middleware.RemoteUploadLimiter(), s.BatchRemoteUploadHandler)
	videos.Post("/refresh-expired", middleware.AuthRequired(), s.TriggerRefreshExpiredHandler)
	videos.Put("/bulk", middleware.AuthRequired(), s.BulkUpdateVideoVisibilityHandler)
	videos.Put("/bulk-adult", middleware.AuthRequired(), s.BulkUpdateVideoAdultHandler)
	videos.Delete("/bulk", middleware.AuthRequired(), s.BulkDeleteVideosHandler)

	// Protected video routes
	videos.Get("/feed/subscriptions", middleware.AuthRequired(), s.SubscriptionFeedHandler)
	videos.Get("/get-upload-url", middleware.AuthRequired(), s.ResolveAdminStatus(), s.EnforceKillSwitch(SwitchFreezeUploads), s.GetUploadUrlHandler)
	videos.Post("/upload-chunk", middleware.AuthRequired(), s.ResolveAdminStatus(), s.EnforceKillSwitch(SwitchFreezeUploads), s.UploadChunkHandler)
	videos.Post("/create-record", middleware.AuthRequired(), s.ResolveAdminStatus(), s.EnforceKillSwitch(SwitchFreezeUploads), s.CreateRecordHandler)

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
	users.Post("/profile/image", middleware.AuthRequired(), s.ResolveAdminStatus(), s.EnforceKillSwitch(SwitchFreezeUploads), s.AvatarUploadHandler)

	// Comments routes
	comments := api.Group("/comments")
	comments.Get("/:videoId", s.ListCommentsHandler)
	comments.Post("/", middleware.AuthRequired(), s.ResolveAdminStatus(), s.EnforceKillSwitch(SwitchFreezeComments), middleware.CommentsLimiter(), s.CreateCommentHandler)
	comments.Get("/:id/replies", s.ListCommentRepliesHandler)
	api.Get("/search/autocomplete", middleware.SearchLimiter(), s.AutocompleteHandler)
	comments.Put("/:id", middleware.AuthRequired(), s.ResolveAdminStatus(), s.EnforceKillSwitch(SwitchFreezeComments), s.UpdateCommentHandler)
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
	playlists.Get("/my-playlists", middleware.AuthRequired(), s.ListPlaylistsHandler)
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

	// Admin routes
	admin := api.Group("/admin", middleware.AuthRequired(), s.AdminAuthRequired())
	admin.Get("/stats", s.AdminStatsHandler)
	admin.Get("/videos", s.AdminListVideosHandler)
	admin.Put("/videos/:id", s.AdminUpdateVideoHandler)
	admin.Delete("/videos/:id", s.AdminDeleteVideoHandler)
	admin.Get("/users", s.AdminListUsersHandler)
	admin.Put("/users/:id/role", s.AdminUpdateUserRoleHandler)
	admin.Delete("/users/:id", s.AdminDeleteUserHandler)
	admin.Post("/actions/refresh-cycle", s.AdminTriggerRefreshHandler)
	admin.Post("/actions/flush-cache", s.AdminFlushCacheHandler)
	admin.Get("/kill-switches", s.AdminGetKillSwitchesHandler)
	admin.Post("/kill-switches", s.AdminToggleKillSwitchHandler)
}
