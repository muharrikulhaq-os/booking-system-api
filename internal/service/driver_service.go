package service

import (
	"context"
	"database/sql"
	"time"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

type DriverService struct {
	q repository.ExtendedQuerier
}

func NewDriverService(db *sql.DB) *DriverService {
	return &DriverService{q: repository.New(db)}
}

type CreateDriverRequest struct {
	UserID        int32  `json:"userId"        validate:"required"`
	LicenseNumber string `json:"licenseNumber" validate:"required"`
	PhoneNumber   string `json:"phoneNumber"   validate:"required"`
}

type UpdateDriverRequest struct {
	LicenseNumber string `json:"licenseNumber" validate:"required"`
	PhoneNumber   string `json:"phoneNumber"   validate:"required"`
}

type AssignDriverRequest struct {
	VehicleID int32 `json:"vehicleId" validate:"required"`
}

func serializeDriverRow(d repository.ListDriversRow, fixedVehicle any) map[string]any {
	// AssignedPlate is always a plain (never-null) string from the query —
	// see the comment on ListDrivers in sql/query/driver.sql for why.
	var plate any
	if d.AssignedPlate != "" {
		plate = d.AssignedPlate
	}
	return map[string]any{
		"id":            d.ID,
		"userId":        d.UserId,
		"name":          d.UserName,
		"employeeId":    d.EmployeeId,
		"email":         d.Email,
		"licenseNumber": d.LicenseNumber,
		"phoneNumber":   d.PhoneNumber,
		"isActive":      d.IsActive,
		"assignedPlate": plate,
		// Kendaraan tetap milik supir ini (vehicles.fixedDriverId) - beda
		// dari assignedPlate yang mengikuti booking aktif, ini permanen
		// sampai admin ubah.
		"fixedVehicle": fixedVehicle,
	}
}

func serializeDriverByID(d repository.GetDriverByIDRow, fixedVehicle any) map[string]any {
	var plate, photo any
	if d.AssignedPlate != "" {
		plate = d.AssignedPlate
	}
	if d.ProfilePhoto.Valid {
		photo = d.ProfilePhoto.String
	}
	return map[string]any{
		"id":            d.ID,
		"userId":        d.UserId,
		"name":          d.UserName,
		"employeeId":    d.EmployeeId,
		"email":         d.Email,
		"profilePhoto":  photo,
		"licenseNumber": d.LicenseNumber,
		"phoneNumber":   d.PhoneNumber,
		"isActive":      d.IsActive,
		"assignedPlate": plate,
		"fixedVehicle":  fixedVehicle,
	}
}

func (s *DriverService) List(ctx context.Context, page, limit int, search *string, isActive *bool) ([]map[string]any, int64, error) {
	params := repository.ListDriversParams{
		Limit:  int32(limit),
		Offset: int32((page - 1) * limit),
	}
	if search != nil {
		params.Search = sql.NullString{String: *search, Valid: true}
	}
	if isActive != nil {
		params.IsActive = sql.NullBool{Bool: *isActive, Valid: true}
	}
	rows, err := s.q.ListDrivers(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	// Count wajib memakai filter yang sama persis dengan List - kalau tidak,
	// total pagination ikut menghitung baris yang justru tersaring keluar.
	total, _ := s.q.CountDrivers(ctx, repository.CountDriversParams{
		Search: params.Search, IsActive: params.IsActive,
	})

	fixedRows, _ := s.q.ListVehiclesWithFixedDriver(ctx)
	fixedByDriver := make(map[int32]any, len(fixedRows))
	for _, fr := range fixedRows {
		fixedByDriver[fr.DriverID] = map[string]any{"id": fr.VehicleID, "plateNumber": fr.PlateNumber}
	}

	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		out[i] = serializeDriverRow(r, fixedByDriver[r.ID])
	}
	return out, total, nil
}

func (s *DriverService) GetByID(ctx context.Context, id int32) (map[string]any, error) {
	d, err := s.q.GetDriverByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	var fixedVehicle any
	if v, verr := s.q.GetVehicleByFixedDriverID(ctx, id); verr == nil {
		fixedVehicle = map[string]any{"id": v.ID, "plateNumber": v.PlateNumber}
	}
	return serializeDriverByID(d, fixedVehicle), nil
}

func (s *DriverService) Create(ctx context.Context, req CreateDriverRequest) (map[string]any, error) {
	if _, err := s.q.GetDriverByUserID(ctx, req.UserID); err == nil {
		return nil, util.NewError(409, "user is already a driver", util.ErrDuplicate)
	}
	d, err := s.q.CreateDriver(ctx, repository.CreateDriverParams{
		UserId:        req.UserID,
		LicenseNumber: req.LicenseNumber,
		PhoneNumber:   req.PhoneNumber,
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, d.ID)
}

func (s *DriverService) Update(ctx context.Context, id int32, req UpdateDriverRequest) (map[string]any, error) {
	if _, err := s.q.GetDriverByID(ctx, id); err != nil {
		return nil, util.ErrNotFound
	}
	if _, err := s.q.UpdateDriver(ctx, repository.UpdateDriverParams{
		ID:            id,
		LicenseNumber: req.LicenseNumber,
		PhoneNumber:   req.PhoneNumber,
	}); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *DriverService) ToggleActive(ctx context.Context, id int32) (map[string]any, error) {
	if _, err := s.q.GetDriverByID(ctx, id); err != nil {
		return nil, util.ErrNotFound
	}
	if _, err := s.q.ToggleDriverActive(ctx, id); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *DriverService) Assign(ctx context.Context, id int32, req AssignDriverRequest) (map[string]any, error) {
	if _, err := s.q.GetDriverByID(ctx, id); err != nil {
		return nil, util.ErrNotFound
	}
	_ = s.q.ReleaseDriver(ctx, id)
	if _, err := s.q.AssignDriverToVehicle(ctx, repository.AssignDriverToVehicleParams{
		DriverId: id, VehicleId: req.VehicleID,
	}); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *DriverService) Release(ctx context.Context, id int32) (map[string]any, error) {
	if _, err := s.q.GetDriverByID(ctx, id); err != nil {
		return nil, util.ErrNotFound
	}
	_ = s.q.ReleaseDriver(ctx, id)
	return s.GetByID(ctx, id)
}

func (s *DriverService) GetAssignments(ctx context.Context, id int32) (any, error) {
	return s.q.GetDriverAssignmentHistory(ctx, id)
}

// SetFixedVehicle assigns (vehicleID != nil) or clears (vehicleID == nil) this
// driver's permanent vehicle pairing - the driver-side entry point for the
// same underlying vehicles.fixedDriverId column VehicleService.SetFixedDriver
// writes, so it stays in sync no matter which form (driver or vehicle) an
// admin edits it from.
func (s *DriverService) SetFixedVehicle(ctx context.Context, id int32, vehicleID *int32) (map[string]any, error) {
	if _, err := s.q.GetDriverByID(ctx, id); err != nil {
		return nil, util.ErrNotFound
	}
	// Lepas kendaraan yang SEKARANG jadi pasangan tetap supir ini (kalau ada) -
	// baik untuk kasus "clear" (vehicleID nil) maupun "pindah ke kendaraan lain".
	if old, err := s.q.GetVehicleByFixedDriverID(ctx, id); err == nil {
		if _, err := s.q.SetVehicleFixedDriver(ctx, repository.SetVehicleFixedDriverParams{
			VehicleID: old.ID, DriverID: sql.NullInt32{},
		}); err != nil {
			return nil, err
		}
	}
	if vehicleID != nil {
		if _, err := s.q.GetVehicleByID(ctx, *vehicleID); err != nil {
			return nil, util.NewError(404, "vehicle not found", util.ErrNotFound)
		}
		if _, err := s.q.SetVehicleFixedDriver(ctx, repository.SetVehicleFixedDriverParams{
			VehicleID: *vehicleID, DriverID: sql.NullInt32{Int32: id, Valid: true},
		}); err != nil {
			return nil, err
		}
	}
	return s.GetByID(ctx, id)
}

func (s *DriverService) GetAvailableDrivers(ctx context.Context, start, end time.Time) ([]map[string]any, error) {
	rows, err := s.q.ListAvailableDrivers(ctx, repository.ListAvailableDriversParams{
		StartFrom: start,
		EndTo:     end,
	})
	if err != nil {
		return nil, err
	}

	spdIDs, _ := s.q.GetDriverIDsWithActiveSpd(ctx)
	spdSet := make(map[int32]bool, len(spdIDs))
	for _, id := range spdIDs {
		spdSet[id] = true
	}

	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		// Supir "kosong" (belum pegang kendaraan) → vehicle null; kapasitas &
		// remainingSeats mengikuti kendaraan yang dibooking (dihitung di FE).
		var vehicleID, plateNumber, vehicleCapacity, remainingSeats any
		if r.VehicleID.Valid {
			vehicleID = r.VehicleID.Int32
		}
		if r.PlateNumber.Valid {
			plateNumber = r.PlateNumber.String
		}
		if r.Capacity.Valid {
			vehicleCapacity = r.Capacity.Int16
			remainingSeats = int(r.Capacity.Int16) - int(r.OverlappingPassengers)
		}
		out[i] = map[string]any{
			"driverId":              r.DriverID,
			"driverName":            r.DriverName,
			"employeeId":            r.EmployeeId,
			"vehicleId":             vehicleID,
			"plateNumber":           plateNumber,
			"vehicleCapacity":       vehicleCapacity,
			"overlappingPassengers": r.OverlappingPassengers,
			"remainingSeats":        remainingSeats,
			"overlappingPurpose":    r.OverlappingPurpose,
			// true bila supir ini sedang terkunci tugas SPD hari ini (lihat
			// GetDriverIDsWithActiveSpd) - badge "Digunakan SPD" di picker.
			"isSpdActive": spdSet[r.DriverID],
		}
	}
	return out, nil
}
