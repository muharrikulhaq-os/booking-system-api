-- name: ListMaintenance :many
SELECT mr.*, r.name AS resource_name, r.type AS resource_type,
       u.name AS created_by_name
FROM maintenance_records mr
JOIN resources r ON r.id = mr."resourceId"
JOIN users u ON u.id = mr."createdById"
WHERE (sqlc.narg(resource_id)::int IS NULL OR mr."resourceId" = sqlc.narg(resource_id)::int)
ORDER BY mr."startDate" DESC
LIMIT $1 OFFSET $2;

-- name: CountMaintenance :one
SELECT COUNT(*) FROM maintenance_records
WHERE (sqlc.narg(resource_id)::int IS NULL OR "resourceId" = sqlc.narg(resource_id)::int);

-- name: GetMaintenanceByID :one
SELECT mr.*, r.name AS resource_name, r.type AS resource_type,
       u.name AS created_by_name
FROM maintenance_records mr
JOIN resources r ON r.id = mr."resourceId"
JOIN users u ON u.id = mr."createdById"
WHERE mr.id = $1 LIMIT 1;

-- name: CreateMaintenance :one
INSERT INTO maintenance_records ("resourceId", description, "startDate", cost, "createdById")
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: UpdateMaintenance :one
UPDATE maintenance_records
SET description = $2, "startDate" = $3, "endDate" = $4, cost = $5
WHERE id = $1 RETURNING *;

-- name: DeleteMaintenance :exec
DELETE FROM maintenance_records WHERE id = $1;
