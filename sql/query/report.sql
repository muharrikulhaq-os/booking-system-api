-- name: ReportBookingSummary :one
SELECT
    COUNT(*) AS total,
    SUM(CASE WHEN status = 'COMPLETED' THEN 1 ELSE 0 END) AS completed,
    SUM(CASE WHEN status = 'PENDING'   THEN 1 ELSE 0 END) AS pending,
    SUM(CASE WHEN status = 'APPROVED'  THEN 1 ELSE 0 END) AS approved,
    SUM(CASE WHEN status = 'ONGOING'   THEN 1 ELSE 0 END) AS ongoing,
    SUM(CASE WHEN status = 'CANCELLED' THEN 1 ELSE 0 END) AS cancelled,
    SUM(CASE WHEN status = 'REJECTED'  THEN 1 ELSE 0 END) AS rejected,
    SUM(CASE WHEN status = 'OVERDUE'   THEN 1 ELSE 0 END) AS overdue
FROM bookings
WHERE (sqlc.narg(start_from)::timestamptz IS NULL OR "startDate" >= sqlc.narg(start_from)::timestamptz)
  AND (sqlc.narg(end_to)::timestamptz IS NULL OR "endDate" <= sqlc.narg(end_to)::timestamptz);

-- name: ReportResourceUsage :many
SELECT * FROM v_vehicle_summary ORDER BY total_bookings DESC;

-- name: ReportFuelExpenses :many
SELECT * FROM v_fuel_expense_summary ORDER BY grand_total DESC;

-- name: ReportMaintenanceCost :many
SELECT mr."resourceId", r.name AS resource_name, r.type AS resource_type,
       COUNT(mr.id) AS total_records,
       COALESCE(SUM(mr.cost), 0) AS total_cost
FROM maintenance_records mr
JOIN resources r ON r.id = mr."resourceId"
GROUP BY mr."resourceId", r.name, r.type
ORDER BY total_cost DESC;

-- name: ReportDriverRatings :many
SELECT * FROM v_driver_ratings_summary ORDER BY average_rating DESC NULLS LAST;

-- name: ReportDriverActivity :many
SELECT d.id AS driver_id, u.name AS driver_name, u."employeeId",
       COUNT(DISTINCT b.id) AS total_bookings,
       SUM(CASE WHEN b.status = 'COMPLETED' THEN 1 ELSE 0 END) AS completed_bookings,
       COALESCE(SUM(fe."totalAmount"), 0) AS total_fuel_expenses
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
