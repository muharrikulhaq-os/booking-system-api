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

func serializeDriverRow(d repository.ListDriversRow) map[string]any {
	var plate any
	if d.AssignedPlate.Valid {
		plate = d.AssignedPlate.String
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
	}
}

func serializeDriverByID(d repository.GetDriverByIDRow) map[string]any {
	var plate, photo any
	if d.AssignedPlate.Valid {
		plate = d.AssignedPlate.String
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
	}
}

func (s *DriverService) List(ctx context.Context, page, limit int, isActive *bool) ([]map[string]any, int64, error) {
	params := repository.ListDriversParams{
		Limit:  int32(limit),
		Offset: int32((page - 1) * limit),
	}
	if isActive != nil {
		params.IsActive = sql.NullBool{Bool: *isActive, Valid: true}
	}
	rows, err := s.q.ListDrivers(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	total, _ := s.q.CountDrivers(ctx, params.IsActive)
	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		out[i] = serializeDriverRow(r)
	}
	return out, total, nil
}

func (s *DriverService) GetByID(ctx context.Context, id int32) (map[string]any, error) {
	d, err := s.q.GetDriverByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	return serializeDriverByID(d), nil
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

func (s *DriverService) GetAvailableDrivers(ctx context.Context, start, end time.Time) ([]map[string]any, error) {
	rows, err := s.q.ListAvailableDrivers(ctx, repository.ListAvailableDriversParams{
		StartFrom: start,
		EndTo:     end,
	})
	if err != nil {
		return nil, err
	}
	
	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		out[i] = map[string]any{
			"driverId":              r.DriverID,
			"driverName":            r.DriverName,
			"employeeId":            r.EmployeeId,
			"vehicleId":             r.VehicleID,
			"plateNumber":           r.PlateNumber,
			"vehicleCapacity":       r.Capacity,
			"overlappingPassengers": r.OverlappingPassengers,
			"remainingSeats":        int(r.Capacity) - int(r.OverlappingPassengers),
		}
	}
	return out, nil
}
