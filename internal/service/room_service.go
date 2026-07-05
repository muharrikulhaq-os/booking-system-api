package service

import (
	"context"
	"database/sql"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

type RoomService struct {
	q repository.ExtendedQuerier
}

func NewRoomService(db *sql.DB) *RoomService {
	return &RoomService{q: repository.New(db)}
}

type CreateRoomRequest struct {
	Name     string `json:"name"     validate:"required"`
	Location string `json:"location" validate:"required"`
	Capacity int16  `json:"capacity" validate:"required,min=1"`
}

type UpdateRoomRequest struct {
	Name     string `json:"name"     validate:"required"`
	Location string `json:"location" validate:"required"`
	Capacity int16  `json:"capacity" validate:"required,min=1"`
}

func serializeRoomRow(r repository.ListRoomsRow) map[string]any {
	return map[string]any{
		"id":         r.ID,
		"resourceId": r.ResourceId,
		"name":       r.ResourceName,
		"location":   r.Location,
		"capacity":   r.Capacity,
		"status":     string(r.ResourceStatus),
		"photoUrl":   nullStr(r.PhotoUrl),
	}
}

func serializeRoomByID(r repository.GetRoomByIDRow) map[string]any {
	return map[string]any{
		"id":         r.ID,
		"resourceId": r.ResourceId,
		"name":       r.ResourceName,
		"location":   r.Location,
		"capacity":   r.Capacity,
		"status":     string(r.ResourceStatus),
		"photoUrl":   nullStr(r.PhotoUrl),
	}
}

func (s *RoomService) List(ctx context.Context, page, limit int, search *string, status *string) ([]map[string]any, int64, error) {
	params := repository.ListRoomsParams{
		Limit:  int32(limit),
		Offset: int32((page - 1) * limit),
	}
	if search != nil {
		params.Search = sql.NullString{String: *search, Valid: true}
	}
	if status != nil {
		params.Status = repository.NullResourceStatus{
			ResourceStatus: repository.ResourceStatus(*status), Valid: true,
		}
	}
	rows, err := s.q.ListRooms(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	total, _ := s.q.CountRooms(ctx, repository.CountRoomsParams{
		Search: params.Search, Status: params.Status,
	})
	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		out[i] = serializeRoomRow(r)
	}
	return out, total, nil
}

func (s *RoomService) GetByID(ctx context.Context, id int32) (map[string]any, error) {
	r, err := s.q.GetRoomByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	return serializeRoomByID(r), nil
}

func (s *RoomService) Create(ctx context.Context, req CreateRoomRequest) (map[string]any, error) {
	res, err := s.q.CreateResource(ctx, repository.CreateResourceParams{
		Name: req.Name, Type: repository.ResourceTypeROOM,
	})
	if err != nil {
		return nil, err
	}
	room, err := s.q.CreateRoom(ctx, repository.CreateRoomParams{
		ResourceId: res.ID, Location: req.Location, Capacity: req.Capacity,
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, room.ID)
}

func (s *RoomService) Update(ctx context.Context, id int32, req UpdateRoomRequest) (map[string]any, error) {
	r, err := s.q.GetRoomByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	_ = s.q.UpdateResourceName(ctx, repository.UpdateResourceNameParams{
		ID: r.ResourceId, Name: req.Name,
	})
	if _, err = s.q.UpdateRoom(ctx, repository.UpdateRoomParams{
		ID: id, Location: req.Location, Capacity: req.Capacity,
	}); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *RoomService) UpdateStatus(ctx context.Context, id int32, status string) (map[string]any, error) {
	r, err := s.q.GetRoomByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if _, err = s.q.UpdateResourceStatus(ctx, repository.UpdateResourceStatusParams{
		ID: r.ResourceId, Status: repository.ResourceStatus(status),
	}); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *RoomService) Delete(ctx context.Context, id int32) error {
	r, err := s.q.GetRoomByID(ctx, id)
	if err != nil {
		return util.ErrNotFound
	}
	return s.q.DeleteResource(ctx, r.ResourceId)
}

func (s *RoomService) UpdatePhoto(ctx context.Context, id int32, photoURL string) (map[string]any, error) {
	_, err := s.q.UpdateRoomPhoto(ctx, repository.UpdateRoomPhotoParams{
		ID:       id,
		PhotoUrl: sql.NullString{String: photoURL, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}
