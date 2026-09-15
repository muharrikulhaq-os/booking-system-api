package service

import (
	"context"
	"database/sql"
	"fmt"

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

func serializeVehicleRow(v repository.ListVehiclesRow, spdActive bool) map[string]any {
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
		// true bila kendaraan sedang diklaim SPD hari ini (lihat
		// GetVehicleIDsWithActiveSpd) - dipakai badge "Digunakan SPD" di
		// picker/list, terpisah dari status resource (AVAILABLE/IN_USE/dst)
		// yang tidak membedakan sebab pemakaiannya.
		"isSpdActive": spdActive,
		"fixedDriver": fixedDriverField(v.FixedDriverId, v.FixedDriverName),
	}
}

func serializeVehicleByID(v repository.GetVehicleByIDRow, spdActive bool) map[string]any {
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
		"isSpdActive":     spdActive,
		"fixedDriver":     fixedDriverField(v.FixedDriverId, v.FixedDriverName),
	}
}

// fixedDriverField builds the {id, name} object surfaced as vehicle.fixedDriver
// (null bila kendaraan tidak punya supir tetap) - dipakai FE untuk auto-pilih
// supir & menyembunyikan pemilihan supir manual saat kendaraan ini dibooking.
func fixedDriverField(id sql.NullInt32, name sql.NullString) any {
	if !id.Valid {
		return nil
	}
	return map[string]any{"id": id.Int32, "name": name.String}
}

func (s *VehicleService) List(ctx context.Context, page, limit int, search *string, categoryID *int32, status *string, sortBy, sortOrder string) ([]map[string]any, int64, error) {
	params := repository.ListVehiclesParams{
		Limit:     int32(limit),
		Offset:    int32((page - 1) * limit),
		SortBy:    sortBy,
		SortOrder: sortOrder,
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

	spdIDs, _ := s.q.GetVehicleIDsWithActiveSpd(ctx)
	spdSet := make(map[int32]bool, len(spdIDs))
	for _, id := range spdIDs {
		spdSet[id] = true
	}

	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		out[i] = serializeVehicleRow(r, spdSet[r.ID])
	}
	return out, total, nil
}

func (s *VehicleService) GetByID(ctx context.Context, id int32) (map[string]any, error) {
	v, err := s.q.GetVehicleByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	spdIDs, _ := s.q.GetVehicleIDsWithActiveSpd(ctx)
	spdActive := false
	for _, sid := range spdIDs {
		if sid == v.ID {
			spdActive = true
			break
		}
	}
	return serializeVehicleByID(v, spdActive), nil
}

func (s *VehicleService) Create(ctx context.Context, req CreateVehicleRequest, actor AuditActor) (map[string]any, error) {
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
	logAudit(ctx, s.q, actor, "CREATE", "Vehicle", v.ID,
		"Membuat kendaraan "+req.Name+" ("+req.PlateNumber+")")
	return s.GetByID(ctx, v.ID)
}

func (s *VehicleService) Update(ctx context.Context, id int32, req UpdateVehicleRequest, actor AuditActor) (map[string]any, error) {
	v, err := s.q.GetVehicleByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	// Odometer kendaraan harus data faktual yang monoton naik - satu-satunya
	// jalan menurunkannya adalah kesalahan input, jadi tolak eksplisit di
	// sini (bukan diam-diam di-clamp) supaya adminnya tahu ada yang salah.
	if req.CurrentOdometer < v.CurrentOdometer {
		return nil, util.NewError(400,
			fmt.Sprintf("odometer tidak boleh kurang dari catatan saat ini (%d km)", v.CurrentOdometer),
			util.ErrBadRequest)
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
	logAudit(ctx, s.q, actor, "UPDATE", "Vehicle", id,
		"Mengubah data kendaraan "+req.Name+" ("+req.PlateNumber+")")

	if req.CurrentOdometer > v.CurrentOdometer {
		checkAndTriggerAutoMaintenance(ctx, s.q, id, actor.UserID)
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

func (s *VehicleService) UpdateStatus(ctx context.Context, id int32, status string, actor AuditActor) (map[string]any, error) {
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
	logAudit(ctx, s.q, actor, "UPDATE_STATUS", "Vehicle", id,
		"Mengubah status kendaraan "+v.PlateNumber+" menjadi "+status)
	return s.GetByID(ctx, id)
}

func (s *VehicleService) Delete(ctx context.Context, id int32, actor AuditActor) error {
	v, err := s.q.GetVehicleByID(ctx, id)
	if err != nil {
		return util.ErrNotFound
	}
	if err := s.q.DeleteResource(ctx, v.ResourceId); err != nil {
		return err
	}
	logAudit(ctx, s.q, actor, "DELETE", "Vehicle", id,
		"Menghapus kendaraan "+v.PlateNumber)
	return nil
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

// SetFixedDriver assigns (driverID != nil) or clears (driverID == nil) this
// vehicle's permanent driver pairing. Assigning first releases whatever
// vehicle currently holds that driver as fixed - the DB UNIQUE constraint
// only allows one vehicle per driver, so without this the second UPDATE
// would just fail instead of "moving" the pairing like an admin expects.
func (s *VehicleService) SetFixedDriver(ctx context.Context, id int32, driverID *int32) (map[string]any, error) {
	if _, err := s.q.GetVehicleByID(ctx, id); err != nil {
		return nil, util.ErrNotFound
	}
	var newDriver sql.NullInt32
	if driverID != nil {
		if _, err := s.q.GetDriverByID(ctx, *driverID); err != nil {
			return nil, util.NewError(404, "driver not found", util.ErrNotFound)
		}
		_ = s.q.ClearVehicleFixedDriverByDriver(ctx, *driverID)
		newDriver = sql.NullInt32{Int32: *driverID, Valid: true}
	}
	if _, err := s.q.SetVehicleFixedDriver(ctx, repository.SetVehicleFixedDriverParams{
		VehicleID: id, DriverID: newDriver,
	}); err != nil {
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
