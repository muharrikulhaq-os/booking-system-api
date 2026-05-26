package http

import (
	"time"

	"booking-system-api/internal/middleware"
	"booking-system-api/internal/service"
	"booking-system-api/internal/util"

	"github.com/gofiber/fiber/v2"
)

// ─── Fuel Expense Handler ─────────────────────────────────────────────────────

type FuelExpenseHandler struct {
	svc *service.FuelExpenseService
}

func NewFuelExpenseHandler(svc *service.FuelExpenseService) *FuelExpenseHandler {
	return &FuelExpenseHandler{svc: svc}
}

func (h *FuelExpenseHandler) Register(r fiber.Router) {
	auth := middleware.Auth()
	admin := middleware.RequireRole("ADMIN")

	g := r.Group("/fuel-expenses", auth)
	g.Get("", h.List)
	g.Get("/:id", h.GetByID)
	g.Post("/bbm", h.CreateBBM)
	g.Post("/listrik", h.CreateListrik)
	g.Delete("/:id", admin, h.Delete)
}

func (h *FuelExpenseHandler) List(c *fiber.Ctx) error {
	page := queryInt(c, "page", 1)
	limit := queryInt(c, "limit", 20)
	data, total, err := h.svc.List(c.Context(), page, limit,
		queryInt32(c, "driverId"), queryInt32(c, "vehicleId"), queryString(c, "fuelType"))
	if err != nil {
		return err
	}
	return util.Paginated(c, "Fuel expenses retrieved", data, total, page, limit)
}

func (h *FuelExpenseHandler) GetByID(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	data, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		return err
	}
	return util.OK(c, "Fuel expense retrieved", data)
}

func (h *FuelExpenseHandler) CreateBBM(c *fiber.Ctx) error {
	var req service.CreateFuelExpenseBBMRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	// get driverId from current user's driver profile
	driverID := queryInt32(c, "driverId")
	if driverID == nil {
		return fiber.NewError(fiber.StatusBadRequest, "driverId is required")
	}
	data, err := h.svc.CreateBBM(c.Context(), req, *driverID)
	if err != nil {
		return err
	}
	return util.Created(c, "BBM expense recorded", data)
}

func (h *FuelExpenseHandler) CreateListrik(c *fiber.Ctx) error {
	var req service.CreateFuelExpenseListrikRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	driverID := queryInt32(c, "driverId")
	if driverID == nil {
		return fiber.NewError(fiber.StatusBadRequest, "driverId is required")
	}
	data, err := h.svc.CreateListrik(c.Context(), req, *driverID)
	if err != nil {
		return err
	}
	return util.Created(c, "Listrik expense recorded", data)
}

func (h *FuelExpenseHandler) Delete(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := h.svc.Delete(c.Context(), id); err != nil {
		return err
	}
	return util.OK(c, "Fuel expense deleted", nil)
}

// ─── Maintenance Handler ──────────────────────────────────────────────────────

type MaintenanceHandler struct {
	svc *service.MaintenanceService
}

func NewMaintenanceHandler(svc *service.MaintenanceService) *MaintenanceHandler {
	return &MaintenanceHandler{svc: svc}
}

func (h *MaintenanceHandler) Register(r fiber.Router) {
	auth := middleware.Auth()
	admin := middleware.RequireRole("ADMIN")

	g := r.Group("/maintenance", auth, admin)
	g.Get("", h.List)
	g.Get("/:id", h.GetByID)
	g.Post("", h.Create)
	g.Put("/:id", h.Update)
	g.Delete("/:id", h.Delete)
}

func (h *MaintenanceHandler) List(c *fiber.Ctx) error {
	page := queryInt(c, "page", 1)
	limit := queryInt(c, "limit", 20)
	data, total, err := h.svc.List(c.Context(), page, limit, queryInt32(c, "resourceId"))
	if err != nil {
		return err
	}
	return util.Paginated(c, "Maintenance records retrieved", data, total, page, limit)
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
	var req service.CreateMaintenanceRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.Create(c.Context(), req, int32(middleware.GetUserID(c)))
	if err != nil {
		return err
	}
	return util.Created(c, "Maintenance record created", data)
}

func (h *MaintenanceHandler) Update(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var req service.UpdateMaintenanceRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.Update(c.Context(), id, req)
	if err != nil {
		return err
	}
	return util.OK(c, "Maintenance record updated", data)
}

func (h *MaintenanceHandler) Delete(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := h.svc.Delete(c.Context(), id); err != nil {
		return err
	}
	return util.OK(c, "Maintenance record deleted", nil)
}

// ─── Attachment Handler (global delete) ──────────────────────────────────────

type AttachmentHandler struct {
	svc *service.AttachmentService
}

func NewAttachmentHandler(svc *service.AttachmentService) *AttachmentHandler {
	return &AttachmentHandler{svc: svc}
}

func (h *AttachmentHandler) Register(r fiber.Router) {
	auth := middleware.Auth()
	g := r.Group("/attachments", auth)
	g.Delete("/:id", h.Delete)
}

func (h *AttachmentHandler) Delete(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := h.svc.Delete(c.Context(), id); err != nil {
		return err
	}
	return util.OK(c, "Attachment deleted", nil)
}

// ─── Guest Booking Handler ────────────────────────────────────────────────────

type GuestBookingHandler struct {
	svc *service.GuestBookingService
}

func NewGuestBookingHandler(svc *service.GuestBookingService) *GuestBookingHandler {
	return &GuestBookingHandler{svc: svc}
}

func (h *GuestBookingHandler) Register(r fiber.Router) {
	auth := middleware.Auth()
	admin := middleware.RequireRole("ADMIN")

	g := r.Group("/guest-bookings")
	// Public
	g.Post("", h.Create)
	g.Get("/:token", h.GetByToken)
	g.Patch("/:token/complete", h.CompleteByToken)
	g.Patch("/:token/cancel", h.CancelByToken)
	// Admin
	g.Get("", auth, admin, h.List)
	g.Post("/:id/approve", auth, admin, h.Approve)
	g.Post("/:id/reject", auth, admin, h.Reject)
	g.Patch("/:id/start", auth, admin, h.Start)
}

func (h *GuestBookingHandler) List(c *fiber.Ctx) error {
	page := queryInt(c, "page", 1)
	limit := queryInt(c, "limit", 20)
	data, total, err := h.svc.List(c.Context(), page, limit, queryString(c, "status"))
	if err != nil {
		return err
	}
	return util.Paginated(c, "Guest bookings retrieved", data, total, page, limit)
}

func (h *GuestBookingHandler) Create(c *fiber.Ctx) error {
	var req service.CreateGuestBookingRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.Create(c.Context(), req)
	if err != nil {
		return err
	}
	return util.Created(c, "Guest booking created", data)
}

func (h *GuestBookingHandler) GetByToken(c *fiber.Ctx) error {
	data, err := h.svc.GetByToken(c.Context(), c.Params("token"))
	if err != nil {
		return err
	}
	return util.OK(c, "Guest booking retrieved", data)
}

func (h *GuestBookingHandler) CompleteByToken(c *fiber.Ctx) error {
	data, err := h.svc.CompleteByToken(c.Context(), c.Params("token"))
	if err != nil {
		return err
	}
	return util.OK(c, "Guest booking completed", data)
}

func (h *GuestBookingHandler) CancelByToken(c *fiber.Ctx) error {
	data, err := h.svc.CancelByToken(c.Context(), c.Params("token"))
	if err != nil {
		return err
	}
	return util.OK(c, "Guest booking cancelled", data)
}

func (h *GuestBookingHandler) Approve(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	data, err := h.svc.Approve(c.Context(), id, int32(middleware.GetUserID(c)))
	if err != nil {
		return err
	}
	return util.OK(c, "Guest booking approved", data)
}

func (h *GuestBookingHandler) Reject(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var req service.RejectGuestBookingRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.Reject(c.Context(), id, req, int32(middleware.GetUserID(c)))
	if err != nil {
		return err
	}
	return util.OK(c, "Guest booking rejected", data)
}

func (h *GuestBookingHandler) Start(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	data, err := h.svc.Start(c.Context(), id)
	if err != nil {
		return err
	}
	return util.OK(c, "Guest booking started", data)
}

// ─── Master Settings Handler ──────────────────────────────────────────────────

type MasterSettingHandler struct {
	svc *service.MasterSettingService
}

func NewMasterSettingHandler(svc *service.MasterSettingService) *MasterSettingHandler {
	return &MasterSettingHandler{svc: svc}
}

func (h *MasterSettingHandler) Register(r fiber.Router) {
	auth := middleware.Auth()
	admin := middleware.RequireRole("ADMIN")

	g := r.Group("/master-settings", auth)
	g.Get("", h.List)
	g.Get("/:key", h.GetByKey)
	g.Put("/:key", admin, h.Upsert)
}

func (h *MasterSettingHandler) List(c *fiber.Ctx) error {
	data, err := h.svc.List(c.Context())
	if err != nil {
		return err
	}
	return util.OK(c, "Settings retrieved", data)
}

func (h *MasterSettingHandler) GetByKey(c *fiber.Ctx) error {
	data, err := h.svc.GetByKey(c.Context(), c.Params("key"))
	if err != nil {
		return err
	}
	return util.OK(c, "Setting retrieved", data)
}

func (h *MasterSettingHandler) Upsert(c *fiber.Ctx) error {
	var req service.UpsertSettingRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.Upsert(c.Context(), c.Params("key"), req)
	if err != nil {
		return err
	}
	return util.OK(c, "Setting updated", data)
}

// ─── Report Handler ───────────────────────────────────────────────────────────

type ReportHandler struct {
	svc *service.ReportService
}

func NewReportHandler(svc *service.ReportService) *ReportHandler {
	return &ReportHandler{svc: svc}
}

func (h *ReportHandler) Register(r fiber.Router) {
	auth := middleware.Auth()
	admin := middleware.RequireRole("ADMIN")

	g := r.Group("/reports", auth, admin)
	g.Get("/bookings", h.BookingSummary)
	g.Get("/resource-usage", h.ResourceUsage)
	g.Get("/fuel-expenses", h.FuelExpenses)
	g.Get("/maintenance-cost", h.MaintenanceCost)
	g.Get("/driver-ratings", h.DriverRatings)
	g.Get("/driver-activity", h.DriverActivity)
	g.Get("/overdue-bookings", h.OverdueBookings)
	g.Get("/audit-logs", h.AuditLogs)
}

func (h *ReportHandler) BookingSummary(c *fiber.Ctx) error {
	var startFrom, endTo *time.Time
	if s := c.Query("startDate"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			startFrom = &t
		}
	}
	if e := c.Query("endDate"); e != "" {
		if t, err := time.Parse(time.RFC3339, e); err == nil {
			endTo = &t
		}
	}
	data, err := h.svc.BookingSummary(c.Context(), startFrom, endTo)
	if err != nil {
		return err
	}
	return util.OK(c, "Booking summary", data)
}

func (h *ReportHandler) ResourceUsage(c *fiber.Ctx) error {
	data, err := h.svc.ResourceUsage(c.Context())
	if err != nil {
		return err
	}
	return util.OK(c, "Resource usage report", data)
}

func (h *ReportHandler) FuelExpenses(c *fiber.Ctx) error {
	data, err := h.svc.FuelExpenses(c.Context())
	if err != nil {
		return err
	}
	return util.OK(c, "Fuel expense report", data)
}

func (h *ReportHandler) MaintenanceCost(c *fiber.Ctx) error {
	data, err := h.svc.MaintenanceCost(c.Context())
	if err != nil {
		return err
	}
	return util.OK(c, "Maintenance cost report", data)
}

func (h *ReportHandler) DriverRatings(c *fiber.Ctx) error {
	data, err := h.svc.DriverRatings(c.Context())
	if err != nil {
		return err
	}
	return util.OK(c, "Driver ratings report", data)
}

func (h *ReportHandler) DriverActivity(c *fiber.Ctx) error {
	data, err := h.svc.DriverActivity(c.Context())
	if err != nil {
		return err
	}
	return util.OK(c, "Driver activity report", data)
}

func (h *ReportHandler) OverdueBookings(c *fiber.Ctx) error {
	data, err := h.svc.OverdueBookings(c.Context())
	if err != nil {
		return err
	}
	return util.OK(c, "Overdue bookings", data)
}

func (h *ReportHandler) AuditLogs(c *fiber.Ctx) error {
	page := queryInt(c, "page", 1)
	limit := queryInt(c, "limit", 50)
	data, total, err := h.svc.AuditLogs(c.Context(), page, limit, queryString(c, "entityType"), queryInt32(c, "userId"))
	if err != nil {
		return err
	}
	return util.Paginated(c, "Audit logs retrieved", data, total, page, limit)
}
