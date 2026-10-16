package middleware

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// isPrivateOrLocalIP checks if an IP belongs to private/loopback/unspecified subnets (trusted proxies).
func isPrivateOrLocalIP(ipStr string) bool {
	host := ipStr
	if h, _, err := net.SplitHostPort(ipStr); err == nil {
		host = h
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()
}

// clientIP extracts the real client IP, only trusting forwarding headers if the immediate peer is a private proxy/gateway.
func clientIP(c *fiber.Ctx) string {
	directIP := c.IP()

	// Only trust forwarding headers if the immediate peer connection is from a trusted proxy / gateway
	if isPrivateOrLocalIP(directIP) {
		if cf := strings.TrimSpace(c.Get("CF-Connecting-IP")); cf != "" && net.ParseIP(cf) != nil {
			return cf
		}
		if xff := strings.TrimSpace(c.Get("X-Forwarded-For")); xff != "" {
			parts := strings.Split(xff, ",")
			first := strings.TrimSpace(parts[0])
			if net.ParseIP(first) != nil {
				return first
			}
		}
		if xri := strings.TrimSpace(c.Get("X-Real-IP")); xri != "" && net.ParseIP(xri) != nil {
			return xri
		}
	}

	return directIP
}

// rateLimitResponse formats a standardized 429 response with Retry-After metadata.
func rateLimitResponse(c *fiber.Ctx, defaultMessage string, expiration time.Duration) error {
	retryAfter := c.Get(fiber.HeaderRetryAfter)
	secs, err := strconv.Atoi(retryAfter)
	if err != nil || secs <= 0 {
		secs = int(expiration.Seconds())
	}
	c.Set(fiber.HeaderRetryAfter, strconv.Itoa(secs))
	return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
		"error":      "rate_limit_exceeded",
		"message":    fmt.Sprintf("%s Please try again in %d seconds.", defaultMessage, secs),
		"retryAfter": secs,
	})
}

// RegisterLimiter prevents mass bot account creation (5 accounts / 10 min per IP).
func RegisterLimiter() fiber.Handler {
	expiration := 10 * time.Minute
	return limiter.New(limiter.Config{
		Max:               5,
		Expiration:        expiration,
		LimiterMiddleware: limiter.SlidingWindow{},
		KeyGenerator: func(c *fiber.Ctx) string {
			return "reg:" + clientIP(c)
		},
		LimitReached: func(c *fiber.Ctx) error {
			return rateLimitResponse(c, "Too many registration attempts.", expiration)
		},
	})
}

// LoginLimiter prevents credential stuffing / brute-force (10 failed attempts / 5 min per IP).
func LoginLimiter() fiber.Handler {
	expiration := 5 * time.Minute
	return limiter.New(limiter.Config{
		Max:                    10,
		Expiration:             expiration,
		LimiterMiddleware:      limiter.SlidingWindow{},
		SkipSuccessfulRequests: true, // Only failed attempts count toward lockout!
		KeyGenerator: func(c *fiber.Ctx) string {
			return "login:" + clientIP(c)
		},
		LimitReached: func(c *fiber.Ctx) error {
			return rateLimitResponse(c, "Too many failed login attempts.", expiration)
		},
	})
}

// AuthLimiter provides backward compatibility, aliasing to LoginLimiter.
func AuthLimiter() fiber.Handler {
	return LoginLimiter()
}

// CheckAvailabilityLimiter prevents username/email enumeration (30 checks / 1 min per IP).
func CheckAvailabilityLimiter() fiber.Handler {
	expiration := 1 * time.Minute
	return limiter.New(limiter.Config{
		Max:               30,
		Expiration:        expiration,
		LimiterMiddleware: limiter.SlidingWindow{},
		KeyGenerator: func(c *fiber.Ctx) string {
			return "check:" + clientIP(c)
		},
		LimitReached: func(c *fiber.Ctx) error {
			return rateLimitResponse(c, "Too many lookup requests.", expiration)
		},
	})
}

// RemoteUploadLimiter prevents abuse of external URL downloads (15 jobs / min per user/IP).
func RemoteUploadLimiter() fiber.Handler {
	expiration := 1 * time.Minute
	return limiter.New(limiter.Config{
		Max:               15,
		Expiration:        expiration,
		LimiterMiddleware: limiter.SlidingWindow{},
		KeyGenerator: func(c *fiber.Ctx) string {
			if uid, ok := c.Locals("userId").(string); ok && uid != "" {
				return "upload:" + uid
			}
			return "upload:" + clientIP(c)
		},
		LimitReached: func(c *fiber.Ctx) error {
			return rateLimitResponse(c, "Remote upload rate limit exceeded.", expiration)
		},
	})
}

// CommentsLimiter prevents automated comment flooding (30 comments / min per user/IP).
func CommentsLimiter() fiber.Handler {
	expiration := 1 * time.Minute
	return limiter.New(limiter.Config{
		Max:               30,
		Expiration:        expiration,
		LimiterMiddleware: limiter.SlidingWindow{},
		KeyGenerator: func(c *fiber.Ctx) string {
			if uid, ok := c.Locals("userId").(string); ok && uid != "" {
				return "comment:" + uid
			}
			return "comment:" + clientIP(c)
		},
		LimitReached: func(c *fiber.Ctx) error {
			return rateLimitResponse(c, "You are posting comments too quickly.", expiration)
		},
	})
}

// SearchLimiter protects search and autocomplete from scraper abuse and query flooding (60 req / min per IP).
func SearchLimiter() fiber.Handler {
	expiration := 1 * time.Minute
	return limiter.New(limiter.Config{
		Max:               60,
		Expiration:        expiration,
		LimiterMiddleware: limiter.SlidingWindow{},
		KeyGenerator: func(c *fiber.Ctx) string {
			return "search:" + clientIP(c)
		},
		LimitReached: func(c *fiber.Ctx) error {
			return rateLimitResponse(c, "Search query limit exceeded.", expiration)
		},
	})
}
