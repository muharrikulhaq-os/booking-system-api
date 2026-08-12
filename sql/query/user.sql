-- name: ListUsers :many
SELECT u.*, r.name AS role_name, d.name AS department_name
FROM users u
JOIN roles r ON r.id = u."roleId"
JOIN departments d ON d.id = u."departmentId"
WHERE (sqlc.narg(search)::text IS NULL
       OR u.name ILIKE '%' || sqlc.narg(search)::text || '%'
       OR u.email ILIKE '%' || sqlc.narg(search)::text || '%'
       OR u."employeeId" ILIKE '%' || sqlc.narg(search)::text || '%')
  AND (sqlc.narg(role_id)::int IS NULL OR u."roleId" = sqlc.narg(role_id)::int)
  AND (sqlc.narg(is_active)::boolean IS NULL OR u."isActive" = sqlc.narg(is_active)::boolean)
  AND (sqlc.narg(department_id)::int IS NULL OR u."departmentId" = sqlc.narg(department_id)::int)
ORDER BY u."createdAt" DESC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*) FROM users u
WHERE (sqlc.narg(search)::text IS NULL
       OR u.name ILIKE '%' || sqlc.narg(search)::text || '%'
       OR u.email ILIKE '%' || sqlc.narg(search)::text || '%'
       OR u."employeeId" ILIKE '%' || sqlc.narg(search)::text || '%')
  AND (sqlc.narg(role_id)::int IS NULL OR u."roleId" = sqlc.narg(role_id)::int)
  AND (sqlc.narg(is_active)::boolean IS NULL OR u."isActive" = sqlc.narg(is_active)::boolean)
  AND (sqlc.narg(department_id)::int IS NULL OR u."departmentId" = sqlc.narg(department_id)::int);

-- name: UserSummaryTotals :one
SELECT
    COUNT(*)                                        AS total_users,
    COUNT(*) FILTER (WHERE u."isActive")            AS active_users,
    COUNT(*) FILTER (WHERE NOT u."isActive")        AS inactive_users,
    COUNT(*) FILTER (WHERE u."profilePhoto" IS NOT NULL) AS with_photo,
    COUNT(*) FILTER (WHERE u."createdAt" >= date_trunc('month', NOW())) AS new_this_month,
    COUNT(*) FILTER (WHERE u."createdAt" >= NOW() - INTERVAL '30 days')  AS new_last_30_days
FROM users u;

-- name: UserSummaryByRole :many
-- LEFT JOIN dari roles supaya role yang belum punya user tetap muncul dengan total 0.
SELECT
    r.id                                     AS role_id,
    r.name                                   AS role_name,
    COUNT(u.id)                              AS total,
    COUNT(u.id) FILTER (WHERE u."isActive")  AS active,
    COUNT(u.id) FILTER (WHERE NOT u."isActive") AS inactive
FROM roles r
LEFT JOIN users u ON u."roleId" = r.id
GROUP BY r.id, r.name
ORDER BY r.id;

-- name: UserSummaryByDepartment :many
SELECT
    d.id                                     AS department_id,
    d.name                                   AS department_name,
    COUNT(u.id)                              AS total,
    COUNT(u.id) FILTER (WHERE u."isActive")  AS active,
    COUNT(u.id) FILTER (WHERE NOT u."isActive") AS inactive
FROM departments d
LEFT JOIN users u ON u."departmentId" = d.id
GROUP BY d.id, d.name
ORDER BY COUNT(u.id) DESC, d.name;

-- name: UserSummaryByRoleDepartment :many
-- Matriks role x department, hanya kombinasi yang benar-benar punya user.
SELECT
    r.id     AS role_id,
    r.name   AS role_name,
    d.id     AS department_id,
    d.name   AS department_name,
    COUNT(u.id)                                 AS total,
    COUNT(u.id) FILTER (WHERE u."isActive")     AS active,
    COUNT(u.id) FILTER (WHERE NOT u."isActive") AS inactive
FROM users u
JOIN roles r       ON r.id = u."roleId"
JOIN departments d ON d.id = u."departmentId"
GROUP BY r.id, r.name, d.id, d.name
ORDER BY r.id, COUNT(u.id) DESC, d.name;

-- name: CreateUser :one
INSERT INTO users ("employeeId", name, email, password, "isActive", "roleId", "departmentId")
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET name = $2, email = $3, "roleId" = $4, "departmentId" = $5, "updatedAt" = NOW()
WHERE id = $1
RETURNING *;

-- name: ToggleUserActive :one
UPDATE users SET "isActive" = NOT "isActive", "updatedAt" = NOW()
WHERE id = $1 RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: UpdateProfilePhoto :one
UPDATE users SET "profilePhoto" = $2, "updatedAt" = NOW()
WHERE id = $1 RETURNING *;

-- name: DeleteProfilePhoto :one
UPDATE users SET "profilePhoto" = NULL, "updatedAt" = NOW()
WHERE id = $1 RETURNING *;

-- name: GetUserByEmployeeID :one
SELECT * FROM users WHERE "employeeId" = $1 LIMIT 1;

-- name: ListRoles :many
SELECT * FROM roles ORDER BY id;

-- name: ListDepartments :many
SELECT * FROM departments ORDER BY name;

-- name: CreateDepartment :one
INSERT INTO departments (name) VALUES ($1) RETURNING *;

-- name: UpdateDepartment :one
UPDATE departments SET name = $2 WHERE id = $1 RETURNING *;

-- name: DeleteDepartment :exec
DELETE FROM departments WHERE id = $1;

-- name: GetDepartmentByID :one
SELECT * FROM departments WHERE id = $1 LIMIT 1;
