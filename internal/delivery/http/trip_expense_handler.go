package http

import (
	"fmt"
	"time"

	"booking-system-api/internal/middleware"
	"booking-system-api/internal/service"
	"booking-system-api/internal/util"

	"github.com/gofiber/fiber/v2"
)

// TripExpenseHandler: biaya perjalanan (tol, parkir, lainnya) —
// docs/RANCANGAN_BIAYA_PERJALANAN.md.
type TripExpenseHandler struct {
	svc *service.TripExpenseService
}

func NewTripExpenseHandler(svc *service.TripExpenseService) *TripExpenseHandler {
	return &TripExpenseHandler{svc: svc}
}

func (h *TripExpenseHandler) Register(r fiber.Router) {
	auth := middleware.Auth()
	admin := middleware.RequireRole("ADMIN")

	r.Get("/trip-expenses/config", auth, h.Config)
	r.Get("/trip-expenses/gates", auth, h.Gates)
	r.Patch("/trip-expenses/:id/void", auth, admin, h.Void)

	r.Get("/bookings/:id/trip-expenses", auth, h.ListForBooking)
	r.Post("/bookings/:id/trip-expenses", auth, middleware.RequireRole("ADMIN", "DRIVER"), h.Create)
	r.Patch("/bookings/:id/toll-budget", auth, admin, h.SetTollBudget)

	r.Get("/trip-reimbursements/pending", auth, admin, h.PendingReimbursements)
	r.Get("/trip-reimbursements", auth, admin, h.ListReimbursements)
	r.Post("/trip-reimbursements", auth, admin, h.PayReimbursement)
}

func (h *TripExpenseHandler) Config(c *fiber.Ctx) error {
	return util.OK(c, "Trip expense config", h.svc.Config(c.Context()))
}

func (h *TripExpenseHandler) Gates(c *fiber.Ctx) error {
	data, err := h.svc.GateSuggestions(c.Context(), c.Query("q"))
	if err != nil {
		return err
	}
	return util.OK(c, "Toll gates", data)
}

func (h *TripExpenseHandler) ListForBooking(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	data, err := h.svc.ListForBooking(c.Context(), id, int32(middleware.GetUserID(c)), middleware.GetUserRole(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Trip expenses", data)
}

// Create (multipart): proofPhoto (wajib), amount, category, tollPoint |
// entryGate/exitGate/paymentMethod, note, occurredAt (RFC3339, opsional).
func (h *TripExpenseHandler) Create(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	req := service.CreateTripExpenseRequest{
		Category:      c.FormValue("category"),
		TollPoint:     c.FormValue("tollPoint"),
		EntryGate:     c.FormValue("entryGate"),
		ExitGate:      c.FormValue("exitGate"),
		Amount:        util.ParseStringToFloat64(c.FormValue("amount")),
		PaymentMethod: c.FormValue("paymentMethod"),
		Note:          c.FormValue("note"),
	}
	if v := c.FormValue("occurredAt"); v != "" {
		t, perr := time.Parse(time.RFC3339, v)
		if perr != nil {
			return fiber.NewError(fiber.StatusBadRequest, "occurredAt harus RFC3339")
		}
		req.OccurredAt = &t
	}
	fh, ferr := c.FormFile("proofPhoto")
	if ferr != nil {
		return fiber.NewError(fiber.StatusBadRequest, "foto struk / bukti bayar wajib dilampirkan")
	}
	if cerr := util.CheckUpload(fh); cerr != nil {
		return fiber.NewError(fiber.StatusBadRequest,
			fmt.Sprintf("foto (%s) tidak bisa diunggah: %s", fh.Filename, uploadErrorID(cerr)))
	}
	path, serr := util.SaveUploadedFile(fh, "trip_expenses")
	if serr != nil {
		return serr
	}
	req.ProofPhotoURL = "/uploads/" + path

	data, err := h.svc.Create(c.Context(), id, req, middleware.GetUserRole(c), auditActor(c))
	if err != nil {
		util.DeleteUploadedFile(path) // catatan ditolak → foto tidak ditinggal yatim
		return err
	}
	return util.Created(c, "Trip expense recorded", data)
}

func (h *TripExpenseHandler) Void(c *fiber.Ctx) error {
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
	if err := h.svc.Void(c.Context(), id, body.Reason, auditActor(c)); err != nil {
		return err
	}
	return util.OK(c, "Trip expense voided", nil)
}

func (h *TripExpenseHandler) SetTollBudget(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var body struct {
		Amount *float64 `json:"amount"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	data, err := h.svc.SetTollBudget(c.Context(), id, body.Amount, auditActor(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Toll budget updated", data)
}

func (h *TripExpenseHandler) PendingReimbursements(c *fiber.Ctx) error {
	data, err := h.svc.PendingReimbursements(c.Context())
	if err != nil {
		return err
	}
	return util.OK(c, "Pending reimbursements", data)
}

func (h *TripExpenseHandler) ListReimbursements(c *fiber.Ctx) error {
	data, err := h.svc.ListReimbursements(c.Context())
	if err != nil {
		return err
	}
	return util.OK(c, "Reimbursements", data)
}

func (h *TripExpenseHandler) PayReimbursement(c *fiber.Ctx) error {
	var req service.PayTripReimbursementRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.PayReimbursement(c.Context(), req, auditActor(c))
	if err != nil {
		return err
	}
	return util.Created(c, "Reimbursement recorded", data)
}
