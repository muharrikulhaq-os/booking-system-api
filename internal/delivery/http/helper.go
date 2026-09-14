package http

import (
	"strconv"
	"strings"

	"booking-system-api/internal/middleware"
	"booking-system-api/internal/service"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

var validate = validator.New()

func parseID(c *fiber.Ctx, param string) (int32, error) {
	id, err := strconv.Atoi(c.Params(param))
	if err != nil {
		return 0, fiber.NewError(fiber.StatusBadRequest, "invalid id parameter")
	}
	return int32(id), nil
}

func bindAndValidate(c *fiber.Ctx, dst any) error {
	if err := c.BodyParser(dst); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if err := validate.Struct(dst); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
	}
	return nil
}

func queryInt(c *fiber.Ctx, key string, def int) int {
	v := c.Query(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func queryString(c *fiber.Ctx, key string) *string {
	v := c.Query(key)
	if v == "" {
		return nil
	}
	return &v
}

func queryInt32(c *fiber.Ctx, key string) *int32 {
	v := c.Query(key)
	if v == "" {
		return nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return nil
	}
	n32 := int32(n)
	return &n32
}

// clientIP resolves the real originating client address. The API now only
// ever gets reached through the IIS/ARR reverse proxy (see the :8443
// firewall issue this was built to route around) - c.IP() alone would
// return the proxy's own loopback address (127.0.0.1) for every single
// request, which is exactly what audit_logs.ipAddress showed before this
// fix. ARR sets X-Forwarded-For, so prefer that (leftmost entry = original
// client, per the de-facto X-Forwarded-For convention) and fall back to
// c.IP() for direct/local requests where the header is absent.
func clientIP(c *fiber.Ctx) string {
	if xff := c.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	return c.IP()
}

// auditActor bundles who's making this request and where from, for services
// that write audit_logs entries - see service.AuditActor.
func auditActor(c *fiber.Ctx) service.AuditActor {
	return service.AuditActor{
		UserID:    int32(middleware.GetUserID(c)),
		IP:        clientIP(c),
		UserAgent: c.Get("User-Agent"),
	}
}
