package http

import (
	"booking-system-api/internal/middleware"
	"booking-system-api/internal/service"
	"booking-system-api/internal/util"

	"github.com/gofiber/fiber/v2"
)

type VehicleHandler struct {
	svc       *service.VehicleService
	attachSvc *service.AttachmentService
}

func NewVehicleHandler(svc *service.VehicleService, attachSvc *service.AttachmentService) *VehicleHandler {
	return &VehicleHandler{svc: svc, attachSvc: attachSvc}
}

func (h *VehicleHandler) Register(r fiber.Router) {
	auth := middleware.Auth()
	admin := middleware.RequireRole("ADMIN")

	g := r.Group("/vehicles", auth)
	g.Get("", h.List)
	g.Get("/categories", h.ListCategories)
	g.Get("/:id", h.GetByID)
	g.Post("", admin, h.Create)
	g.Post("/categories", admin, h.CreateCategory)
	g.Put("/:id", admin, h.Update)
	g.Patch("/:id/status", admin, h.UpdateStatus)
	g.Patch("/:id/photo", admin, h.UpdatePhoto)
	g.Delete("/:id", admin, h.Delete)
	g.Delete("/categories/:id", admin, h.DeleteCategory)
	g.Get("/:id/maintenance-status", h.GetMaintenanceStatus)
	g.Get("/:id/attachments", h.ListAttachments)
	g.Post("/:id/attachments", admin, h.UploadAttachment)
}

func (h *VehicleHandler) List(c *fiber.Ctx) error {
	page := queryInt(c, "page", 1)
	limit := queryInt(c, "limit", 20)
	data, total, err := h.svc.List(c.Context(), page, limit, queryString(c, "search"), queryInt32(c, "categoryId"), queryString(c, "status"))
	if err != nil {
		return err
	}
	return util.Paginated(c, "Vehicles retrieved", data, total, page, limit)
}

func (h *VehicleHandler) GetByID(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	data, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		return err
	}
	return util.OK(c, "Vehicle retrieved", data)
}

func (h *VehicleHandler) Create(c *fiber.Ctx) error {
	var req service.CreateVehicleRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.Create(c.Context(), req)
	if err != nil {
		return err
	}
	return util.Created(c, "Vehicle created", data)
}

func (h *VehicleHandler) Update(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var req service.UpdateVehicleRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.Update(c.Context(), id, req, int32(middleware.GetUserID(c)))
	if err != nil {
		return err
	}
	return util.OK(c, "Vehicle updated", data)
}

func (h *VehicleHandler) GetMaintenanceStatus(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	data, err := h.svc.GetMaintenanceStatus(c.Context(), id)
	if err != nil {
		return err
	}
	return util.OK(c, "Vehicle maintenance status retrieved", data)
}

func (h *VehicleHandler) UpdateStatus(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var req service.UpdateStatusRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.UpdateStatus(c.Context(), id, req.Status)
	if err != nil {
		return err
	}
	return util.OK(c, "Vehicle status updated", data)
}

func (h *VehicleHandler) UpdatePhoto(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	fh, err := c.FormFile("photo")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "photo file is required")
	}
	savedPath, saveErr := util.SaveUploadedFile(fh, "vehicle")
	if saveErr != nil {
		return fiber.NewError(fiber.StatusBadRequest, saveErr.Error())
	}
	data, err := h.svc.UpdatePhoto(c.Context(), id, "/files/"+savedPath)
	if err != nil {
		util.DeleteUploadedFile(savedPath)
		return err
	}
	return util.OK(c, "Vehicle photo updated", data)
}

func (h *VehicleHandler) Delete(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := h.svc.Delete(c.Context(), id); err != nil {
		return err
	}
	return util.OK(c, "Vehicle deleted", nil)
}

func (h *VehicleHandler) ListCategories(c *fiber.Ctx) error {
	data, err := h.svc.ListCategories(c.Context())
	if err != nil {
		return err
	}
	return util.OK(c, "Categories retrieved", data)
}

func (h *VehicleHandler) CreateCategory(c *fiber.Ctx) error {
	var body struct {
		Name string `json:"name" validate:"required"`
	}
	if err := bindAndValidate(c, &body); err != nil {
		return err
	}
	data, err := h.svc.CreateCategory(c.Context(), body.Name)
	if err != nil {
		return err
	}
	return util.Created(c, "Category created", data)
}

func (h *VehicleHandler) DeleteCategory(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := h.svc.DeleteCategory(c.Context(), id); err != nil {
		return err
	}
	return util.OK(c, "Category deleted", nil)
}

func (h *VehicleHandler) ListAttachments(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	data, err := h.attachSvc.ListByVehicle(c.Context(), id)
	if err != nil {
		return err
	}
	return util.OK(c, "Attachments retrieved", data)
}

func (h *VehicleHandler) UploadAttachment(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	fh, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "file is required")
	}
	desc := c.FormValue("description")
	uploaderID := int32(middleware.GetUserID(c))
	data, err := h.attachSvc.UploadForVehicle(c.Context(), id, uploaderID, fh, desc)
	if err != nil {
		return err
	}
	return util.Created(c, "Attachment uploaded", data)
}
