package api

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgtype"
	"gorm.io/gorm/clause"
)

const (
	SwitchMaintenanceMode    = "maintenance_mode"
	SwitchFreezeUploads      = "freeze_uploads"
	SwitchFreezeRegistration = "freeze_registration"
	SwitchFreezeComments     = "freeze_comments"
)

var AllKillSwitches = []string{
	SwitchMaintenanceMode,
	SwitchFreezeUploads,
	SwitchFreezeRegistration,
	SwitchFreezeComments,
}

// IsKillSwitchActive checks if a platform kill switch is currently engaged.
// Checks Valkey/Redis first for sub-millisecond response, falling back to PostgreSQL.
func (s *Server) IsKillSwitchActive(ctx context.Context, key string) bool {
	if s == nil {
		return false
	}

	valkeyKey := "system:flag:" + key

	// 1. Check Valkey/Redis cache
	if s.Cache != nil {
		val, err := s.Cache.Get(ctx, valkeyKey).Result()
		if err == nil {
			return val == "true" || val == "1"
		}
	}

	// 2. Fall back to PostgreSQL database
	if s.Repository != nil {
		var setting db.SystemSetting
		err := s.Repository.GetORM().WithContext(ctx).Where("key = ?", key).First(&setting).Error
		if err == nil {
			active := setting.Value == "true" || setting.Value == "1"
			// Warm Valkey cache
			if s.Cache != nil {
				_ = s.Cache.Set(ctx, valkeyKey, setting.Value, 24*time.Hour).Err()
			}
			return active
		}
	}

	return false
}

// SetKillSwitch updates the persistent and cached status of a platform kill switch.
func (s *Server) SetKillSwitch(ctx context.Context, key string, enabled bool, updatedBy pgtype.UUID) error {
	valStr := "false"
	if enabled {
		valStr = "true"
	}

	// 1. Persist to PostgreSQL
	if s.Repository != nil {
		setting := db.SystemSetting{
			Key:       key,
			Value:     valStr,
			UpdatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
			UpdatedBy: updatedBy,
		}

		err := s.Repository.GetORM().WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at", "updated_by"}),
		}).Create(&setting).Error
		if err != nil {
			return err
		}
	}

	// 2. Synchronize to Valkey/Redis immediately
	if s.Cache != nil {
		valkeyKey := "system:flag:" + key
		_ = s.Cache.Set(ctx, valkeyKey, valStr, 0).Err()
	}

	log.Printf("[Superpower] Admin kill switch %q set to %v by %v\n", key, enabled, formatUUID(updatedBy))
	return nil
}

// EnforceKillSwitch returns a middleware that blocks non-admin requests when the specified switch is active.
// Administrators always bypass kill switch restrictions.
func (s *Server) EnforceKillSwitch(key string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Administrators always bypass kill switches
		if isAdmin, ok := c.Locals("isAdmin").(bool); ok && isAdmin {
			return c.Next()
		}

		if s.IsKillSwitchActive(c.UserContext(), key) {
			messages := map[string]string{
				SwitchMaintenanceMode:    "Platform is currently undergoing scheduled maintenance. Please check back shortly.",
				SwitchFreezeUploads:      "Video uploads are temporarily disabled by platform administration.",
				SwitchFreezeRegistration: "New user registrations are temporarily closed by platform administration.",
				SwitchFreezeComments:     "Commenting is temporarily paused by platform administration.",
			}
			msg := messages[key]
			if msg == "" {
				msg = "This action is temporarily suspended by platform administration."
			}

			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error":   "kill_switch_active",
				"flag":    key,
				"message": msg,
			})
		}

		return c.Next()
	}
}

// MaintenanceModeMiddleware halts mutating requests across the platform when maintenance mode is active.
func (s *Server) MaintenanceModeMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		method := c.Method()
		// Safe/read-only requests are allowed so visitors can see the platform status
		if method == fiber.MethodGet || method == fiber.MethodHead || method == fiber.MethodOptions {
			return c.Next()
		}

		path := c.Path()
		// Allow auth endpoints so administrators can authenticate and resolve maintenance
		if strings.HasPrefix(path, "/api/auth/login") ||
			strings.HasPrefix(path, "/api/auth/logout") ||
			strings.HasPrefix(path, "/api/health") ||
			strings.HasPrefix(path, "/api/system/") ||
			strings.HasPrefix(path, "/api/admin/") {
			return c.Next()
		}

		// Admins bypass maintenance mode
		if isAdmin, ok := c.Locals("isAdmin").(bool); ok && isAdmin {
			return c.Next()
		}

		if s.IsKillSwitchActive(c.UserContext(), SwitchMaintenanceMode) {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error":   "maintenance_mode_active",
				"flag":    SwitchMaintenanceMode,
				"message": "Platform is currently under emergency maintenance. State modifications are temporarily restricted.",
			})
		}

		return c.Next()
	}
}

// AdminGetKillSwitchesHandler returns the current operational status of all platform kill switches.
func (s *Server) AdminGetKillSwitchesHandler(c *fiber.Ctx) error {
	ctx := c.UserContext()
	result := make(map[string]fiber.Map)

	var settings []db.SystemSetting
	if s.Repository != nil {
		_ = s.Repository.GetORM().WithContext(ctx).Find(&settings).Error
	}

	settingMap := make(map[string]db.SystemSetting)
	for _, st := range settings {
		settingMap[st.Key] = st
	}

	for _, key := range AllKillSwitches {
		st, exists := settingMap[key]
		enabled := false
		var updatedAt *time.Time
		var updatedBy string

		if exists {
			enabled = st.Value == "true" || st.Value == "1"
			if st.UpdatedAt.Valid {
				t := st.UpdatedAt.Time
				updatedAt = &t
			}
			updatedBy = formatUUID(st.UpdatedBy)
		} else {
			// Verify if active in Valkey cache
			enabled = s.IsKillSwitchActive(ctx, key)
		}

		result[key] = fiber.Map{
			"key":       key,
			"enabled":   enabled,
			"updatedAt": updatedAt,
			"updatedBy": updatedBy,
		}
	}

	return c.JSON(fiber.Map{
		"switches": result,
	})
}

// AdminToggleKillSwitchRequest represents the payload to engage or disengage a kill switch.
type AdminToggleKillSwitchRequest struct {
	Key     string `json:"key"`
	Enabled bool   `json:"enabled"`
}

// AdminToggleKillSwitchHandler flips a platform kill switch on or off.
func (s *Server) AdminToggleKillSwitchHandler(c *fiber.Ctx) error {
	var req AdminToggleKillSwitchRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request payload"})
	}

	req.Key = strings.TrimSpace(req.Key)
	isValid := false
	for _, k := range AllKillSwitches {
		if k == req.Key {
			isValid = true
			break
		}
	}
	if !isValid {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid kill switch identifier"})
	}

	adminUUID, _ := s.requireAuth(c)
	if err := s.SetKillSwitch(c.UserContext(), req.Key, req.Enabled, adminUUID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to update kill switch state: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Kill switch updated successfully.",
		"key":     req.Key,
		"enabled": req.Enabled,
	})
}

// PublicSystemStatusHandler returns the public status of platform feature flags.
func (s *Server) PublicSystemStatusHandler(c *fiber.Ctx) error {
	ctx := c.UserContext()

	return c.JSON(fiber.Map{
		"maintenanceMode":    s.IsKillSwitchActive(ctx, SwitchMaintenanceMode),
		"freezeUploads":      s.IsKillSwitchActive(ctx, SwitchFreezeUploads),
		"freezeRegistration": s.IsKillSwitchActive(ctx, SwitchFreezeRegistration),
		"freezeComments":     s.IsKillSwitchActive(ctx, SwitchFreezeComments),
	})
}
