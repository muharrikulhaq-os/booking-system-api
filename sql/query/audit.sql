-- name: CreateAuditLog :one
INSERT INTO audit_logs ("userId", action, "entityType", "entityId", description)
VALUES ($1, $2, $3, $4, $5) RETURNING *;
