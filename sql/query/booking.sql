-- name: ListBookings :many
SELECT b.*,
       u.name AS user_name, u."employeeId", dept.name AS department_name,
       r.name AS resource_name, r.type AS resource_type, r.status AS resource_status,
       COALESCE(rv."photoUrl", rr."photoUrl") AS resource_photo_url,
       ab.name AS approver_name,
       drv.id AS driver_id, du.name AS driver_name, drv."phoneNumber" AS driver_phone,
       v.id AS vehicle_id, v."plateNumber", v.brand, v.model, v.capacity,
       (
           SELECT EXISTS (
               SELECT 1 FROM bookings b2
               WHERE b2.id != b.id
                 AND b2.status = 'PENDING'
                 AND b.status = 'PENDING'
                 AND b2."resourceId" = b."resourceId"
                 AND (
                     (b2."startDate" <= b."endDate" AND b2."endDate" >= b."startDate")
                 )
           )
       ) AS has_merge_suggestion,
       orig.name AS original_resource_name,
       merged_by."primaryBookingId" AS merged_into_id,
       (SELECT COUNT(*) FROM booking_merges bm WHERE bm."primaryBookingId" = b.id) AS merge_count
FROM bookings b
JOIN users u ON u.id = b."userId"
JOIN departments dept ON dept.id = u."departmentId"
JOIN resources r ON r.id = b."resourceId"
LEFT JOIN vehicles rv ON rv."resourceId" = r.id
LEFT JOIN rooms rr ON rr."resourceId" = r.id
LEFT JOIN resources orig ON orig.id = b."originalResourceId"
LEFT JOIN users ab ON ab.id = b."approvedById"
LEFT JOIN drivers drv ON drv.id = b."assignedDriverId"
LEFT JOIN users du ON du.id = drv."userId"
LEFT JOIN vehicles v ON v.id = b."assignedVehicleId"
LEFT JOIN booking_merges merged_by ON merged_by."mergedBookingId" = b.id
WHERE (sqlc.narg(user_id)::int IS NULL OR b."userId" = sqlc.narg(user_id)::int)
  AND (sqlc.narg(status)::booking_status IS NULL OR b.status = sqlc.narg(status)::booking_status)
  AND (sqlc.narg(resource_id)::int IS NULL OR b."resourceId" = sqlc.narg(resource_id)::int)
  AND (sqlc.narg(resource_type)::resource_type IS NULL OR r.type = sqlc.narg(resource_type)::resource_type)
  AND (sqlc.narg(driver_id)::int IS NULL OR b."assignedDriverId" = sqlc.narg(driver_id)::int)
  AND (sqlc.narg(start_from)::timestamptz IS NULL OR b."startDate" >= sqlc.narg(start_from)::timestamptz)
  AND (sqlc.narg(end_to)::timestamptz IS NULL OR b."endDate" <= sqlc.narg(end_to)::timestamptz)
  AND (sqlc.narg(search)::text IS NULL OR r.name ILIKE '%' || sqlc.narg(search)::text || '%' OR u.name ILIKE '%' || sqlc.narg(search)::text || '%')
ORDER BY b."createdAt" DESC
LIMIT $1 OFFSET $2;

-- name: CountBookings :one
SELECT COUNT(*) FROM bookings b
JOIN users u ON u.id = b."userId"
JOIN resources r ON r.id = b."resourceId"
WHERE (sqlc.narg(user_id)::int IS NULL OR b."userId" = sqlc.narg(user_id)::int)
  AND (sqlc.narg(status)::booking_status IS NULL OR b.status = sqlc.narg(status)::booking_status)
  AND (sqlc.narg(resource_id)::int IS NULL OR b."resourceId" = sqlc.narg(resource_id)::int)
  AND (sqlc.narg(resource_type)::resource_type IS NULL OR r.type = sqlc.narg(resource_type)::resource_type)
  AND (sqlc.narg(driver_id)::int IS NULL OR b."assignedDriverId" = sqlc.narg(driver_id)::int)
  AND (sqlc.narg(start_from)::timestamptz IS NULL OR b."startDate" >= sqlc.narg(start_from)::timestamptz)
  AND (sqlc.narg(end_to)::timestamptz IS NULL OR b."endDate" <= sqlc.narg(end_to)::timestamptz)
  AND (sqlc.narg(search)::text IS NULL OR r.name ILIKE '%' || sqlc.narg(search)::text || '%' OR u.name ILIKE '%' || sqlc.narg(search)::text || '%');

-- name: GetBookingByID :one
SELECT b.*,
       u.name AS user_name, u."employeeId", dept.name AS department_name,
       r.name AS resource_name, r.type AS resource_type, r.status AS resource_status,
       COALESCE(rv."photoUrl", rr."photoUrl") AS resource_photo_url,
       ab.name AS approver_name,
       drv.id AS driver_id, du.name AS driver_name, drv."phoneNumber" AS driver_phone,
       v.id AS vehicle_id, v."plateNumber", v.brand, v.model, v.capacity,
       (
           SELECT EXISTS (
               SELECT 1 FROM bookings b2
               WHERE b2.id != b.id
                 AND b2.status = 'PENDING'
                 AND b.status = 'PENDING'
                 AND b2."resourceId" = b."resourceId"
                 AND (
                     (b2."startDate" <= b."endDate" AND b2."endDate" >= b."startDate")
                 )
           )
       ) AS has_merge_suggestion,
       orig.name AS original_resource_name,
       orig.type AS original_resource_type,
       b."odometerStart", b."startLocation", b."startPhotoUrl"
FROM bookings b
JOIN users u ON u.id = b."userId"
JOIN departments dept ON dept.id = u."departmentId"
JOIN resources r ON r.id = b."resourceId"
LEFT JOIN vehicles rv ON rv."resourceId" = r.id
LEFT JOIN rooms rr ON rr."resourceId" = r.id
LEFT JOIN resources orig ON orig.id = b."originalResourceId"
LEFT JOIN users ab ON ab.id = b."approvedById"
LEFT JOIN drivers drv ON drv.id = b."assignedDriverId"
LEFT JOIN users du ON du.id = drv."userId"
LEFT JOIN vehicles v ON v.id = b."assignedVehicleId"
WHERE b.id = $1 LIMIT 1;

-- name: CreateBooking :one
INSERT INTO bookings (
    "userId", "resourceId", "startDate", "endDate", purpose,
    "passengerCount", "assignedDriverId", "assignedVehicleId", status, "bookingType"
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'PENDING', $9) RETURNING *;

-- name: UpdateBookingStatus :one
UPDATE bookings SET status = $2, "updatedAt" = NOW() WHERE id = $1 RETURNING *;

-- name: ApproveBooking :one
UPDATE bookings
SET status = 'APPROVED', "approvedById" = $2, "approvedAt" = NOW(), "updatedAt" = NOW()
WHERE id = $1 RETURNING *;

-- name: RejectBooking :one
UPDATE bookings
SET status = 'REJECTED', "approvedById" = $2, "approvedAt" = NOW(), "updatedAt" = NOW()
WHERE id = $1 RETURNING *;

-- name: AssignVehicleToBooking :one
UPDATE bookings
SET "assignedDriverId" = $2, "assignedVehicleId" = $3, "assignedAt" = NOW(), "updatedAt" = NOW()
WHERE id = $1 RETURNING *;

-- name: StartBooking :one
UPDATE bookings SET status = 'ONGOING', "updatedAt" = NOW() WHERE id = $1 RETURNING *;

-- name: CompleteBooking :one
UPDATE bookings
SET status = 'COMPLETED', "returnedAt" = NOW(), "updatedAt" = NOW()
WHERE id = $1 RETURNING *;

-- name: CancelBooking :one
UPDATE bookings SET status = 'CANCELLED', "updatedAt" = NOW() WHERE id = $1 RETURNING *;

-- name: CheckBookingConflict :one
SELECT COUNT(*) FROM bookings
WHERE "resourceId" = $1
  AND status IN ('PENDING', 'APPROVED', 'ONGOING')
  AND "startDate" < sqlc.arg(check_end)
  AND "endDate" > sqlc.arg(check_start)
  AND (sqlc.narg(exclude_id)::int IS NULL OR id != sqlc.narg(exclude_id)::int);

-- name: CheckVehicleConflict :one
SELECT COUNT(*) FROM bookings
WHERE "assignedVehicleId" = $1
  AND status IN ('APPROVED', 'ONGOING')
  AND "startDate" < sqlc.arg(check_end)
  AND "endDate" > sqlc.arg(check_start)
  AND id != sqlc.arg(exclude_id);

-- name: CheckVehicleSpdConflict :one
-- Konflik KHUSUS SPD - hanya menyala kalau booking existing ATAU booking
-- yang sedang dicek (new_is_spd) adalah SPD. SPD mengklaim seluruh hari
-- kalender yang disentuh jadwalnya, bukan cuma jam spesifik - sisi existing
-- di-expand di sini via date_trunc, sisi baru di-expand oleh caller (lihat
-- effectiveConflictWindow di booking_service.go) sebelum jadi check_start/
-- check_end. Kalau kedua sisi NON_SPD, query ini sengaja tidak match apa
-- pun - overlap jam biasa tetap lewat CheckVehicleConflict di atas (bisa
-- di-resolve via merge). Hand-written di booking_extra.go (sqlc tidak
-- tersedia di environment ini) - definisi di sini untuk regen berikutnya.
SELECT COUNT(*) FROM bookings
WHERE "assignedVehicleId" = sqlc.arg(vehicle_id)
  AND status IN ('APPROVED', 'ONGOING')
  AND id != sqlc.arg(exclude_id)
  AND (sqlc.arg(new_is_spd)::bool OR "bookingType" = 'SPD')
  AND CASE WHEN "bookingType" = 'SPD'
        THEN date_trunc('day', "startDate")
        ELSE "startDate" END < sqlc.arg(check_end)::timestamptz
  AND CASE WHEN "bookingType" = 'SPD'
        THEN date_trunc('day', "endDate") + INTERVAL '1 day'
        ELSE "endDate" END > sqlc.arg(check_start)::timestamptz;

-- name: CheckDriverSpdConflict :one
-- Sama seperti CheckVehicleSpdConflict, untuk supir (assignedDriverId).
SELECT COUNT(*) FROM bookings
WHERE "assignedDriverId" = sqlc.arg(driver_id)
  AND status IN ('APPROVED', 'ONGOING')
  AND id != sqlc.arg(exclude_id)
  AND (sqlc.arg(new_is_spd)::bool OR "bookingType" = 'SPD')
  AND CASE WHEN "bookingType" = 'SPD'
        THEN date_trunc('day', "startDate")
        ELSE "startDate" END < sqlc.arg(check_end)::timestamptz
  AND CASE WHEN "bookingType" = 'SPD'
        THEN date_trunc('day', "endDate") + INTERVAL '1 day'
        ELSE "endDate" END > sqlc.arg(check_start)::timestamptz;

-- name: CreateApprovalLog :one
INSERT INTO approval_logs ("bookingId", "approverId", action, note)
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: GetApprovalLogs :many
SELECT al.*, u.name AS approver_name
FROM approval_logs al
JOIN users u ON u.id = al."approverId"
WHERE al."bookingId" = $1
ORDER BY al."createdAt" ASC;

-- name: CreateDriverRating :one
INSERT INTO driver_ratings ("bookingId", "driverId", "ratedById", rating, review)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: GetDriverRatingByBooking :one
SELECT * FROM driver_ratings WHERE "bookingId" = $1 LIMIT 1;

-- name: GetDriverRatings :many
SELECT dr.*, u.name AS rated_by_name
FROM driver_ratings dr
JOIN users u ON u.id = dr."ratedById"
WHERE dr."driverId" = $1
ORDER BY dr."createdAt" DESC;

-- name: UpdateBookingResource :one
UPDATE bookings SET "resourceId" = $2, "updatedAt" = NOW() WHERE id = $1 RETURNING *;

-- name: MarkOverdueBookings :many
-- Booking ONGOING tapi lewat endDate → OVERDUE (sudah mulai tapi belum selesai tepat waktu)
UPDATE bookings SET status = 'OVERDUE', "updatedAt" = NOW()
WHERE status = 'ONGOING' AND "endDate" < NOW()
RETURNING *;

-- name: MarkExpiredBookings :many
-- Booking APPROVED tapi tidak pernah dimulai (tidak jadi ONGOING) selama masa bookingnya → EXPIRED
UPDATE bookings SET status = 'EXPIRED', "updatedAt" = NOW()
WHERE status = 'APPROVED' AND "endDate" < NOW()
RETURNING *;

-- name: MarkIgnoredBookings :many
-- Booking PENDING tapi admin tidak merespons sampai masa bookingnya habis → IGNORED
UPDATE bookings SET status = 'IGNORED', "updatedAt" = NOW()
WHERE status = 'PENDING' AND "endDate" < NOW()
RETURNING *;

-- name: UpdateMergedBooking :one
UPDATE bookings
SET "startDate" = $2, "endDate" = $3, "purpose" = $4,
    "assignedDriverId" = $5, "assignedVehicleId" = $6,
    "resourceId" = $7, "updatedAt" = NOW()
WHERE id = $1 RETURNING *;

-- name: GetOverlappingPassengerCount :one
SELECT COALESCE(SUM("passengerCount"), 0)::int FROM bookings
WHERE "assignedVehicleId" = $1
  AND status IN ('APPROVED', 'ONGOING')
  AND "startDate" < sqlc.arg(check_end)
  AND "endDate" > sqlc.arg(check_start)
  AND id != sqlc.arg(exclude_id);
