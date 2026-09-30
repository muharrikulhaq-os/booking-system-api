package http

import (
	"time"

	"booking-system-api/internal/middleware"
	"booking-system-api/internal/service"
	"booking-system-api/internal/util"

	"github.com/gofiber/fiber/v2"
)

// FuelLedgerHandler: saldo BBM kendaraan, SPBU mitra, dan voucher BBM
// (docs/RANCANGAN_VOUCHER_BBM.md).
type FuelLedgerHandler struct {
	svc *service.FuelLedgerService
}

func NewFuelLedgerHandler(svc *service.FuelLedgerService) *FuelLedgerHandler {
	return &FuelLedgerHandler{svc: svc}
}

func (h *FuelLedgerHandler) Register(r fiber.Router) {
	auth := middleware.Auth()
	admin := middleware.RequireRole("ADMIN")
	adminDriver := middleware.RequireRole("ADMIN", "DRIVER")

	b := r.Group("/fuel-balances", auth)
	b.Get("", admin, h.ListBalances)
	b.Get("/:vehicleId", adminDriver, h.GetBalance)
	b.Get("/:vehicleId/ledger", admin, h.ListLedger)
	b.Put("/:vehicleId/profile", admin, h.UpdateProfile)
	b.Post("/:vehicleId/adjustments", admin, h.Adjust)

	st := r.Group("/fuel-stations", auth)
	st.Get("", h.ListStations)
	st.Post("", admin, h.CreateStation)
	st.Put("/:id", admin, h.UpdateStation)
	st.Delete("/:id", admin, h.DeleteStation)

	v := r.Group("/fuel-vouchers", auth)
	v.Get("", adminDriver, h.ListVouchers)
	v.Post("/preview", admin, h.PreviewVoucher)
	v.Post("/reconcile", admin, h.Reconcile)
	v.Post("", admin, h.IssueVoucher)
	v.Get("/:id", adminDriver, h.GetVoucher)
	v.Patch("/:id/use", adminDriver, h.UseVoucher)
	v.Patch("/:id/cancel", admin, h.CancelVoucher)
}

// ─── Saldo ──────────────────────────────────────────────────────────────────

func (h *FuelLedgerHandler) ListBalances(c *fiber.Ctx) error {
	data, err := h.svc.ListBalances(c.Context())
	if err != nil {
		return err
	}
	return util.OK(c, "Fuel balances retrieved", data)
}

func (h *FuelLedgerHandler) GetBalance(c *fiber.Ctx) error {
	id, err := parseID(c, "vehicleId")
	if err != nil {
		return err
	}
	data, err := h.svc.GetBalance(c.Context(), id)
	if err != nil {
		return err
	}
	return util.OK(c, "Fuel balance retrieved", data)
}

func (h *FuelLedgerHandler) ListLedger(c *fiber.Ctx) error {
	id, err := parseID(c, "vehicleId")
	if err != nil {
		return err
	}
	page := queryInt(c, "page", 1)
	limit := queryInt(c, "limit", 20)
	data, total, err := h.svc.ListLedger(c.Context(), id, queryString(c, "energy"), page, limit)
	if err != nil {
		return err
	}
	return util.Paginated(c, "Fuel ledger retrieved", data, total, page, limit)
}

func (h *FuelLedgerHandler) UpdateProfile(c *fiber.Ctx) error {
	id, err := parseID(c, "vehicleId")
	if err != nil {
		return err
	}
	var req service.FuelProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	data, err := h.svc.UpdateProfile(c.Context(), id, req, auditActor(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Fuel profile updated", data)
}

func (h *FuelLedgerHandler) Adjust(c *fiber.Ctx) error {
	id, err := parseID(c, "vehicleId")
	if err != nil {
		return err
	}
	var req service.FuelAdjustmentRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	data, err := h.svc.Adjust(c.Context(), id, req, auditActor(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Fuel balance adjusted", data)
}

// ─── SPBU mitra ─────────────────────────────────────────────────────────────

func (h *FuelLedgerHandler) ListStations(c *fiber.Ctx) error {
	data, err := h.svc.ListStations(c.Context(), c.Query("active") == "true")
	if err != nil {
		return err
	}
	return util.OK(c, "Fuel stations retrieved", data)
}

func (h *FuelLedgerHandler) CreateStation(c *fiber.Ctx) error {
	var req service.FuelStationRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	data, err := h.svc.CreateStation(c.Context(), req, auditActor(c))
	if err != nil {
		return err
	}
	return util.Created(c, "Fuel station created", data)
}

func (h *FuelLedgerHandler) UpdateStation(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var req service.FuelStationRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	data, err := h.svc.UpdateStation(c.Context(), id, req, auditActor(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Fuel station updated", data)
}

func (h *FuelLedgerHandler) DeleteStation(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := h.svc.DeleteStation(c.Context(), id, auditActor(c)); err != nil {
		return err
	}
	return util.OK(c, "Fuel station deleted", nil)
}

// ─── Voucher ────────────────────────────────────────────────────────────────

func (h *FuelLedgerHandler) PreviewVoucher(c *fiber.Ctx) error {
	var req service.FuelVoucherRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	data, err := h.svc.PreviewVoucher(c.Context(), req)
	if err != nil {
		return err
	}
	return util.OK(c, "Fuel voucher preview", data)
}

func (h *FuelLedgerHandler) IssueVoucher(c *fiber.Ctx) error {
	var req service.FuelVoucherRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	data, err := h.svc.IssueVoucher(c.Context(), req, auditActor(c))
	if err != nil {
		return err
	}
	return util.Created(c, "Fuel voucher issued", data)
}

func queryTime(c *fiber.Ctx, key string) *time.Time {
	v := c.Query(key)
	if v == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil
	}
	return &t
}

func queryBool(c *fiber.Ctx, key string) *bool {
	switch c.Query(key) {
	case "true":
		b := true
		return &b
	case "false":
		b := false
		return &b
	}
	return nil
}

func (h *FuelLedgerHandler) ListVouchers(c *fiber.Ctx) error {
	page := queryInt(c, "page", 1)
	limit := queryInt(c, "limit", 20)
	f := service.VoucherListFilter{
		Status:     queryString(c, "status"),
		VehicleID:  queryInt32(c, "vehicleId"),
		StationID:  queryInt32(c, "stationId"),
		From:       queryTime(c, "from"),
		To:         queryTime(c, "to"),
		Reconciled: queryBool(c, "reconciled"),
		Search:     queryString(c, "search"),
	}
	data, total, summary, err := h.svc.ListVouchers(c.Context(), f, page, limit,
		int32(middleware.GetUserID(c)), middleware.GetUserRole(c))
	if err != nil {
		return err
	}
	totalPages := (total + int64(limit) - 1) / int64(limit)
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Fuel vouchers retrieved",
		"data":    data,
		"summary": summary,
		"pagination": fiber.Map{
			"page": page, "limit": limit, "total": total, "totalPages": totalPages,
		},
	})
}

func (h *FuelLedgerHandler) GetVoucher(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	data, err := h.svc.GetVoucher(c.Context(), id, int32(middleware.GetUserID(c)), middleware.GetUserRole(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Fuel voucher retrieved", data)
}

// UseVoucher (multipart): receiptPhoto (wajib untuk driver), odometer, note.
func (h *FuelLedgerHandler) UseVoucher(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	req := service.UseVoucherRequest{
		Odometer: int32(util.ParseStringToInt(c.FormValue("odometer"))),
		Note:     c.FormValue("note"),
	}
	if file, ferr := c.FormFile("receiptPhoto"); ferr == nil {
		path, err := util.SaveUploadedFile(file, "fuel_proofs")
		if err != nil {
			return err
		}
		req.ReceiptPhotoUrl = "/uploads/" + path
	}
	data, err := h.svc.UseVoucher(c.Context(), id, req, auditActor(c), middleware.GetUserRole(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Fuel voucher used", data)
}

func (h *FuelLedgerHandler) CancelVoucher(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	data, err := h.svc.CancelVoucher(c.Context(), id, body.Reason, auditActor(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Fuel voucher cancelled", data)
}

func (h *FuelLedgerHandler) Reconcile(c *fiber.Ctx) error {
	var req service.ReconcileRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	n, err := h.svc.Reconcile(c.Context(), req, auditActor(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Fuel vouchers reconciled", fiber.Map{"reconciled": n})
}
