package service

import (
	"context"
	"database/sql"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

type VehicleService struct {
	q repository.ExtendedQuerier
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
	EnergyType      string `json:"energyType"      validate:"omitempty,oneof=BBM LISTRIK HYBRID"`
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
	EnergyType      string `json:"energyType"  validate:"omitempty,oneof=BBM LISTRIK HYBRID"`
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
		"energyType":      string(v.EnergyType),
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
		"energyType":      string(v.EnergyType),
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

	energyType := req.EnergyType
	if energyType == "" {
		energyType = string(repository.EnergyTypeBBM)
	}
	_, err = s.q.CreateVehicle(ctx, repository.CreateVehicleParams{
		ResourceId: r.ID, PlateNumber: req.PlateNumber, Brand: req.Brand,
		Model: req.Model, Year: req.Year, CurrentOdometer: req.CurrentOdometer,
		CategoryId: req.CategoryID, Capacity: req.Capacity,
		EnergyType: repository.EnergyType(energyType),
	})
	if err != nil {
		return nil, err
	}

	v, _ := s.q.GetVehicleByPlate(ctx, req.PlateNumber)
	return s.GetByID(ctx, v.ID)
}

func (s *VehicleService) Update(ctx context.Context, id int32, req UpdateVehicleRequest, actorID int32) (map[string]any, error) {
	v, err := s.q.GetVehicleByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}

	_ = s.q.UpdateResourceName(ctx, repository.UpdateResourceNameParams{
		ID: v.ResourceId, Name: req.Name,
	})
	energyType := repository.EnergyType(req.EnergyType)
	if req.EnergyType == "" {
		energyType = v.EnergyType
	}
	_, err = s.q.UpdateVehicle(ctx, repository.UpdateVehicleParams{
		ID: id, PlateNumber: req.PlateNumber, Brand: req.Brand,
		Model: req.Model, Year: req.Year, CurrentOdometer: req.CurrentOdometer,
		CategoryId: req.CategoryID, Capacity: req.Capacity,
		EnergyType: energyType,
	})
	if err != nil {
		return nil, err
	}

	if req.CurrentOdometer > v.CurrentOdometer {
		checkAndTriggerAutoMaintenance(ctx, s.q, id, actorID)
	}

	return s.GetByID(ctx, id)
}

// GetMaintenanceStatus returns how many kilometers remain before this
// vehicle's next scheduled (odometer-based) maintenance is auto-triggered.
func (s *VehicleService) GetMaintenanceStatus(ctx context.Context, id int32) (map[string]any, error) {
	v, err := s.q.GetVehicleByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	nextDueAt := v.LastMaintenanceOdometer + v.MaintenanceIntervalKm
	remaining := nextDueAt - v.CurrentOdometer
	if remaining < 0 {
		remaining = 0
	}
	return map[string]any{
		"vehicleId":               v.ID,
		"currentOdometer":         v.CurrentOdometer,
		"lastMaintenanceOdometer": v.LastMaintenanceOdometer,
		"maintenanceIntervalKm":   v.MaintenanceIntervalKm,
		"nextMaintenanceDueAt":    nextDueAt,
		"kmUntilDue":              remaining,
		"isDue":                   v.CurrentOdometer-v.LastMaintenanceOdometer >= v.MaintenanceIntervalKm,
	}, nil
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
