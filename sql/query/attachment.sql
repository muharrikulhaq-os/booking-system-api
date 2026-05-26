-- name: ListAttachmentsByVehicle :many
SELECT a.*, u.name AS uploader_name
FROM attachments a
JOIN users u ON u.id = a."uploadedById"
WHERE a."vehicleId" = $1
ORDER BY a."createdAt" DESC;

-- name: ListAttachmentsByRoom :many
SELECT a.*, u.name AS uploader_name
FROM attachments a
JOIN users u ON u.id = a."uploadedById"
WHERE a."roomId" = $1
ORDER BY a."createdAt" DESC;

-- name: ListAttachmentsByBooking :many
SELECT a.*, u.name AS uploader_name
FROM attachments a
JOIN users u ON u.id = a."uploadedById"
WHERE a."bookingId" = $1
ORDER BY a."createdAt" DESC;

-- name: GetAttachmentByID :one
SELECT a.*, u.name AS uploader_name
FROM attachments a
JOIN users u ON u.id = a."uploadedById"
WHERE a.id = $1 LIMIT 1;

-- name: CreateAttachmentForVehicle :one
INSERT INTO attachments ("uploadedById", "vehicleId", "filePath", "fileName", "fileType", "fileSize", description)
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;

-- name: CreateAttachmentForRoom :one
INSERT INTO attachments ("uploadedById", "roomId", "filePath", "fileName", "fileType", "fileSize", description)
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;

-- name: CreateAttachmentForBooking :one
INSERT INTO attachments ("uploadedById", "bookingId", "filePath", "fileName", "fileType", "fileSize", description)
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;

-- name: DeleteAttachment :exec
DELETE FROM attachments WHERE id = $1;
