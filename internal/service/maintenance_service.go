package service

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
	ResourceID  int32     `json:"resourceId"  validate:"required"`
	Description string    `json:"description" validate:"required"`
	StartDate   time.Time `json:"startDate"   validate:"required"`
	Cost        *float64  `json:"cost"`
}

type UpdateMaintenanceRequest struct {
	Description string     `json:"description" validate:"required"`
	StartDate   time.Time  `json:"startDate"   validate:"required"`
	EndDate     *time.Time `json:"endDate"`
	Cost        *float64   `json:"cost"`
}

func serializeMaintenanceRow(m repository.ListMaintenanceRow) map[string]any {
	return map[string]any{
		"id":           m.ID,
		"resourceId":   m.ResourceId,
		"resourceName": m.ResourceName,
		"resourceType": string(m.ResourceType),
		"description":  m.Description,
		"startDate":    m.StartDate,
		"endDate":      nullTime(m.EndDate),
		"cost":         m.Cost,
		"createdBy":    m.CreatedByName,
		"createdAt":    m.CreatedAt,
	}
}

func serializeMaintenanceByID(m repository.GetMaintenanceByIDRow) map[string]any {
	return map[string]any{
		"id":           m.ID,
		"resourceId":   m.ResourceId,
		"resourceName": m.ResourceName,
		"resourceType": string(m.ResourceType),
		"description":  m.Description,
		"startDate":    m.StartDate,
		"endDate":      nullTime(m.EndDate),
		"cost":         m.Cost,
		"createdBy":    m.CreatedByName,
		"createdAt":    m.CreatedAt,
	}
}

func (s *MaintenanceService) List(ctx context.Context, page, limit int, resourceID *int32) ([]map[string]any, int64, error) {
	params := repository.ListMaintenanceParams{
		Limit:  int32(limit),
		Offset: int32((page - 1) * limit),
	}
	if resourceID != nil {
		params.ResourceID = sql.NullInt32{Int32: *resourceID, Valid: true}
	}
	rows, err := s.q.ListMaintenance(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	total, _ := s.q.CountMaintenance(ctx, params.ResourceID)
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

func (s *MaintenanceService) Create(ctx context.Context, req CreateMaintenanceRequest, createdByID int32) (map[string]any, error) {
	m, err := s.q.CreateMaintenance(ctx, repository.CreateMaintenanceParams{
		ResourceId:  req.ResourceID,
		Description: req.Description,
		StartDate:   req.StartDate,
		Cost:        nullNumeric(req.Cost),
		CreatedById: createdByID,
	})
	if err != nil {
		return nil, err
	}

	// set resource status to MAINTENANCE
	_, _ = s.q.UpdateResourceStatus(ctx, repository.UpdateResourceStatusParams{
		ID:     req.ResourceID,
		Status: repository.ResourceStatusMAINTENANCE,
	})

	return s.GetByID(ctx, m.ID)
}

func (s *MaintenanceService) Update(ctx context.Context, id int32, req UpdateMaintenanceRequest) (map[string]any, error) {
	m, err := s.q.GetMaintenanceByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}

	var endDate sql.NullTime
	if req.EndDate != nil {
		endDate = sql.NullTime{Time: *req.EndDate, Valid: true}
	}

	if _, err = s.q.UpdateMaintenance(ctx, repository.UpdateMaintenanceParams{
		ID:          id,
		Description: req.Description,
		StartDate:   req.StartDate,
		EndDate:     endDate,
		Cost:        nullNumeric(req.Cost),
	}); err != nil {
		return nil, err
	}

	// if endDate provided, mark resource AVAILABLE again
	if req.EndDate != nil {
		_, _ = s.q.UpdateResourceStatus(ctx, repository.UpdateResourceStatusParams{
			ID:     m.ResourceId,
			Status: repository.ResourceStatusAVAILABLE,
		})
	}

	return s.GetByID(ctx, id)
}

func (s *MaintenanceService) Delete(ctx context.Context, id int32) error {
	if _, err := s.q.GetMaintenanceByID(ctx, id); err != nil {
		return util.ErrNotFound
	}
	return s.q.DeleteMaintenance(ctx, id)
}
