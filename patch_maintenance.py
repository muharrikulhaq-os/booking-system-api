import os

maintenance_service_code = """package service

import (
	"context"
	"database/sql"
	"time"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

type MaintenanceService struct {
	q *repository.Queries
}

func NewMaintenanceService(db *sql.DB) *MaintenanceService {
	return &MaintenanceService{q: repository.New(db)}
}

type CreateMaintenanceRequest struct {
	VehicleID         int32      `json:"vehicleId"         validate:"required"`
	MaintenanceTypeID *int32     `json:"maintenanceTypeId"`
	Description       string     `json:"description"       validate:"required"`
	Odometer          *int32     `json:"odometer"`
	TotalCost         *float64   `json:"totalCost"`
	VendorName        string     `json:"vendorName"`
	Location          string     `json:"location"          validate:"required"`
	StartDate         time.Time  `json:"startDate"         validate:"required"`
	EndDate           *time.Time `json:"endDate"`
}

type UpdateMaintenanceRequest struct {
	VehicleID         int32      `json:"vehicleId"         validate:"required"`
	MaintenanceTypeID *int32     `json:"maintenanceTypeId"`
	Description       string     `json:"description"       validate:"required"`
	Odometer          *int32     `json:"odometer"`
	TotalCost         *float64   `json:"totalCost"`
	VendorName        string     `json:"vendorName"`
	Location          string     `json:"location"          validate:"required"`
	StartDate         time.Time  `json:"startDate"         validate:"required"`
	EndDate           *time.Time `json:"endDate"`
}

func serializeMaintenanceRow(m repository.ListMaintenanceRow) map[string]any {
	return map[string]any{
		"id":                m.ID,
		"vehicleId":         m.VehicleId,
		"vehicleName":       m.VehicleName,
		"plateNumber":       m.PlateNumber,
		"maintenanceTypeId": m.MaintenanceTypeId.Int32,
		"description":       m.Description,
		"odometer":          m.Odometer.Int32,
		"totalCost":         m.TotalCost.String,
		"vendorName":        m.VendorName.String,
		"location":          m.Location,
		"startDate":         m.StartDate,
		"endDate":           nullTime(m.EndDate),
		"createdBy":         m.CreatedByName,
		"createdAt":         m.CreatedAt,
	}
}

func serializeMaintenanceByID(m repository.GetMaintenanceByIDRow) map[string]any {
	return map[string]any{
		"id":                m.ID,
		"vehicleId":         m.VehicleId,
		"vehicleName":       m.VehicleName,
		"plateNumber":       m.PlateNumber,
		"maintenanceTypeId": m.MaintenanceTypeId.Int32,
		"description":       m.Description,
		"odometer":          m.Odometer.Int32,
		"totalCost":         m.TotalCost.String,
		"vendorName":        m.VendorName.String,
		"location":          m.Location,
		"startDate":         m.StartDate,
		"endDate":           nullTime(m.EndDate),
		"createdBy":         m.CreatedByName,
		"createdAt":         m.CreatedAt,
	}
}

func nullInt32Ptr(v *int32) sql.NullInt32 {
	if v == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: *v, Valid: true}
}

func (s *MaintenanceService) List(ctx context.Context, page, limit int, vehicleID *int32) ([]map[string]any, int64, error) {
	params := repository.ListMaintenanceParams{
		Limit:  int32(limit),
		Offset: int32((page - 1) * limit),
	}
	if vehicleID != nil {
		params.VehicleID = sql.NullInt32{Int32: *vehicleID, Valid: true}
	}
	rows, err := s.q.ListMaintenance(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	total, _ := s.q.CountMaintenance(ctx, params.VehicleID)
	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		out[i] = serializeMaintenanceRow(r)
	}
	return out, total, nil
}

func (s *MaintenanceService) GetByID(ctx context.Context, id int32) (map[string]any, error) {
	m, err := s.q.GetMaintenanceByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	return serializeMaintenanceByID(m), nil
}

type MaintenanceCreateResponse struct {
	Data    map[string]any `json:"data"`
	Warning string         `json:"warning,omitempty"`
}

func (s *MaintenanceService) Create(ctx context.Context, req CreateMaintenanceRequest, createdByID int32) (MaintenanceCreateResponse, error) {
	// First, get the vehicle to find its resourceId for overlapping check
	vehicle, err := s.q.GetVehicleByID(ctx, req.VehicleID)
	if err != nil {
		return MaintenanceCreateResponse{}, util.NewError(404, "Vehicle not found", util.ErrNotFound)
	}

	endDate := req.StartDate.Add(24 * time.Hour) // just an estimate if EndDate is not provided
	if req.EndDate != nil {
		endDate = *req.EndDate
	}

	// Check overlapping bookings on the resource
	overlapCount, _ := s.q.CheckBookingConflict(ctx, repository.CheckBookingConflictParams{
		ResourceID: vehicle.ResourceId,
		StartDate:  req.StartDate,
		EndDate:    endDate,
	})

	warningMsg := ""
	if overlapCount > 0 {
		warningMsg = "Warning: There are active bookings that overlap with this maintenance schedule."
	}

	var ed sql.NullTime
	if req.EndDate != nil {
		ed = sql.NullTime{Time: *req.EndDate, Valid: true}
	}

	m, err := s.q.CreateMaintenance(ctx, repository.CreateMaintenanceParams{
		VehicleId:         req.VehicleID,
		MaintenanceTypeId: nullInt32Ptr(req.MaintenanceTypeID),
		Description:       req.Description,
		Odometer:          nullInt32Ptr(req.Odometer),
		TotalCost:         nullNumeric(req.Cost),
		VendorName:        sql.NullString{String: req.VendorName, Valid: req.VendorName != ""},
		Location:          req.Location,
		StartDate:         req.StartDate,
		EndDate:           ed,
		RecordedById:      createdByID,
	})
	if err != nil {
		return MaintenanceCreateResponse{}, err
	}

	// set resource status to MAINTENANCE
	_, _ = s.q.UpdateResourceStatus(ctx, repository.UpdateResourceStatusParams{
		ID:     vehicle.ResourceId,
		Status: repository.ResourceStatusMAINTENANCE,
	})
	
	// if end date is filled and is in the past, immediately unlock (just in case)
	if req.EndDate != nil && req.EndDate.Before(time.Now()) {
		_, _ = s.q.UpdateResourceStatus(ctx, repository.UpdateResourceStatusParams{
			ID:     vehicle.ResourceId,
			Status: repository.ResourceStatusAVAILABLE,
		})
	}

	res, _ := s.GetByID(ctx, m.ID)
	return MaintenanceCreateResponse{
		Data:    res,
		Warning: warningMsg,
	}, nil
}

func (s *MaintenanceService) Update(ctx context.Context, id int32, req UpdateMaintenanceRequest) (map[string]any, error) {
	m, err := s.q.GetMaintenanceByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	
	vehicle, err := s.q.GetVehicleByID(ctx, req.VehicleID)
	if err != nil {
		return nil, util.NewError(404, "Vehicle not found", util.ErrNotFound)
	}

	var endDate sql.NullTime
	if req.EndDate != nil {
		endDate = sql.NullTime{Time: *req.EndDate, Valid: true}
	}

	if _, err = s.q.UpdateMaintenance(ctx, repository.UpdateMaintenanceParams{
		ID:                id,
		VehicleId:         req.VehicleID,
		MaintenanceTypeId: nullInt32Ptr(req.MaintenanceTypeID),
		Description:       req.Description,
		Odometer:          nullInt32Ptr(req.Odometer),
		TotalCost:         nullNumeric(req.Cost),
		VendorName:        sql.NullString{String: req.VendorName, Valid: req.VendorName != ""},
		Location:          req.Location,
		StartDate:         req.StartDate,
		EndDate:           endDate,
	}); err != nil {
		return nil, err
	}

	// if endDate provided and is <= now, mark resource AVAILABLE again
	if req.EndDate != nil && (req.EndDate.Before(time.Now()) || req.EndDate.Equal(time.Now())) {
		_, _ = s.q.UpdateResourceStatus(ctx, repository.UpdateResourceStatusParams{
			ID:     vehicle.ResourceId,
			Status: repository.ResourceStatusAVAILABLE,
		})
	} else if req.EndDate == nil || req.EndDate.After(time.Now()) {
	    _, _ = s.q.UpdateResourceStatus(ctx, repository.UpdateResourceStatusParams{
			ID:     vehicle.ResourceId,
			Status: repository.ResourceStatusMAINTENANCE,
		})
	}

	return s.GetByID(ctx, id)
}

func (s *MaintenanceService) Delete(ctx context.Context, id int32) error {
	m, err := s.q.GetMaintenanceByID(ctx, id)
	if err != nil {
		return util.ErrNotFound
	}
	
	vehicle, err := s.q.GetVehicleByID(ctx, m.VehicleId)
	
	err = s.q.DeleteMaintenance(ctx, id)
	
	// Recover to AVAILABLE if deleted
	if err == nil && vehicle.ID != 0 {
		_, _ = s.q.UpdateResourceStatus(ctx, repository.UpdateResourceStatusParams{
			ID:     vehicle.ResourceId,
			Status: repository.ResourceStatusAVAILABLE,
		})
	}
	return err
}
"""

with open('internal/service/maintenance_service.go', 'w') as f:
    f.write(maintenance_service_code)
print("Updated internal/service/maintenance_service.go")

"""
Now I also need to update MaintenanceHandler in remaining_handlers.go to handle vehicleId
and the new response from Create() (it returns MaintenanceCreateResponse).
"""
with open('internal/delivery/http/remaining_handlers.go', 'r') as f:
    handlers = f.read()

old_list = """func (h *MaintenanceHandler) List(c *fiber.Ctx) error {
	page := queryInt(c, "page", 1)
	limit := queryInt(c, "limit", 20)
	data, total, err := h.svc.List(c.Context(), page, limit, queryInt32(c, "resourceId"))
	if err != nil {
		return err
	}
	return util.Paginated(c, "Maintenance records retrieved", data, total, page, limit)
}"""
new_list = """func (h *MaintenanceHandler) List(c *fiber.Ctx) error {
	page := queryInt(c, "page", 1)
	limit := queryInt(c, "limit", 20)
	data, total, err := h.svc.List(c.Context(), page, limit, queryInt32(c, "vehicleId"))
	if err != nil {
		return err
	}
	return util.Paginated(c, "Maintenance records retrieved", data, total, page, limit)
}"""
handlers = handlers.replace(old_list, new_list)

old_create = """func (h *MaintenanceHandler) Create(c *fiber.Ctx) error {
	var req service.CreateMaintenanceRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.Create(c.Context(), req, int32(middleware.GetUserID(c)))
	if err != nil {
		return err
	}
	return util.Created(c, "Maintenance record created", data)
}"""
new_create = """func (h *MaintenanceHandler) Create(c *fiber.Ctx) error {
	var req service.CreateMaintenanceRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	resp, err := h.svc.Create(c.Context(), req, int32(middleware.GetUserID(c)))
	if err != nil {
		return err
	}
	if resp.Warning != "" {
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"status":  "success",
			"message": "Maintenance record created with warning",
			"warning": resp.Warning,
			"data":    resp.Data,
		})
	}
	return util.Created(c, "Maintenance record created", resp.Data)
}"""
handlers = handlers.replace(old_create, new_create)

with open('internal/delivery/http/remaining_handlers.go', 'w') as f:
    f.write(handlers)
print("Patched handlers for maintenance.")
