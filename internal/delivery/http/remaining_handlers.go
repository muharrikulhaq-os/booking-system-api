package http

import (
	"time"

	"booking-system-api/internal/middleware"
	"booking-system-api/internal/service"
	"mime/multipart"
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
	g.Post("", h.Create)
	g.Delete("/:id", admin, h.Delete)
}

func (h *FuelExpenseHandler) List(c *fiber.Ctx) error {
	page := queryInt(c, "page", 1)
	limit := queryInt(c, "limit", 20)
	role := middleware.GetUserRole(c)
	actorID := int32(middleware.GetUserID(c))

	data, total, err := h.svc.List(c.Context(), page, limit,
		queryInt32(c, "driverId"), queryInt32(c, "vehicleId"), queryString(c, "fuelType"),
		queryInt32(c, "bookingId"), actorID, role)
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

func (h *FuelExpenseHandler) Create(c *fiber.Ctx) error {
	var req service.CreateFuelExpenseRequest
	
	// Read multipart fields
	req.VehicleID = int32(util.ParseStringToInt(c.FormValue("vehicleId")))
	req.FuelTypeID = int32(util.ParseStringToInt(c.FormValue("fuelTypeId")))
	if bid := c.FormValue("bookingId"); bid != "" {
		parsed := int32(util.ParseStringToInt(bid))
		req.BookingID = &parsed
	}
	
	req.FuelGrade = c.FormValue("fuelGrade")
	req.Liter = util.ParseStringToFloat64(c.FormValue("liter"))
	req.PricePerLiter = util.ParseStringToFloat64(c.FormValue("pricePerLiter"))
	req.Kwh = util.ParseStringToFloat64(c.FormValue("kwh"))
	req.PricePerKwh = util.ParseStringToFloat64(c.FormValue("pricePerKwh"))
	req.OdometerBefore = int32(util.ParseStringToInt(c.FormValue("odometerBefore")))
	req.OdometerAfter = int32(util.ParseStringToInt(c.FormValue("odometerAfter")))
	req.Note = c.FormValue("note")

	// Handle proofPhoto
	file, err := c.FormFile("proofPhoto")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "proofPhoto is required")
	}
	
	// Check driver ID from token
	recordedByID := int32(middleware.GetUserID(c))
	
	// In a real implementation we would get driver ID based on user ID if the user is a driver.
	// We'll pass the recordedByID for now, or you could do a query to find the driver ID.
	driverID := &recordedByID // Simplified, typically we lookup driver by user_id

	// Save photo logic should be in service or delivery. We'll let service handle saving,
	// or we can save it here. Saving here is typical for handlers.
	filePath, err := util.SaveUploadedFile(file, "fuel_proofs")
	if err != nil {
		return err
	}
	req.ProofPhotoUrl = "/uploads/" + filePath

	data, err := h.svc.Create(c.Context(), req, recordedByID, driverID)
	if err != nil {
		return err
	}
	return util.Created(c, "Fuel expense recorded", data)
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
	g.Patch("/:id/complete", h.Complete)
	g.Delete("/:id", h.Delete)
}

func (h *MaintenanceHandler) List(c *fiber.Ctx) error {
	page := queryInt(c, "page", 1)
	limit := queryInt(c, "limit", 20)
	data, total, err := h.svc.List(c.Context(), page, limit, queryInt32(c, "vehicleId"))
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
	resp, err := h.svc.Create(c.Context(), req, int32(middleware.GetUserID(c)))
	if err != nil {
		return err
	}
	if resp.Warning != "" {
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"success": true,
			"message": "Maintenance record created with warning",
			"warning": resp.Warning,
			"data":    resp.Data,
		})
	}
	return util.Created(c, "Maintenance record created", resp.Data)
}

func (h *MaintenanceHandler) Complete(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	
	form, formErr := c.MultipartForm()
	var photos []*multipart.FileHeader
	if formErr == nil {
		photos = form.File["photos[]"]
		if len(photos) == 0 {
			photos = form.File["photo"] // fallback
		}
	}
	
	data, err := h.svc.Complete(c.Context(), id, photos)
	if err != nil {
		return err
	}
	return util.OK(c, "Maintenance record marked as complete", data)
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

	// New frontend settings endpoints
	s := r.Group("/settings", auth)
	s.Get("/fuel-prices", h.ListFuelPrices)
	s.Put("/:key", admin, h.Upsert)
}

func (h *MasterSettingHandler) ListFuelPrices(c *fiber.Ctx) error {
	data, err := h.svc.ListFuelPrices(c.Context())
	if err != nil {
		return err
	}
	return util.OK(c, "Fuel prices retrieved", data)
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
	// Existing endpoints
	g.Get("/bookings", h.BookingSummary)
	g.Get("/resource-usage", h.ResourceUsage)
	g.Get("/fuel-expenses", h.FuelExpenses)
	g.Get("/maintenance-cost", h.MaintenanceCost)
	g.Get("/driver-ratings", h.DriverRatings)
	g.Get("/driver-activity", h.DriverActivity)
	g.Get("/overdue-bookings", h.OverdueBookings)
	g.Get("/audit-logs", h.AuditLogs)
	// New endpoints
	g.Get("/overview", h.Overview)
	g.Get("/bookings/trend", h.BookingTrend)
	g.Get("/bookings/by-department", h.BookingsByDepartment)
	g.Get("/bookings/by-resource", h.BookingsByResource)
	g.Get("/bookings/approval-performance", h.ApprovalPerformance)
	g.Get("/cost-summary", h.CostSummary)
	g.Get("/cost/by-vehicle", h.CostByVehicle)
	g.Get("/cost/by-department", h.CostByDepartment)
	g.Get("/cost/trend", h.CostTrend)
	g.Get("/driver-performance", h.DriverPerformance)
	g.Get("/department-summary", h.DepartmentSummary)
	g.Get("/driver-trips", h.DriverTrips)
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
	data, err := h.svc.ResourceUsage(c.Context(), parseDate(c, "startDate"), parseDate(c, "endDate"))
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
	data, err := h.svc.DriverRatings(c.Context(), parseDate(c, "startDate"), parseDate(c, "endDate"))
	if err != nil {
		return err
	}
	return util.OK(c, "Driver ratings report", data)
}

func (h *ReportHandler) DriverActivity(c *fiber.Ctx) error {
	data, err := h.svc.DriverActivity(c.Context(), parseDate(c, "startDate"), parseDate(c, "endDate"))
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
	data, total, err := h.svc.AuditLogs(c.Context(), page, limit,
		queryString(c, "entityType"), queryInt32(c, "userId"),
		parseDate(c, "startDate"), parseDate(c, "endDate"))
	if err != nil {
		return err
	}
	return util.Paginated(c, "Audit logs retrieved", data, total, page, limit)
}

// parseDate parses an RFC3339 date from a query param; returns nil if absent or invalid.
func parseDate(c *fiber.Ctx, key string) *time.Time {
	s := c.Query(key)
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}

func (h *ReportHandler) Overview(c *fiber.Ctx) error {
	data, err := h.svc.Overview(c.Context(), parseDate(c, "startDate"), parseDate(c, "endDate"))
	if err != nil {
		return err
	}
	return util.OK(c, "Overview report", data)
}

// rangeOrDefault resolves optional startDate/endDate query params to a
// concrete [start, end] window, falling back to "now minus N months" when
// either side is missing - dipakai trend chart (BookingTrend/CostTrend)
// yang butuh bound konkret, bukan nullable, untuk generate_series-nya.
func rangeOrDefault(c *fiber.Ctx, defaultMonthsBack int) (time.Time, time.Time) {
	end := time.Now()
	if e := parseDate(c, "endDate"); e != nil {
		end = *e
	}
	start := end.AddDate(0, -defaultMonthsBack, 0)
	if s := parseDate(c, "startDate"); s != nil {
		start = *s
	}
	return start, end
}

func (h *ReportHandler) BookingTrend(c *fiber.Ctx) error {
	groupBy := c.Query("groupBy", "monthly")
	start, end := rangeOrDefault(c, 12)
	data, err := h.svc.BookingTrend(c.Context(), groupBy, start, end)
	if err != nil {
		return err
	}
	return util.OK(c, "Booking trend", data)
}

func (h *ReportHandler) BookingsByDepartment(c *fiber.Ctx) error {
	data, err := h.svc.BookingsByDepartment(c.Context(), parseDate(c, "startDate"), parseDate(c, "endDate"))
	if err != nil {
		return err
	}
	return util.OK(c, "Bookings by department", data)
}

func (h *ReportHandler) BookingsByResource(c *fiber.Ctx) error {
	data, err := h.svc.BookingsByResource(c.Context(), parseDate(c, "startDate"), parseDate(c, "endDate"))
	if err != nil {
		return err
	}
	return util.OK(c, "Bookings by resource", data)
}

func (h *ReportHandler) ApprovalPerformance(c *fiber.Ctx) error {
	data, err := h.svc.ApprovalPerformance(c.Context(), parseDate(c, "startDate"), parseDate(c, "endDate"))
	if err != nil {
		return err
	}
	return util.OK(c, "Approval performance", data)
}

func (h *ReportHandler) CostSummary(c *fiber.Ctx) error {
	data, err := h.svc.CostSummary(c.Context(), parseDate(c, "startDate"), parseDate(c, "endDate"))
	if err != nil {
		return err
	}
	return util.OK(c, "Cost summary", data)
}

func (h *ReportHandler) CostByVehicle(c *fiber.Ctx) error {
	data, err := h.svc.CostByVehicle(c.Context(), parseDate(c, "startDate"), parseDate(c, "endDate"))
	if err != nil {
		return err
	}
	return util.OK(c, "Cost by vehicle", data)
}

func (h *ReportHandler) CostByDepartment(c *fiber.Ctx) error {
	data, err := h.svc.CostByDepartment(c.Context(), parseDate(c, "startDate"), parseDate(c, "endDate"))
	if err != nil {
		return err
	}
	return util.OK(c, "Cost by department", data)
}

func (h *ReportHandler) CostTrend(c *fiber.Ctx) error {
	groupBy := c.Query("groupBy", "monthly")
	start, end := rangeOrDefault(c, 6)
	data, err := h.svc.CostTrend(c.Context(), groupBy, start, end)
	if err != nil {
		return err
	}
	return util.OK(c, "Cost trend", data)
}

func (h *ReportHandler) DriverPerformance(c *fiber.Ctx) error {
	data, err := h.svc.DriverPerformance(c.Context(), parseDate(c, "startDate"), parseDate(c, "endDate"))
	if err != nil {
		return err
	}
	return util.OK(c, "Driver performance", data)
}

func (h *ReportHandler) DepartmentSummary(c *fiber.Ctx) error {
	data, err := h.svc.DepartmentSummary(c.Context(), parseDate(c, "startDate"), parseDate(c, "endDate"))
	if err != nil {
		return err
	}
	return util.OK(c, "Department summary", data)
}

func (h *ReportHandler) DriverTrips(c *fiber.Ctx) error {
	data, err := h.svc.DriverTrips(c.Context(), parseDate(c, "startDate"), parseDate(c, "endDate"))
	if err != nil {
		return err
	}
	return util.OK(c, "Driver trips report retrieved", data)
}
