-- name: ListGuestBookings :many
SELECT gb.*, r.name AS resource_name, r.type AS resource_type,
       u.name AS approver_name
FROM guest_bookings gb
JOIN resources r ON r.id = gb."resourceId"
LEFT JOIN users u ON u.id = gb."approvedById"
WHERE (sqlc.narg(status)::booking_status IS NULL OR gb.status = sqlc.narg(status)::booking_status)
ORDER BY gb."createdAt" DESC
LIMIT $1 OFFSET $2;

-- name: CountGuestBookings :one
SELECT COUNT(*) FROM guest_bookings
WHERE (sqlc.narg(status)::booking_status IS NULL OR status = sqlc.narg(status)::booking_status);

-- name: GetGuestBookingByID :one
SELECT gb.*, r.name AS resource_name, r.type AS resource_type,
       u.name AS approver_name
FROM guest_bookings gb
JOIN resources r ON r.id = gb."resourceId"
LEFT JOIN users u ON u.id = gb."approvedById"
WHERE gb.id = $1 LIMIT 1;

-- name: GetGuestBookingByToken :one
SELECT gb.*, r.name AS resource_name, r.type AS resource_type
FROM guest_bookings gb
JOIN resources r ON r.id = gb."resourceId"
WHERE gb."accessToken" = $1 LIMIT 1;

-- name: CreateGuestBooking :one
INSERT INTO guest_bookings (
    "guestName", "guestEmail", "guestPhone", "departmentName",
    "resourceId", "startDate", "endDate", purpose, "accessToken"
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING *;

-- name: UpdateGuestBookingStatus :one
UPDATE guest_bookings SET status = $2, "updatedAt" = NOW() WHERE id = $1 RETURNING *;

-- name: ApproveGuestBooking :one
UPDATE guest_bookings
SET status = 'APPROVED', "approvedById" = $2, "approvedAt" = NOW(), "updatedAt" = NOW()
WHERE id = $1 RETURNING *;

-- name: RejectGuestBooking :one
UPDATE guest_bookings
SET status = 'REJECTED', "approvedById" = $2, "approvedAt" = NOW(),
    "rejectionNote" = $3, "updatedAt" = NOW()
WHERE id = $1 RETURNING *;

-- name: StartGuestBooking :one
UPDATE guest_bookings SET status = 'ONGOING', "updatedAt" = NOW() WHERE id = $1 RETURNING *;

-- name: CompleteGuestBookingByToken :one
UPDATE guest_bookings
SET status = 'COMPLETED', "returnedAt" = NOW(), "updatedAt" = NOW()
WHERE "accessToken" = $1 RETURNING *;

-- name: CancelGuestBookingByToken :one
UPDATE guest_bookings
SET status = 'CANCELLED', "updatedAt" = NOW()
WHERE "accessToken" = $1 RETURNING *;
