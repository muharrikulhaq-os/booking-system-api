-- name: CreateDriverOvertime :one
INSERT INTO driver_overtimes (
    "bookingId", "driverId", "scheduledEndAt", "actualEndAt", "overtimeMinutes"
) VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: GetOvertimeByBooking :one
SELECT * FROM driver_overtimes WHERE "bookingId" = $1 LIMIT 1;

-- name: ListOvertimeByDriver :many
SELECT ot.*, b.purpose, b."startDate", b."endDate" AS "bookingEndDate"
FROM driver_overtimes ot
JOIN bookings b ON b.id = ot."bookingId"
WHERE ot."driverId" = $1
ORDER BY ot."createdAt" DESC;
