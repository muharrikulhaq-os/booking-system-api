package middleware

import (
	"strings"

	"booking-system-api/internal/util"

	"github.com/gofiber/fiber/v2"
)

const (
	LocalUserID   = "userID"
	LocalUserRole = "userRole"
)

func Auth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		token := strings.TrimPrefix(header, "Bearer ")
		
		if token == "" {
			token = c.Query("token")
		}

		if token == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "missing authorization token")
		}

		claims, err := util.ParseToken(token)
		if err != nil || claims.Type != "access" {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired token")
		}

		c.Locals(LocalUserID, claims.UserID)
		c.Locals(LocalUserRole, claims.Role)
		return c.Next()
	}
}

func RequireRole(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, _ := c.Locals(LocalUserRole).(string)
		for _, r := range roles {
			if r == role {
				return c.Next()
			}
		}
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}
}

func GetUserID(c *fiber.Ctx) int {
	id, _ := c.Locals(LocalUserID).(int)
	return id
}

func GetUserRole(c *fiber.Ctx) string {
	r, _ := c.Locals(LocalUserRole).(string)
	return r
}
