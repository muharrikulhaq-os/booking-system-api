package service

import (
	"context"
	"database/sql"
	"strings"

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

// withRoomKeepers menambahkan penjaga ruangan (N:N, migrasi 000024):
//   - roomKeepers: semua penjaga (aktif maupun tidak)
//   - hasActiveKeeper: false = ruangan belum bisa dibooking
//   - roomKeeper: penjaga pertama - hanya untuk aplikasi versi lama
func withRoomKeepers(out map[string]any, keepers []repository.RoomKeeperRef) map[string]any {
	list := make([]map[string]any, 0, len(keepers))
	active := false
	for _, k := range keepers {
		list = append(list, roomKeeperPayload(k))
		active = active || k.IsActive
	}
	out["roomKeepers"] = list
	out["hasActiveKeeper"] = active
	out["roomKeeper"] = nil
	if len(keepers) > 0 {
		out["roomKeeper"] = map[string]any{"id": keepers[0].RoomKeeperID, "name": keepers[0].Name}
	}
	return out
}

func serializeRoomRow(r repository.ListRoomsRow, keepers []repository.RoomKeeperRef) map[string]any {
	return withRoomKeepers(map[string]any{
		"id":         r.ID,
		"resourceId": r.ResourceId,
		"name":       r.ResourceName,
		"location":   r.Location,
		"capacity":   r.Capacity,
		"status":     string(r.ResourceStatus),
		"photoUrl":   nullStr(r.PhotoUrl),
	}, keepers)
}

func serializeRoomByID(r repository.GetRoomByIDRow, keepers []repository.RoomKeeperRef) map[string]any {
	return withRoomKeepers(map[string]any{
		"id":         r.ID,
		"resourceId": r.ResourceId,
		"name":       r.ResourceName,
		"location":   r.Location,
		"capacity":   r.Capacity,
		"status":     string(r.ResourceStatus),
		"photoUrl":   nullStr(r.PhotoUrl),
	}, keepers)
}

// bookable = hanya yang bisa dipesan (lihat VehicleService.List).
func (s *RoomService) List(ctx context.Context, page, limit int, search *string, status *string, bookable bool, sortBy, sortOrder string) ([]map[string]any, int64, error) {
	params := repository.ListRoomsParams{
		Limit:        int32(limit),
		Offset:       int32((page - 1) * limit),
		SortBy:       sortBy,
		SortOrder:    sortOrder,
		BookableOnly: bookable,
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
		Search: params.Search, Status: params.Status, BookableOnly: bookable,
	})
	refs, _ := s.q.ListAllRoomKeeperRefs(ctx)
	byRoom := map[int32][]repository.RoomKeeperRef{}
	for _, k := range refs {
		byRoom[k.RoomID] = append(byRoom[k.RoomID], k)
	}
	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		out[i] = serializeRoomRow(r, byRoom[r.ID])
	}
	return out, total, nil
}

func (s *RoomService) GetByID(ctx context.Context, id int32) (map[string]any, error) {
	r, err := s.q.GetRoomByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	keepers, err := s.q.ListRoomKeepersOfRoom(ctx, id)
	if err != nil {
		return nil, err
	}
	return serializeRoomByID(r, keepers), nil
}

func (s *RoomService) Create(ctx context.Context, req CreateRoomRequest, actor AuditActor) (map[string]any, error) {
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
	logAudit(ctx, s.q, actor, "CREATE", "Room", room.ID,
		"Membuat ruangan "+req.Name+" ("+req.Location+")")
	return s.GetByID(ctx, room.ID)
}

func (s *RoomService) Update(ctx context.Context, id int32, req UpdateRoomRequest, actor AuditActor) (map[string]any, error) {
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
	logAudit(ctx, s.q, actor, "UPDATE", "Room", id,
		"Mengubah data ruangan "+req.Name+" ("+req.Location+")")
	return s.GetByID(ctx, id)
}

func (s *RoomService) UpdateStatus(ctx context.Context, id int32, status string, actor AuditActor, role string) (map[string]any, error) {
	r, err := s.q.GetRoomByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	// Penjaga ruangan hanya boleh mengubah ruangan yang dijaganya (B19).
	if role == "ROOM_KEEPER" {
		rk, rkErr := s.q.GetRoomKeeperByUserID(ctx, actor.UserID)
		if rkErr != nil || !rk.IsActive {
			return nil, util.NewError(403, "Anda bukan penjaga ruangan ini", util.ErrForbidden)
		}
		if ok, err := s.q.IsRoomKeeperOfResource(ctx, rk.ID, r.ResourceId); err != nil || !ok {
			return nil, util.NewError(403, "Anda bukan penjaga ruangan ini", util.ErrForbidden)
		}
	}
	if err := guardManualStatusChange(ctx, s.q, r.ResourceId, repository.ResourceStatus(status)); err != nil {
		return nil, err
	}
	if _, err = s.q.UpdateResourceStatus(ctx, repository.UpdateResourceStatusParams{
		ID: r.ResourceId, Status: repository.ResourceStatus(status),
	}); err != nil {
		return nil, err
	}
	logAudit(ctx, s.q, actor, "UPDATE_STATUS", "Room", id,
		"Mengubah status ruangan "+r.ResourceName+" menjadi "+status)
	return s.GetByID(ctx, id)
}

func (s *RoomService) Delete(ctx context.Context, id int32, actor AuditActor) error {
	r, err := s.q.GetRoomByID(ctx, id)
	if err != nil {
		return util.ErrNotFound
	}
	if err := s.q.DeleteResource(ctx, r.ResourceId); err != nil {
		return inUseError(err, "ruangan ini masih punya riwayat booking dan tidak bisa dihapus - ubah statusnya jadi INACTIVE saja")
	}
	logAudit(ctx, s.q, actor, "DELETE", "Room", id,
		"Menghapus ruangan "+r.ResourceName)
	return nil
}

// SetRoomKeepers mengganti seluruh penjaga ruangan (N:N - satu ruangan bisa
// dijaga beberapa orang, satu orang bisa menjaga beberapa ruangan). Daftar
// kosong = tanpa penjaga: ruangan tidak bisa dibooking sampai dipilihkan lagi.
// Booking yang sudah ada tetap berjalan.
func (s *RoomService) SetRoomKeepers(ctx context.Context, id int32, keeperIDs []int32, actor AuditActor) (map[string]any, error) {
	r, err := s.q.GetRoomByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	ids := make([]int32, 0, len(keeperIDs))
	names := make([]string, 0, len(keeperIDs))
	seen := map[int32]bool{}
	for _, kid := range keeperIDs {
		if seen[kid] {
			continue
		}
		seen[kid] = true
		k, err := s.q.GetRoomKeeperByID(ctx, kid)
		if err != nil {
			return nil, util.NewError(404, "penjaga ruangan tidak ditemukan", util.ErrNotFound)
		}
		ids = append(ids, kid)
		names = append(names, k.UserName)
	}
	if err := s.q.SetRoomKeepers(ctx, id, ids); err != nil {
		return nil, err
	}
	desc := "Penjaga ruangan " + r.ResourceName + " dikosongkan"
	if len(names) > 0 {
		desc = "Penjaga ruangan " + r.ResourceName + ": " + strings.Join(names, ", ")
	}
	logAudit(ctx, s.q, actor, "UPDATE", "Room", id, desc)
	return s.GetByID(ctx, id)
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
