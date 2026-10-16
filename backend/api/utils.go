package api

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgtype"
)

// formatUUID converts a pgtype.UUID to a standard RFC 4122 string representation.
func formatUUID(id pgtype.UUID) string {
	b := id.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// parseUUID parses a standard UUID string into a pgtype.UUID.
func parseUUID(idStr string) (pgtype.UUID, error) {
	var u pgtype.UUID
	err := u.Scan(idStr)
	return u, err
}

// pageLimit extracts page and limit from query parameters with validation and sane defaults.
func pageLimit(c *fiber.Ctx, defaultLimit, maxLimit int) (int, int) {
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(c.Query("limit", strconv.Itoa(defaultLimit)))
	if err != nil || limit < 1 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return page, limit
}
