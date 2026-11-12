-- name: ListRooms :many
-- ORDER BY di sini cuma placeholder/default untuk dokumentasi - versi
-- runtime-nya (internal/repository/room.sql.go) mengganti klausanya secara
-- dinamis lewat BuildOrderBy (internal/repository/sort.go), lihat catatan di
-- ListVehicles (sql/query/vehicle.sql).
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
  AND (NOT sqlc.arg(bookable_only)::boolean OR (r.status NOT IN ('INACTIVE', 'MAINTENANCE')
       AND EXISTS (
           SELECT 1 FROM room_keeper_rooms rkr
           JOIN room_keepers rk ON rk.id = rkr."roomKeeperId"
           JOIN users ku ON ku.id = rk."userId"
           WHERE rkr."roomId" = rm.id AND rk."isActive" AND ku."isActive")))
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
       ))
  AND (NOT sqlc.arg(bookable_only)::boolean OR (r.status NOT IN ('INACTIVE', 'MAINTENANCE')
       AND EXISTS (
           SELECT 1 FROM room_keeper_rooms rkr
           JOIN room_keepers rk ON rk.id = rkr."roomKeeperId"
           JOIN users ku ON ku.id = rk."userId"
           WHERE rkr."roomId" = rm.id AND rk."isActive" AND ku."isActive")));

-- name: GetRoomByID :one
SELECT rm.id, rm."resourceId", rm.location, rm.capacity, rm."photoUrl", r.name AS resource_name, r.status AS resource_status
FROM rooms rm
JOIN resources r ON r.id = rm."resourceId"
WHERE rm.id = $1 LIMIT 1;

-- name: CreateRoom :one
INSERT INTO rooms ("resourceId", location, capacity) VALUES ($1, $2, $3) RETURNING *;

-- name: UpdateRoom :one
UPDATE rooms SET location = $2, capacity = $3 WHERE id = $1 RETURNING *;

-- name: UpdateRoomPhoto :one
UPDATE rooms SET "photoUrl" = $2 WHERE id = $1 RETURNING *;

-- Penjaga ruangan N:N (room_keeper_rooms, migrasi 000024). Versi
-- runtime ada di internal/repository/room_extra.go.

-- name: ListRoomKeepersByResource :many
SELECT rkr."roomId", rk.id, u.id, u.name, COALESCE(NULLIF(rk."phoneNumber", ''), u."phoneNumber", ''),
       (rk."isActive" AND u."isActive")
FROM room_keeper_rooms rkr
JOIN room_keepers rk ON rk.id = rkr."roomKeeperId"
JOIN users u ON u.id = rk."userId"
JOIN rooms rm ON rm.id = rkr."roomId"
WHERE rm."resourceId" = $1
ORDER BY u.name ASC;

-- name: SetRoomKeepers :exec
WITH del AS (
    DELETE FROM room_keeper_rooms
    WHERE "roomId" = $1 AND NOT ("roomKeeperId" = ANY($2::int[]))
)
INSERT INTO room_keeper_rooms ("roomId", "roomKeeperId")
SELECT $1, k FROM unnest($2::int[]) AS k
ON CONFLICT DO NOTHING;

-- name: IsRoomKeeperOfResource :one
SELECT EXISTS (
    SELECT 1 FROM room_keeper_rooms rkr
    JOIN rooms rm ON rm.id = rkr."roomId"
    WHERE rkr."roomKeeperId" = $1 AND rm."resourceId" = $2
);

-- name: GetRoomsByRoomKeeperID :many
SELECT rm.id, rm."resourceId", rm.location, rm.capacity, rm."photoUrl", rkr."roomKeeperId", r.name AS resource_name, r.status AS resource_status
FROM room_keeper_rooms rkr
JOIN rooms rm ON rm.id = rkr."roomId"
JOIN resources r ON r.id = rm."resourceId"
WHERE rkr."roomKeeperId" = $1
ORDER BY r.name ASC;

-- name: CreateRoomRating :one
INSERT INTO room_ratings ("bookingId", "roomId", "roomKeeperId", "ratedById", rating, review)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: GetRoomRatingByBooking :one
SELECT * FROM room_ratings WHERE "bookingId" = $1 LIMIT 1;

-- name: ListRoomRatingsByBooking :many
-- Satu booking bisa punya beberapa rating: satu per penjaga.
SELECT rr."roomKeeperId", u.name, rr.rating, rr.review, rr."createdAt"
FROM room_ratings rr
LEFT JOIN room_keepers rk ON rk.id = rr."roomKeeperId"
LEFT JOIN users u ON u.id = rk."userId"
WHERE rr."bookingId" = $1
ORDER BY rr."createdAt" ASC;

-- name: GetRoomRatings :many
-- Ratings are attributed to the room KEEPER, not the room itself.
SELECT rr.*, u.name AS rated_by_name
FROM room_ratings rr
JOIN users u ON u.id = rr."ratedById"
WHERE rr."roomKeeperId" = $1
ORDER BY rr."createdAt" DESC;
