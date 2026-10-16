package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgtype"
)

// requireAuth extracts the authenticated user's UUID from fiber context locals.
// If the user is unauthenticated or the user ID is invalid, it writes a 401 JSON response
// and returns (pgtype.UUID{}, false). Callers should immediately return nil if ok is false.
func (s *Server) requireAuth(c *fiber.Ctx) (pgtype.UUID, bool) {
	userIdLocal := c.Locals("userId")
	if userIdLocal == nil {
		_ = c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Unauthorized",
			"error":   "Unauthorized",
		})
		return pgtype.UUID{}, false
	}

	userIdStr, ok := userIdLocal.(string)
	if !ok || userIdStr == "" {
		_ = c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Unauthorized",
			"error":   "Unauthorized",
		})
		return pgtype.UUID{}, false
	}

	uid, err := parseUUID(userIdStr)
	if err != nil {
		_ = c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Invalid user ID",
			"error":   "Invalid user ID",
		})
		return pgtype.UUID{}, false
	}

	return uid, true
}

// getOptionalUserID extracts the user's UUID if present and valid in fiber context locals.
// Returns (zeroUUID, false) if the user is unauthenticated, guest, or the ID is malformed.
func (s *Server) getOptionalUserID(c *fiber.Ctx) (pgtype.UUID, bool) {
	userIdLocal := c.Locals("userId")
	if userIdLocal == nil {
		return pgtype.UUID{}, false
	}

	userIdStr, ok := userIdLocal.(string)
	if !ok || userIdStr == "" {
		return pgtype.UUID{}, false
	}

	uid, err := parseUUID(userIdStr)
	if err != nil {
		return pgtype.UUID{}, false
	}

	return uid, true
}
