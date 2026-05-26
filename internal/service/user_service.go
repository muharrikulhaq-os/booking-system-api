package service

import (
	"context"
	"database/sql"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

type UserService struct {
	q *repository.Queries
}

func NewUserService(db *sql.DB) *UserService {
	return &UserService{q: repository.New(db)}
}

type CreateUserRequest struct {
	EmployeeID   string `json:"employeeId"   validate:"required"`
	Name         string `json:"name"         validate:"required"`
	Email        string `json:"email"        validate:"required,email"`
	Password     string `json:"password"     validate:"required,min=8"`
	RoleID       int32  `json:"roleId"       validate:"required"`
	DepartmentID int32  `json:"departmentId" validate:"required"`
}

type UpdateUserRequest struct {
	Name         string `json:"name"         validate:"required"`
	Email        string `json:"email"        validate:"required,email"`
	RoleID       int32  `json:"roleId"       validate:"required"`
	DepartmentID int32  `json:"departmentId" validate:"required"`
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

func (s *UserService) List(ctx context.Context, page, limit int, search *string, roleID *int32, isActive *bool) ([]map[string]any, int64, error) {
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

	rows, err := s.q.ListUsers(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.q.CountUsers(ctx, repository.CountUsersParams{
		Search: params.Search, RoleID: params.RoleID, IsActive: params.IsActive,
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
	}, nil
}

func (s *UserService) Create(ctx context.Context, req CreateUserRequest) (map[string]any, error) {
	if _, err := s.q.GetUserByEmail(ctx, req.Email); err == nil {
		return nil, util.ErrDuplicate
	}
	hashed, err := util.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	user, err := s.q.CreateUser(ctx, repository.CreateUserParams{
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
	full, _ := s.q.GetUserByID(ctx, user.ID)
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
	_, err := s.q.UpdateUser(ctx, repository.UpdateUserParams{
		ID: id, Name: req.Name, Email: req.Email,
		RoleId: req.RoleID, DepartmentId: req.DepartmentID,
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
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

func (s *UserService) ListRoles(ctx context.Context) (any, error) {
	return s.q.ListRoles(ctx)
}

func (s *UserService) ListDepartments(ctx context.Context) (any, error) {
	return s.q.ListDepartments(ctx)
}
