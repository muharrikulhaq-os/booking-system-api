package service

import (
	"context"
	"database/sql"
	"time"

	"booking-system-api/internal/repository"
)

type ReportService struct {
	q *repository.Queries
}

func NewReportService(db *sql.DB) *ReportService {
	return &ReportService{q: repository.New(db)}
}

func (s *ReportService) BookingSummary(ctx context.Context, startFrom, endTo *time.Time) (any, error) {
	params := repository.ReportBookingSummaryParams{}
	if startFrom != nil {
		params.StartFrom = sql.NullTime{Time: *startFrom, Valid: true}
	}
	if endTo != nil {
		params.EndTo = sql.NullTime{Time: *endTo, Valid: true}
	}
	return s.q.ReportBookingSummary(ctx, params)
}

func (s *ReportService) ResourceUsage(ctx context.Context) (any, error) {
	return s.q.ReportResourceUsage(ctx)
}

func (s *ReportService) FuelExpenses(ctx context.Context) (any, error) {
	return s.q.ReportFuelExpenses(ctx)
}

func (s *ReportService) MaintenanceCost(ctx context.Context) (any, error) {
	return s.q.ReportMaintenanceCost(ctx)
}

func (s *ReportService) DriverRatings(ctx context.Context) (any, error) {
	return s.q.ReportDriverRatings(ctx)
}

func (s *ReportService) DriverActivity(ctx context.Context) (any, error) {
	return s.q.ReportDriverActivity(ctx)
}

func (s *ReportService) OverdueBookings(ctx context.Context) (any, error) {
	return s.q.ReportOverdueBookings(ctx)
}

func (s *ReportService) AuditLogs(
	ctx context.Context,
	page, limit int,
	entityType *string,
	userID *int32,
) (any, int64, error) {

	params := repository.ReportAuditLogsParams{
		Limit:  int32(limit),
		Offset: int32((page - 1) * limit),
	}

	if entityType != nil {
		params.EntityType = sql.NullString{
			String: *entityType,
			Valid:  true,
		}
	}

	if userID != nil {
		params.UserID = sql.NullInt32{
			Int32: *userID,
			Valid: true,
		}
	}

	rows, err := s.q.ReportAuditLogs(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	resp := make([]repository.AuditLogResponse, 0, len(rows))
	for _, row := range rows {
		resp = append(resp, toAuditLogResponse(row))
	}

	total, err := s.q.CountAuditLogs(ctx, repository.CountAuditLogsParams{
		EntityType: params.EntityType,
		UserID:     params.UserID,
	})
	if err != nil {
		return nil, 0, err
	}

	return resp, total, nil
}

func toAuditLogResponse(row repository.ReportAuditLogsRow) repository.AuditLogResponse {
	var userID *int32
	if row.UserId.Valid {
		userID = &row.UserId.Int32
	}

	var entityID *int32
	if row.EntityId.Valid {
		entityID = &row.EntityId.Int32
	}

	var description *string
	if row.Description.Valid {
		description = &row.Description.String
	}

	var userName *string
	if row.UserName.Valid {
		userName = &row.UserName.String
	}

	return repository.AuditLogResponse{
		ID:          row.ID,
		UserID:      userID,
		Action:      row.Action,
		EntityType:  row.EntityType,
		EntityID:    entityID,
		Description: description,
		CreatedAt:   row.CreatedAt.Format("2006-05-01"),
		UserName:    userName,
	}
}
