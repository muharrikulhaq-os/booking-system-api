package http

import (
	"bytes"
	"path/filepath"
	"strconv"
	"strings"

	"booking-system-api/internal/middleware"
	"booking-system-api/internal/service"
	"booking-system-api/internal/util"

	"github.com/gofiber/fiber/v2"
	"github.com/xuri/excelize/v2"
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
	g.Get("/bulk-template", admin, h.BulkTemplate)
	g.Post("/bulk-import", admin, h.BulkImport)
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

// ─── Bulk Import (Excel) ──────────────────────────────────────────────────────

// bulkColumnAliases memetakan berbagai penulisan header ke field kanonik,
// sehingga urutan/penamaan kolom di Excel tidak harus persis.
var bulkColumnAliases = map[string]string{
	"employeeid": "employeeId", "employee id": "employeeId", "nip": "employeeId",
	"id karyawan": "employeeId", "idkaryawan": "employeeId",
	"name": "name", "nama": "name", "nama lengkap": "name",
	"email": "email", "e-mail": "email", "surel": "email",
	"password": "password", "kata sandi": "password", "sandi": "password",
	"role": "role", "peran": "role", "jabatan": "role",
	"department": "department", "departemen": "department", "divisi": "department",
}

func normalizeHeader(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// BulkTemplate mengunduh file Excel template beserta sheet referensi
// (daftar role & departemen yang valid).
func (h *UserHandler) BulkTemplate(c *fiber.Ctx) error {
	roles, departments, err := h.svc.BulkImportRefData(c.Context())
	if err != nil {
		return err
	}

	f := excelize.NewFile()
	defer f.Close()

	const sheet = "Users"
	f.SetSheetName(f.GetSheetName(0), sheet)

	headers := []string{"employeeId", "name", "email", "password", "role", "department"}
	for i, hd := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, hd)
	}

	// Contoh baris (memakai role & departemen pertama yang tersedia).
	exampleRole := "EMPLOYEE"
	if len(roles) > 0 {
		exampleRole = roles[0]
	}
	exampleDept := "Information Technology"
	if len(departments) > 0 {
		exampleDept = departments[0]
	}
	example := []any{"EMP001", "Budi Santoso", "budi@perusahaan.com", "Password123!", exampleRole, exampleDept}
	for i, v := range example {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		_ = f.SetCellValue(sheet, cell, v)
	}

	for i, w := range []float64{16, 26, 30, 18, 16, 26} {
		col, _ := excelize.ColumnNumberToName(i + 1)
		_ = f.SetColWidth(sheet, col, col, w)
	}

	// Sheet referensi: role & departemen valid + catatan.
	const ref = "Referensi"
	if _, err := f.NewSheet(ref); err != nil {
		return err
	}
	_ = f.SetCellValue(ref, "A1", "ROLE (valid)")
	for i, r := range roles {
		_ = f.SetCellValue(ref, "A"+strconv.Itoa(i+2), r)
	}
	_ = f.SetCellValue(ref, "C1", "DEPARTEMEN (valid)")
	for i, d := range departments {
		_ = f.SetCellValue(ref, "C"+strconv.Itoa(i+2), d)
	}
	_ = f.SetCellValue(ref, "E1", "Catatan")
	_ = f.SetCellValue(ref, "E2", "Hapus baris contoh sebelum mengunggah.")
	_ = f.SetCellValue(ref, "E3", "password boleh dikosongkan → otomatis "+service.DefaultBulkPassword)
	_ = f.SetCellValue(ref, "E4", "role & department ditulis sesuai daftar di sheet ini.")
	_ = f.SetColWidth(ref, "A", "A", 20)
	_ = f.SetColWidth(ref, "C", "C", 28)
	_ = f.SetColWidth(ref, "E", "E", 60)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return err
	}
	c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set("Content-Disposition", `attachment; filename="template-import-users.xlsx"`)
	return c.Send(buf.Bytes())
}

// BulkImport menerima file Excel (field "file") dan membuat user secara massal.
func (h *UserHandler) BulkImport(c *fiber.Ctx) error {
	fh, err := c.FormFile("file")
	if err != nil || fh == nil {
		return fiber.NewError(fiber.StatusBadRequest, "file is required")
	}
	if ext := strings.ToLower(filepath.Ext(fh.Filename)); ext != ".xlsx" && ext != ".xlsm" {
		return fiber.NewError(fiber.StatusBadRequest, "file must be .xlsx")
	}

	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	var buf bytes.Buffer
	if _, err = buf.ReadFrom(src); err != nil {
		return err
	}
	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "cannot read excel file")
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "excel file has no sheet")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "cannot read rows")
	}
	if len(rows) < 2 {
		return fiber.NewError(fiber.StatusBadRequest, "no data rows found (baris pertama dianggap header)")
	}

	// Petakan posisi kolom dari header.
	idx := map[string]int{}
	for i, cell := range rows[0] {
		if canon, ok := bulkColumnAliases[normalizeHeader(cell)]; ok {
			if _, dup := idx[canon]; !dup {
				idx[canon] = i
			}
		}
	}
	for _, required := range []string{"employeeId", "name", "email", "role", "department"} {
		if _, ok := idx[required]; !ok {
			return fiber.NewError(fiber.StatusBadRequest,
				"kolom '"+required+"' tidak ditemukan di header. Gunakan template yang disediakan.")
		}
	}

	at := func(row []string, key string) string {
		i, ok := idx[key]
		if !ok || i >= len(row) {
			return ""
		}
		return row[i]
	}

	parsed := make([]service.BulkUserRow, 0, len(rows)-1)
	for n, row := range rows[1:] {
		// Lewati baris yang benar-benar kosong.
		empty := true
		for _, cell := range row {
			if strings.TrimSpace(cell) != "" {
				empty = false
				break
			}
		}
		if empty {
			continue
		}
		parsed = append(parsed, service.BulkUserRow{
			Row:            n + 2, // +2: header di baris 1, index mulai 0
			EmployeeID:     at(row, "employeeId"),
			Name:           at(row, "name"),
			Email:          at(row, "email"),
			Password:       at(row, "password"),
			RoleName:       at(row, "role"),
			DepartmentName: at(row, "department"),
		})
	}

	data, err := h.svc.BulkImport(c.Context(), parsed)
	if err != nil {
		return err
	}
	return util.OK(c, "Bulk import selesai", data)
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
