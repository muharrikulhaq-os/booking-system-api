package http

import (
	"path/filepath"
	"strings"

	"booking-system-api/internal/middleware"
	"booking-system-api/internal/service"
	"booking-system-api/internal/util"

	"github.com/gofiber/fiber/v2"
)

type UserHandler struct {
	svc        *service.UserService
	attachSvc  *service.AttachmentService
}

func NewUserHandler(svc *service.UserService, attachSvc *service.AttachmentService) *UserHandler {
	return &UserHandler{svc: svc, attachSvc: attachSvc}
}

func (h *UserHandler) Register(r fiber.Router) {
	auth := middleware.Auth()
	admin := middleware.RequireRole("ADMIN")

	g := r.Group("/users", auth)
	g.Get("", admin, h.List)
	g.Get("/me", h.GetMe)
	g.Get("/roles", h.ListRoles)
	g.Get("/departments", h.ListDepartments)
	g.Post("/departments", admin, h.CreateDepartment)
	g.Put("/departments/:id", admin, h.UpdateDepartment)
	g.Delete("/departments/:id", admin, h.DeleteDepartment)
	g.Get("/:id", admin, h.GetByID)
	g.Post("", admin, h.Create)
	g.Put("/:id", admin, h.Update)
	g.Patch("/:id/toggle-active", admin, h.ToggleActive)
	g.Delete("/:id", admin, h.Delete)
	g.Put("/me/profile-photo", h.UploadMyPhoto)
	g.Delete("/me/profile-photo", h.DeleteMyPhoto)
	g.Put("/:id/profile-photo", admin, h.UploadUserPhoto)
}

func (h *UserHandler) List(c *fiber.Ctx) error {
	page := queryInt(c, "page", 1)
	limit := queryInt(c, "limit", 20)
	data, total, err := h.svc.List(c.Context(), page, limit, queryString(c, "search"), queryInt32(c, "roleId"), nil, queryInt32(c, "departmentId"))
	if err != nil {
		return err
	}
	return util.Paginated(c, "Users retrieved", data, total, page, limit)
}

func (h *UserHandler) GetMe(c *fiber.Ctx) error {
	data, err := h.svc.GetByID(c.Context(), int32(middleware.GetUserID(c)))
	if err != nil {
		return err
	}
	return util.OK(c, "Profile retrieved", data)
}

func (h *UserHandler) GetByID(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	data, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		return err
	}
	return util.OK(c, "User retrieved", data)
}

func (h *UserHandler) Create(c *fiber.Ctx) error {
	var req service.CreateUserRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.Create(c.Context(), req)
	if err != nil {
		return err
	}
	return util.Created(c, "User created", data)
}

func (h *UserHandler) Update(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var req service.UpdateUserRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.Update(c.Context(), id, req)
	if err != nil {
		return err
	}
	return util.OK(c, "User updated", data)
}

func (h *UserHandler) ToggleActive(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	data, err := h.svc.ToggleActive(c.Context(), id)
	if err != nil {
		return err
	}
	return util.OK(c, "User status toggled", data)
}

func (h *UserHandler) Delete(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := h.svc.Delete(c.Context(), id); err != nil {
		return err
	}
	return util.OK(c, "User deleted", nil)
}

func (h *UserHandler) ListRoles(c *fiber.Ctx) error {
	data, err := h.svc.ListRoles(c.Context())
	if err != nil {
		return err
	}
	return util.OK(c, "Roles retrieved", data)
}

func (h *UserHandler) ListDepartments(c *fiber.Ctx) error {
	data, err := h.svc.ListDepartments(c.Context())
	if err != nil {
		return err
	}
	return util.OK(c, "Departments retrieved", data)
}

func (h *UserHandler) CreateDepartment(c *fiber.Ctx) error {
	var body struct {
		Name string `json:"name" validate:"required"`
	}
	if err := bindAndValidate(c, &body); err != nil {
		return err
	}
	data, err := h.svc.CreateDepartment(c.Context(), body.Name)
	if err != nil {
		return err
	}
	return util.Created(c, "Department created", data)
}

func (h *UserHandler) UpdateDepartment(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var body struct {
		Name string `json:"name" validate:"required"`
	}
	if err := bindAndValidate(c, &body); err != nil {
		return err
	}
	data, err := h.svc.UpdateDepartment(c.Context(), id, body.Name)
	if err != nil {
		return err
	}
	return util.OK(c, "Department updated", data)
}

func (h *UserHandler) DeleteDepartment(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := h.svc.DeleteDepartment(c.Context(), id); err != nil {
		return err
	}
	return util.OK(c, "Department deleted", nil)
}

func (h *UserHandler) UploadMyPhoto(c *fiber.Ctx) error {
	fh, err := c.FormFile("photo")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "photo file is required")
	}
	savedPath, err := util.SaveUploadedFile(fh, "profile")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	userID := int32(middleware.GetUserID(c))
	data, err := h.svc.UpdateProfilePhoto(c.Context(), userID, savedPath)
	if err != nil {
		util.DeleteUploadedFile(savedPath)
		return err
	}
	return util.OK(c, "Profile photo updated", data)
}

func (h *UserHandler) DeleteMyPhoto(c *fiber.Ctx) error {
	userID := int32(middleware.GetUserID(c))
	if err := h.svc.DeleteProfilePhoto(c.Context(), userID); err != nil {
		return err
	}
	return util.OK(c, "Profile photo deleted", nil)
}

func (h *UserHandler) UploadUserPhoto(c *fiber.Ctx) error {
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	fh, err := c.FormFile("photo")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "photo file is required")
	}
	savedPath, saveErr := util.SaveUploadedFile(fh, "profile")
	if saveErr != nil {
		return fiber.NewError(fiber.StatusBadRequest, saveErr.Error())
	}
	data, err := h.svc.UpdateProfilePhoto(c.Context(), id, savedPath)
	if err != nil {
		util.DeleteUploadedFile(savedPath)
		return err
	}
	return util.OK(c, "Profile photo updated", data)
}

// ServeFile serves uploaded files with basic path traversal protection.
// Images are served inline (displayed in browser); PDFs and other types trigger download.
func ServeFile(uploadDir string) fiber.Handler {
	imageTypes := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
	}
	return func(c *fiber.Ctx) error {
		rel := c.Params("*")
		clean := filepath.Clean(rel)
		if len(clean) == 0 || clean[0] == '.' {
			return fiber.NewError(fiber.StatusForbidden, "invalid path")
		}
		abs := filepath.Join(uploadDir, clean)
		ext := strings.ToLower(filepath.Ext(abs))
		if imageTypes[ext] {
			c.Set("Content-Disposition", "inline")
		}
		return c.SendFile(abs)
	}
}
