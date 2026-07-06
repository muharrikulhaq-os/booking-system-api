package http

import (
	"booking-system-api/internal/middleware"
	"booking-system-api/internal/service"
	"booking-system-api/internal/util"
	"github.com/gofiber/fiber/v2"
	"strconv"
)

type NotificationHandler struct {
	svc *service.NotificationService
}

func NewNotificationHandler(svc *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

func (h *NotificationHandler) Register(r fiber.Router) {
	auth := middleware.Auth()
	g := r.Group("/users/me/notifications", auth)

	g.Get("/", h.GetMyNotifications)
	g.Patch("/:id/read", h.MarkAsRead)
	g.Patch("/read-all", h.MarkAllAsRead)
}

func (h *NotificationHandler) GetMyNotifications(c *fiber.Ctx) error {
	userID := c.Locals("userId").(int)
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	data, err := h.svc.GetMyNotifications(c.Context(), int32(userID), page, limit)
	if err != nil {
		return err
	}
	return util.OK(c, "Notifications retrieved", data)
}

func (h *NotificationHandler) MarkAsRead(c *fiber.Ctx) error {
	userID := c.Locals("userId").(int)
	notifID, _ := strconv.Atoi(c.Params("id"))

	err := h.svc.MarkAsRead(c.Context(), int32(notifID), int32(userID))
	if err != nil {
		return err
	}
	return util.OK(c, "Notification marked as read", nil)
}

func (h *NotificationHandler) MarkAllAsRead(c *fiber.Ctx) error {
	userID := c.Locals("userId").(int)

	err := h.svc.MarkAllAsRead(c.Context(), int32(userID))
	if err != nil {
		return err
	}
	return util.OK(c, "All notifications marked as read", nil)
}
