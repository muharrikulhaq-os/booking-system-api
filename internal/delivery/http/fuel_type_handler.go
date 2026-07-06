package http

import (
	"booking-system-api/internal/middleware"
	"booking-system-api/internal/service"
	"booking-system-api/internal/util"
	"github.com/gofiber/fiber/v2"
	"strconv"
)

type FuelTypeHandler struct {
	svc *service.FuelTypeService
}

func NewFuelTypeHandler(svc *service.FuelTypeService) *FuelTypeHandler {
	return &FuelTypeHandler{svc: svc}
}

func (h *FuelTypeHandler) Register(r fiber.Router) {
	auth := middleware.Auth()
	admin := middleware.RequireRole("ADMIN")

	g := r.Group("/fuel-types", auth)
	g.Get("", h.List)
	g.Post("", admin, h.Create)
	g.Put("/:id", admin, h.Update)
	g.Delete("/:id", admin, h.Delete)
}

func (h *FuelTypeHandler) List(c *fiber.Ctx) error {
	data, err := h.svc.List(c.Context())
	if err != nil {
		return err
	}
	return util.OK(c, "Fuel types retrieved", data)
}

func (h *FuelTypeHandler) Create(c *fiber.Ctx) error {
	var req service.FuelTypeRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.Create(c.Context(), req)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Fuel type created",
		"data":    data,
	})
}

func (h *FuelTypeHandler) Update(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var req service.FuelTypeRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.Update(c.Context(), int32(id), req)
	if err != nil {
		return err
	}
	return util.OK(c, "Fuel type updated", data)
}

func (h *FuelTypeHandler) Delete(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	if err := h.svc.Delete(c.Context(), int32(id)); err != nil {
		return err
	}
	return util.OK(c, "Fuel type deleted", nil)
}
