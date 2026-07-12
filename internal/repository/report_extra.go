package repository

import (
	"context"
	"database/sql"
	"time"
)

// ─── Overview ────────────────────────────────────────────────────────────────

type OverviewRow struct {
	TotalBookings  int64   `json:"totalBookings"`
	TotalCost      float64 `json:"totalCost"`
	AvgUtilization float64 `json:"avgUtilization"`
	OverdueCount   int64   `json:"overdueCount"`
}

func (q *Queries) ReportOverview(ctx context.Context, start, end time.Time) (OverviewRow, error) {
	query := `
		SELECT
		    COUNT(b.id)                                                   AS total_bookings,
		    COALESCE((SELECT SUM(fe."totalCost")::float8 FROM fuel_expenses fe
		              JOIN bookings fb ON fb.id = fe."bookingId"
		              WHERE fb."startDate" >= $1 AND fb."endDate" <= $2), 0) +
		    COALESCE((SELECT SUM(mr."totalCost")::float8 FROM maintenance_records mr
		              WHERE mr."startDate" >= $1 AND mr."startDate" <= $2), 0) AS total_cost,
		    COALESCE(
		        (COUNT(CASE WHEN b.status IN ('ONGOING','COMPLETED') THEN 1 END)::float8 /
		         NULLIF(COUNT(b.id),0)) * 100, 0)                        AS avg_utilization,
		    COUNT(CASE WHEN b.status = 'OVERDUE' THEN 1 END)             AS overdue_count
		FROM bookings b
		WHERE b."startDate" >= $1 AND b."endDate" <= $2`
	var r OverviewRow
	err := q.db.QueryRowContext(ctx, query, start, end).Scan(
		&r.TotalBookings, &r.TotalCost, &r.AvgUtilization, &r.OverdueCount,
	)
	return r, err
}

// ─── Booking Trend ────────────────────────────────────────────────────────────

type BookingTrendRow struct {
	Period  string `json:"period"`
	Count   int64  `json:"count"`
	Vehicle int64  `json:"vehicle"`
	Room    int64  `json:"room"`
}

func (q *Queries) ReportBookingTrend(ctx context.Context, groupBy string, periods int) ([]BookingTrendRow, error) {
	var trunc string
	var fmt string
	switch groupBy {
	case "daily":
		trunc = "day"
		fmt = "YYYY-MM-DD"
	case "weekly":
		trunc = "week"
		fmt = "IYYY-\"W\"IW"
	default:
		trunc = "month"
		fmt = "YYYY-MM"
	}
	query := `
		SELECT
		    TO_CHAR(DATE_TRUNC($1, b."startDate"), $2) AS period,
		    COUNT(b.id)                                  AS count,
		    COUNT(CASE WHEN r.type = 'VEHICLE' THEN 1 END) AS vehicle,
		    COUNT(CASE WHEN r.type = 'ROOM'    THEN 1 END) AS room
		FROM bookings b
		JOIN resources r ON r.id = b."resourceId"
		WHERE b."startDate" >= NOW() - ($3::int || ' ' || $1)::INTERVAL
		GROUP BY DATE_TRUNC($1, b."startDate")
		ORDER BY DATE_TRUNC($1, b."startDate") ASC`
	rows, err := q.db.QueryContext(ctx, query, trunc, fmt, periods)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []BookingTrendRow
	for rows.Next() {
		var r BookingTrendRow
		if err := rows.Scan(&r.Period, &r.Count, &r.Vehicle, &r.Room); err != nil {
			return nil, err
		}
		items = append(items, r)
	}
	return items, rows.Err()
}

// ─── Bookings by Department ───────────────────────────────────────────────────

type BookingByDepartmentRow struct {
	DepartmentID   int32  `json:"departmentId"`
	DepartmentName string `json:"departmentName"`
	Total          int64  `json:"total"`
	Pending        int64  `json:"pending"`
	Approved       int64  `json:"approved"`
	Completed      int64  `json:"completed"`
	Cancelled      int64  `json:"cancelled"`
	Rejected       int64  `json:"rejected"`
}

func (q *Queries) ReportBookingsByDepartment(ctx context.Context, start, end sql.NullTime) ([]BookingByDepartmentRow, error) {
	query := `
		SELECT
		    d.id                                                          AS department_id,
		    d.name                                                        AS department_name,
		    COUNT(b.id)                                                   AS total,
		    COUNT(CASE WHEN b.status = 'PENDING'   THEN 1 END)           AS pending,
		    COUNT(CASE WHEN b.status = 'APPROVED'  THEN 1 END)           AS approved,
		    COUNT(CASE WHEN b.status = 'COMPLETED' THEN 1 END)           AS completed,
		    COUNT(CASE WHEN b.status = 'CANCELLED' THEN 1 END)           AS cancelled,
		    COUNT(CASE WHEN b.status = 'REJECTED'  THEN 1 END)           AS rejected
		FROM departments d
		LEFT JOIN users u ON u."departmentId" = d.id
		LEFT JOIN bookings b ON b."userId" = u.id
		    AND ($1::timestamptz IS NULL OR b."startDate" >= $1::timestamptz)
		    AND ($2::timestamptz IS NULL OR b."endDate"   <= $2::timestamptz)
		GROUP BY d.id, d.name
		ORDER BY total DESC`
	rows, err := q.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []BookingByDepartmentRow
	for rows.Next() {
		var r BookingByDepartmentRow
		if err := rows.Scan(&r.DepartmentID, &r.DepartmentName, &r.Total,
			&r.Pending, &r.Approved, &r.Completed, &r.Cancelled, &r.Rejected); err != nil {
			return nil, err
		}
		items = append(items, r)
	}
	return items, rows.Err()
}

// ─── Bookings by Resource ─────────────────────────────────────────────────────

type BookingByResourceRow struct {
	ResourceID    int32        `json:"resourceId"`
	ResourceName  string       `json:"resourceName"`
	ResourceType  ResourceType `json:"resourceType"`
	TotalBookings int64        `json:"totalBookings"`
	TotalHours    float64      `json:"totalHours"`
}

func (q *Queries) ReportBookingsByResource(ctx context.Context, start, end sql.NullTime) ([]BookingByResourceRow, error) {
	query := `
		SELECT
		    r.id,
		    r.name,
		    r.type,
		    COUNT(b.id)                                                AS total_bookings,
		    COALESCE(SUM(EXTRACT(EPOCH FROM (b."endDate" - b."startDate"))/3600), 0) AS total_hours
		FROM resources r
		LEFT JOIN bookings b ON b."resourceId" = r.id
		    AND ($1::timestamptz IS NULL OR b."startDate" >= $1::timestamptz)
		    AND ($2::timestamptz IS NULL OR b."endDate"   <= $2::timestamptz)
		GROUP BY r.id, r.name, r.type
		ORDER BY total_bookings DESC`
	rows, err := q.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []BookingByResourceRow
	for rows.Next() {
		var r BookingByResourceRow
		if err := rows.Scan(&r.ResourceID, &r.ResourceName, &r.ResourceType, &r.TotalBookings, &r.TotalHours); err != nil {
			return nil, err
		}
		items = append(items, r)
	}
	return items, rows.Err()
}

// ─── Approval Performance ─────────────────────────────────────────────────────

type ApprovalPerformanceRow struct {
	AvgApprovalTimeHours float64 `json:"avgApprovalTimeHours"`
	ApprovedWithin24h    float64 `json:"approvedWithin24h"`
	TotalProcessed       int64   `json:"totalProcessed"`
}

func (q *Queries) ReportApprovalPerformance(ctx context.Context, start, end sql.NullTime) (ApprovalPerformanceRow, error) {
	query := `
		SELECT
		    COALESCE(AVG(EXTRACT(EPOCH FROM (b."approvedAt" - b."createdAt"))/3600), 0) AS avg_hours,
		    COALESCE(
		        COUNT(CASE WHEN EXTRACT(EPOCH FROM (b."approvedAt" - b."createdAt"))/3600 <= 24 THEN 1 END)::float8 /
		        NULLIF(COUNT(b.id)::float8, 0) * 100, 0)                                AS within_24h_pct,
		    COUNT(b.id)                                                                  AS total_processed
		FROM bookings b
		WHERE b."approvedAt" IS NOT NULL
		  AND b.status IN ('APPROVED','COMPLETED','ONGOING','OVERDUE')
		  AND ($1::timestamptz IS NULL OR b."createdAt" >= $1::timestamptz)
		  AND ($2::timestamptz IS NULL OR b."createdAt" <= $2::timestamptz)`
	var r ApprovalPerformanceRow
	err := q.db.QueryRowContext(ctx, query, start, end).Scan(
		&r.AvgApprovalTimeHours, &r.ApprovedWithin24h, &r.TotalProcessed,
	)
	return r, err
}

// ─── Cost Summary ─────────────────────────────────────────────────────────────

type CostSummaryRow struct {
	TotalFuelCost        float64 `json:"totalFuelCost"`
	TotalMaintenanceCost float64 `json:"totalMaintenanceCost"`
	TotalCost            float64 `json:"totalCost"`
}

func (q *Queries) ReportCostSummary(ctx context.Context, start, end sql.NullTime) (CostSummaryRow, error) {
	query := `
		SELECT
		    COALESCE((SELECT SUM(fe."totalCost")::float8 FROM fuel_expenses fe
		              WHERE ($1::timestamptz IS NULL OR fe."createdAt" >= $1::timestamptz)
		                AND ($2::timestamptz IS NULL OR fe."createdAt" <= $2::timestamptz)), 0) AS fuel_cost,
		    COALESCE((SELECT SUM(mr."totalCost")::float8 FROM maintenance_records mr
		              WHERE ($1::timestamptz IS NULL OR mr."startDate" >= $1::timestamptz)
		                AND ($2::timestamptz IS NULL OR mr."startDate" <= $2::timestamptz)), 0) AS maint_cost`
	var fuelCost, maintCost float64
	if err := q.db.QueryRowContext(ctx, query, start, end).Scan(&fuelCost, &maintCost); err != nil {
		return CostSummaryRow{}, err
	}
	return CostSummaryRow{
		TotalFuelCost:        fuelCost,
		TotalMaintenanceCost: maintCost,
		TotalCost:            fuelCost + maintCost,
	}, nil
}

// ─── Cost by Vehicle ──────────────────────────────────────────────────────────

type CostByVehicleRow struct {
	VehicleID       int32   `json:"vehicleId"`
	Name            string  `json:"name"`
	PlateNumber     string  `json:"plateNumber"`
	FuelCost        float64 `json:"fuelCost"`
	MaintenanceCost float64 `json:"maintenanceCost"`
	TotalCost       float64 `json:"totalCost"`
	TotalKm         float64 `json:"totalKm"`
	AvgCostPerKm    float64 `json:"avgCostPerKm"`
}

func (q *Queries) ReportCostByVehicle(ctx context.Context, start, end sql.NullTime) ([]CostByVehicleRow, error) {
	query := `
		SELECT
		    v.id,
		    r.name,
		    v."plateNumber",
		    COALESCE(SUM(fe."totalCost")::float8, 0)                       AS fuel_cost,
		    COALESCE((SELECT SUM(mr."totalCost")::float8 FROM maintenance_records mr
		              WHERE mr."vehicleId" = v.id
		                AND ($1::timestamptz IS NULL OR mr."startDate" >= $1::timestamptz)
		                AND ($2::timestamptz IS NULL OR mr."startDate" <= $2::timestamptz)), 0) AS maint_cost,
		    COALESCE(SUM(CASE WHEN fe."odometerAfter" IS NOT NULL AND fe."odometerBefore" IS NOT NULL
		                  THEN (fe."odometerAfter" - fe."odometerBefore")::float8 ELSE 0 END), 0) AS total_km
		FROM vehicles v
		JOIN resources r ON r.id = v."resourceId"
		LEFT JOIN fuel_expenses fe ON fe."vehicleId" = v.id
		    AND ($1::timestamptz IS NULL OR fe."createdAt" >= $1::timestamptz)
		    AND ($2::timestamptz IS NULL OR fe."createdAt" <= $2::timestamptz)
		GROUP BY v.id, r.name, v."plateNumber", v."resourceId"
		ORDER BY fuel_cost DESC`
	rows, err := q.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []CostByVehicleRow
	for rows.Next() {
		var r CostByVehicleRow
		if err := rows.Scan(&r.VehicleID, &r.Name, &r.PlateNumber, &r.FuelCost, &r.MaintenanceCost, &r.TotalKm); err != nil {
			return nil, err
		}
		r.TotalCost = r.FuelCost + r.MaintenanceCost
		if r.TotalKm > 0 {
			r.AvgCostPerKm = r.TotalCost / r.TotalKm
		}
		items = append(items, r)
	}
	return items, rows.Err()
}

// ─── Cost by Department ───────────────────────────────────────────────────────

type CostByDepartmentRow struct {
	DepartmentID    int32   `json:"departmentId"`
	DepartmentName  string  `json:"departmentName"`
	BookingCount    int64   `json:"bookingCount"`
	FuelCost        float64 `json:"fuelCost"`
	MaintenanceCost float64 `json:"maintenanceCost"`
	TotalCost       float64 `json:"totalCost"`
}

func (q *Queries) ReportCostByDepartment(ctx context.Context, start, end sql.NullTime) ([]CostByDepartmentRow, error) {
	query := `
		SELECT
		    d.id,
		    d.name,
		    COUNT(DISTINCT b.id)                                           AS booking_count,
		    COALESCE(SUM(fe."totalCost")::float8, 0)                    AS fuel_cost,
		    0::float8                                                      AS maint_cost
		FROM departments d
		LEFT JOIN users u ON u."departmentId" = d.id
		LEFT JOIN bookings b ON b."userId" = u.id
		    AND ($1::timestamptz IS NULL OR b."startDate" >= $1::timestamptz)
		    AND ($2::timestamptz IS NULL OR b."endDate"   <= $2::timestamptz)
		LEFT JOIN fuel_expenses fe ON fe."bookingId" = b.id
		GROUP BY d.id, d.name
		ORDER BY fuel_cost DESC`
	rows, err := q.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []CostByDepartmentRow
	for rows.Next() {
		var r CostByDepartmentRow
		if err := rows.Scan(&r.DepartmentID, &r.DepartmentName, &r.BookingCount, &r.FuelCost, &r.MaintenanceCost); err != nil {
			return nil, err
		}
		r.TotalCost = r.FuelCost + r.MaintenanceCost
		items = append(items, r)
	}
	return items, rows.Err()
}

// ─── Cost Trend ───────────────────────────────────────────────────────────────

type CostTrendRow struct {
	Period          string  `json:"period"`
	FuelCost        float64 `json:"fuelCost"`
	MaintenanceCost float64 `json:"maintenanceCost"`
	TotalCost       float64 `json:"totalCost"`
}

func (q *Queries) ReportCostTrend(ctx context.Context, groupBy string, periods int) ([]CostTrendRow, error) {
	var trunc, fmtStr string
	switch groupBy {
	case "daily":
		trunc = "day"
		fmtStr = "YYYY-MM-DD"
	case "weekly":
		trunc = "week"
		fmtStr = "IYYY-\"W\"IW"
	default:
		trunc = "month"
		fmtStr = "YYYY-MM"
	}
	query := `
		WITH periods AS (
		    SELECT generate_series(
		        DATE_TRUNC($1, NOW() - ($3::int || ' ' || $1)::INTERVAL),
		        DATE_TRUNC($1, NOW()),
		        ('1 ' || $1)::INTERVAL
		    ) AS p
		)
		SELECT
		    TO_CHAR(p, $2) AS period,
		    COALESCE((SELECT SUM(fe."totalCost")::float8 FROM fuel_expenses fe
		              WHERE DATE_TRUNC($1, fe."createdAt") = p), 0) AS fuel_cost,
		    COALESCE((SELECT SUM(mr."totalCost")::float8 FROM maintenance_records mr
		              WHERE DATE_TRUNC($1, mr."startDate") = p), 0) AS maint_cost
		FROM periods
		ORDER BY p ASC`
	rows, err := q.db.QueryContext(ctx, query, trunc, fmtStr, periods)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []CostTrendRow
	for rows.Next() {
		var r CostTrendRow
		if err := rows.Scan(&r.Period, &r.FuelCost, &r.MaintenanceCost); err != nil {
			return nil, err
		}
		r.TotalCost = r.FuelCost + r.MaintenanceCost
		items = append(items, r)
	}
	return items, rows.Err()
}

// ─── Driver Performance ───────────────────────────────────────────────────────

type DriverPerformanceRow struct {
	DriverID     int32   `json:"driverId"`
	DriverName   string  `json:"driverName"`
	TotalTrips   int64   `json:"totalTrips"`
	TotalKm      float64 `json:"totalKm"`
	TotalFuelCost float64 `json:"totalFuelCost"`
	AvgCostPerKm float64 `json:"avgCostPerKm"`
	AvgRating    float64 `json:"avgRating"`
	TotalReviews int64   `json:"totalReviews"`
	OnTimeRate   float64 `json:"onTimeRate"`
	LateCount    int64   `json:"lateCount"`
}

func (q *Queries) ReportDriverPerformance(ctx context.Context, start, end sql.NullTime) ([]DriverPerformanceRow, error) {
	query := `
		SELECT
		    d.id,
		    u.name,
		    COUNT(DISTINCT b.id)                                                AS total_trips,
		    COALESCE(SUM(CASE WHEN fe."odometerAfter" IS NOT NULL AND fe."odometerBefore" IS NOT NULL
		                  THEN (fe."odometerAfter" - fe."odometerBefore")::float8 ELSE 0 END), 0) AS total_km,
		    COALESCE(SUM(fe."totalCost")::float8, 0)                         AS total_fuel_cost,
		    COALESCE(AVG(dr.rating::float8), 0)                                AS avg_rating,
		    COUNT(DISTINCT dr.id)                                               AS total_reviews,
		    COUNT(CASE WHEN b.status = 'OVERDUE' THEN 1 END)                   AS late_count
		FROM drivers d
		JOIN users u ON u.id = d."userId"
		LEFT JOIN bookings b ON b."assignedDriverId" = d.id
		    AND ($1::timestamptz IS NULL OR b."startDate" >= $1::timestamptz)
		    AND ($2::timestamptz IS NULL OR b."endDate"   <= $2::timestamptz)
		LEFT JOIN fuel_expenses fe ON fe."driverId" = d.id
		    AND ($1::timestamptz IS NULL OR fe."createdAt" >= $1::timestamptz)
		    AND ($2::timestamptz IS NULL OR fe."createdAt" <= $2::timestamptz)
		LEFT JOIN driver_ratings dr ON dr."driverId" = d.id
		GROUP BY d.id, u.name
		ORDER BY total_trips DESC`
	rows, err := q.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DriverPerformanceRow
	for rows.Next() {
		var r DriverPerformanceRow
		if err := rows.Scan(&r.DriverID, &r.DriverName, &r.TotalTrips, &r.TotalKm,
			&r.TotalFuelCost, &r.AvgRating, &r.TotalReviews, &r.LateCount); err != nil {
			return nil, err
		}
		if r.TotalKm > 0 {
			r.AvgCostPerKm = r.TotalFuelCost / r.TotalKm
		}
		if r.TotalTrips > 0 {
			onTime := r.TotalTrips - r.LateCount
			if onTime < 0 {
				onTime = 0
			}
			r.OnTimeRate = float64(onTime) / float64(r.TotalTrips) * 100
		}
		items = append(items, r)
	}
	return items, rows.Err()
}

// ─── Department Summary ───────────────────────────────────────────────────────

type DepartmentSummaryRow struct {
	DepartmentID    int32   `json:"departmentId"`
	DepartmentName  string  `json:"departmentName"`
	BookingCount    int64   `json:"bookingCount"`
	FuelCost        float64 `json:"fuelCost"`
	MaintenanceCost float64 `json:"maintenanceCost"`
	TotalCost       float64 `json:"totalCost"`
	TopResource     string  `json:"topResource"`
}

func (q *Queries) ReportDepartmentSummary(ctx context.Context, start, end sql.NullTime) ([]DepartmentSummaryRow, error) {
	query := `
		WITH dept_stats AS (
		    SELECT
		        d.id AS dept_id,
		        d.name AS dept_name,
		        COUNT(DISTINCT b.id)                                 AS booking_count,
		        COALESCE(SUM(fe."totalCost")::float8, 0)           AS fuel_cost
		    FROM departments d
		    LEFT JOIN users u ON u."departmentId" = d.id
		    LEFT JOIN bookings b ON b."userId" = u.id
		        AND ($1::timestamptz IS NULL OR b."startDate" >= $1::timestamptz)
		        AND ($2::timestamptz IS NULL OR b."endDate"   <= $2::timestamptz)
		    LEFT JOIN fuel_expenses fe ON fe."bookingId" = b.id
		    GROUP BY d.id, d.name
		),
		top_resource AS (
		    SELECT
		        u."departmentId" AS dept_id,
		        r.name AS resource_name,
		        COUNT(*) AS usage_count,
		        ROW_NUMBER() OVER (PARTITION BY u."departmentId" ORDER BY COUNT(*) DESC) AS rn
		    FROM bookings b
		    JOIN users u ON u.id = b."userId"
		    JOIN resources r ON r.id = b."resourceId"
		    WHERE ($1::timestamptz IS NULL OR b."startDate" >= $1::timestamptz)
		      AND ($2::timestamptz IS NULL OR b."endDate"   <= $2::timestamptz)
		    GROUP BY u."departmentId", r.name
		)
		SELECT
		    ds.dept_id, ds.dept_name, ds.booking_count, ds.fuel_cost,
		    0::float8 AS maint_cost,
		    COALESCE(tr.resource_name, '') AS top_resource
		FROM dept_stats ds
		LEFT JOIN top_resource tr ON tr.dept_id = ds.dept_id AND tr.rn = 1
		ORDER BY ds.booking_count DESC`
	rows, err := q.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DepartmentSummaryRow
	for rows.Next() {
		var r DepartmentSummaryRow
		if err := rows.Scan(&r.DepartmentID, &r.DepartmentName, &r.BookingCount,
			&r.FuelCost, &r.MaintenanceCost, &r.TopResource); err != nil {
			return nil, err
		}
		r.TotalCost = r.FuelCost + r.MaintenanceCost
		items = append(items, r)
	}
	return items, rows.Err()
}
