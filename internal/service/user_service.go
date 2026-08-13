package service

import (
	"context"
	"database/sql"
	"math"
	"strings"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

type UserService struct {
	q  repository.ExtendedQuerier
	db *sql.DB
}

func NewUserService(db *sql.DB) *UserService {
	return &UserService{
		q:  repository.New(db),
		db: db,
	}
}

type CreateUserRequest struct {
	EmployeeID    string `json:"employeeId"    validate:"required"`
	Name          string `json:"name"          validate:"required"`
	Email         string `json:"email"         validate:"required,email"`
	Password      string `json:"password"      validate:"required,min=8"`
	RoleID        int32  `json:"roleId"        validate:"required"`
	DepartmentID  int32  `json:"departmentId"  validate:"required"`
	LicenseNumber string `json:"licenseNumber"`
	PhoneNumber   string `json:"phoneNumber"`
}

type UpdateUserRequest struct {
	Name          string `json:"name"          validate:"required"`
	Email         string `json:"email"         validate:"required,email"`
	RoleID        int32  `json:"roleId"        validate:"required"`
	DepartmentID  int32  `json:"departmentId"  validate:"required"`
	LicenseNumber string `json:"licenseNumber"`
	PhoneNumber   string `json:"phoneNumber"`
}

func serializeUser(u repository.ListUsersRow) map[string]any {
	return map[string]any{
		"id":           u.ID,
		"employeeId":   u.EmployeeId,
		"name":         u.Name,
		"email":        u.Email,
		"profilePhoto": nullStr(u.ProfilePhoto),
		"isActive":     u.IsActive,
		"role":         map[string]any{"id": u.RoleId, "name": string(u.RoleName)},
		"department":   map[string]any{"id": u.DepartmentId, "name": u.DepartmentName},
		"createdAt":    u.CreatedAt,
	}
}

func (s *UserService) List(ctx context.Context, page, limit int, search *string, roleID *int32, isActive *bool, departmentID *int32) ([]map[string]any, int64, error) {
	params := repository.ListUsersParams{
		Limit:  int32(limit),
		Offset: int32((page - 1) * limit),
	}
	if search != nil {
		params.Search = sql.NullString{String: *search, Valid: true}
	}
	if roleID != nil {
		params.RoleID = sql.NullInt32{Int32: *roleID, Valid: true}
	}
	if isActive != nil {
		params.IsActive = sql.NullBool{Bool: *isActive, Valid: true}
	}
	if departmentID != nil {
		params.DepartmentID = sql.NullInt32{Int32: *departmentID, Valid: true}
	}

	rows, err := s.q.ListUsers(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.q.CountUsers(ctx, repository.CountUsersParams{
		Search: params.Search, RoleID: params.RoleID, IsActive: params.IsActive, DepartmentID: params.DepartmentID,
	})
	if err != nil {
		return nil, 0, err
	}

	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		out[i] = serializeUser(r)
	}
	return out, total, nil
}

func (s *UserService) GetByID(ctx context.Context, id int32) (map[string]any, error) {
	u, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}

	out := map[string]any{
		"id":           u.ID,
		"employeeId":   u.EmployeeId,
		"name":         u.Name,
		"email":        u.Email,
		"profilePhoto": nullStr(u.ProfilePhoto),
		"isActive":     u.IsActive,
		"role":         map[string]any{"id": u.RoleId, "name": string(u.RoleName)},
		"department":   map[string]any{"id": u.DepartmentId, "name": u.DepartmentName},
		"createdAt":    u.CreatedAt,
	}

	if string(u.RoleName) == "DRIVER" {
		if d, err := s.q.GetDriverByUserID(ctx, id); err == nil {
			out["licenseNumber"] = d.LicenseNumber
			out["phoneNumber"] = d.PhoneNumber
		}
	}

	if string(u.RoleName) == "ROOM_KEEPER" {
		if rk, err := s.q.GetRoomKeeperByUserID(ctx, id); err == nil {
			out["phoneNumber"] = rk.PhoneNumber
		}
	}

	return out, nil
}

func (s *UserService) Create(ctx context.Context, req CreateUserRequest) (map[string]any, error) {
	if _, err := s.q.GetUserByEmail(ctx, req.Email); err == nil {
		return nil, util.ErrDuplicate
	}
	hashed, err := util.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	qtx := repository.New(tx)

	user, err := qtx.CreateUser(ctx, repository.CreateUserParams{
		EmployeeId:   req.EmployeeID,
		Name:         req.Name,
		Email:        req.Email,
		Password:     hashed,
		IsActive:     true,
		RoleId:       req.RoleID,
		DepartmentId: req.DepartmentID,
	})
	if err != nil {
		return nil, err
	}
	full, _ := qtx.GetUserByID(ctx, user.ID)

	if string(full.RoleName) == "DRIVER" {
		if req.LicenseNumber == "" || req.PhoneNumber == "" {
			return nil, util.NewError(400, "licenseNumber and phoneNumber are required for DRIVER role", util.ErrBadRequest)
		}

		_, err = qtx.CreateDriver(ctx, repository.CreateDriverParams{
			UserId:        full.ID,
			LicenseNumber: req.LicenseNumber,
			PhoneNumber:   req.PhoneNumber,
		})

		if err != nil {
			return nil, err
		}
	}

	if string(full.RoleName) == "ROOM_KEEPER" {
		if req.PhoneNumber == "" {
			return nil, util.NewError(400, "phoneNumber is required for ROOM_KEEPER role", util.ErrBadRequest)
		}

		_, err = qtx.CreateRoomKeeper(ctx, repository.CreateRoomKeeperParams{
			UserId:      full.ID,
			PhoneNumber: req.PhoneNumber,
		})

		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return map[string]any{
		"id":         full.ID,
		"employeeId": full.EmployeeId,
		"name":       full.Name,
		"email":      full.Email,
		"role":       map[string]any{"id": full.RoleId, "name": string(full.RoleName)},
		"department": map[string]any{"id": full.DepartmentId, "name": full.DepartmentName},
	}, nil
}

func (s *UserService) Update(ctx context.Context, id int32, req UpdateUserRequest) (map[string]any, error) {
	if _, err := s.q.GetUserByID(ctx, id); err != nil {
		return nil, util.ErrNotFound
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	qtx := repository.New(tx)

	_, err = qtx.UpdateUser(ctx, repository.UpdateUserParams{
		ID: id, Name: req.Name, Email: req.Email,
		RoleId: req.RoleID, DepartmentId: req.DepartmentID,
	})
	if err != nil {
		return nil, err
	}

	full, _ := qtx.GetUserByID(ctx, id)

	if string(full.RoleName) == "DRIVER" {
		if req.LicenseNumber == "" || req.PhoneNumber == "" {
			return nil, util.NewError(400, "licenseNumber dan phoneNumber wajib diisi khusus untuk role Driver", nil)
		}

		if d, err := qtx.GetDriverByUserID(ctx, id); err == nil {
			_, err = qtx.UpdateDriver(ctx, repository.UpdateDriverParams{
				ID:            d.ID,
				LicenseNumber: req.LicenseNumber,
				PhoneNumber:   req.PhoneNumber,
			})
			if err != nil {
				return nil, err
			}
		} else {
			_, err = qtx.CreateDriver(ctx, repository.CreateDriverParams{
				UserId:        id,
				LicenseNumber: req.LicenseNumber,
				PhoneNumber:   req.PhoneNumber,
			})
			if err != nil {
				return nil, err
			}
		}
	}

	if string(full.RoleName) == "ROOM_KEEPER" {
		if req.PhoneNumber == "" {
			return nil, util.NewError(400, "phoneNumber wajib diisi khusus untuk role Room Keeper", util.ErrBadRequest)
		}

		if rk, err := qtx.GetRoomKeeperByUserID(ctx, id); err == nil {
			_, err = qtx.UpdateRoomKeeper(ctx, repository.UpdateRoomKeeperParams{
				ID:          rk.ID,
				PhoneNumber: req.PhoneNumber,
			})
			if err != nil {
				return nil, err
			}
		} else {
			_, err = qtx.CreateRoomKeeper(ctx, repository.CreateRoomKeeperParams{
				UserId:      id,
				PhoneNumber: req.PhoneNumber,
			})
			if err != nil {
				return nil, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.GetByID(ctx, id)
}

// ResetPasswordRequest — reset password oleh ADMIN (tanpa OTP).
// Alur user biasa (lupa password) tetap lewat /auth: forgot-password → verify-otp → reset-password.
type AdminResetPasswordRequest struct {
	NewPassword string `json:"newPassword" validate:"required,min=8"`
}

// ResetPassword mengganti password user secara langsung. Hanya untuk ADMIN —
// tidak ada verifikasi OTP karena admin dianggap sudah terautentikasi & berwenang.
func (s *UserService) ResetPassword(ctx context.Context, id int32, newPassword string) error {
	if _, err := s.q.GetUserByID(ctx, id); err != nil {
		return util.ErrNotFound
	}
	hashed, err := util.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.q.UpdateUserPassword(ctx, repository.UpdateUserPasswordParams{
		Password: hashed,
		ID:       id,
	})
}

func (s *UserService) ToggleActive(ctx context.Context, id int32) (map[string]any, error) {
	if _, err := s.q.GetUserByID(ctx, id); err != nil {
		return nil, util.ErrNotFound
	}
	_, err := s.q.ToggleUserActive(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *UserService) Delete(ctx context.Context, id int32) error {
	if _, err := s.q.GetUserByID(ctx, id); err != nil {
		return util.ErrNotFound
	}
	return s.q.DeleteUser(ctx, id)
}

func (s *UserService) UpdateProfilePhoto(ctx context.Context, id int32, path string) (map[string]any, error) {
	_, err := s.q.UpdateProfilePhoto(ctx, repository.UpdateProfilePhotoParams{
		ID:           id,
		ProfilePhoto: sql.NullString{String: path, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *UserService) DeleteProfilePhoto(ctx context.Context, id int32) error {
	_, err := s.q.DeleteProfilePhoto(ctx, id)
	return err
}

// ─── Summary ──────────────────────────────────────────────────────────────────

// percentOf mengembalikan porsi n terhadap total dalam persen (2 desimal).
func percentOf(n, total int64) float64 {
	if total == 0 {
		return 0
	}
	return math.Round(float64(n)/float64(total)*10000) / 100
}

// Summary merangkum jumlah user: total keseluruhan, rincian per role,
// per departemen, dan matriks role × departemen.
func (s *UserService) Summary(ctx context.Context) (map[string]any, error) {
	totals, err := s.q.UserSummaryTotals(ctx)
	if err != nil {
		return nil, err
	}

	roleRows, err := s.q.UserSummaryByRole(ctx)
	if err != nil {
		return nil, err
	}
	deptRows, err := s.q.UserSummaryByDepartment(ctx)
	if err != nil {
		return nil, err
	}
	matrixRows, err := s.q.UserSummaryByRoleDepartment(ctx)
	if err != nil {
		return nil, err
	}

	// Rincian per role + versi ringkas {ROLE: total} agar mudah dipakai front-end.
	byRole := make([]map[string]any, len(roleRows))
	roleCount := make(map[string]int64, len(roleRows))
	for i, r := range roleRows {
		roleCount[string(r.RoleName)] = r.Total
		byRole[i] = map[string]any{
			"roleId":     r.RoleID,
			"role":       string(r.RoleName),
			"total":      r.Total,
			"active":     r.Active,
			"inactive":   r.Inactive,
			"percentage": percentOf(r.Total, totals.TotalUsers),
		}
	}

	byDepartment := make([]map[string]any, len(deptRows))
	for i, d := range deptRows {
		byDepartment[i] = map[string]any{
			"departmentId": d.DepartmentID,
			"department":   d.DepartmentName,
			"total":        d.Total,
			"active":       d.Active,
			"inactive":     d.Inactive,
			"percentage":   percentOf(d.Total, totals.TotalUsers),
		}
	}

	// Matriks digabung menjadi satu entri per role berisi daftar departemennya.
	type roleGroup struct {
		payload map[string]any
		depts   []map[string]any
	}
	groups := make(map[int32]*roleGroup, len(roleRows))
	order := make([]int32, 0, len(roleRows))
	for _, m := range matrixRows {
		g, ok := groups[m.RoleID]
		if !ok {
			g = &roleGroup{payload: map[string]any{
				"roleId": m.RoleID,
				"role":   string(m.RoleName),
			}}
			groups[m.RoleID] = g
			order = append(order, m.RoleID)
		}
		g.depts = append(g.depts, map[string]any{
			"departmentId": m.DepartmentID,
			"department":   m.DepartmentName,
			"total":        m.Total,
			"active":       m.Active,
			"inactive":     m.Inactive,
		})
	}
	byRoleDepartment := make([]map[string]any, 0, len(order))
	for _, id := range order {
		g := groups[id]
		g.payload["departments"] = g.depts
		byRoleDepartment = append(byRoleDepartment, g.payload)
	}

	return map[string]any{
		"totals": map[string]any{
			"total":            totals.TotalUsers,
			"active":           totals.ActiveUsers,
			"inactive":         totals.InactiveUsers,
			"withProfilePhoto": totals.WithPhoto,
			"newThisMonth":     totals.NewThisMonth,
			"newLast30Days":    totals.NewLast30Days,
			"totalRoles":       len(roleRows),
			"totalDepartments": len(deptRows),
		},
		"roleCount":        roleCount,
		"byRole":           byRole,
		"byDepartment":     byDepartment,
		"byRoleDepartment": byRoleDepartment,
	}, nil
}

func (s *UserService) ListRoles(ctx context.Context) (any, error) {
	return s.q.ListRoles(ctx)
}

func (s *UserService) ListDepartments(ctx context.Context) (any, error) {
	return s.q.ListDepartments(ctx)
}

func (s *UserService) CreateDepartment(ctx context.Context, name string) (any, error) {
	d, err := s.q.CreateDepartment(ctx, name)
	if err != nil {
		return nil, util.ErrDuplicate
	}
	return d, nil
}

func (s *UserService) UpdateDepartment(ctx context.Context, id int32, name string) (any, error) {
	if _, err := s.q.GetDepartmentByID(ctx, id); err != nil {
		return nil, util.ErrNotFound
	}
	d, err := s.q.UpdateDepartment(ctx, repository.UpdateDepartmentParams{ID: id, Name: name})
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (s *UserService) DeleteDepartment(ctx context.Context, id int32) error {
	if _, err := s.q.GetDepartmentByID(ctx, id); err != nil {
		return util.ErrNotFound
	}
	return s.q.DeleteDepartment(ctx, id)
}

// ─── Bulk Import (Excel) ──────────────────────────────────────────────────────

// DefaultBulkPassword dipakai bila kolom password di Excel dikosongkan.
const DefaultBulkPassword = "Password123!"

// BulkUserRow = satu baris data dari Excel (role & department berupa NAMA, bukan id).
type BulkUserRow struct {
	Row            int // nomor baris di file Excel (untuk pesan error)
	EmployeeID     string
	Name           string
	Email          string
	Password       string
	RoleName       string
	DepartmentName string
}

// BulkImportRowResult = hasil per baris agar admin tahu baris mana yang gagal.
type BulkImportRowResult struct {
	Row        int    `json:"row"`
	EmployeeID string `json:"employeeId"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

// BulkImport memvalidasi & menyimpan banyak user sekaligus.
// Baris yang gagal TIDAK membatalkan baris lain — hasil dilaporkan per baris.
func (s *UserService) BulkImport(ctx context.Context, rows []BulkUserRow) (map[string]any, error) {
	if len(rows) == 0 {
		return nil, util.NewError(400, "no data rows found in the file", util.ErrBadRequest)
	}

	// Peta nama → id (case-insensitive) untuk role & department.
	roles, err := s.q.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	roleByName := make(map[string]int32, len(roles))
	roleNames := make([]string, 0, len(roles))
	for _, r := range roles {
		roleByName[strings.ToUpper(strings.TrimSpace(string(r.Name)))] = r.ID
		roleNames = append(roleNames, string(r.Name))
	}

	depts, err := s.q.ListDepartments(ctx)
	if err != nil {
		return nil, err
	}
	deptByName := make(map[string]int32, len(depts))
	for _, d := range depts {
		deptByName[strings.ToLower(strings.TrimSpace(d.Name))] = d.ID
	}

	results := make([]BulkImportRowResult, 0, len(rows))
	seenEmail := make(map[string]bool, len(rows))
	seenEmp := make(map[string]bool, len(rows))
	successCount := 0

	for _, r := range rows {
		employeeID := strings.TrimSpace(r.EmployeeID)
		name := strings.TrimSpace(r.Name)
		email := strings.ToLower(strings.TrimSpace(r.Email))
		roleName := strings.ToUpper(strings.TrimSpace(r.RoleName))
		deptName := strings.ToLower(strings.TrimSpace(r.DepartmentName))
		password := strings.TrimSpace(r.Password)

		res := BulkImportRowResult{Row: r.Row, EmployeeID: employeeID, Name: name, Email: email}

		fail := func(msg string) {
			res.Success = false
			res.Error = msg
			results = append(results, res)
		}

		switch {
		case employeeID == "":
			fail("employeeId wajib diisi")
			continue
		case name == "":
			fail("name wajib diisi")
			continue
		case email == "":
			fail("email wajib diisi")
			continue
		case !strings.Contains(email, "@") || !strings.Contains(email, "."):
			fail("format email tidak valid")
			continue
		}

		if password == "" {
			password = DefaultBulkPassword
		} else if len(password) < 8 {
			fail("password minimal 8 karakter")
			continue
		}

		roleID, ok := roleByName[roleName]
		if !ok {
			fail("role tidak dikenal: '" + r.RoleName + "' (pilihan: " + strings.Join(roleNames, ", ") + ")")
			continue
		}
		deptID, ok := deptByName[deptName]
		if !ok {
			fail("departemen tidak ditemukan: '" + r.DepartmentName + "'")
			continue
		}

		// Duplikat di dalam file itu sendiri.
		if seenEmail[email] {
			fail("email duplikat di dalam file")
			continue
		}
		if seenEmp[strings.ToLower(employeeID)] {
			fail("employeeId duplikat di dalam file")
			continue
		}

		// Duplikat terhadap data yang sudah ada.
		if _, err := s.q.GetUserByEmail(ctx, email); err == nil {
			fail("email sudah terdaftar")
			continue
		}

		hashed, herr := util.HashPassword(password)
		if herr != nil {
			fail("gagal memproses password")
			continue
		}

		if _, cerr := s.q.CreateUser(ctx, repository.CreateUserParams{
			EmployeeId:   employeeID,
			Name:         name,
			Email:        email,
			Password:     hashed,
			IsActive:     true,
			RoleId:       roleID,
			DepartmentId: deptID,
		}); cerr != nil {
			// employeeId & email UNIQUE di DB — terjemahkan agar mudah dipahami.
			msg := cerr.Error()
			switch {
			case strings.Contains(msg, "employeeId"):
				fail("employeeId sudah terdaftar")
			case strings.Contains(msg, "email"):
				fail("email sudah terdaftar")
			default:
				fail("gagal menyimpan: " + msg)
			}
			continue
		}

		seenEmail[email] = true
		seenEmp[strings.ToLower(employeeID)] = true
		successCount++
		res.Success = true
		results = append(results, res)
	}

	return map[string]any{
		"total":        len(rows),
		"successCount": successCount,
		"failedCount":  len(rows) - successCount,
		"results":      results,
	}, nil
}

// BulkImportRefData = daftar role & department valid untuk sheet referensi template.
func (s *UserService) BulkImportRefData(ctx context.Context) (roles []string, departments []string, err error) {
	rs, err := s.q.ListRoles(ctx)
	if err != nil {
		return nil, nil, err
	}
	for _, r := range rs {
		roles = append(roles, string(r.Name))
	}
	ds, err := s.q.ListDepartments(ctx)
	if err != nil {
		return nil, nil, err
	}
	for _, d := range ds {
		departments = append(departments, d.Name)
	}
	return roles, departments, nil
}
