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

type BookingMerge struct {
	ID               int32          `json:"id"`
	PrimaryBookingID int32          `json:"primaryBookingId"`
	MergedBookingID  int32          `json:"mergedBookingId"`
	MergedByID       int32          `json:"mergedById"`
	Reason           sql.NullString `json:"reason"`
	CreatedAt        time.Time      `json:"createdAt"`
}

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
	).Scan(&m.ID, &m.PrimaryBookingID, &m.MergedBookingID, &m.MergedByID, &m.Reason, &m.CreatedAt)
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

// UpdateBookingDates updates the startDate and endDate of a booking (used when merging).
func (q *Queries) UpdateBookingDates(ctx context.Context, bookingID int32, startDate, endDate time.Time) error {
	query := `UPDATE bookings SET "startDate" = $2, "endDate" = $3, "updatedAt" = NOW() WHERE id = $1`
	_, err := q.db.ExecContext(ctx, query, bookingID, startDate, endDate)
	return err
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

// ─── Room Keepers ────────────────────────────────────────────────────────────

type RoomKeeper struct {
	ID          int32     `json:"id"`
	UserID      int32     `json:"userId"`
	PhoneNumber string    `json:"phoneNumber"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (q *Queries) GetRoomKeeperByUserID(ctx context.Context, userID int32) (RoomKeeper, error) {
	query := `SELECT id, "userId", "phoneNumber", "isActive", "createdAt" FROM room_keepers WHERE "userId" = $1 LIMIT 1`
	var rk RoomKeeper
	err := q.db.QueryRowContext(ctx, query, userID).Scan(
		&rk.ID, &rk.UserID, &rk.PhoneNumber, &rk.IsActive, &rk.CreatedAt,
	)
	return rk, err
}
