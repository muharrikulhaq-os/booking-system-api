package http

import (
	"time"

	"booking-system-api/internal/middleware"
	"booking-system-api/internal/service"
	"booking-system-api/internal/util"

	"github.com/gofiber/fiber/v2"
)

type DriverHandler struct {
	svc *service.DriverService
}

func NewDriverHandler(svc *service.DriverService) *DriverHandler {
	return &DriverHandler{svc: svc}
}

func (h *DriverHandler) Register(r fiber.Router) {
	auth := middleware.Auth()
	admin := middleware.RequireRole("ADMIN")

	g := r.Group("/drivers", auth)
	g.Get("", h.List)
	g.Get("/available", h.AvailableDrivers)
	g.Get("/:driver_id", h.GetByID)
	g.Post("", admin, h.Create)
	g.Put("/:driver_id", admin, h.Update)
	g.Patch("/:driver_id/toggle-active", admin, h.ToggleActive)
	g.Post("/:driver_id/assign", admin, h.Assign)
	g.Patch("/:driver_id/release", admin, h.Release)
	g.Get("/:driver_id/assignments", admin, h.Assignments)
	g.Patch("/:driver_id/fixed-vehicle", admin, h.SetFixedVehicle)
}

func (h *DriverHandler) List(c *fiber.Ctx) error {
	page := queryInt(c, "page", 1)
	limit := queryInt(c, "limit", 20)
	data, total, err := h.svc.List(c.Context(), page, limit, queryString(c, "search"), nil)
	if err != nil {
		return err
	}
	return util.Paginated(c, "Drivers retrieved", data, total, page, limit)
}

func (h *DriverHandler) GetByID(c *fiber.Ctx) error {
	id, err := parseID(c, "driver_id")
	if err != nil {
		return err
	}
	data, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		return err
	}
	return util.OK(c, "Driver retrieved", data)
}

func (h *DriverHandler) Create(c *fiber.Ctx) error {
	var req service.CreateDriverRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.Create(c.Context(), req)
	if err != nil {
		return err
	}
	return util.Created(c, "Driver created", data)
}

func (h *DriverHandler) Update(c *fiber.Ctx) error {
	id, err := parseID(c, "driver_id")
	if err != nil {
		return err
	}
	var req service.UpdateDriverRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.Update(c.Context(), id, req)
	if err != nil {
		return err
	}
	return util.OK(c, "Driver updated", data)
}

func (h *DriverHandler) ToggleActive(c *fiber.Ctx) error {
	id, err := parseID(c, "driver_id")
	if err != nil {
		return err
	}
	data, err := h.svc.ToggleActive(c.Context(), id)
	if err != nil {
		return err
	}
	return util.OK(c, "Driver status toggled", data)
}

func (h *DriverHandler) Assign(c *fiber.Ctx) error {
	id, err := parseID(c, "driver_id")
	if err != nil {
		return err
	}
	var req service.AssignDriverRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.Assign(c.Context(), id, req)
	if err != nil {
		return err
	}
	return util.OK(c, "Driver assigned", data)
}

func (h *DriverHandler) Release(c *fiber.Ctx) error {
	id, err := parseID(c, "driver_id")
	if err != nil {
		return err
	}
	data, err := h.svc.Release(c.Context(), id)
	if err != nil {
		return err
	}
	return util.OK(c, "Driver released", data)
}

func (h *DriverHandler) Assignments(c *fiber.Ctx) error {
	id, err := parseID(c, "driver_id")
	if err != nil {
		return err
	}
	data, err := h.svc.GetAssignments(c.Context(), id)
	if err != nil {
		return err
	}
	return util.OK(c, "Assignment history retrieved", data)
}

func (h *DriverHandler) SetFixedVehicle(c *fiber.Ctx) error {
	id, err := parseID(c, "driver_id")
	if err != nil {
		return err
	}
	var body struct {
		VehicleID *int32 `json:"vehicleId"`
	}
	if err := bindAndValidate(c, &body); err != nil {
		return err
	}
	data, err := h.svc.SetFixedVehicle(c.Context(), id, body.VehicleID)
	if err != nil {
		return err
	}
	return util.OK(c, "Driver fixed vehicle updated", data)
}

// attach the middleware reference so it can be used in handler
var _ = middleware.GetUserID

func (h *DriverHandler) AvailableDrivers(c *fiber.Ctx) error {
	s := c.Query("startDate")
	e := c.Query("endDate")
	if s == "" || e == "" {
		return fiber.NewError(fiber.StatusBadRequest, "startDate and endDate are required")
	}
	start, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid startDate format")
	}
	end, err := time.Parse(time.RFC3339, e)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid endDate format")
	}
	data, err := h.svc.GetAvailableDrivers(c.Context(), start, end)
	if err != nil {
		return err
	}
	return util.OK(c, "Available drivers retrieved", data)
}
