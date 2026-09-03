-- name: ListVehicles :many
-- Saat status filter = AVAILABLE, resource yang punya booking APPROVED/ONGOING
-- yang overlap dengan waktu sekarang ikut disembunyikan — bukan cuma yang sudah
-- di-Start (resources.status = IN_USE), tapi juga yang baru APPROVED tapi jadwalnya
-- sudah berjalan. Filter status lain (MAINTENANCE/INACTIVE/IN_USE) tidak terpengaruh.
SELECT v.*, r.name AS resource_name, r.status AS resource_status,
       vc.name AS category_name, fdu.name AS fixed_driver_name
FROM vehicles v
JOIN resources r ON r.id = v."resourceId"
JOIN vehicle_categories vc ON vc.id = v."categoryId"
LEFT JOIN drivers fd ON fd.id = v."fixedDriverId"
LEFT JOIN users fdu ON fdu.id = fd."userId"
WHERE (sqlc.narg(search)::text IS NULL
       OR r.name ILIKE '%' || sqlc.narg(search)::text || '%'
       OR v."plateNumber" ILIKE '%' || sqlc.narg(search)::text || '%'
       OR v.brand ILIKE '%' || sqlc.narg(search)::text || '%')
  AND (sqlc.narg(category_id)::int IS NULL OR v."categoryId" = sqlc.narg(category_id)::int)
  AND (sqlc.narg(status)::resource_status IS NULL OR r.status = sqlc.narg(status)::resource_status)
  AND (sqlc.narg(status)::resource_status IS DISTINCT FROM 'AVAILABLE'::resource_status
       OR NOT EXISTS (
           SELECT 1 FROM bookings bk
           WHERE bk."resourceId" = r.id
             AND bk.status IN ('APPROVED', 'ONGOING')
             AND NOW() BETWEEN bk."startDate" AND bk."endDate"
       ))
ORDER BY r."createdAt" DESC
LIMIT $1 OFFSET $2;

-- name: CountVehicles :one
SELECT COUNT(*) FROM vehicles v
JOIN resources r ON r.id = v."resourceId"
WHERE (sqlc.narg(search)::text IS NULL
       OR r.name ILIKE '%' || sqlc.narg(search)::text || '%'
       OR v."plateNumber" ILIKE '%' || sqlc.narg(search)::text || '%')
  AND (sqlc.narg(category_id)::int IS NULL OR v."categoryId" = sqlc.narg(category_id)::int)
  AND (sqlc.narg(status)::resource_status IS NULL OR r.status = sqlc.narg(status)::resource_status)
  AND (sqlc.narg(status)::resource_status IS DISTINCT FROM 'AVAILABLE'::resource_status
       OR NOT EXISTS (
           SELECT 1 FROM bookings bk
           WHERE bk."resourceId" = r.id
             AND bk.status IN ('APPROVED', 'ONGOING')
             AND NOW() BETWEEN bk."startDate" AND bk."endDate"
       ));

-- name: GetVehicleByID :one
SELECT v.*, r.name AS resource_name, r.status AS resource_status,
       vc.name AS category_name, fdu.name AS fixed_driver_name
FROM vehicles v
JOIN resources r ON r.id = v."resourceId"
JOIN vehicle_categories vc ON vc.id = v."categoryId"
LEFT JOIN drivers fd ON fd.id = v."fixedDriverId"
LEFT JOIN users fdu ON fdu.id = fd."userId"
WHERE v.id = $1 LIMIT 1;

-- name: GetVehicleByPlate :one
SELECT * FROM vehicles WHERE "plateNumber" = $1 LIMIT 1;

-- name: CreateResource :one
INSERT INTO resources (name, type, status) VALUES ($1, $2, 'AVAILABLE') RETURNING *;

-- name: CreateVehicle :one
INSERT INTO vehicles ("resourceId", "plateNumber", brand, model, year,
                       "currentOdometer", "categoryId", capacity, energy_type)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING *;

-- name: UpdateVehicle :one
UPDATE vehicles
SET "plateNumber" = $2, brand = $3, model = $4, year = $5,
    "currentOdometer" = $6, "categoryId" = $7, capacity = $8, energy_type = $9
WHERE id = $1 RETURNING *;

-- name: UpdateResourceName :exec
UPDATE resources SET name = $2, "updatedAt" = NOW() WHERE id = $1;

-- name: UpdateResourceStatus :one
UPDATE resources SET status = $2, "updatedAt" = NOW() WHERE id = $1 RETURNING *;

-- name: DeleteResource :exec
DELETE FROM resources WHERE id = $1;

-- name: UpdateVehiclePhoto :one
UPDATE vehicles SET "photoUrl" = $2 WHERE id = $1 RETURNING *;

-- name: ListVehicleCategories :many
SELECT * FROM vehicle_categories ORDER BY name;

-- name: GetVehicleCategoryByID :one
SELECT * FROM vehicle_categories WHERE id = $1 LIMIT 1;

-- name: CreateVehicleCategory :one
INSERT INTO vehicle_categories (name) VALUES ($1) RETURNING *;

-- name: DeleteVehicleCategory :exec
DELETE FROM vehicle_categories WHERE id = $1;

-- name: GetResourceByID :one
SELECT * FROM resources WHERE id = $1 LIMIT 1;

-- name: UpdateVehicleOdometer :one
-- Only moves the odometer forward; ignores lower/stale readings.
UPDATE vehicles
SET "currentOdometer" = GREATEST("currentOdometer", $2)
WHERE id = $1 RETURNING *;

-- name: UpdateVehicleLastMaintenanceOdometer :exec
UPDATE vehicles SET "lastMaintenanceOdometer" = $2 WHERE id = $1;

-- name: SetVehicleFixedDriver :one
UPDATE vehicles SET "fixedDriverId" = $2 WHERE id = $1 RETURNING *;

-- name: ClearVehicleFixedDriverByDriver :exec
-- Dipanggil sebelum menetapkan supir tetap baru ke kendaraan lain - UNIQUE
-- di kolom fixedDriverId cuma izinkan satu kendaraan per supir, jadi ikatan
-- lama harus dilepas dulu supaya UPDATE berikutnya tidak bentrok constraint.
UPDATE vehicles SET "fixedDriverId" = NULL WHERE "fixedDriverId" = $1;

-- name: GetVehicleByFixedDriverID :one
SELECT * FROM vehicles WHERE "fixedDriverId" = $1 LIMIT 1;
