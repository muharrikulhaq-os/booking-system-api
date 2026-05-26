-- name: GetUserByEmail :one
SELECT u.*, r.name AS role_name, d.name AS department_name
FROM users u
JOIN roles r ON r.id = u."roleId"
JOIN departments d ON d.id = u."departmentId"
WHERE u.email = $1 LIMIT 1;

-- name: GetUserByID :one
SELECT u.*, r.name AS role_name, d.name AS department_name
FROM users u
JOIN roles r ON r.id = u."roleId"
JOIN departments d ON d.id = u."departmentId"
WHERE u.id = $1 LIMIT 1;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens ("userId", token, "expiresAt", revoked)
VALUES ($1, $2, $3, FALSE)
RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens
WHERE token = $1 AND "userId" = $2 AND revoked = FALSE LIMIT 1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens SET revoked = TRUE
WHERE token = $1 AND "userId" = $2;

-- name: CreateOTP :one
INSERT INTO password_reset_otps ("userId", "otpCode", "expiresAt", "isUsed")
VALUES ($1, $2, $3, FALSE)
RETURNING *;

-- name: InvalidatePreviousOTPs :exec
UPDATE password_reset_otps SET "isUsed" = TRUE
WHERE "userId" = $1 AND "isUsed" = FALSE;

-- name: GetValidOTP :one
SELECT * FROM password_reset_otps
WHERE "userId" = $1 AND "otpCode" = $2 AND "isUsed" = FALSE
ORDER BY id DESC LIMIT 1;

-- name: MarkOTPUsed :exec
UPDATE password_reset_otps SET "isUsed" = TRUE WHERE id = $1;

-- name: UpdateUserPassword :exec
UPDATE users SET password = $1, "updatedAt" = NOW() WHERE id = $2;
