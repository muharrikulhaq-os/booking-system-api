-- name: ListDrivers :many
SELECT d.*, u.name AS user_name, u."employeeId", u.email,
       v."plateNumber" AS assigned_plate
FROM drivers d
JOIN users u ON u.id = d."userId"
LEFT JOIN driver_assignments da ON da."driverId" = d.id AND da."releasedAt" IS NULL
LEFT JOIN vehicles v ON v.id = da."vehicleId"
WHERE (sqlc.narg(is_active)::boolean IS NULL OR d."isActive" = sqlc.narg(is_active)::boolean)
ORDER BY d."createdAt" DESC
LIMIT $1 OFFSET $2;

-- name: CountDrivers :one
SELECT COUNT(*) FROM drivers d
WHERE (sqlc.narg(is_active)::boolean IS NULL OR d."isActive" = sqlc.narg(is_active)::boolean);

-- name: GetDriverByID :one
SELECT d.*, u.name AS user_name, u."employeeId", u.email, u."profilePhoto",
       v."plateNumber" AS assigned_plate
FROM drivers d
JOIN users u ON u.id = d."userId"
LEFT JOIN driver_assignments da ON da."driverId" = d.id AND da."releasedAt" IS NULL
LEFT JOIN vehicles v ON v.id = da."vehicleId"
WHERE d.id = $1 LIMIT 1;

-- name: GetDriverByUserID :one
SELECT * FROM drivers WHERE "userId" = $1 LIMIT 1;

-- name: CreateDriver :one
INSERT INTO drivers ("userId", "licenseNumber", "phoneNumber", "isActive")
VALUES ($1, $2, $3, TRUE) RETURNING *;

-- name: UpdateDriver :one
UPDATE drivers SET "licenseNumber" = $2, "phoneNumber" = $3 WHERE id = $1 RETURNING *;

-- name: ToggleDriverActive :one
UPDATE drivers SET "isActive" = NOT "isActive" WHERE id = $1 RETURNING *;

-- name: AssignDriverToVehicle :one
INSERT INTO driver_assignments ("driverId", "vehicleId") VALUES ($1, $2) RETURNING *;

-- name: ReleaseDriver :exec
UPDATE driver_assignments SET "releasedAt" = NOW()
WHERE "driverId" = $1 AND "releasedAt" IS NULL;

-- name: GetDriverCurrentAssignment :one
SELECT da.*, v."plateNumber", v.brand, v.model
FROM driver_assignments da
JOIN vehicles v ON v.id = da."vehicleId"
WHERE da."driverId" = $1 AND da."releasedAt" IS NULL LIMIT 1;

-- name: GetDriverAssignmentHistory :many
SELECT da.*, v."plateNumber", v.brand, v.model
FROM driver_assignments da
JOIN vehicles v ON v.id = da."vehicleId"
WHERE da."driverId" = $1
ORDER BY da."assignedAt" DESC;
