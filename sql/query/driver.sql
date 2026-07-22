-- name: ListDrivers :many
SELECT d.*, u.name AS user_name, u."employeeId", u.email,
       (SELECT v2."plateNumber" FROM bookings ab2
          JOIN vehicles v2 ON v2.id = ab2."assignedVehicleId"
          WHERE ab2."assignedDriverId" = d.id AND ab2.status IN ('APPROVED','ONGOING')
          ORDER BY ab2."startDate" DESC LIMIT 1) AS assigned_plate
FROM drivers d
JOIN users u ON u.id = d."userId"
WHERE (sqlc.narg(is_active)::boolean IS NULL OR d."isActive" = sqlc.narg(is_active)::boolean)
ORDER BY d."createdAt" DESC
LIMIT $1 OFFSET $2;

-- name: CountDrivers :one
SELECT COUNT(*) FROM drivers d
WHERE (sqlc.narg(is_active)::boolean IS NULL OR d."isActive" = sqlc.narg(is_active)::boolean);

-- name: GetDriverByID :one
SELECT d.*, u.name AS user_name, u."employeeId", u.email, u."profilePhoto",
       (SELECT v2."plateNumber" FROM bookings ab2
          JOIN vehicles v2 ON v2.id = ab2."assignedVehicleId"
          WHERE ab2."assignedDriverId" = d.id AND ab2.status IN ('APPROVED','ONGOING')
          ORDER BY ab2."startDate" DESC LIMIT 1) AS assigned_plate
FROM drivers d
JOIN users u ON u.id = d."userId"
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

-- name: ListAvailableDrivers :many
SELECT d.id AS driver_id, u.name AS driver_name, u."employeeId",
       v.id AS vehicle_id, v."plateNumber", v.capacity,
       COALESCE(
           (SELECT SUM(b."passengerCount")::int
            FROM bookings b
            WHERE b."assignedVehicleId" = v.id
              AND b.status IN ('APPROVED', 'ONGOING')
              AND b."startDate" < sqlc.arg(end_to)::timestamptz
              AND b."endDate" > sqlc.arg(start_from)::timestamptz
           ), 0
       )::int AS overlapping_passengers,
       COALESCE((SELECT b.purpose
        FROM bookings b
        WHERE b."assignedDriverId" = d.id
          AND b.status IN ('APPROVED', 'ONGOING')
          AND b."startDate" < sqlc.arg(end_to)::timestamptz
          AND b."endDate" > sqlc.arg(start_from)::timestamptz
        ORDER BY b."startDate" ASC LIMIT 1), '') AS overlapping_purpose
FROM drivers d
JOIN users u ON u.id = d."userId"
LEFT JOIN driver_assignments da ON da."driverId" = d.id AND da."releasedAt" IS NULL
LEFT JOIN vehicles v ON v.id = da."vehicleId"
WHERE d."isActive" = TRUE
ORDER BY overlapping_passengers ASC;
