-- name: ListFuelTypes :many
SELECT * FROM fuel_types
ORDER BY id ASC;

-- name: GetFuelType :one
SELECT * FROM fuel_types
WHERE id = $1 LIMIT 1;

-- name: CreateFuelType :one
INSERT INTO fuel_types (
    name, type, unit, default_price, is_active
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: UpdateFuelType :one
UPDATE fuel_types
SET name = $2,
    type = $3,
    unit = $4,
    default_price = $5,
    is_active = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteFuelType :exec
DELETE FROM fuel_types
WHERE id = $1;
