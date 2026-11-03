package repository

import (
	"context"
	"database/sql"
	"time"
)

// Uang parkir per booking — docs/RANCANGAN_PARKIR_PENUMPANG.md §2.
// Catatan parkir = trip_expenses category 'PARKING'. Booking yang digabung
// (merge) dihitung sebagai satu perjalanan atas booking utamanya.

func (q *Queries) GetBookingParkingAllowance(ctx context.Context, bookingID int32) (sql.NullFloat64, error) {
	var v sql.NullFloat64
	err := q.db.QueryRowContext(ctx, `SELECT "parkingAllowance"::float8 FROM bookings WHERE id = $1`, bookingID).Scan(&v)
	return v, err
}

func (q *Queries) SetBookingParkingAllowance(ctx context.Context, bookingID int32, amount sql.NullFloat64) error {
	_, err := q.db.ExecContext(ctx, `UPDATE bookings SET "parkingAllowance" = $2 WHERE id = $1`, bookingID, amount)
	return err
}

type ParkingSettlement struct {
	ID              int32          `json:"id"`
	BookingID       int32          `json:"bookingId"`
	DriverID        int32          `json:"driverId"`
	DriverName      string         `json:"driverName"`
	Direction       string         `json:"direction"`
	Amount          float64        `json:"amount"`
	Allowance       float64        `json:"allowance"`
	Used            float64        `json:"used"`
	ReimbursementID sql.NullInt32  `json:"reimbursementId"`
	SettledAt       time.Time      `json:"settledAt"`
	Note            sql.NullString `json:"note"`
	CreatedByName   string         `json:"createdByName"`
	BookingPurpose  string         `json:"bookingPurpose"`
	BookingStart    time.Time      `json:"bookingStart"`
}

const parkingSettlementSelect = `
	SELECT ps.id, ps."bookingId", ps."driverId", du.name, ps.direction, ps.amount::float8,
	       ps.allowance::float8, ps.used::float8, ps."reimbursementId", ps."settledAt", ps.note,
	       cu.name, b.purpose, b."startDate"
	FROM parking_settlements ps
	JOIN bookings b ON b.id = ps."bookingId"
	JOIN drivers d ON d.id = ps."driverId"
	JOIN users du ON du.id = d."userId"
	JOIN users cu ON cu.id = ps."createdById"`

func scanParkingSettlements(rows *sql.Rows) ([]ParkingSettlement, error) {
	defer rows.Close()
	var out []ParkingSettlement
	for rows.Next() {
		var p ParkingSettlement
		if err := rows.Scan(&p.ID, &p.BookingID, &p.DriverID, &p.DriverName, &p.Direction, &p.Amount,
			&p.Allowance, &p.Used, &p.ReimbursementID, &p.SettledAt, &p.Note,
			&p.CreatedByName, &p.BookingPurpose, &p.BookingStart); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetParkingSettlement: penyelesaian parkir booking ini (sql.ErrNoRows = belum).
func (q *Queries) GetParkingSettlement(ctx context.Context, bookingID int32) (ParkingSettlement, error) {
	rows, err := q.db.QueryContext(ctx, parkingSettlementSelect+` WHERE ps."bookingId" = $1`, bookingID)
	if err != nil {
		return ParkingSettlement{}, err
	}
	list, err := scanParkingSettlements(rows)
	if err != nil {
		return ParkingSettlement{}, err
	}
	if len(list) == 0 {
		return ParkingSettlement{}, sql.ErrNoRows
	}
	return list[0], nil
}

func (q *Queries) ListParkingSettlements(ctx context.Context, limit int32) ([]ParkingSettlement, error) {
	rows, err := q.db.QueryContext(ctx, parkingSettlementSelect+`
		ORDER BY ps."settledAt" DESC, ps.id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	return scanParkingSettlements(rows)
}

// PendingParking: perjalanan yang selesai dengan selisih parkir belum diselesaikan.
type PendingParking struct {
	BookingID    int32     `json:"bookingId"`
	DriverID     int32     `json:"driverId"`
	DriverName   string    `json:"driverName"`
	Allowance    float64   `json:"allowance"`
	Used         float64   `json:"used"`
	Count        int64     `json:"count"`
	Purpose      string    `json:"purpose"`
	StartDate    time.Time `json:"startDate"`
	ResourceName string    `json:"resourceName"`
	PlateNumber  string    `json:"plateNumber"`
	Status       string    `json:"status"`
}

// ListPendingParking: booking utama (bukan sisi "digabung") berstatus
// RETURNED/COMPLETED, punya supir, belum diselesaikan, dan terpakai ≠ bekal.
// bookingIDs (opsional) membatasi ke booking tertentu.
func (q *Queries) ListPendingParking(ctx context.Context, bookingIDs []int32) ([]PendingParking, error) {
	filter := ""
	args := []any{}
	if len(bookingIDs) > 0 {
		filter = ` AND b.id = ANY($1::int[])`
		args = append(args, intArrayLiteral(bookingIDs))
	}
	rows, err := q.db.QueryContext(ctx, `
		WITH grp AS (
		    SELECT id AS lead, id AS member FROM bookings
		    UNION
		    SELECT "primaryBookingId", "mergedBookingId" FROM booking_merges
		), used AS (
		    SELECT g.lead, COALESCE(SUM(te.amount), 0)::float8 AS used, COUNT(te.id) AS cnt
		    FROM grp g
		    LEFT JOIN trip_expenses te ON te."bookingId" = g.member
		         AND te.category = 'PARKING' AND te."voidedAt" IS NULL
		    GROUP BY g.lead
		)
		SELECT b.id, b."assignedDriverId", du.name, COALESCE(b."parkingAllowance", 0)::float8,
		       u.used, u.cnt, b.purpose, b."startDate", r.name, COALESCE(v."plateNumber", ''), b.status::text
		FROM bookings b
		JOIN used u ON u.lead = b.id
		JOIN drivers d ON d.id = b."assignedDriverId"
		JOIN users du ON du.id = d."userId"
		JOIN resources r ON r.id = b."resourceId"
		LEFT JOIN vehicles v ON v.id = b."assignedVehicleId"
		WHERE b.status IN ('RETURNED', 'COMPLETED')
		  AND NOT EXISTS (SELECT 1 FROM booking_merges x WHERE x."mergedBookingId" = b.id)
		  AND NOT EXISTS (SELECT 1 FROM parking_settlements ps WHERE ps."bookingId" = b.id)
		  AND round(u.used::numeric, 2) <> round(COALESCE(b."parkingAllowance", 0), 2)`+filter+`
		ORDER BY du.name, b."startDate", b.id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PendingParking
	for rows.Next() {
		var p PendingParking
		if err := rows.Scan(&p.BookingID, &p.DriverID, &p.DriverName, &p.Allowance, &p.Used, &p.Count,
			&p.Purpose, &p.StartDate, &p.ResourceName, &p.PlateNumber, &p.Status); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

type CreateParkingSettlementParams struct {
	BookingID       int32
	DriverID        int32
	Direction       string
	Amount          float64
	Allowance       float64
	Used            float64
	ReimbursementID sql.NullInt32
	SettledAt       time.Time
	Note            sql.NullString
	CreatedByID     int32
}

// CreateParkingSettlement: false bila booking itu sudah diselesaikan (UNIQUE).
func (q *Queries) CreateParkingSettlement(ctx context.Context, p CreateParkingSettlementParams) (bool, error) {
	res, err := q.db.ExecContext(ctx, `
		INSERT INTO parking_settlements ("bookingId", "driverId", direction, amount, allowance, used,
		                                 "reimbursementId", "settledAt", note, "createdById")
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT ("bookingId") DO NOTHING`,
		p.BookingID, p.DriverID, p.Direction, p.Amount, p.Allowance, p.Used,
		p.ReimbursementID, p.SettledAt, p.Note, p.CreatedByID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

// CreateTripReimbursement: satu pembayaran ke supir (tanpa menandai transaksi).
func (q *Queries) CreateTripReimbursement(ctx context.Context, driverID int32, amount float64, paidAt time.Time, note sql.NullString, byID int32) (int32, error) {
	var id int32
	err := q.db.QueryRowContext(ctx, `
		INSERT INTO trip_reimbursements ("driverId", amount, "paidAt", note, "createdById")
		VALUES ($1, $2, $3, $4, $5) RETURNING id`, driverID, amount, paidAt, note, byID).Scan(&id)
	return id, err
}
