package api

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/divyanshmehta355/aurahub/backend/internal/httpclient"
	"github.com/divyanshmehta355/aurahub/backend/internal/kafka/workers"
	"github.com/divyanshmehta355/aurahub/backend/internal/opensearch"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgtype"
)

// AdminAuthRequired verifies that the requesting user is authenticated and has administrative rights.
func (s *Server) AdminAuthRequired() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userIdLocal := c.Locals("userId")
		if userIdLocal == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Authentication required"})
		}
		userIdStr, ok := userIdLocal.(string)
		if !ok || userIdStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid session"})
		}
		userId, err := parseUUID(userIdStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid user ID"})
		}
		user, err := s.Repository.GetUser(context.Background(), userId)
		if err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "User not found or access denied"})
		}
		if !isUserAdmin(user) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "Administrator privileges required"})
		}
		c.Locals("adminUser", user)
		return c.Next()
	}
}

// AdminStatsHandler returns complete platform-wide health, moderation, and infrastructure metrics.
func (s *Server) AdminStatsHandler(c *fiber.Ctx) error {
	ctx := context.Background()
	orm := s.Repository.GetORM().WithContext(ctx)

	var totalUsers, adultConsentUsers, recentSignups int64
	orm.Model(&db.User{}).Count(&totalUsers)
	orm.Model(&db.User{}).Where("show_adult_content = ?", true).Count(&adultConsentUsers)
	orm.Model(&db.User{}).Where("created_at >= ?", time.Now().Add(-7*24*time.Hour)).Count(&recentSignups)

	var totalVideos, safeVideos, adultVideos, totalShorts int64
	orm.Model(&db.Video{}).Count(&totalVideos)
	orm.Model(&db.Video{}).Where("is_adult = ?", false).Count(&safeVideos)
	orm.Model(&db.Video{}).Where("is_adult = ?", true).Count(&adultVideos)
	orm.Model(&db.Video{}).Where("is_short = ?", true).Count(&totalShorts)

	var totalViews int64
	orm.Model(&db.Video{}).Select("COALESCE(SUM(views), 0)").Scan(&totalViews)

	var totalComments, totalLikes, totalPlaylists int64
	orm.Model(&db.Comment{}).Count(&totalComments)
	orm.Model(&db.UserActivity{}).Where("interaction_type = ?", "like").Count(&totalLikes)
	orm.Model(&db.Playlist{}).Count(&totalPlaylists)

	var activeStreamtape, deadStreamtape, pendingStreamtape int64
	orm.Model(&db.Video{}).Where("streamtape_status = ?", "active").Count(&activeStreamtape)
	orm.Model(&db.Video{}).Where("streamtape_status = ?", "dead").Count(&deadStreamtape)
	orm.Model(&db.Video{}).Where("streamtape_status = ?", "pending").Count(&pendingStreamtape)

	var needingRefresh int64
	cutoff28Days := time.Now().Add(-28 * 24 * time.Hour)
	orm.Model(&db.Video{}).Where("last_refreshed_at < ? OR last_refreshed_at IS NULL", cutoff28Days).Count(&needingRefresh)

	return c.JSON(fiber.Map{
		"users": fiber.Map{
			"total":         totalUsers,
			"adultEnabled":  adultConsentUsers,
			"recentSignups": recentSignups,
		},
		"videos": fiber.Map{
			"total":        totalVideos,
			"safe":         safeVideos,
			"adult":        adultVideos,
			"shorts":       totalShorts,
			"totalViews":   totalViews,
		},
		"engagement": fiber.Map{
			"comments":  totalComments,
			"likes":     totalLikes,
			"playlists": totalPlaylists,
		},
		"streamtape": fiber.Map{
			"active":         activeStreamtape,
			"dead":           deadStreamtape,
			"pending":        pendingStreamtape,
			"needingRefresh": needingRefresh,
		},
		"system": fiber.Map{
			"opensearchEnabled": s.OpenSearch != nil && s.OpenSearch.Enabled(),
			"cacheEnabled":      s.Cache != nil,
			"serverTime":        time.Now(),
		},
	})
}

// AdminListVideosHandler returns paginated videos platform-wide with moderation filters.
func (s *Server) AdminListVideosHandler(c *fiber.Ctx) error {
	ctx := context.Background()
	orm := s.Repository.GetORM().WithContext(ctx)

	page, limit := pageLimit(c, 15, 100)
	offset := (page - 1) * limit

	queryBuilder := orm.Model(&db.Video{}).
		Select("videos.*, users.username as uploader_username, users.avatar as uploader_avatar").
		Joins("LEFT JOIN users ON users.id = videos.uploader_id")

	// Filter by search keyword
	if q := strings.TrimSpace(c.Query("q")); q != "" {
		queryBuilder = queryBuilder.Where("videos.title ILIKE ? OR users.username ILIKE ?", "%"+q+"%", "%"+q+"%")
	}

	// Filter by adult status
	if adultFilter := c.Query("adult"); adultFilter == "true" {
		queryBuilder = queryBuilder.Where("videos.is_adult = ?", true)
	} else if adultFilter == "false" {
		queryBuilder = queryBuilder.Where("videos.is_adult = ?", false)
	}

	// Filter by visibility
	if vis := c.Query("visibility"); vis != "" && vis != "all" {
		queryBuilder = queryBuilder.Where("videos.visibility = ?", vis)
	}

	// Filter by streamtape status
	if status := c.Query("status"); status != "" && status != "all" {
		queryBuilder = queryBuilder.Where("videos.streamtape_status = ?", status)
	}

	// Filter by category
	if cat := c.Query("category"); cat != "" && cat != "All" && cat != "all" {
		queryBuilder = queryBuilder.Where("videos.category = ?", cat)
	}

	var total int64
	if err := queryBuilder.Count(&total).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to count videos"})
	}

	// Sort
	switch c.Query("sort", "newest") {
	case "views":
		queryBuilder = queryBuilder.Order("videos.views DESC, videos.created_at DESC")
	case "oldest":
		queryBuilder = queryBuilder.Order("videos.created_at ASC")
	case "needs_refresh":
		queryBuilder = queryBuilder.Order("videos.last_refreshed_at ASC NULLS FIRST, videos.created_at DESC")
	default:
		queryBuilder = queryBuilder.Order("videos.created_at DESC")
	}

	type AdminVideoRow struct {
		db.Video
		UploaderUsername string      `gorm:"column:uploader_username"`
		UploaderAvatar   pgtype.Text `gorm:"column:uploader_avatar"`
	}

	var rows []AdminVideoRow
	if err := queryBuilder.Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to query videos"})
	}

	results := make([]fiber.Map, 0, len(rows))
	for _, v := range rows {
		results = append(results, fiber.Map{
			"id":               formatUUID(v.ID),
			"fileId":           v.FileID,
			"title":            v.Title,
			"description":      v.Description.String,
			"thumbnailUrl":     v.ThumbnailUrl.String,
			"category":         v.Category.String,
			"visibility":       v.Visibility.VideoVisibility,
			"isShort":          v.IsShort.Bool,
			"isAdult":          v.IsAdult.Bool,
			"views":            v.Views.Int32,
			"streamtapeStatus": v.StreamtapeStatus.StreamtapeStatus,
			"lastRefreshedAt":  v.LastRefreshedAt.Time,
			"createdAt":        v.CreatedAt.Time,
			"uploader": fiber.Map{
				"id":       formatUUID(v.UploaderID),
				"username": v.UploaderUsername,
				"avatar":   v.UploaderAvatar.String,
			},
		})
	}

	return c.JSON(fiber.Map{
		"videos":      results,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"totalPages":  (total + int64(limit) - 1) / int64(limit),
	})
}

// AdminUpdateVideoHandler allows moderators to update video categorization, adult classification, or visibility.
func (s *Server) AdminUpdateVideoHandler(c *fiber.Ctx) error {
	idStr := c.Params("id")
	ctx := context.Background()

	var video db.Video
	query := s.Repository.GetORM().WithContext(ctx)
	if id, err := parseUUID(idStr); err == nil {
		query = query.Where("id = ? OR file_id = ?", id, idStr)
	} else {
		query = query.Where("file_id = ?", idStr)
	}

	if err := query.First(&video).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"message": "Video not found"})
	}

	var req struct {
		Title            *string `json:"title"`
		Category         *string `json:"category"`
		IsAdult          *bool   `json:"isAdult"`
		Visibility       *string `json:"visibility"`
		StreamtapeStatus *string `json:"streamtapeStatus"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid request body"})
	}

	updates := make(map[string]interface{})
	if req.Title != nil && strings.TrimSpace(*req.Title) != "" {
		updates["title"] = strings.TrimSpace(*req.Title)
		video.Title = strings.TrimSpace(*req.Title)
	}
	if req.Category != nil {
		updates["category"] = strings.TrimSpace(*req.Category)
		video.Category = pgtype.Text{String: strings.TrimSpace(*req.Category), Valid: true}
	}
	if req.IsAdult != nil {
		updates["is_adult"] = *req.IsAdult
		video.IsAdult = pgtype.Bool{Bool: *req.IsAdult, Valid: true}
	}
	if req.Visibility != nil {
		vis := strings.ToLower(strings.TrimSpace(*req.Visibility))
		if vis == "public" || vis == "unlisted" || vis == "private" {
			updates["visibility"] = vis
			video.Visibility = db.NullVideoVisibility{VideoVisibility: db.VideoVisibility(vis), Valid: true}
		}
	}
	if req.StreamtapeStatus != nil {
		st := strings.ToLower(strings.TrimSpace(*req.StreamtapeStatus))
		if st == "active" || st == "dead" || st == "pending" {
			updates["streamtape_status"] = st
		}
	}

	updates["updated_at"] = time.Now()
	if err := s.Repository.GetORM().WithContext(ctx).Model(&db.Video{}).Where("id = ?", video.ID).Updates(updates).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to update video: " + err.Error()})
	}

	// Sync with OpenSearch if enabled
	s.indexVideoInOpenSearch(ctx, video)

	// Invalidate feed & search caches
	if s.Cache != nil {
		s.InvalidateFeedCache(ctx)
		_ = s.Cache.Del(ctx, "thumbnail:"+idStr, "stream:"+idStr).Err()
		if video.FileID != "" {
			_ = s.Cache.Del(ctx, "thumbnail:"+video.FileID, "stream:"+video.FileID).Err()
		}
	}

	return c.JSON(fiber.Map{
		"message": "Video updated successfully",
		"id":      formatUUID(video.ID),
		"isAdult": video.IsAdult.Bool,
	})
}

// AdminDeleteVideoHandler deletes any video across PostgreSQL, OpenSearch, and Streamtape.
func (s *Server) AdminDeleteVideoHandler(c *fiber.Ctx) error {
	idStr := c.Params("id")
	ctx := context.Background()

	var video db.Video
	query := s.Repository.GetORM().WithContext(ctx)
	if id, err := parseUUID(idStr); err == nil {
		query = query.Where("id = ? OR file_id = ?", id, idStr)
	} else {
		query = query.Where("file_id = ?", idStr)
	}

	if err := query.First(&video).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"message": "Video not found"})
	}

	// Delete from Streamtape asynchronously
	go func(fileID string) {
		if fileID != "" {
			_ = httpclient.DeleteStreamtapeFile(context.Background(), fileID)
		}
	}(video.FileID)

	// Delete from OpenSearch
	if s.OpenSearch != nil && s.OpenSearch.Enabled() {
		_ = s.OpenSearch.DeleteVideo(ctx, formatUUID(video.ID))
	}

	// Delete from PostgreSQL
	if err := s.Repository.DeleteVideo(ctx, video.ID); err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to delete video: " + err.Error()})
	}

	// Clear cache
	if s.Cache != nil {
		s.InvalidateFeedCache(ctx)
		_ = s.Cache.Del(ctx, "thumbnail:"+idStr, "stream:"+idStr).Err()
		if video.FileID != "" {
			_ = s.Cache.Del(ctx, "thumbnail:"+video.FileID, "stream:"+video.FileID).Err()
		}
	}

	return c.JSON(fiber.Map{"message": "Video deleted successfully"})
}

// AdminListUsersHandler returns paginated users with video metrics and admin status.
func (s *Server) AdminListUsersHandler(c *fiber.Ctx) error {
	ctx := context.Background()
	orm := s.Repository.GetORM().WithContext(ctx)

	page, limit := pageLimit(c, 20, 100)
	offset := (page - 1) * limit

	queryBuilder := orm.Model(&db.User{})

	if q := strings.TrimSpace(c.Query("q")); q != "" {
		queryBuilder = queryBuilder.Where("username ILIKE ? OR email ILIKE ?", "%"+q+"%", "%"+q+"%")
	}

	var total int64
	if err := queryBuilder.Count(&total).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to count users"})
	}

	var users []db.User
	if err := queryBuilder.Order("created_at DESC").Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to query users"})
	}

	userResults := make([]fiber.Map, 0, len(users))
	for _, u := range users {
		var videoCount int64
		orm.Model(&db.Video{}).Where("uploader_id = ?", u.ID).Count(&videoCount)

		var subscriberCount int64
		orm.Model(&db.Subscription{}).Where("subscribed_to_id = ?", u.ID).Count(&subscriberCount)

		userResults = append(userResults, fiber.Map{
			"id":               formatUUID(u.ID),
			"username":         u.Username,
			"email":            u.Email,
			"avatar":           u.Avatar.String,
			"bio":              u.Bio.String,
			"showAdultContent": u.ShowAdultContent.Bool,
			"isAdmin":          isUserAdmin(u),
			"createdAt":        u.CreatedAt.Time,
			"videoCount":       videoCount,
			"subscriberCount":  subscriberCount,
		})
	}

	return c.JSON(fiber.Map{
		"users":      userResults,
		"total":      total,
		"page":       page,
		"limit":      limit,
		"totalPages": (total + int64(limit) - 1) / int64(limit),
	})
}

// AdminUpdateUserRoleHandler toggles or assigns admin status to any registered user.
func (s *Server) AdminUpdateUserRoleHandler(c *fiber.Ctx) error {
	idStr := c.Params("id")
	userId, err := parseUUID(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid user ID"})
	}

	var req struct {
		IsAdmin bool `json:"isAdmin"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid request body"})
	}

	ctx := context.Background()
	if err := s.Repository.GetORM().WithContext(ctx).Model(&db.User{}).Where("id = ?", userId).Update("is_admin", req.IsAdmin).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to update user role"})
	}

	return c.JSON(fiber.Map{
		"message": "User admin status updated successfully",
		"id":      idStr,
		"isAdmin": req.IsAdmin,
	})
}

// AdminDeleteUserHandler cascades and removes a user and their uploads.
func (s *Server) AdminDeleteUserHandler(c *fiber.Ctx) error {
	idStr := c.Params("id")
	userId, err := parseUUID(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid user ID"})
	}

	ctx := context.Background()
	// Protect against self-deletion by requesting admin
	adminUser := c.Locals("adminUser").(db.User)
	if formatUUID(adminUser.ID) == idStr {
		return c.Status(400).JSON(fiber.Map{"message": "Cannot delete your own administrator account"})
	}

	if err := s.Repository.GetORM().WithContext(ctx).Where("id = ?", userId).Delete(&db.User{}).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to delete user"})
	}

	return c.JSON(fiber.Map{"message": "User deleted successfully"})
}

// AdminFlushCacheHandler clears Valkey/Redis cache.
func (s *Server) AdminFlushCacheHandler(c *fiber.Ctx) error {
	if s.Cache == nil {
		return c.Status(400).JSON(fiber.Map{"message": "Redis cache is not configured"})
	}
	if err := s.Cache.FlushDB(c.UserContext()).Err(); err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to flush cache: " + err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Cache flushed successfully"})
}

// AdminTriggerRefreshHandler runs the 28-day auto-refresh cycle on demand.
func (s *Server) AdminTriggerRefreshHandler(c *fiber.Ctx) error {
	days := 28
	if qDays := c.Query("days"); qDays != "" {
		if d, err := strconv.Atoi(qDays); err == nil && d > 0 {
			days = d
		}
	}
	limit := 25
	if qLimit := c.Query("limit"); qLimit != "" {
		if l, err := strconv.Atoi(qLimit); err == nil && l > 0 {
			limit = l
		}
	}

	olderThan := time.Duration(days) * 24 * time.Hour
	log.Printf("[Admin] Manual re-upload cycle triggered (older than %d days, limit %d)\n", days, limit)
	refreshed, err := workers.RunAutoRefreshCycle(c.UserContext(), s.Repository, s.Cache, s.Kafka, olderThan, limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"message":   "Error running refresh cycle: " + err.Error(),
			"refreshed": refreshed,
		})
	}

	return c.JSON(fiber.Map{
		"message":   fmt.Sprintf("Auto-refresh cycle completed. %d video(s) processed.", refreshed),
		"refreshed": refreshed,
		"days":      days,
	})
}

func (s *Server) indexVideoInOpenSearch(ctx context.Context, v db.Video) {
	if s.OpenSearch == nil || !s.OpenSearch.Enabled() {
		return
	}
	doc := opensearch.VideoDocument{
		ID:           formatUUID(v.ID),
		FileID:       v.FileID,
		Title:        v.Title,
		Description:  v.Description.String,
		Category:     v.Category.String,
		Tags:         v.Tags,
		ThumbnailUrl: v.ThumbnailUrl.String,
		Views:        v.Views.Int32,
		IsShort:      v.IsShort.Bool,
		IsAdult:      v.IsAdult.Bool,
		Visibility:   string(v.Visibility.VideoVisibility),
		UploaderID:   formatUUID(v.UploaderID),
		CreatedAt:    v.CreatedAt.Time,
	}
	_ = s.OpenSearch.IndexVideo(ctx, doc)
}

