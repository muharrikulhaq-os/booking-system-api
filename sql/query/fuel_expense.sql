-- name: ListFuelExpenses :many
SELECT fe.*, d_u.name AS driver_name, v."plateNumber", r.name AS vehicle_name, ft.type AS fuel_category_name
FROM fuel_expenses fe
LEFT JOIN drivers d ON d.id = fe."driverId"
LEFT JOIN users d_u ON d_u.id = d."userId"
JOIN vehicles v ON v.id = fe."vehicleId"
JOIN resources r ON r.id = v."resourceId"
JOIN fuel_types ft ON ft.id = fe."fuelTypeId"
WHERE (sqlc.narg(driver_id)::int IS NULL OR fe."driverId" = sqlc.narg(driver_id)::int)
  AND (sqlc.narg(vehicle_id)::int IS NULL OR fe."vehicleId" = sqlc.narg(vehicle_id)::int)
  AND (sqlc.narg(fuel_category)::fuel_category IS NULL OR ft.type = sqlc.narg(fuel_category)::fuel_category)
ORDER BY fe."createdAt" DESC
LIMIT $1 OFFSET $2;

-- name: CountFuelExpenses :one
SELECT COUNT(*) FROM fuel_expenses fe
JOIN fuel_types ft ON ft.id = fe."fuelTypeId"
WHERE (sqlc.narg(driver_id)::int IS NULL OR fe."driverId" = sqlc.narg(driver_id)::int)
  AND (sqlc.narg(vehicle_id)::int IS NULL OR fe."vehicleId" = sqlc.narg(vehicle_id)::int)
  AND (sqlc.narg(fuel_category)::fuel_category IS NULL OR ft.type = sqlc.narg(fuel_category)::fuel_category);

-- name: GetFuelExpenseByID :one
SELECT fe.*, d_u.name AS driver_name, v."plateNumber", r.name AS vehicle_name, ft.type AS fuel_category_name
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
