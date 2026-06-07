package http

import (
	"booking-system-api/internal/middleware"
	"booking-system-api/internal/service"
	"booking-system-api/internal/util"

	"github.com/gofiber/fiber/v2"
)

type RoomHandler struct {
	svc       *service.RoomService
	attachSvc *service.AttachmentService
}

func NewRoomHandler(svc *service.RoomService, attachSvc *service.AttachmentService) *RoomHandler {
	return &RoomHandler{svc: svc, attachSvc: attachSvc}
}

func (h *RoomHandler) Register(r fiber.Router) {
	auth := middleware.Auth()
	admin := middleware.RequireRole("ADMIN")

	g := r.Group("/rooms", auth)
	g.Get("", h.List)
	g.Get("/:id", h.GetByID)
	g.Post("", admin, h.Create)
	g.Put("/:id", admin, h.Update)
	g.Patch("/:id/status", admin, h.UpdateStatus)
	g.Patch("/:id/photo", admin, h.UpdatePhoto)
	g.Delete("/:id", admin, h.Delete)
	g.Get("/:id/attachments", h.ListAttachments)
	g.Post("/:id/attachments", admin, h.UploadAttachment)
}

func (h *RoomHandler) List(c *fiber.Ctx) error {
	page := queryInt(c, "page", 1)
	limit := queryInt(c, "limit", 20)
	data, total, err := h.svc.List(c.Context(), page, limit, queryString(c, "search"), queryString(c, "status"))
	if err != nil {
		return err
	}
	return util.Paginated(c, "Rooms retrieved", data, total, page, limit)
}

func (h *RoomHandler) GetByID(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	data, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		return err
	}
	return util.OK(c, "Room retrieved", data)
}

func (h *RoomHandler) Create(c *fiber.Ctx) error {
	var req service.CreateRoomRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.Create(c.Context(), req)
	if err != nil {
		return err
	}
	return util.Created(c, "Room created", data)
}

func (h *RoomHandler) Update(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var req service.UpdateRoomRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.Update(c.Context(), id, req)
	if err != nil {
		return err
	}
	return util.OK(c, "Room updated", data)
}

func (h *RoomHandler) UpdateStatus(c *fiber.Ctx) error {
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
	return util.OK(c, "Room status updated", data)
}

func (h *RoomHandler) UpdatePhoto(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	fh, err := c.FormFile("photo")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "photo file is required")
	}
	savedPath, saveErr := util.SaveUploadedFile(fh, "room")
	if saveErr != nil {
		return fiber.NewError(fiber.StatusBadRequest, saveErr.Error())
	}
	data, err := h.svc.UpdatePhoto(c.Context(), id, "/files/"+savedPath)
	if err != nil {
		util.DeleteUploadedFile(savedPath)
		return err
	}
	return util.OK(c, "Room photo updated", data)
}

func (h *RoomHandler) Delete(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := h.svc.Delete(c.Context(), id); err != nil {
		return err
	}
	return util.OK(c, "Room deleted", nil)
}

func (h *RoomHandler) ListAttachments(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	data, err := h.attachSvc.ListByRoom(c.Context(), id)
	if err != nil {
		return err
	}
	return util.OK(c, "Attachments retrieved", data)
}

func (h *RoomHandler) UploadAttachment(c *fiber.Ctx) error {
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
	data, err := h.attachSvc.UploadForRoom(c.Context(), id, uploaderID, fh, desc)
	if err != nil {
		return err
	}
	return util.Created(c, "Attachment uploaded", data)
}
