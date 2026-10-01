package http

import (
	"mime/multipart"

	"booking-system-api/internal/middleware"
	"booking-system-api/internal/service"
	"booking-system-api/internal/util"

	"github.com/gofiber/fiber/v2"
)

// ─── Maintenance (vendor/bengkel luar) ───────────────────────────────────────

type MaintenanceHandler struct {
	svc *service.MaintenanceService
}

func NewMaintenanceHandler(svc *service.MaintenanceService) *MaintenanceHandler {
	return &MaintenanceHandler{svc: svc}
}

func (h *MaintenanceHandler) Register(r fiber.Router) {
	g := r.Group("/maintenance", middleware.Auth(), middleware.RequireRole("ADMIN"))
	g.Get("", h.List)
	g.Get("/options", h.Options)
	g.Get("/:id", h.GetByID)
	g.Post("", h.Create)
	g.Put("/:id", h.Update)
	g.Delete("/:id", h.Delete)
	g.Post("/:id/submit", h.Submit)
	g.Post("/:id/schedule", h.Schedule)
	g.Post("/:id/handover", h.Handover)
	g.Post("/:id/return", h.Return)
	g.Post("/:id/cancel", h.Cancel)
	g.Patch("/:id/cost", h.Cost)
	g.Post("/:id/documents", h.UploadDocuments)
	g.Delete("/:id/documents/:docId", h.DeleteDocument)
	g.Get("/:id/pdf/:kind", h.PDF)
}

// action: respons standar + peringatan opsional.
func action(c *fiber.Ctx, status int, msg string, resp service.MaintenanceActionResponse) error {
	body := fiber.Map{"success": true, "message": msg, "data": resp.Data}
	if resp.Warning != "" {
		body["warning"] = resp.Warning
	}
	return c.Status(status).JSON(body)
}

func (h *MaintenanceHandler) List(c *fiber.Ctx) error {
	page := queryInt(c, "page", 1)
	limit := queryInt(c, "limit", 20)
	data, total, err := h.svc.List(c.Context(), page, limit, service.MaintenanceListFilter{
		VehicleID: queryInt32(c, "vehicleId"), VendorID: queryInt32(c, "vendorId"),
		Status: c.Query("status"), Search: c.Query("search"),
		SortBy: c.Query("sortBy"), SortOrder: c.Query("sortOrder"),
	})
	if err != nil {
		return err
	}
	return util.Paginated(c, "Maintenance records retrieved", data, total, page, limit)
}

func (h *MaintenanceHandler) Options(c *fiber.Ctx) error {
	return util.OK(c, "Maintenance options", service.MaintenanceOptions())
}

func (h *MaintenanceHandler) GetByID(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	data, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		return err
	}
	return util.OK(c, "Maintenance record retrieved", data)
}

func (h *MaintenanceHandler) Create(c *fiber.Ctx) error {
	var req service.MaintenancePlanRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	resp, err := h.svc.Create(c.Context(), req, auditActor(c))
	if err != nil {
		return err
	}
	return action(c, fiber.StatusCreated, "Maintenance created", resp)
}

func (h *MaintenanceHandler) Update(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var req service.MaintenancePlanRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	resp, err := h.svc.Update(c.Context(), id, req, auditActor(c))
	if err != nil {
		return err
	}
	return action(c, fiber.StatusOK, "Maintenance updated", resp)
}

func (h *MaintenanceHandler) Delete(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := h.svc.Delete(c.Context(), id, auditActor(c)); err != nil {
		return err
	}
	return util.OK(c, "Maintenance deleted", nil)
}

func (h *MaintenanceHandler) Submit(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	resp, err := h.svc.Submit(c.Context(), id, auditActor(c))
	if err != nil {
		return err
	}
	return action(c, fiber.StatusOK, "Maintenance submitted", resp)
}

func (h *MaintenanceHandler) Schedule(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var req service.ScheduleMaintenanceRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	resp, err := h.svc.Schedule(c.Context(), id, req, auditActor(c))
	if err != nil {
		return err
	}
	return action(c, fiber.StatusOK, "Maintenance scheduled", resp)
}

func (h *MaintenanceHandler) Handover(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var req service.HandoverMaintenanceRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	resp, err := h.svc.Handover(c.Context(), id, req, auditActor(c))
	if err != nil {
		return err
	}
	return action(c, fiber.StatusOK, "Vehicle handed over to vendor", resp)
}

func (h *MaintenanceHandler) Return(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var req service.ReturnMaintenanceRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	resp, err := h.svc.Return(c.Context(), id, req, auditActor(c))
	if err != nil {
		return err
	}
	return action(c, fiber.StatusOK, "Vehicle returned from vendor", resp)
}

func (h *MaintenanceHandler) Cancel(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var req service.CancelMaintenanceRequest
	_ = c.BodyParser(&req)
	resp, err := h.svc.Cancel(c.Context(), id, req, auditActor(c))
	if err != nil {
		return err
	}
	return action(c, fiber.StatusOK, "Maintenance cancelled", resp)
}

func (h *MaintenanceHandler) Cost(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var req service.MaintenanceCostRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.UpdateCost(c.Context(), id, req, auditActor(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Maintenance cost updated", data)
}

func (h *MaintenanceHandler) UploadDocuments(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	form, err := c.MultipartForm()
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "gunakan multipart/form-data")
	}
	files := form.File["files[]"]
	if len(files) == 0 {
		files = form.File["file"]
	}
	kind := c.FormValue("kind")
	data, err := h.svc.UploadDocuments(c.Context(), id, kind, files, auditActor(c))
	if err != nil {
		return err
	}
	return util.Created(c, "Documents uploaded", data)
}

func (h *MaintenanceHandler) DeleteDocument(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	docID, err := parseID(c, "docId")
	if err != nil {
		return err
	}
	if err := h.svc.DeleteDocument(c.Context(), id, docID, auditActor(c)); err != nil {
		return err
	}
	return util.OK(c, "Document deleted", nil)
}

func (h *MaintenanceHandler) PDF(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	data, name, err := h.svc.GeneratePDF(c.Context(), id, c.Params("kind"))
	if err != nil {
		return err
	}
	c.Set(fiber.HeaderContentType, "application/pdf")
	disp := "inline"
	if c.Query("download") == "1" {
		disp = "attachment"
	}
	c.Set(fiber.HeaderContentDisposition, disp+`; filename="`+name+`"`)
	return c.Send(data)
}

// ─── Vendor ──────────────────────────────────────────────────────────────────

type VendorHandler struct {
	svc *service.VendorService
}

func NewVendorHandler(svc *service.VendorService) *VendorHandler {
	return &VendorHandler{svc: svc}
}

func (h *VendorHandler) Register(r fiber.Router) {
	g := r.Group("/vendors", middleware.Auth(), middleware.RequireRole("ADMIN"))
	g.Get("", h.List)
	g.Get("/:id", h.GetByID)
	g.Post("", h.Create)
	g.Put("/:id", h.Update)
	g.Patch("/:id/toggle-active", h.Toggle)
	g.Delete("/:id", h.Delete)
}

func (h *VendorHandler) List(c *fiber.Ctx) error {
	var active *bool
	switch c.Query("isActive") {
	case "true":
		t := true
		active = &t
	case "false":
		f := false
		active = &f
	}
	data, err := h.svc.List(c.Context(), c.Query("search"), c.Query("type"), active)
	if err != nil {
		return err
	}
	return util.OK(c, "Vendors retrieved", data)
}

func (h *VendorHandler) GetByID(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	data, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		return err
	}
	return util.OK(c, "Vendor retrieved", data)
}

func (h *VendorHandler) Create(c *fiber.Ctx) error {
	var req service.VendorRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.Create(c.Context(), req, auditActor(c))
	if err != nil {
		return err
	}
	return util.Created(c, "Vendor created", data)
}

func (h *VendorHandler) Update(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var req service.VendorRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.Update(c.Context(), id, req, auditActor(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Vendor updated", data)
}

func (h *VendorHandler) Toggle(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	data, err := h.svc.ToggleActive(c.Context(), id, auditActor(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Vendor status toggled", data)
}

func (h *VendorHandler) Delete(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := h.svc.Delete(c.Context(), id, auditActor(c)); err != nil {
		return err
	}
	return util.OK(c, "Vendor deleted", nil)
}

// ─── Laporan kendala kendaraan ───────────────────────────────────────────────

type VehicleIssueHandler struct {
	svc *service.VehicleIssueService
}

func NewVehicleIssueHandler(svc *service.VehicleIssueService) *VehicleIssueHandler {
	return &VehicleIssueHandler{svc: svc}
}

func (h *VehicleIssueHandler) Register(r fiber.Router) {
	admin := middleware.RequireRole("ADMIN")
	g := r.Group("/vehicle-issues", middleware.Auth(), middleware.RequireRole("ADMIN", "DRIVER"))
	g.Get("", h.List)
	g.Get("/my-vehicles", h.MyVehicles)
	g.Get("/:id", h.GetByID)
	g.Post("", h.Create)
	g.Post("/:id/convert", admin, h.Convert)
	g.Post("/:id/dismiss", admin, h.Dismiss)
}

func (h *VehicleIssueHandler) List(c *fiber.Ctx) error {
	page := queryInt(c, "page", 1)
	limit := queryInt(c, "limit", 20)
	data, total, err := h.svc.List(c.Context(), page, limit, c.Query("status"), queryInt32(c, "vehicleId"),
		middleware.GetUserRole(c), int32(middleware.GetUserID(c)))
	if err != nil {
		return err
	}
	return util.Paginated(c, "Vehicle issues retrieved", data, total, page, limit)
}

func (h *VehicleIssueHandler) MyVehicles(c *fiber.Ctx) error {
	data, err := h.svc.MyVehicles(c.Context(), int32(middleware.GetUserID(c)))
	if err != nil {
		return err
	}
	return util.OK(c, "Vehicles retrieved", data)
}

func (h *VehicleIssueHandler) GetByID(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	data, err := h.svc.GetByID(c.Context(), id, middleware.GetUserRole(c), int32(middleware.GetUserID(c)))
	if err != nil {
		return err
	}
	return util.OK(c, "Vehicle issue retrieved", data)
}

func (h *VehicleIssueHandler) Create(c *fiber.Ctx) error {
	var req service.CreateVehicleIssueRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if err := validate.Struct(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
	}
	var photos []*multipart.FileHeader
	if form, ferr := c.MultipartForm(); ferr == nil {
		photos = form.File["photos[]"]
		if len(photos) == 0 {
			photos = form.File["photo"]
		}
	}
	data, err := h.svc.Create(c.Context(), req, photos, middleware.GetUserRole(c), auditActor(c))
	if err != nil {
		return err
	}
	return util.Created(c, "Vehicle issue reported", data)
}

func (h *VehicleIssueHandler) Convert(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	data, err := h.svc.Convert(c.Context(), id, auditActor(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Vehicle issue converted", data)
}

func (h *VehicleIssueHandler) Dismiss(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var req service.ResolveVehicleIssueRequest
	_ = c.BodyParser(&req)
	data, err := h.svc.Dismiss(c.Context(), id, req, auditActor(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Vehicle issue dismissed", data)
}

// ─── Pengaturan dokumen ──────────────────────────────────────────────────────

type DocumentSettingsHandler struct {
	svc *service.DocumentSettingsService
}

func NewDocumentSettingsHandler(svc *service.DocumentSettingsService) *DocumentSettingsHandler {
	return &DocumentSettingsHandler{svc: svc}
}

func (h *DocumentSettingsHandler) Register(r fiber.Router) {
	g := r.Group("/document-settings", middleware.Auth(), middleware.RequireRole("ADMIN"))
	g.Get("", h.Get)
	g.Put("", h.Update)
	g.Post("/logo", h.SetLogo)
	g.Delete("/logo", h.DeleteLogo)
}

func (h *DocumentSettingsHandler) Get(c *fiber.Ctx) error {
	data, err := h.svc.Get(c.Context())
	if err != nil {
		return err
	}
	return util.OK(c, "Document settings retrieved", data)
}

func (h *DocumentSettingsHandler) Update(c *fiber.Ctx) error {
	var req service.DocumentSettingsRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.Update(c.Context(), req, auditActor(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Document settings updated", data)
}

func (h *DocumentSettingsHandler) SetLogo(c *fiber.Ctx) error {
	fh, err := c.FormFile("logo")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "file logo wajib diunggah (field: logo)")
	}
	data, err := h.svc.SetLogo(c.Context(), fh, auditActor(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Logo updated", data)
}

func (h *DocumentSettingsHandler) DeleteLogo(c *fiber.Ctx) error {
	data, err := h.svc.DeleteLogo(c.Context(), auditActor(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Logo removed", data)
}
