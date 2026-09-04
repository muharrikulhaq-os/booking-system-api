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
SELECT rm.*, r.name AS resource_name, r.status AS resource_status, rku.name AS room_keeper_name
FROM rooms rm
JOIN resources r ON r.id = rm."resourceId"
LEFT JOIN room_keepers rk ON rk.id = rm."roomKeeperId"
LEFT JOIN users rku ON rku.id = rk."userId"
WHERE rm.id = $1 LIMIT 1;

-- name: CreateRoom :one
INSERT INTO rooms ("resourceId", location, capacity) VALUES ($1, $2, $3) RETURNING *;

-- name: UpdateRoom :one
UPDATE rooms SET location = $2, capacity = $3 WHERE id = $1 RETURNING *;

-- name: UpdateRoomPhoto :one
UPDATE rooms SET "photoUrl" = $2 WHERE id = $1 RETURNING *;

-- name: GetRoomKeeperIDByResourceID :one
-- Resolves a booking's resourceId down to the room's assigned room keeper
-- (nullable - a room may not have one) - used by RateRoom to attribute the
-- rating correctly.
SELECT id, "roomKeeperId" FROM rooms WHERE "resourceId" = $1 LIMIT 1;

-- name: SetRoomKeeper :one
UPDATE rooms SET "roomKeeperId" = $2 WHERE id = $1 RETURNING *;

-- name: GetRoomsByRoomKeeperID :many
SELECT rm.*, r.name AS resource_name, r.status AS resource_status
FROM rooms rm
JOIN resources r ON r.id = rm."resourceId"
WHERE rm."roomKeeperId" = $1
ORDER BY r.name ASC;

-- name: CreateRoomRating :one
INSERT INTO room_ratings ("bookingId", "roomId", "roomKeeperId", "ratedById", rating, review)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: GetRoomRatingByBooking :one
SELECT * FROM room_ratings WHERE "bookingId" = $1 LIMIT 1;

-- name: GetRoomRatings :many
-- Ratings are attributed to the room KEEPER, not the room itself.
SELECT rr.*, u.name AS rated_by_name
FROM room_ratings rr
JOIN users u ON u.id = rr."ratedById"
WHERE rr."roomKeeperId" = $1
ORDER BY rr."createdAt" DESC;
