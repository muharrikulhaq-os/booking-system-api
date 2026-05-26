-- name: ListMasterSettings :many
SELECT * FROM master_settings ORDER BY key;

-- name: GetMasterSettingByKey :one
SELECT * FROM master_settings WHERE key = $1 LIMIT 1;

-- name: UpsertMasterSetting :one
INSERT INTO master_settings (key, value, unit, description)
VALUES ($1, $2, $3, $4)
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    unit = EXCLUDED.unit,
    description = EXCLUDED.description,
    "updatedAt" = NOW()
RETURNING *;
