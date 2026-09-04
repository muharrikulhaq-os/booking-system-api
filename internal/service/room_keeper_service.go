package service

import (
	"context"
	"database/sql"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

type RoomKeeperService struct {
	q repository.ExtendedQuerier
}

func NewRoomKeeperService(db *sql.DB) *RoomKeeperService {
	return &RoomKeeperService{q: repository.New(db)}
}

type CreateRoomKeeperRequest struct {
	UserID      int32  `json:"userId"      validate:"required"`
	PhoneNumber string `json:"phoneNumber" validate:"required"`
}

type UpdateRoomKeeperRequest struct {
	PhoneNumber string `json:"phoneNumber" validate:"required"`
}

func serializeRoomKeeperRow(rk repository.ListRoomKeepersRow) map[string]any {
	return map[string]any{
		"id":          rk.ID,
		"userId":      rk.UserId,
		"name":        rk.UserName,
		"employeeId":  rk.EmployeeId,
		"email":       rk.Email,
		"phoneNumber": rk.PhoneNumber,
		"isActive":    rk.IsActive,
	}
}

func serializeRoomKeeperByID(rk repository.GetRoomKeeperByIDRow, rooms []map[string]any) map[string]any {
	var photo any
	if rk.ProfilePhoto.Valid {
		photo = rk.ProfilePhoto.String
	}
	return map[string]any{
		"id":           rk.ID,
		"userId":       rk.UserId,
		"name":         rk.UserName,
		"employeeId":   rk.EmployeeId,
		"email":        rk.Email,
		"profilePhoto": photo,
		"phoneNumber":  rk.PhoneNumber,
		"isActive":     rk.IsActive,
		// Ruangan yang jadi tanggung jawab room keeper ini - bisa lebih dari
		// satu (N:1, beda dari pasangan tetap supir<->kendaraan yang 1:1).
		"rooms": rooms,
	}
}

func (s *RoomKeeperService) List(ctx context.Context, page, limit int, isActive *bool) ([]map[string]any, int64, error) {
	params := repository.ListRoomKeepersParams{
		Limit:  int32(limit),
		Offset: int32((page - 1) * limit),
	}
	if isActive != nil {
		params.IsActive = sql.NullBool{Bool: *isActive, Valid: true}
	}
	rows, err := s.q.ListRoomKeepers(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	total, _ := s.q.CountRoomKeepers(ctx, params.IsActive)
	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		out[i] = serializeRoomKeeperRow(r)
	}
	return out, total, nil
}

func (s *RoomKeeperService) GetByID(ctx context.Context, id int32) (map[string]any, error) {
	rk, err := s.q.GetRoomKeeperByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	roomRows, _ := s.q.GetRoomsByRoomKeeperID(ctx, id)
	rooms := make([]map[string]any, len(roomRows))
	for i, r := range roomRows {
		rooms[i] = map[string]any{
			"id":       r.ID,
			"name":     r.ResourceName,
			"location": r.Location,
		}
	}
	return serializeRoomKeeperByID(rk, rooms), nil
}

func (s *RoomKeeperService) Create(ctx context.Context, req CreateRoomKeeperRequest) (map[string]any, error) {
	if _, err := s.q.GetRoomKeeperByUserID(ctx, req.UserID); err == nil {
		return nil, util.NewError(409, "user is already a room keeper", util.ErrDuplicate)
	}
	rk, err := s.q.CreateRoomKeeper(ctx, repository.CreateRoomKeeperParams{
		UserId:      req.UserID,
		PhoneNumber: req.PhoneNumber,
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, rk.ID)
}

func (s *RoomKeeperService) Update(ctx context.Context, id int32, req UpdateRoomKeeperRequest) (map[string]any, error) {
	if _, err := s.q.GetRoomKeeperByID(ctx, id); err != nil {
		return nil, util.ErrNotFound
	}
	if _, err := s.q.UpdateRoomKeeper(ctx, repository.UpdateRoomKeeperParams{
		ID:          id,
		PhoneNumber: req.PhoneNumber,
	}); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *RoomKeeperService) ToggleActive(ctx context.Context, id int32) (map[string]any, error) {
	if _, err := s.q.GetRoomKeeperByID(ctx, id); err != nil {
		return nil, util.ErrNotFound
	}
	if _, err := s.q.ToggleRoomKeeperActive(ctx, id); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}
