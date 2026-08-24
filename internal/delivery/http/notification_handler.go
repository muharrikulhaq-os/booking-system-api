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
	g.Get("/unread-count", h.UnreadCount)
	g.Patch("/:id/read", h.MarkAsRead)
	g.Patch("/read-all", h.MarkAllAsRead)

	// FCM device token (push notification saat app ditutup).
	d := r.Group("/users/me/device-tokens", auth)
	d.Post("/", h.SaveDeviceToken)
	d.Delete("/", h.DeleteDeviceToken)
}

type deviceTokenRequest struct {
	Token    string `json:"token" validate:"required"`
	Platform string `json:"platform"`
}

func (h *NotificationHandler) SaveDeviceToken(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	var req deviceTokenRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.svc.SaveDeviceToken(c.Context(), int32(userID), req.Token, req.Platform); err != nil {
		return err
	}
	return util.OK(c, "Device token saved", nil)
}

func (h *NotificationHandler) DeleteDeviceToken(c *fiber.Ctx) error {
	var req deviceTokenRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.svc.RemoveDeviceToken(c.Context(), req.Token); err != nil {
		return err
	}
	return util.OK(c, "Device token removed", nil)
}

func (h *NotificationHandler) UnreadCount(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	count, err := h.svc.UnreadCount(c.Context(), int32(userID))
	if err != nil {
		return err
	}
	return util.OK(c, "Unread count retrieved", fiber.Map{"count": count})
}

func (h *NotificationHandler) GetMyNotifications(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	data, total, err := h.svc.GetMyNotifications(c.Context(), int32(userID), page, limit)
	if err != nil {
		return err
	}
	return util.Paginated(c, "Notifications retrieved", data, total, page, limit)
}

func (h *NotificationHandler) MarkAsRead(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	notifID, _ := strconv.Atoi(c.Params("id"))

	err := h.svc.MarkAsRead(c.Context(), int32(notifID), int32(userID))
	if err != nil {
		return err
	}
	return util.OK(c, "Notification marked as read", nil)
}

func (h *NotificationHandler) MarkAllAsRead(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	err := h.svc.MarkAllAsRead(c.Context(), int32(userID))
	if err != nil {
		return err
	}
	return util.OK(c, "All notifications marked as read", nil)
}
