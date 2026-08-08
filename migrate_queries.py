import os

fuel_expense_sql = """-- name: ListFuelExpenses :many
SELECT fe.*, d_u.name AS driver_name, v."plateNumber", r.name AS vehicle_name, ft.type AS fuel_type_name
FROM fuel_expenses fe
LEFT JOIN drivers d ON d.id = fe."driverId"
LEFT JOIN users d_u ON d_u.id = d."userId"
JOIN vehicles v ON v.id = fe."vehicleId"
JOIN resources r ON r.id = v."resourceId"
JOIN fuel_types ft ON ft.id = fe."fuelTypeId"
WHERE (sqlc.narg(driver_id)::int IS NULL OR fe."driverId" = sqlc.narg(driver_id)::int)
  AND (sqlc.narg(vehicle_id)::int IS NULL OR fe."vehicleId" = sqlc.narg(vehicle_id)::int)
  AND (sqlc.narg(fuel_type)::fuel_type IS NULL OR ft.type = sqlc.narg(fuel_type)::fuel_type)
ORDER BY fe."createdAt" DESC
LIMIT $1 OFFSET $2;

-- name: CountFuelExpenses :one
SELECT COUNT(*) FROM fuel_expenses fe
JOIN fuel_types ft ON ft.id = fe."fuelTypeId"
WHERE (sqlc.narg(driver_id)::int IS NULL OR fe."driverId" = sqlc.narg(driver_id)::int)
  AND (sqlc.narg(vehicle_id)::int IS NULL OR fe."vehicleId" = sqlc.narg(vehicle_id)::int)
  AND (sqlc.narg(fuel_type)::fuel_type IS NULL OR ft.type = sqlc.narg(fuel_type)::fuel_type);

-- name: GetFuelExpenseByID :one
SELECT fe.*, d_u.name AS driver_name, v."plateNumber", r.name AS vehicle_name, ft.type AS fuel_type_name
FROM fuel_expenses fe
LEFT JOIN drivers d ON d.id = fe."driverId"
LEFT JOIN users d_u ON d_u.id = d."userId"
JOIN vehicles v ON v.id = fe."vehicleId"
JOIN resources r ON r.id = v."resourceId"
JOIN fuel_types ft ON ft.id = fe."fuelTypeId"
WHERE fe.id = $1 LIMIT 1;

-- name: CreateFuelExpense :one
INSERT INTO fuel_expenses (
    "vehicleId", "fuelTypeId", "bookingId", "driverId", "recordedById",
    odometer, quantity, "pricePerUnit", "totalCost",
    "batteryBefore", "batteryAfter", location, "stationName", note
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14) RETURNING *;

-- name: UpdateFuelExpense :one
UPDATE fuel_expenses
SET "vehicleId" = $2, "fuelTypeId" = $3, "bookingId" = $4, "driverId" = $5,
    "recordedById" = $6, odometer = $7, quantity = $8, "pricePerUnit" = $9,
    "totalCost" = $10, "batteryBefore" = $11, "batteryAfter" = $12,
    location = $13, "stationName" = $14, note = $15
WHERE id = $1 RETURNING *;

-- name: DeleteFuelExpense :exec
DELETE FROM fuel_expenses WHERE id = $1;
"""

maintenance_sql = """-- name: ListMaintenance :many
SELECT mr.*, r.name AS vehicle_name, v."plateNumber",
       u.name AS created_by_name
FROM maintenance_records mr
JOIN vehicles v ON v.id = mr."vehicleId"
JOIN resources r ON r.id = v."resourceId"
JOIN users u ON u.id = mr."recordedById"
WHERE (sqlc.narg(vehicle_id)::int IS NULL OR mr."vehicleId" = sqlc.narg(vehicle_id)::int)
ORDER BY mr."startDate" DESC
LIMIT $1 OFFSET $2;

-- name: CountMaintenance :one
SELECT COUNT(*) FROM maintenance_records
WHERE (sqlc.narg(vehicle_id)::int IS NULL OR "vehicleId" = sqlc.narg(vehicle_id)::int);

-- name: GetMaintenanceByID :one
SELECT mr.*, r.name AS vehicle_name, v."plateNumber",
       u.name AS created_by_name
FROM maintenance_records mr
JOIN vehicles v ON v.id = mr."vehicleId"
JOIN resources r ON r.id = v."resourceId"
JOIN users u ON u.id = mr."recordedById"
WHERE mr.id = $1 LIMIT 1;

-- name: CreateMaintenance :one
INSERT INTO maintenance_records (
    "vehicleId", "maintenanceTypeId", description, odometer,
    "totalCost", "vendorName", location, "startDate", "endDate", "recordedById"
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING *;

-- name: UpdateMaintenance :one
UPDATE maintenance_records
SET "vehicleId" = $2, "maintenanceTypeId" = $3, description = $4, odometer = $5,
    "totalCost" = $6, "vendorName" = $7, location = $8, "startDate" = $9, "endDate" = $10
WHERE id = $1 RETURNING *;

-- name: DeleteMaintenance :exec
DELETE FROM maintenance_records WHERE id = $1;
"""

with open('sql/query/fuel_expense.sql', 'w') as f:
    f.write(fuel_expense_sql)

with open('sql/query/maintenance.sql', 'w') as f:
    f.write(maintenance_sql)

# Patch report.sql
with open('sql/query/report.sql', 'r') as f:
    report = f.read()

report = report.replace("""-- name: ReportMaintenanceCost :many
SELECT mr."resourceId", r.name AS resource_name, r.type AS resource_type,
       COUNT(mr.id) AS total_records,
       COALESCE(SUM(mr.cost), 0) AS total_cost
FROM maintenance_records mr
JOIN resources r ON r.id = mr."resourceId"
GROUP BY mr."resourceId", r.name, r.type
ORDER BY total_cost DESC;""", """-- name: ReportMaintenanceCost :many
SELECT mr."vehicleId", r.name AS resource_name, 'VEHICLE' AS resource_type,
       COUNT(mr.id) AS total_records,
       COALESCE(SUM(mr."totalCost"), 0) AS total_cost
FROM maintenance_records mr
JOIN vehicles v ON v.id = mr."vehicleId"
JOIN resources r ON r.id = v."resourceId"
GROUP BY mr."vehicleId", r.name
ORDER BY total_cost DESC;""")

report = report.replace("""COALESCE(SUM(fe."totalAmount"), 0) AS total_fuel_expenses""", """COALESCE(SUM(fe."totalCost"), 0) AS total_fuel_expenses""")

with open('sql/query/report.sql', 'w') as f:
    f.write(report)

print("Queries updated successfully.")
