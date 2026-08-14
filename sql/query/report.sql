-- name: ReportBookingSummary :one
SELECT
    COUNT(*) AS total,
    COALESCE(SUM(CASE WHEN status = 'COMPLETED' THEN 1 ELSE 0 END), 0)::bigint AS completed,
    COALESCE(SUM(CASE WHEN status = 'PENDING'   THEN 1 ELSE 0 END), 0)::bigint AS pending,
    COALESCE(SUM(CASE WHEN status = 'APPROVED'  THEN 1 ELSE 0 END), 0)::bigint AS approved,
    COALESCE(SUM(CASE WHEN status = 'ONGOING'   THEN 1 ELSE 0 END), 0)::bigint AS ongoing,
    COALESCE(SUM(CASE WHEN status = 'CANCELLED' THEN 1 ELSE 0 END), 0)::bigint AS cancelled,
    COALESCE(SUM(CASE WHEN status = 'REJECTED'  THEN 1 ELSE 0 END), 0)::bigint AS rejected,
    COALESCE(SUM(CASE WHEN status = 'OVERDUE'   THEN 1 ELSE 0 END), 0)::bigint AS overdue
FROM bookings
WHERE (sqlc.narg(start_from)::timestamptz IS NULL OR "startDate" >= sqlc.narg(start_from)::timestamptz)
  AND (sqlc.narg(end_to)::timestamptz IS NULL OR "endDate" <= sqlc.narg(end_to)::timestamptz);

-- name: ReportResourceUsage :many
SELECT * FROM v_vehicle_summary ORDER BY total_bookings DESC;

-- name: ReportFuelExpenses :many
SELECT * FROM v_fuel_expense_summary ORDER BY grand_total DESC;

-- name: ReportMaintenanceCost :many
SELECT mr."vehicleId", r.name AS resource_name, 'VEHICLE' AS resource_type,
       COUNT(mr.id) AS total_records,
       COALESCE(SUM(mr."totalCost"), 0) AS total_cost
FROM maintenance_records mr
JOIN vehicles v ON v.id = mr."vehicleId"
JOIN resources r ON r.id = v."resourceId"
GROUP BY mr."vehicleId", r.name
ORDER BY total_cost DESC;

-- name: ReportDriverRatings :many
SELECT * FROM v_driver_ratings_summary ORDER BY average_rating DESC NULLS LAST;

-- name: ReportDriverActivity :many
SELECT d.id AS driver_id, u.name AS driver_name, u."employeeId",
       COUNT(DISTINCT b.id) AS total_bookings,
       COALESCE(SUM(CASE WHEN b.status = 'COMPLETED' THEN 1 ELSE 0 END), 0)::bigint AS completed_bookings,
       COALESCE(SUM(fe."totalCost"), 0) AS total_fuel_expenses
FROM drivers d
JOIN users u ON u.id = d."userId"
LEFT JOIN bookings b ON b."assignedDriverId" = d.id
LEFT JOIN fuel_expenses fe ON fe."driverId" = d.id
GROUP BY d.id, u.name, u."employeeId"
ORDER BY total_bookings DESC;

-- name: ReportOverdueBookings :many
SELECT b.*,
       u.name AS user_name, u."employeeId",
       r.name AS resource_name, r.type AS resource_type
FROM bookings b
JOIN users u ON u.id = b."userId"
JOIN resources r ON r.id = b."resourceId"
WHERE b.status = 'OVERDUE'
ORDER BY b."endDate" ASC;

-- name: ReportAuditLogs :many
SELECT al.*, u.name AS user_name
FROM audit_logs al
LEFT JOIN users u ON u.id = al."userId"
WHERE (sqlc.narg(entity_type)::text IS NULL OR al."entityType" = sqlc.narg(entity_type)::text)
  AND (sqlc.narg(user_id)::int IS NULL OR al."userId" = sqlc.narg(user_id)::int)
ORDER BY al."createdAt" DESC
LIMIT $1 OFFSET $2;

-- name: CountAuditLogs :one
SELECT COUNT(*) FROM audit_logs
WHERE (sqlc.narg(entity_type)::text IS NULL OR "entityType" = sqlc.narg(entity_type)::text)
  AND (sqlc.narg(user_id)::int IS NULL OR "userId" = sqlc.narg(user_id)::int);

-- name: ReportDriverTrips :many
-- Rekap jumlah trip SPD vs Non-SPD dan total overtime per driver.
SELECT
    d.id AS driver_id, u.name AS driver_name, u."employeeId",
    COUNT(DISTINCT b.id) FILTER (WHERE b."bookingType" = 'SPD')     AS spd_trips,
    COUNT(DISTINCT b.id) FILTER (WHERE b."bookingType" = 'NON_SPD') AS non_spd_trips,
    COUNT(DISTINCT ot.id)                                           AS overtime_trips,
    COALESCE(SUM(ot."overtimeMinutes"), 0)::int                     AS total_overtime_minutes
FROM drivers d
JOIN users u ON u.id = d."userId"
LEFT JOIN bookings b ON b."assignedDriverId" = d.id
    AND b.status = 'COMPLETED'
    AND (sqlc.narg(start_from)::timestamptz IS NULL OR b."startDate" >= sqlc.narg(start_from)::timestamptz)
    AND (sqlc.narg(end_to)::timestamptz IS NULL OR b."endDate" <= sqlc.narg(end_to)::timestamptz)
LEFT JOIN driver_overtimes ot ON ot."bookingId" = b.id
GROUP BY d.id, u.name, u."employeeId"
ORDER BY (spd_trips + non_spd_trips) DESC;
