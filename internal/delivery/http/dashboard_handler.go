package http

import (
	"booking-system-api/internal/middleware"
	"booking-system-api/internal/service"

	"github.com/gofiber/fiber/v2"
)

type DashboardHandler struct {
	svc *service.DashboardService
}

func NewDashboardHandler(svc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

func (h *DashboardHandler) Register(r fiber.Router) {
	auth := middleware.Auth()
	admin := middleware.RequireRole("ADMIN")

	g := r.Group("/dashboard", auth, admin)
	g.Get("/summary", h.GetSummary)
}

func (h *DashboardHandler) GetSummary(c *fiber.Ctx) error {
	summary, err := h.svc.GetSummary(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(summary)
}
