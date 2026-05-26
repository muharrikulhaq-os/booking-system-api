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

func (s *ReportService) AuditLogs(ctx context.Context, page, limit int, entityType *string, userID *int32) (any, int64, error) {
	params := repository.ReportAuditLogsParams{
		Limit:  int32(limit),
		Offset: int32((page - 1) * limit),
	}
	if entityType != nil {
		params.EntityType = sql.NullString{String: *entityType, Valid: true}
	}
	if userID != nil {
		params.UserID = sql.NullInt32{Int32: *userID, Valid: true}
	}
	rows, err := s.q.ReportAuditLogs(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	total, _ := s.q.CountAuditLogs(ctx, repository.CountAuditLogsParams{
		EntityType: params.EntityType, UserID: params.UserID,
	})
	return rows, total, nil
}
