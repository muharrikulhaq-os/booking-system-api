-- name: ListFuelExpenses :many
SELECT fe.*, d_u.name AS driver_name, v."plateNumber", r.name AS vehicle_name
FROM fuel_expenses fe
JOIN drivers d ON d.id = fe."driverId"
JOIN users d_u ON d_u.id = d."userId"
JOIN vehicles v ON v.id = fe."vehicleId"
JOIN resources r ON r.id = v."resourceId"
WHERE (sqlc.narg(driver_id)::int IS NULL OR fe."driverId" = sqlc.narg(driver_id)::int)
  AND (sqlc.narg(vehicle_id)::int IS NULL OR fe."vehicleId" = sqlc.narg(vehicle_id)::int)
  AND (sqlc.narg(fuel_type)::fuel_type IS NULL OR fe."fuelType" = sqlc.narg(fuel_type)::fuel_type)
ORDER BY fe."createdAt" DESC
LIMIT $1 OFFSET $2;

-- name: CountFuelExpenses :one
SELECT COUNT(*) FROM fuel_expenses fe
WHERE (sqlc.narg(driver_id)::int IS NULL OR fe."driverId" = sqlc.narg(driver_id)::int)
  AND (sqlc.narg(vehicle_id)::int IS NULL OR fe."vehicleId" = sqlc.narg(vehicle_id)::int)
  AND (sqlc.narg(fuel_type)::fuel_type IS NULL OR fe."fuelType" = sqlc.narg(fuel_type)::fuel_type);

-- name: GetFuelExpenseByID :one
SELECT fe.*, d_u.name AS driver_name, v."plateNumber", r.name AS vehicle_name
FROM fuel_expenses fe
JOIN drivers d ON d.id = fe."driverId"
JOIN users d_u ON d_u.id = d."userId"
JOIN vehicles v ON v.id = fe."vehicleId"
JOIN resources r ON r.id = v."resourceId"
WHERE fe.id = $1 LIMIT 1;

-- name: CreateFuelExpenseBBM :one
INSERT INTO fuel_expenses (
    "driverId", "vehicleId", "bookingId", "fuelType",
    liter, "pricePerLiter", "odometerBefore", "odometerAfter",
    "totalAmount", note
) VALUES ($1, $2, $3, 'BBM', $4, $5, $6, $7, $8, $9) RETURNING *;

-- name: CreateFuelExpenseListrik :one
INSERT INTO fuel_expenses (
    "driverId", "vehicleId", "bookingId", "fuelType",
    kwh, "pricePerKwh", "batteryBefore", "batteryAfter",
    "totalAmount", note
) VALUES ($1, $2, $3, 'LISTRIK', $4, $5, $6, $7, $8, $9) RETURNING *;

-- name: UpdateFuelExpenseBBM :one
UPDATE fuel_expenses
SET liter = $2, "pricePerLiter" = $3, "odometerBefore" = $4,
    "odometerAfter" = $5, "totalAmount" = $6, note = $7
WHERE id = $1 RETURNING *;

-- name: UpdateFuelExpenseListrik :one
UPDATE fuel_expenses
SET kwh = $2, "pricePerKwh" = $3, "batteryBefore" = $4,
    "batteryAfter" = $5, "totalAmount" = $6, note = $7
WHERE id = $1 RETURNING *;

-- name: DeleteFuelExpense :exec
DELETE FROM fuel_expenses WHERE id = $1;
