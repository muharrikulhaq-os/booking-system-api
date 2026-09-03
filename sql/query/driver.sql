-- name: ListDrivers :many
-- assigned_plate: sqlc v1.31.1's nullability inference for anything but a
-- direct table.column reference is unreliable here — a scalar subquery
-- silently flipped this to non-nullable `string` on a regen (the original
-- bug), LEFT JOIN LATERAL didn't propagate nullability at all, and
-- NULLIF()/CASE tricks to force nullable produced flatly wrong types
-- (bool, interface{}). Rather than fight the inference, COALESCE(...,
-- '')::text pins it to a plain, always-non-null `string` deterministically
-- — same pattern already used for overlapping_purpose below. The Go side
-- (driver_service.go) checks `!= ""` instead of `.Valid`.
-- DISTINCT ON precomputes "latest assigned vehicle per driver" without
-- correlation, so it joins like an ordinary table (also avoids the original
-- correlated-subquery instability).
SELECT d.*, u.name AS user_name, u."employeeId", u.email,
       COALESCE(ap."plateNumber", '')::text AS assigned_plate
FROM drivers d
JOIN users u ON u.id = d."userId"
LEFT JOIN (
    SELECT DISTINCT ON (ab2."assignedDriverId")
        ab2."assignedDriverId" AS driver_id, v2."plateNumber"
    FROM bookings ab2
    JOIN vehicles v2 ON v2.id = ab2."assignedVehicleId"
    WHERE ab2.status IN ('APPROVED','ONGOING')
    ORDER BY ab2."assignedDriverId", ab2."startDate" DESC
) ap ON ap.driver_id = d.id
WHERE (sqlc.narg(search)::text IS NULL
       OR u.name ILIKE '%' || sqlc.narg(search)::text || '%'
       OR u.email ILIKE '%' || sqlc.narg(search)::text || '%'
       OR u."employeeId" ILIKE '%' || sqlc.narg(search)::text || '%')
  AND (sqlc.narg(is_active)::boolean IS NULL OR d."isActive" = sqlc.narg(is_active)::boolean)
ORDER BY d."createdAt" DESC
LIMIT $1 OFFSET $2;

-- name: CountDrivers :one
-- JOIN users wajib di sini: filter search menyentuh kolom milik users,
-- jadi COUNT harus melewati join yang sama dengan ListDrivers supaya
-- total pagination tetap konsisten dengan baris yang ditampilkan.
SELECT COUNT(*) FROM drivers d
JOIN users u ON u.id = d."userId"
WHERE (sqlc.narg(search)::text IS NULL
       OR u.name ILIKE '%' || sqlc.narg(search)::text || '%'
       OR u.email ILIKE '%' || sqlc.narg(search)::text || '%'
       OR u."employeeId" ILIKE '%' || sqlc.narg(search)::text || '%')
  AND (sqlc.narg(is_active)::boolean IS NULL OR d."isActive" = sqlc.narg(is_active)::boolean);

-- name: GetDriverByID :one
-- See note on ListDrivers above.
SELECT d.*, u.name AS user_name, u."employeeId", u.email, u."profilePhoto",
       COALESCE(ap."plateNumber", '')::text AS assigned_plate
FROM drivers d
JOIN users u ON u.id = d."userId"
LEFT JOIN (
    SELECT DISTINCT ON (ab2."assignedDriverId")
        ab2."assignedDriverId" AS driver_id, v2."plateNumber"
    FROM bookings ab2
    JOIN vehicles v2 ON v2.id = ab2."assignedVehicleId"
    WHERE ab2.status IN ('APPROVED','ONGOING')
    ORDER BY ab2."assignedDriverId", ab2."startDate" DESC
) ap ON ap.driver_id = d.id
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

-- name: GetVehicleCurrentAssignment :one
-- Supir yang sedang aktif memegang kendaraan ini sekarang (kalau ada).
SELECT * FROM driver_assignments
WHERE "vehicleId" = $1 AND "releasedAt" IS NULL LIMIT 1;

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
       -- Explicit ::text cast: sqlc can't always resolve the type of a
       -- COALESCE wrapped around a scalar subquery on its own and falls
       -- back to `interface{}` (seen in the generated code before this
       -- cast was added) — pin it so it stays `string`, not `sql.NullString`.
       COALESCE((SELECT b.purpose
        FROM bookings b
        WHERE b."assignedDriverId" = d.id
          AND b.status IN ('APPROVED', 'ONGOING')
          AND b."startDate" < sqlc.arg(end_to)::timestamptz
          AND b."endDate" > sqlc.arg(start_from)::timestamptz
        ORDER BY b."startDate" ASC LIMIT 1), '')::text AS overlapping_purpose
FROM drivers d
JOIN users u ON u.id = d."userId"
LEFT JOIN driver_assignments da ON da."driverId" = d.id AND da."releasedAt" IS NULL
LEFT JOIN vehicles v ON v.id = da."vehicleId"
WHERE d."isActive" = TRUE
ORDER BY overlapping_passengers ASC;

-- name: GetDriverIDsWithActiveSpd :many
-- Supir yang sedang terkunci tugas SPD hari ini (hari kalender penuh WIB) -
-- sama seperti GetVehicleIDsWithActiveSpd tapi sisi supir, dipakai untuk
-- badge "Digunakan SPD" di daftar pemilihan supir saat create booking.
SELECT DISTINCT "assignedDriverId" FROM bookings
WHERE "bookingType" = 'SPD'
  AND status IN ('APPROVED', 'ONGOING')
  AND "assignedDriverId" IS NOT NULL
  AND date_trunc('day', "startDate" AT TIME ZONE 'Asia/Jakarta') <= date_trunc('day', NOW() AT TIME ZONE 'Asia/Jakarta')
  AND date_trunc('day', "endDate" AT TIME ZONE 'Asia/Jakarta')   >= date_trunc('day', NOW() AT TIME ZONE 'Asia/Jakarta');
