package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"mime/multipart"
	"strings"
	"github.com/sqlc-dev/pqtype"
	"fmt"
	"time"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

type MaintenanceService struct {
	q repository.ExtendedQuerier
}

func NewMaintenanceService(db *sql.DB) *MaintenanceService {
	return &MaintenanceService{q: repository.New(db)}
}

type CreateMaintenanceRequest struct {
	VehicleID         int32      `json:"vehicleId"         validate:"required"`
	MaintenanceTypeID *int32     `json:"maintenanceTypeId"`
	Type              string     `json:"type"              validate:"required"` // e.g. routine, repair
	Status            string     `json:"status"            validate:"required"` // e.g. pending, completed
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
	Type              string     `json:"type"              validate:"required"`
	Status            string     `json:"status"            validate:"required"`
	Description       string     `json:"description"       validate:"required"`
	Odometer          *int32     `json:"odometer"`
	TotalCost         *float64   `json:"totalCost"`
	VendorName        string     `json:"vendorName"`
	Location          string     `json:"location"          validate:"required"`
	StartDate         time.Time  `json:"startDate"         validate:"required"`
	EndDate           *time.Time `json:"endDate"`
}

func serializeMaintenanceRow(m repository.ListMaintenanceRow) map[string]any {
	var proofPhotos []string
	if m.ProofPhotos.Valid && len(m.ProofPhotos.RawMessage) > 0 {
		_ = json.Unmarshal(m.ProofPhotos.RawMessage, &proofPhotos)
	}

	return map[string]any{
		"id":                m.ID,
		"vehicleId":         m.VehicleId,
		"vehicleName":       m.VehicleName,
		"plateNumber":       m.PlateNumber,
		"maintenanceTypeId": m.MaintenanceTypeId.Int32,
		"type":              m.Type,
		"status":            m.Status,
		"description":       m.Description,
		"odometer":          m.Odometer.Int32,
		"totalCost":         m.TotalCost.String,
		"vendorName":        m.VendorName.String,
		"location":          m.Location,
		"startDate":         m.StartDate,
		"endDate":           nullTime(m.EndDate),
		"completedAt":       nullTime(m.CompletedAt),
		"proofPhotos":       proofPhotos,
		"createdBy":         m.CreatedByName,
		"createdAt":         m.CreatedAt,
	}
}

func serializeMaintenanceByID(m repository.GetMaintenanceByIDRow) map[string]any {
	var proofPhotos []string
	if m.ProofPhotos.Valid && len(m.ProofPhotos.RawMessage) > 0 {
		_ = json.Unmarshal(m.ProofPhotos.RawMessage, &proofPhotos)
	}

	return map[string]any{
		"id":                m.ID,
		"vehicleId":         m.VehicleId,
		"vehicleName":       m.VehicleName,
		"plateNumber":       m.PlateNumber,
		"maintenanceTypeId": m.MaintenanceTypeId.Int32,
		"type":              m.Type,
		"status":            m.Status,
		"description":       m.Description,
		"odometer":          m.Odometer.Int32,
		"totalCost":         m.TotalCost.String,
		"vendorName":        m.VendorName.String,
		"location":          m.Location,
		"startDate":         m.StartDate,
		"endDate":           nullTime(m.EndDate),
		"completedAt":       nullTime(m.CompletedAt),
		"proofPhotos":       proofPhotos,
		"createdBy":         m.CreatedByName,
		"createdAt":         m.CreatedAt,
	}
}

func nullNumeric(f *float64) sql.NullString {
	if f == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: fmt.Sprintf("%g", *f), Valid: true}
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
		ResourceId: vehicle.ResourceId,
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

	var completedAt sql.NullTime
	if req.Status == "completed" {
		completedAt = sql.NullTime{Time: time.Now(), Valid: true}
	}

	m, err := s.q.CreateMaintenance(ctx, repository.CreateMaintenanceParams{
		VehicleId:         req.VehicleID,
		MaintenanceTypeId: nullInt32Ptr(req.MaintenanceTypeID),
		Description:       req.Description,
		Type:              req.Type,
		Status:            req.Status,
		Odometer:          nullInt32Ptr(req.Odometer),
		TotalCost:         nullNumeric(req.TotalCost),
		VendorName:        sql.NullString{String: req.VendorName, Valid: req.VendorName != ""},
		Location:          req.Location,
		StartDate:         req.StartDate,
		EndDate:           ed,
		CompletedAt:       completedAt,
		ProofPhotos:       pqtype.NullRawMessage{},
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
	existing, err := s.q.GetMaintenanceByID(ctx, id)
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

	var completedAt sql.NullTime
	if req.Status == "completed" {
		completedAt = sql.NullTime{Time: time.Now(), Valid: true}
	} else {
		// if not completed, retain existing completedAt? or clear it?
		// Usually if changing back to pending, we clear it.
		completedAt = sql.NullTime{}
	}
	if req.Status == existing.Status {
		completedAt = existing.CompletedAt
	}

	if _, err = s.q.UpdateMaintenance(ctx, repository.UpdateMaintenanceParams{
		ID:                id,
		VehicleId:         req.VehicleID,
		MaintenanceTypeId: nullInt32Ptr(req.MaintenanceTypeID),
		Description:       req.Description,
		Type:              req.Type,
		Status:            req.Status,
		Odometer:          nullInt32Ptr(req.Odometer),
		TotalCost:         nullNumeric(req.TotalCost),
		VendorName:        sql.NullString{String: req.VendorName, Valid: req.VendorName != ""},
		Location:          req.Location,
		StartDate:         req.StartDate,
		EndDate:           endDate,
		CompletedAt:       completedAt,
		ProofPhotos:       existing.ProofPhotos,
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

func (s *MaintenanceService) Complete(ctx context.Context, id int32, photos []*multipart.FileHeader) (map[string]any, error) {
	existing, err := s.q.GetMaintenanceByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if existing.Status == "completed" {
		return nil, util.NewError(400, "Maintenance is already completed", util.ErrBadRequest)
	}

	var proofPhotos []string
	if existing.ProofPhotos.Valid && len(existing.ProofPhotos.RawMessage) > 0 {
		_ = json.Unmarshal(existing.ProofPhotos.RawMessage, &proofPhotos)
	}

	for _, fh := range photos {
		filePath, err := util.SaveUploadedFile(fh, "maintenance")
		if err == nil {
			if !strings.HasPrefix(filePath, "/uploads/") {
				filePath = "/uploads/" + filePath
			}
			proofPhotos = append(proofPhotos, filePath)
		}
	}

	photosJSON, _ := json.Marshal(proofPhotos)

	if _, err = s.q.UpdateMaintenance(ctx, repository.UpdateMaintenanceParams{
		ID:                id,
		VehicleId:         existing.VehicleId,
		MaintenanceTypeId: existing.MaintenanceTypeId,
		Type:              existing.Type,
		Status:            "completed",
		Description:       existing.Description,
		Odometer:          existing.Odometer,
		TotalCost:         existing.TotalCost,
		VendorName:        existing.VendorName,
		Location:          existing.Location,
		StartDate:         existing.StartDate,
		EndDate:           existing.EndDate,
		CompletedAt:       sql.NullTime{Time: time.Now(), Valid: true},
		ProofPhotos:       pqtype.NullRawMessage{RawMessage: photosJSON, Valid: true},
	}); err != nil {
		return nil, err
	}

	// Update resource status to AVAILABLE
	vehicle, _ := s.q.GetVehicleByID(ctx, existing.VehicleId)
	if vehicle.ResourceId != 0 {
		_, _ = s.q.UpdateResourceStatus(ctx, repository.UpdateResourceStatusParams{
			ID:     vehicle.ResourceId,
			Status: repository.ResourceStatusAVAILABLE,
		})
	}

	return s.GetByID(ctx, id)
}
