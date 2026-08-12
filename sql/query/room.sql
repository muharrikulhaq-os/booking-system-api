-- name: ListRooms :many
-- Saat status filter = AVAILABLE, resource yang punya booking APPROVED/ONGOING
-- yang overlap dengan waktu sekarang ikut disembunyikan — lihat catatan di ListVehicles.
SELECT rm.*, r.name AS resource_name, r.status AS resource_status
FROM rooms rm
JOIN resources r ON r.id = rm."resourceId"
WHERE (sqlc.narg(search)::text IS NULL
       OR r.name ILIKE '%' || sqlc.narg(search)::text || '%'
       OR rm.location ILIKE '%' || sqlc.narg(search)::text || '%')
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

-- name: CountRooms :one
SELECT COUNT(*) FROM rooms rm
JOIN resources r ON r.id = rm."resourceId"
WHERE (sqlc.narg(search)::text IS NULL
       OR r.name ILIKE '%' || sqlc.narg(search)::text || '%')
  AND (sqlc.narg(status)::resource_status IS NULL OR r.status = sqlc.narg(status)::resource_status)
  AND (sqlc.narg(status)::resource_status IS DISTINCT FROM 'AVAILABLE'::resource_status
       OR NOT EXISTS (
           SELECT 1 FROM bookings bk
           WHERE bk."resourceId" = r.id
             AND bk.status IN ('APPROVED', 'ONGOING')
             AND NOW() BETWEEN bk."startDate" AND bk."endDate"
       ));

-- name: GetRoomByID :one
SELECT rm.*, r.name AS resource_name, r.status AS resource_status
FROM rooms rm
JOIN resources r ON r.id = rm."resourceId"
WHERE rm.id = $1 LIMIT 1;

-- name: CreateRoom :one
INSERT INTO rooms ("resourceId", location, capacity) VALUES ($1, $2, $3) RETURNING *;

-- name: UpdateRoom :one
UPDATE rooms SET location = $2, capacity = $3 WHERE id = $1 RETURNING *;

-- name: UpdateRoomPhoto :one
UPDATE rooms SET "photoUrl" = $2 WHERE id = $1 RETURNING *;
