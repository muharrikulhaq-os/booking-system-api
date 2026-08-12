-- name: ListRoomKeepers :many
SELECT rk.*, u.name AS user_name, u."employeeId", u.email
FROM room_keepers rk
JOIN users u ON u.id = rk."userId"
WHERE (sqlc.narg(is_active)::boolean IS NULL OR rk."isActive" = sqlc.narg(is_active)::boolean)
ORDER BY rk."createdAt" DESC
LIMIT $1 OFFSET $2;

-- name: CountRoomKeepers :one
SELECT COUNT(*) FROM room_keepers rk
WHERE (sqlc.narg(is_active)::boolean IS NULL OR rk."isActive" = sqlc.narg(is_active)::boolean);

-- name: GetRoomKeeperByID :one
SELECT rk.*, u.name AS user_name, u."employeeId", u.email, u."profilePhoto"
FROM room_keepers rk
JOIN users u ON u.id = rk."userId"
WHERE rk.id = $1 LIMIT 1;

-- name: CreateRoomKeeper :one
INSERT INTO room_keepers ("userId", "phoneNumber", "isActive")
VALUES ($1, $2, TRUE) RETURNING *;

-- name: UpdateRoomKeeper :one
UPDATE room_keepers SET "phoneNumber" = $2 WHERE id = $1 RETURNING *;

-- name: ToggleRoomKeeperActive :one
UPDATE room_keepers SET "isActive" = NOT "isActive" WHERE id = $1 RETURNING *;
