package http

import (
	"booking-system-api/internal/middleware"
	"booking-system-api/internal/service"
	"booking-system-api/internal/util"

	"github.com/gofiber/fiber/v2"
)

type RoomKeeperHandler struct {
	svc *service.RoomKeeperService
}

func NewRoomKeeperHandler(svc *service.RoomKeeperService) *RoomKeeperHandler {
	return &RoomKeeperHandler{svc: svc}
}

func (h *RoomKeeperHandler) Register(r fiber.Router) {
	auth := middleware.Auth()
	admin := middleware.RequireRole("ADMIN")

	g := r.Group("/room-keepers", auth)
	g.Get("", h.List)
	g.Get("/:room_keeper_id", h.GetByID)
	g.Post("", admin, h.Create)
	g.Put("/:room_keeper_id", admin, h.Update)
	g.Patch("/:room_keeper_id/toggle-active", admin, h.ToggleActive)
}

func (h *RoomKeeperHandler) List(c *fiber.Ctx) error {
	page := queryInt(c, "page", 1)
	limit := queryInt(c, "limit", 20)
	data, total, err := h.svc.List(c.Context(), page, limit, nil)
	if err != nil {
		return err
	}
	return util.Paginated(c, "Room keepers retrieved", data, total, page, limit)
}

func (h *RoomKeeperHandler) GetByID(c *fiber.Ctx) error {
	id, err := parseID(c, "room_keeper_id")
	if err != nil {
		return err
	}
	data, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		return err
	}
	return util.OK(c, "Room keeper retrieved", data)
}

func (h *RoomKeeperHandler) Create(c *fiber.Ctx) error {
	var req service.CreateRoomKeeperRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.Create(c.Context(), req)
	if err != nil {
		return err
	}
	return util.Created(c, "Room keeper created", data)
}

func (h *RoomKeeperHandler) Update(c *fiber.Ctx) error {
	id, err := parseID(c, "room_keeper_id")
	if err != nil {
		return err
	}
	var req service.UpdateRoomKeeperRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.Update(c.Context(), id, req)
	if err != nil {
		return err
	}
	return util.OK(c, "Room keeper updated", data)
}

func (h *RoomKeeperHandler) ToggleActive(c *fiber.Ctx) error {
	id, err := parseID(c, "room_keeper_id")
	if err != nil {
		return err
	}
	data, err := h.svc.ToggleActive(c.Context(), id)
	if err != nil {
		return err
	}
	return util.OK(c, "Room keeper status toggled", data)
}
