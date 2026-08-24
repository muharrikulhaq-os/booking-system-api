package service

import (
	"context"
	"database/sql"
	"time"

	"booking-system-api/internal/repository"
)

type ReportService struct {
	q  repository.ExtendedQuerier
	db *sql.DB
}

func NewReportService(db *sql.DB) *ReportService {
	return &ReportService{q: repository.New(db), db: db}
}

// nullableTime converts optional *time.Time to sql.NullTime.
func nullableTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// periodBounds returns start/end times for the given named period (monthly/quarterly/yearly).
func periodBounds(period string) (start, end time.Time) {
	now := time.Now().UTC()
	switch period {
	case "quarterly":
		q := (int(now.Month())-1)/3 + 1
		start = time.Date(now.Year(), time.Month((q-1)*3+1), 1, 0, 0, 0, 0, time.UTC)
		end = start.AddDate(0, 3, 0).Add(-time.Nanosecond)
	case "yearly":
		start = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		end = start.AddDate(1, 0, 0).Add(-time.Nanosecond)
	default: // monthly
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		end = start.AddDate(0, 1, 0).Add(-time.Nanosecond)
	}
	return
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

// ─── New report methods ───────────────────────────────────────────────────────

func (s *ReportService) Overview(ctx context.Context, period string) (any, error) {
	start, end := periodBounds(period)
	prevStart, prevEnd := start.AddDate(0, 0, -int(end.Sub(start)/24/time.Hour)), start.Add(-time.Nanosecond)

	cur, err := s.q.ReportOverview(ctx, start, end)
	if err != nil {
		return nil, err
	}
	prev, _ := s.q.ReportOverview(ctx, prevStart, prevEnd)

	pct := func(cur, prev float64) float64 {
		if prev == 0 {
			return 0
		}
		return (cur - prev) / prev * 100
	}
	return map[string]any{
		"totalBookings":  cur.TotalBookings,
		"totalCost":      cur.TotalCost,
		"avgUtilization": cur.AvgUtilization,
		"overdueCount":   cur.OverdueCount,
		"previousPeriod": map[string]any{
			"totalBookings":  prev.TotalBookings,
			"totalCost":      prev.TotalCost,
			"avgUtilization": prev.AvgUtilization,
			"overdueCount":   prev.OverdueCount,
		},
		"changePercent": map[string]any{
			"bookings":    pct(float64(cur.TotalBookings), float64(prev.TotalBookings)),
			"cost":        pct(cur.TotalCost, prev.TotalCost),
			"utilization": pct(cur.AvgUtilization, prev.AvgUtilization),
			"overdue":     pct(float64(cur.OverdueCount), float64(prev.OverdueCount)),
		},
	}, nil
}

func (s *ReportService) BookingTrend(ctx context.Context, groupBy string, periods int) (any, error) {
	return s.q.ReportBookingTrend(ctx, groupBy, periods)
}

func (s *ReportService) BookingsByDepartment(ctx context.Context, start, end *time.Time) (any, error) {
	return s.q.ReportBookingsByDepartment(ctx, nullableTime(start), nullableTime(end))
}

func (s *ReportService) BookingsByResource(ctx context.Context, start, end *time.Time) (any, error) {
	return s.q.ReportBookingsByResource(ctx, nullableTime(start), nullableTime(end))
}

func (s *ReportService) ApprovalPerformance(ctx context.Context, start, end *time.Time) (any, error) {
	return s.q.ReportApprovalPerformance(ctx, nullableTime(start), nullableTime(end))
}

func (s *ReportService) CostSummary(ctx context.Context, start, end *time.Time) (any, error) {
	cur, err := s.q.ReportCostSummary(ctx, nullableTime(start), nullableTime(end))
	if err != nil {
		return nil, err
	}
	// Previous period: same duration ending just before start
	var prev repository.CostSummaryRow
	if start != nil && end != nil {
		dur := end.Sub(*start)
		prevEnd := start.Add(-time.Nanosecond)
		prevStart := prevEnd.Add(-dur)
		prev, _ = s.q.ReportCostSummary(ctx, nullableTime(&prevStart), nullableTime(&prevEnd))
	}
	pct := func(c, p float64) float64 {
		if p == 0 {
			return 0
		}
		return (c - p) / p * 100
	}
	return map[string]any{
		"totalFuelCost":        cur.TotalFuelCost,
		"totalMaintenanceCost": cur.TotalMaintenanceCost,
		"totalCost":            cur.TotalCost,
		"previousPeriod": map[string]any{
			"totalFuelCost":        prev.TotalFuelCost,
			"totalMaintenanceCost": prev.TotalMaintenanceCost,
			"totalCost":            prev.TotalCost,
		},
		"changePercent": map[string]any{
			"fuel":        pct(cur.TotalFuelCost, prev.TotalFuelCost),
			"maintenance": pct(cur.TotalMaintenanceCost, prev.TotalMaintenanceCost),
			"total":       pct(cur.TotalCost, prev.TotalCost),
		},
	}, nil
}

func (s *ReportService) CostByVehicle(ctx context.Context, start, end *time.Time) (any, error) {
	return s.q.ReportCostByVehicle(ctx, nullableTime(start), nullableTime(end))
}

func (s *ReportService) CostByDepartment(ctx context.Context, start, end *time.Time) (any, error) {
	return s.q.ReportCostByDepartment(ctx, nullableTime(start), nullableTime(end))
}

func (s *ReportService) CostTrend(ctx context.Context, groupBy string, periods int) (any, error) {
	return s.q.ReportCostTrend(ctx, groupBy, periods)
}

func (s *ReportService) DriverPerformance(ctx context.Context, start, end *time.Time) (any, error) {
	return s.q.ReportDriverPerformance(ctx, nullableTime(start), nullableTime(end))
}

func (s *ReportService) DepartmentSummary(ctx context.Context, start, end *time.Time) (any, error) {
	return s.q.ReportDepartmentSummary(ctx, nullableTime(start), nullableTime(end))
}

// DriverTrips reports, per driver, how many completed trips were SPD (surat
// perintah dinas) vs NON_SPD, plus two DISTINCT overtime-adjacent metrics
// that must not be conflated:
//   - "overtime" (keterlambatan): the trip finished after its OWN booking
//     endDate - driver_overtimes, NON_SPD only, relative to that trip's plan.
//   - "lembur": the trip finished after the fixed 18:00 WIB workday cutoff,
//     regardless of the booking's own schedule. NON_SPD only, same as
//     "overtime" - SPD never counts toward lembur, by explicit business rule.
// Also reports average rating - unlike DriverPerformance.avgRating (all-time),
// this one is scoped to the same [start, end) window as everything else here.
func (s *ReportService) DriverTrips(ctx context.Context, start, end *time.Time) (any, error) {
	rows, err := s.q.ReportDriverTrips(ctx, repository.ReportDriverTripsParams{
		StartFrom: nullableTime(start), EndTo: nullableTime(end),
	})
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		out[i] = map[string]any{
			"driverId":             r.DriverID,
			"driverName":           r.DriverName,
			"employeeId":           r.EmployeeId,
			"spdTrips":             r.SpdTrips,
			"nonSpdTrips":          r.NonSpdTrips,
			"totalTrips":           r.SpdTrips + r.NonSpdTrips,
			"overtimeTrips":        r.OvertimeTrips,
			"totalOvertimeMinutes": r.TotalOvertimeMinutes,
			"totalOvertimeHours":   float64(r.TotalOvertimeMinutes) / 60,
			"lemburTrips":          r.LemburTrips,
			"totalLemburMinutes":   r.TotalLemburMinutes,
			"totalLemburHours":     float64(r.TotalLemburMinutes) / 60,
			"avgRating":            r.AvgRating,
			"totalReviews":         r.TotalReviews,
		}
	}
	return out, nil
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
