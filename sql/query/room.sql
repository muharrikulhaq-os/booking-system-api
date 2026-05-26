-- name: ListRooms :many
SELECT rm.*, r.name AS resource_name, r.status AS resource_status
FROM rooms rm
JOIN resources r ON r.id = rm."resourceId"
WHERE (sqlc.narg(search)::text IS NULL
       OR r.name ILIKE '%' || sqlc.narg(search)::text || '%'
       OR rm.location ILIKE '%' || sqlc.narg(search)::text || '%')
  AND (sqlc.narg(status)::resource_status IS NULL OR r.status = sqlc.narg(status)::resource_status)
ORDER BY r."createdAt" DESC
LIMIT $1 OFFSET $2;

-- name: CountRooms :one
SELECT COUNT(*) FROM rooms rm
JOIN resources r ON r.id = rm."resourceId"
WHERE (sqlc.narg(search)::text IS NULL
       OR r.name ILIKE '%' || sqlc.narg(search)::text || '%')
  AND (sqlc.narg(status)::resource_status IS NULL OR r.status = sqlc.narg(status)::resource_status);

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
