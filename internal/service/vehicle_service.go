package service

import (
	"context"
	"database/sql"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

type VehicleService struct {
	q *repository.Queries
}

func NewVehicleService(db *sql.DB) *VehicleService {
	return &VehicleService{q: repository.New(db)}
}

type CreateVehicleRequest struct {
	Name            string `json:"name"            validate:"required"`
	PlateNumber     string `json:"plateNumber"     validate:"required"`
	Brand           string `json:"brand"           validate:"required"`
	Model           string `json:"model"           validate:"required"`
	Year            int16  `json:"year"            validate:"required"`
	CurrentOdometer int32  `json:"currentOdometer"`
	CategoryID      int32  `json:"categoryId"      validate:"required"`
	Capacity        int16  `json:"capacity"        validate:"required,min=1"`
}

type UpdateVehicleRequest struct {
	Name            string `json:"name"        validate:"required"`
	PlateNumber     string `json:"plateNumber" validate:"required"`
	Brand           string `json:"brand"       validate:"required"`
	Model           string `json:"model"       validate:"required"`
	Year            int16  `json:"year"        validate:"required"`
	CurrentOdometer int32  `json:"currentOdometer"`
	CategoryID      int32  `json:"categoryId"  validate:"required"`
	Capacity        int16  `json:"capacity"    validate:"required,min=1"`
}

type UpdateStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=AVAILABLE MAINTENANCE INACTIVE"`
}

func serializeVehicleRow(v repository.ListVehiclesRow) map[string]any {
	return map[string]any{
		"id":              v.ID,
		"resourceId":      v.ResourceId,
		"name":            v.ResourceName,
		"plateNumber":     v.PlateNumber,
		"brand":           v.Brand,
		"model":           v.Model,
		"year":            v.Year,
		"currentOdometer": v.CurrentOdometer,
		"capacity":        v.Capacity,
		"category":        map[string]any{"id": v.CategoryId, "name": v.CategoryName},
		"status":          string(v.ResourceStatus),
		"photoUrl":        nullStr(v.PhotoUrl),
	}
}

func serializeVehicleByID(v repository.GetVehicleByIDRow) map[string]any {
	return map[string]any{
		"id":              v.ID,
		"resourceId":      v.ResourceId,
		"name":            v.ResourceName,
		"plateNumber":     v.PlateNumber,
		"brand":           v.Brand,
		"model":           v.Model,
		"year":            v.Year,
		"currentOdometer": v.CurrentOdometer,
		"capacity":        v.Capacity,
		"category":        map[string]any{"id": v.CategoryId, "name": v.CategoryName},
		"status":          string(v.ResourceStatus),
		"photoUrl":        nullStr(v.PhotoUrl),
	}
}

func (s *VehicleService) List(ctx context.Context, page, limit int, search *string, categoryID *int32, status *string) ([]map[string]any, int64, error) {
	params := repository.ListVehiclesParams{
		Limit:  int32(limit),
		Offset: int32((page - 1) * limit),
	}
	if search != nil {
		params.Search = sql.NullString{String: *search, Valid: true}
	}
	if categoryID != nil {
		params.CategoryID = sql.NullInt32{Int32: *categoryID, Valid: true}
	}
	if status != nil {
		params.Status = repository.NullResourceStatus{
			ResourceStatus: repository.ResourceStatus(*status), Valid: true,
		}
	}

	rows, err := s.q.ListVehicles(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	total, _ := s.q.CountVehicles(ctx, repository.CountVehiclesParams{
		Search: params.Search, CategoryID: params.CategoryID, Status: params.Status,
	})

	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		out[i] = serializeVehicleRow(r)
	}
	return out, total, nil
}

func (s *VehicleService) GetByID(ctx context.Context, id int32) (map[string]any, error) {
	v, err := s.q.GetVehicleByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	return serializeVehicleByID(v), nil
}

func (s *VehicleService) Create(ctx context.Context, req CreateVehicleRequest) (map[string]any, error) {
	if _, err := s.q.GetVehicleByPlate(ctx, req.PlateNumber); err == nil {
		return nil, util.NewError(409, "plate number already exists", util.ErrDuplicate)
	}

	r, err := s.q.CreateResource(ctx, repository.CreateResourceParams{
		Name: req.Name, Type: repository.ResourceTypeVEHICLE,
	})
	if err != nil {
		return nil, err
	}

	_, err = s.q.CreateVehicle(ctx, repository.CreateVehicleParams{
		ResourceId: r.ID, PlateNumber: req.PlateNumber, Brand: req.Brand,
		Model: req.Model, Year: req.Year, CurrentOdometer: req.CurrentOdometer,
		CategoryId: req.CategoryID, Capacity: req.Capacity,
	})
	if err != nil {
		return nil, err
	}

	v, _ := s.q.GetVehicleByPlate(ctx, req.PlateNumber)
	return s.GetByID(ctx, v.ID)
}

func (s *VehicleService) Update(ctx context.Context, id int32, req UpdateVehicleRequest) (map[string]any, error) {
	v, err := s.q.GetVehicleByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}

	_ = s.q.UpdateResourceName(ctx, repository.UpdateResourceNameParams{
		ID: v.ResourceId, Name: req.Name,
	})
	_, err = s.q.UpdateVehicle(ctx, repository.UpdateVehicleParams{
		ID: id, PlateNumber: req.PlateNumber, Brand: req.Brand,
		Model: req.Model, Year: req.Year, CurrentOdometer: req.CurrentOdometer,
		CategoryId: req.CategoryID, Capacity: req.Capacity,
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *VehicleService) UpdateStatus(ctx context.Context, id int32, status string) (map[string]any, error) {
	v, err := s.q.GetVehicleByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	_, err = s.q.UpdateResourceStatus(ctx, repository.UpdateResourceStatusParams{
		ID: v.ResourceId, Status: repository.ResourceStatus(status),
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *VehicleService) Delete(ctx context.Context, id int32) error {
	v, err := s.q.GetVehicleByID(ctx, id)
	if err != nil {
		return util.ErrNotFound
	}
	return s.q.DeleteResource(ctx, v.ResourceId)
}

func (s *VehicleService) UpdatePhoto(ctx context.Context, id int32, photoURL string) (map[string]any, error) {
	_, err := s.q.UpdateVehiclePhoto(ctx, repository.UpdateVehiclePhotoParams{
		ID:       id,
		PhotoUrl: sql.NullString{String: photoURL, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *VehicleService) ListCategories(ctx context.Context) (any, error) {
	return s.q.ListVehicleCategories(ctx)
}

func (s *VehicleService) CreateCategory(ctx context.Context, name string) (any, error) {
	return s.q.CreateVehicleCategory(ctx, name)
}

func (s *VehicleService) DeleteCategory(ctx context.Context, id int32) error {
	if _, err := s.q.GetVehicleCategoryByID(ctx, id); err != nil {
		return util.ErrNotFound
	}
	return s.q.DeleteVehicleCategory(ctx, id)
}
