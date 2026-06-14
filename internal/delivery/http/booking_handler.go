package http

import (
	"time"

	"booking-system-api/internal/middleware"
	"booking-system-api/internal/service"
	"booking-system-api/internal/util"

	"github.com/gofiber/fiber/v2"
)

type BookingHandler struct {
	svc       *service.BookingService
	attachSvc *service.AttachmentService
}

func NewBookingHandler(svc *service.BookingService, attachSvc *service.AttachmentService) *BookingHandler {
	return &BookingHandler{svc: svc, attachSvc: attachSvc}
}

func (h *BookingHandler) Register(r fiber.Router) {
	auth := middleware.Auth()
	admin := middleware.RequireRole("ADMIN")
	adminOrDriver := middleware.RequireRole("ADMIN", "DRIVER", "ROOM_KEEPER")
	adminOrRoomKeeper := middleware.RequireRole("ADMIN", "ROOM_KEEPER")

	g := r.Group("/bookings", auth)
	g.Get("", h.List)
	g.Get("/drivers/:driver_id/ratings", admin, h.GetDriverRatings)
	g.Get("/:id", h.GetByID)
	g.Post("", h.Create)
	g.Patch("/:id/cancel", h.Cancel)
	g.Post("/:id/approve", admin, h.Approve)
	g.Post("/:id/reject", admin, h.Reject)
	g.Patch("/:id/substitute-resource", admin, h.SubstituteResource)
	g.Post("/:id/assign-vehicle", admin, h.AssignVehicle)
	g.Patch("/:id/start", adminOrDriver, h.Start)
	g.Patch("/:id/complete", adminOrRoomKeeper, h.Complete)
	g.Post("/:id/merge", admin, h.MergeBooking)
	g.Get("/:id/merge-info", auth, h.MergeInfo)
	g.Post("/:id/rate-driver", h.RateDriver)
	g.Get("/:id/approval-log", admin, h.ApprovalLog)
	g.Get("/:id/activity", auth, h.Activity)
	g.Get("/:id/attachments", h.ListAttachments)
	g.Post("/:id/attachments", h.UploadAttachment)
	g.Post("/:id/return-report", middleware.RequireRole("DRIVER"), h.SubmitReturnReport)
	g.Get("/:id/return-report", h.GetReturnReport)
}

func (h *BookingHandler) List(c *fiber.Ctx) error {
	page := queryInt(c, "page", 1)
	limit := queryInt(c, "limit", 20)

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

	data, total, err := h.svc.List(c.Context(), page, limit,
		queryInt32(c, "userId"), queryString(c, "status"),
		queryInt32(c, "resourceId"), queryString(c, "resourceType"),
		queryInt32(c, "driverId"), startFrom, endTo,
		middleware.GetUserID(c), middleware.GetUserRole(c),
		queryString(c, "search"),
	)
	if err != nil {
		return err
	}
	return util.Paginated(c, "Bookings retrieved", data, total, page, limit)
}

func (h *BookingHandler) GetByID(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	data, err := h.svc.GetByID(c.Context(), id, middleware.GetUserID(c), middleware.GetUserRole(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Booking retrieved", data)
}

func (h *BookingHandler) Create(c *fiber.Ctx) error {
	var req service.CreateBookingRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.Create(c.Context(), req, middleware.GetUserID(c))
	if err != nil {
		return err
	}
	return util.Created(c, "Booking created", data)
}

func (h *BookingHandler) Cancel(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	data, err := h.svc.Cancel(c.Context(), id, middleware.GetUserID(c), middleware.GetUserRole(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Booking cancelled", data)
}

func (h *BookingHandler) Approve(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var req service.ApproveBookingRequest
	_ = c.BodyParser(&req)
	data, err := h.svc.Approve(c.Context(), id, req, middleware.GetUserID(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Booking approved", data)
}

func (h *BookingHandler) Reject(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var req service.RejectBookingRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.Reject(c.Context(), id, req, middleware.GetUserID(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Booking rejected", data)
}

func (h *BookingHandler) SubstituteResource(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var req service.SubstituteResourceRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.SubstituteResource(c.Context(), id, req, middleware.GetUserID(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Resource substituted", data)
}

func (h *BookingHandler) AssignVehicle(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var req service.AssignVehicleRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.AssignVehicle(c.Context(), id, req, middleware.GetUserID(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Vehicle assigned", data)
}

func (h *BookingHandler) Start(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	data, err := h.svc.Start(c.Context(), id, middleware.GetUserID(c), middleware.GetUserRole(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Booking started", data)
}

func (h *BookingHandler) Complete(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	data, err := h.svc.Complete(c.Context(), id, middleware.GetUserID(c), middleware.GetUserRole(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Booking completed", data)
}

func (h *BookingHandler) MergeBooking(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var req service.MergeBookingRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.MergeBookings(c.Context(), id, req, middleware.GetUserID(c))
	if err != nil {
		return err
	}
	return util.Created(c, "Bookings merged", data)
}

func (h *BookingHandler) MergeInfo(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	data, err := h.svc.GetMergeInfo(c.Context(), id, middleware.GetUserID(c), middleware.GetUserRole(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Merge info retrieved", data)
}

func (h *BookingHandler) RateDriver(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var req service.RateDriverRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.RateDriver(c.Context(), id, req, middleware.GetUserID(c))
	if err != nil {
		return err
	}
	return util.Created(c, "Driver rated", data)
}

func (h *BookingHandler) GetDriverRatings(c *fiber.Ctx) error {
	id, err := parseID(c, "driver_id")
	if err != nil {
		return err
	}
	data, err := h.svc.GetDriverRatings(c.Context(), id)
	if err != nil {
		return err
	}
	return util.OK(c, "Driver ratings retrieved", data)
}

func (h *BookingHandler) ApprovalLog(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	data, err := h.svc.GetApprovalLog(c.Context(), id)
	if err != nil {
		return err
	}
	return util.OK(c, "Approval log retrieved", data)
}

func (h *BookingHandler) Activity(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	data, err := h.svc.GetActivity(c.Context(), id, middleware.GetUserID(c), middleware.GetUserRole(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Activity log retrieved", data)
}

func (h *BookingHandler) ListAttachments(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	data, err := h.attachSvc.ListByBooking(c.Context(), id)
	if err != nil {
		return err
	}
	return util.OK(c, "Attachments retrieved", data)
}

func (h *BookingHandler) UploadAttachment(c *fiber.Ctx) error {
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
	data, err := h.attachSvc.UploadForBooking(c.Context(), id, uploaderID, fh, desc)
	if err != nil {
		return err
	}
	return util.Created(c, "Attachment uploaded", data)
}

func (h *BookingHandler) SubmitReturnReport(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}

	note := c.FormValue("note")
	location := c.FormValue("location")
	if note == "" {
		return fiber.NewError(fiber.StatusBadRequest, "note is required")
	}
	if location == "" {
		return fiber.NewError(fiber.StatusBadRequest, "location is required")
	}

	uploaderID := int32(middleware.GetUserID(c))
	if err = h.svc.SubmitReturnReport(c.Context(), id, note, location, middleware.GetUserID(c)); err != nil {
		return err
	}

	// upload photos (field name "photos[]" or "photo", multiple allowed)
	form, formErr := c.MultipartForm()
	uploadedPhotos := make([]map[string]any, 0)
	if formErr == nil {
		for _, fh := range form.File["photos[]"] {
			p, pErr := h.attachSvc.UploadReturnPhoto(c.Context(), id, uploaderID, fh)
			if pErr == nil {
				uploadedPhotos = append(uploadedPhotos, p)
			}
		}
		// also accept field name "photo" (single)
		for _, fh := range form.File["photo"] {
			p, pErr := h.attachSvc.UploadReturnPhoto(c.Context(), id, uploaderID, fh)
			if pErr == nil {
				uploadedPhotos = append(uploadedPhotos, p)
			}
		}
	}

	return util.Created(c, "Return report submitted", map[string]any{
		"bookingId": id,
		"note":      note,
		"location":  location,
		"photos":    uploadedPhotos,
	})
}

func (h *BookingHandler) GetReturnReport(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	data, err := h.svc.GetReturnReport(c.Context(), id, middleware.GetUserID(c), middleware.GetUserRole(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Return report retrieved", data)
}
