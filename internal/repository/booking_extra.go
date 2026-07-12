package repository

import (
	"context"
	"database/sql"
	"time"
)

// AssignVehicleAndUpdateResource assigns driver+vehicle to a booking and, if the vehicle
// belongs to a different resource than the one originally booked, updates resourceId and
// saves the original in originalResourceId. This fixes calendar/availability tracking.
func (q *Queries) AssignVehicleAndUpdateResource(
	ctx context.Context,
	bookingID, driverID, vehicleID, vehicleResourceID int32,
) error {
	query := `
		UPDATE bookings
		SET "assignedDriverId"   = $2,
		    "assignedVehicleId"  = $3,
		    "originalResourceId" = CASE
		        WHEN "resourceId" != $4 AND "originalResourceId" IS NULL
		        THEN "resourceId"
		        ELSE "originalResourceId"
		    END,
		    "resourceId" = $4,
		    "assignedAt" = NOW(),
		    "updatedAt"  = NOW()
		WHERE id = $1`
	_, err := q.db.ExecContext(ctx, query, bookingID, driverID, vehicleID, vehicleResourceID)
	return err
}

// ─── Booking Merges ──────────────────────────────────────────────────────────

func (q *Queries) CreateBookingMerge(
	ctx context.Context,
	primaryBookingID, mergedBookingID, mergedByID int32,
	reason string,
) (BookingMerge, error) {
	query := `
		INSERT INTO booking_merges ("primaryBookingId", "mergedBookingId", "mergedById", reason)
		VALUES ($1, $2, $3, $4)
		RETURNING id, "primaryBookingId", "mergedBookingId", "mergedById", reason, "createdAt"`
	var m BookingMerge
	err := q.db.QueryRowContext(ctx, query,
		primaryBookingID, mergedBookingID, mergedByID,
		sql.NullString{String: reason, Valid: reason != ""},
	).Scan(&m.ID, &m.PrimaryBookingId, &m.MergedBookingId, &m.MergedById, &m.Reason, &m.CreatedAt)
	return m, err
}

type BookingMergeInfoRow struct {
	ID               int32          `json:"id"`
	PrimaryBookingID int32          `json:"primaryBookingId"`
	MergedBookingID  int32          `json:"mergedBookingId"`
	MergedByName     string         `json:"mergedByName"`
	Reason           sql.NullString `json:"reason"`
	CreatedAt        time.Time      `json:"createdAt"`
	OtherBookingID   int32          `json:"otherBookingId"`
	OtherUserID      int32          `json:"otherUserId"`
	OtherUserName    string         `json:"otherUserName"`
	OtherEmployeeID  string         `json:"otherEmployeeId"`
	OtherDepartment  string         `json:"otherDepartment"`
	OtherPurpose     string         `json:"otherPurpose"`
	IsPrimary        bool           `json:"isPrimary"`
}

// GetBookingMerges returns merge relationships for a booking (as primary or secondary).
func (q *Queries) GetBookingMerges(ctx context.Context, bookingID int32) ([]BookingMergeInfoRow, error) {
	query := `
		SELECT
		    bm.id,
		    bm."primaryBookingId",
		    bm."mergedBookingId",
		    mb_user.name  AS merged_by_name,
		    bm.reason,
		    bm."createdAt",
		    b_other.id   AS other_booking_id,
		    u_other.id   AS other_user_id,
		    u_other.name AS other_user_name,
		    u_other."employeeId" AS other_employee_id,
		    dept_other.name AS other_department,
		    b_other.purpose   AS other_purpose,
		    (bm."primaryBookingId" = $1) AS is_primary
		FROM booking_merges bm
		JOIN users mb_user ON mb_user.id = bm."mergedById"
		JOIN bookings b_other ON b_other.id = CASE
		    WHEN bm."primaryBookingId" = $1 THEN bm."mergedBookingId"
		    ELSE bm."primaryBookingId"
		END
		JOIN users u_other ON u_other.id = b_other."userId"
		JOIN departments dept_other ON dept_other.id = u_other."departmentId"
		WHERE bm."primaryBookingId" = $1 OR bm."mergedBookingId" = $1
		ORDER BY bm."createdAt" ASC`

	rows, err := q.db.QueryContext(ctx, query, bookingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []BookingMergeInfoRow
	for rows.Next() {
		var r BookingMergeInfoRow
		if err := rows.Scan(
			&r.ID, &r.PrimaryBookingID, &r.MergedBookingID,
			&r.MergedByName, &r.Reason, &r.CreatedAt,
			&r.OtherBookingID, &r.OtherUserID, &r.OtherUserName,
			&r.OtherEmployeeID, &r.OtherDepartment, &r.OtherPurpose,
			&r.IsPrimary,
		); err != nil {
			return nil, err
		}
		items = append(items, r)
	}
	return items, rows.Err()
}

// InheritMergeDriverVehicle copies the driver and vehicle assignment from the primary booking
// to the merged booking so both bookings share the same trip logistics.
func (q *Queries) InheritMergeDriverVehicle(
	ctx context.Context,
	mergedBookingID, driverID, vehicleID int32,
	driverValid, vehicleValid bool,
) error {
	query := `
		UPDATE bookings
		SET "assignedDriverId"  = CASE WHEN $2 THEN $3::int ELSE "assignedDriverId" END,
		    "assignedVehicleId" = CASE WHEN $4 THEN $5::int ELSE "assignedVehicleId" END,
		    "updatedAt" = NOW()
		WHERE id = $1`
	_, err := q.db.ExecContext(ctx, query, mergedBookingID, driverValid, driverID, vehicleValid, vehicleID)
	return err
}

// InheritMergeResourceDriverVehicle moves the merged (target) booking ONTO the primary
// booking's resource (vehicle slot) and copies its driver/vehicle assignment, so both
// bookings share ONE vehicle (hemat kendaraan). The target's original resource is saved
// in originalResourceId — freeing that vehicle on the calendar and marking the booking as
// reassigned. Driver/vehicle are only overwritten when the primary actually has them.
func (q *Queries) InheritMergeResourceDriverVehicle(
	ctx context.Context,
	mergedBookingID, primaryResourceID, driverID, vehicleID int32,
	driverValid, vehicleValid bool,
) error {
	query := `
		UPDATE bookings
		SET "originalResourceId" = CASE
		        WHEN "resourceId" != $2 AND "originalResourceId" IS NULL
		        THEN "resourceId"
		        ELSE "originalResourceId"
		    END,
		    "resourceId"        = $2,
		    "assignedDriverId"  = CASE WHEN $3 THEN $4::int ELSE "assignedDriverId" END,
		    "assignedVehicleId" = CASE WHEN $5 THEN $6::int ELSE "assignedVehicleId" END,
		    "assignedAt"        = NOW(),
		    "updatedAt"         = NOW()
		WHERE id = $1`
	_, err := q.db.ExecContext(ctx, query,
		mergedBookingID, primaryResourceID, driverValid, driverID, vehicleValid, vehicleID)
	return err
}

// UpdateBookingDates updates the startDate and endDate of a booking (used when merging).
func (q *Queries) UpdateBookingDates(ctx context.Context, bookingID int32, startDate, endDate time.Time) error {
	query := `UPDATE bookings SET "startDate" = $2, "endDate" = $3, "updatedAt" = NOW() WHERE id = $1`
	_, err := q.db.ExecContext(ctx, query, bookingID, startDate, endDate)
	return err
}

// GetVehicleIDByResourceID returns the vehicle id backing a resource (error if the
// resource is not a vehicle, e.g. a room).
func (q *Queries) GetVehicleIDByResourceID(ctx context.Context, resourceID int32) (int32, error) {
	var id int32
	err := q.db.QueryRowContext(ctx,
		`SELECT id FROM vehicles WHERE "resourceId" = $1 LIMIT 1`, resourceID).Scan(&id)
	return id, err
}

// GetFreeDriver returns one active driver with no current vehicle assignment (i.e. not
// tied to an active booking = "kosong/senggang"). Error if none available.
func (q *Queries) GetFreeDriver(ctx context.Context) (int32, error) {
	var id int32
	err := q.db.QueryRowContext(ctx, `
		SELECT d.id FROM drivers d
		LEFT JOIN driver_assignments da ON da."driverId" = d.id AND da."releasedAt" IS NULL
		WHERE d."isActive" = TRUE AND da.id IS NULL
		ORDER BY d.id ASC LIMIT 1`).Scan(&id)
	return id, err
}

// CountActiveBookingsByDriver counts APPROVED/ONGOING bookings assigned to a driver,
// excluding one booking id. Used to decide whether to release the driver's vehicle
// ownership when a booking completes (release only when no other active booking).
func (q *Queries) CountActiveBookingsByDriver(ctx context.Context, driverID, excludeBookingID int32) (int64, error) {
	query := `SELECT COUNT(*) FROM bookings
		WHERE "assignedDriverId" = $1 AND status IN ('APPROVED','ONGOING') AND id != $2`
	var n int64
	err := q.db.QueryRowContext(ctx, query, driverID, excludeBookingID).Scan(&n)
	return n, err
}

// CheckBookingAlreadyMerged returns true if both booking IDs are already in the same merge group.
func (q *Queries) CheckBookingAlreadyMerged(ctx context.Context, bookingA, bookingB int32) (bool, error) {
	query := `
		SELECT COUNT(*) FROM booking_merges
		WHERE ("primaryBookingId" = $1 AND "mergedBookingId" = $2)
		   OR ("primaryBookingId" = $2 AND "mergedBookingId" = $1)`
	var count int64
	err := q.db.QueryRowContext(ctx, query, bookingA, bookingB).Scan(&count)
	return count > 0, err
}

// ─── Booking Return Reports ──────────────────────────────────────────────────

type BookingReturnReportRow struct {
	BookingReturnReport
	SubmitterName string `json:"submitterName"`
}

// EnsureReturnReportTable creates the booking_return_reports table if it doesn't exist.
func EnsureReturnReportTable(ctx context.Context, db DBTX) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS booking_return_reports (
			id               SERIAL       PRIMARY KEY,
			"bookingId"      INTEGER      NOT NULL UNIQUE REFERENCES bookings(id) ON DELETE CASCADE,
			"submittedById"  INTEGER      NOT NULL REFERENCES users(id),
			note             TEXT         NOT NULL,
			location         VARCHAR(500) NOT NULL,
			"submittedAt"    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
		)`)
	return err
}

func (q *Queries) CreateReturnReport(ctx context.Context, bookingID, submittedByID int32, note, location string) (BookingReturnReport, error) {
	query := `
		INSERT INTO booking_return_reports ("bookingId", "submittedById", note, location)
		VALUES ($1, $2, $3, $4)
		RETURNING id, "bookingId", "submittedById", note, location, "submittedAt"`
	var r BookingReturnReport
	err := q.db.QueryRowContext(ctx, query, bookingID, submittedByID, note, location).
		Scan(&r.ID, &r.BookingId, &r.SubmittedById, &r.Note, &r.Location, &r.SubmittedAt)
	return r, err
}

func (q *Queries) GetReturnReport(ctx context.Context, bookingID int32) (BookingReturnReportRow, error) {
	query := `
		SELECT r.id, r."bookingId", r."submittedById", r.note, r.location, r."submittedAt",
		       u.name AS submitter_name
		FROM booking_return_reports r
		JOIN users u ON u.id = r."submittedById"
		WHERE r."bookingId" = $1`
	var r BookingReturnReportRow
	err := q.db.QueryRowContext(ctx, query, bookingID).
		Scan(&r.ID, &r.BookingId, &r.SubmittedById, &r.Note, &r.Location, &r.SubmittedAt, &r.SubmitterName)
	return r, err
}

// ─── Room Keepers ────────────────────────────────────────────────────────────

func (q *Queries) GetRoomKeeperByUserID(ctx context.Context, userID int32) (RoomKeeper, error) {
	query := `SELECT id, "userId", "phoneNumber", "isActive", "createdAt" FROM room_keepers WHERE "userId" = $1 LIMIT 1`
	var rk RoomKeeper
	err := q.db.QueryRowContext(ctx, query, userID).Scan(
		&rk.ID, &rk.UserId, &rk.PhoneNumber, &rk.IsActive, &rk.CreatedAt,
	)
	return rk, err
}
