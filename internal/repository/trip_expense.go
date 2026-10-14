package repository

import (
	"context"
	"database/sql"
	"time"
)

// Biaya perjalanan (tol, parkir, lainnya) — docs/RANCANGAN_BIAYA_PERJALANAN.md.

type TripExpense struct {
	ID              int32          `json:"id"`
	BookingID       int32          `json:"bookingId"`
	VehicleID       sql.NullInt32  `json:"vehicleId"`
	PlateNumber     sql.NullString `json:"plateNumber"`
	DriverID        sql.NullInt32  `json:"driverId"`
	DriverName      sql.NullString `json:"driverName"`
	Category        string         `json:"category"`
	TollPoint       sql.NullString `json:"tollPoint"`
	EntryGate       sql.NullString `json:"entryGate"`
	ExitGate        sql.NullString `json:"exitGate"`
	Amount          float64        `json:"amount"`
	PaymentMethod   sql.NullString `json:"paymentMethod"`
	ProofPhotoURL   string         `json:"proofPhotoUrl"`
	Note            sql.NullString `json:"note"`
	OccurredAt      time.Time      `json:"occurredAt"`
	RecordedByID    int32          `json:"recordedById"`
	RecordedByName  string         `json:"recordedByName"`
	ReimbursementID sql.NullInt32  `json:"reimbursementId"`
	ReimbursedAt    sql.NullTime   `json:"reimbursedAt"`
	VoidedAt        sql.NullTime   `json:"voidedAt"`
	VoidedByName    sql.NullString `json:"voidedByName"`
	VoidReason      sql.NullString `json:"voidReason"`
	CreatedAt       time.Time      `json:"createdAt"`
}

const tripExpenseSelect = `
	SELECT te.id, te."bookingId", te."vehicleId", v."plateNumber", te."driverId", du.name,
	       te.category, te."tollPoint", te."entryGate", te."exitGate", te.amount::float8,
	       te."paymentMethod", te."proofPhotoUrl", te.note, te."occurredAt",
	       te."recordedById", ru.name, te."reimbursementId", tr."paidAt",
	       te."voidedAt", vu.name, te."voidReason", te."createdAt"
	FROM trip_expenses te
	JOIN users ru ON ru.id = te."recordedById"
	LEFT JOIN vehicles v ON v.id = te."vehicleId"
	LEFT JOIN drivers d ON d.id = te."driverId"
	LEFT JOIN users du ON du.id = d."userId"
	LEFT JOIN users vu ON vu.id = te."voidedById"
	LEFT JOIN trip_reimbursements tr ON tr.id = te."reimbursementId"`

func scanTripExpenses(rows *sql.Rows) ([]TripExpense, error) {
	defer rows.Close()
	var out []TripExpense
	for rows.Next() {
		var e TripExpense
		if err := rows.Scan(&e.ID, &e.BookingID, &e.VehicleID, &e.PlateNumber, &e.DriverID, &e.DriverName,
			&e.Category, &e.TollPoint, &e.EntryGate, &e.ExitGate, &e.Amount,
			&e.PaymentMethod, &e.ProofPhotoURL, &e.Note, &e.OccurredAt,
			&e.RecordedByID, &e.RecordedByName, &e.ReimbursementID, &e.ReimbursedAt,
			&e.VoidedAt, &e.VoidedByName, &e.VoidReason, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

type CreateTripExpenseParams struct {
	BookingID     int32
	VehicleID     sql.NullInt32
	DriverID      sql.NullInt32
	Category      string
	TollPoint     sql.NullString
	EntryGate     sql.NullString
	ExitGate      sql.NullString
	Amount        float64
	PaymentMethod sql.NullString
	ProofPhotoURL string
	Note          sql.NullString
	OccurredAt    time.Time
	RecordedByID  int32
}

func (q *Queries) CreateTripExpense(ctx context.Context, p CreateTripExpenseParams) (int32, error) {
	var id int32
	err := q.db.QueryRowContext(ctx, `
		INSERT INTO trip_expenses ("bookingId", "vehicleId", "driverId", category, "tollPoint",
		    "entryGate", "exitGate", amount, "paymentMethod", "proofPhotoUrl", note, "occurredAt", "recordedById")
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id`,
		p.BookingID, p.VehicleID, p.DriverID, p.Category, p.TollPoint,
		p.EntryGate, p.ExitGate, p.Amount, p.PaymentMethod, p.ProofPhotoURL, p.Note, p.OccurredAt, p.RecordedByID,
	).Scan(&id)
	return id, err
}

func (q *Queries) GetTripExpense(ctx context.Context, id int32) (TripExpense, error) {
	rows, err := q.db.QueryContext(ctx, tripExpenseSelect+` WHERE te.id = $1`, id)
	if err != nil {
		return TripExpense{}, err
	}
	list, err := scanTripExpenses(rows)
	if err != nil {
		return TripExpense{}, err
	}
	if len(list) == 0 {
		return TripExpense{}, sql.ErrNoRows
	}
	return list[0], nil
}

// ListTripExpensesByBooking: semua catatan booking (termasuk yang dibatalkan),
// urut waktu kejadian.
func (q *Queries) ListTripExpensesByBooking(ctx context.Context, bookingID int32) ([]TripExpense, error) {
	rows, err := q.db.QueryContext(ctx, tripExpenseSelect+`
		WHERE te."bookingId" = $1 ORDER BY te."occurredAt", te.id`, bookingID)
	if err != nil {
		return nil, err
	}
	return scanTripExpenses(rows)
}

// VoidTripExpense membatalkan catatan yang belum dibatalkan & belum
// direimburse. false = tidak ada baris yang memenuhi syarat.
func (q *Queries) VoidTripExpense(ctx context.Context, id, byID int32, reason string) (bool, error) {
	res, err := q.db.ExecContext(ctx, `
		UPDATE trip_expenses SET "voidedAt" = now(), "voidedById" = $2, "voidReason" = $3
		WHERE id = $1 AND "voidedAt" IS NULL AND "reimbursementId" IS NULL`, id, byID, reason)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// ListTripGateSuggestions: nama gerbang tol yang pernah dipakai (masuk/keluar),
// paling sering dulu — saran saat mengetik.
func (q *Queries) ListTripGateSuggestions(ctx context.Context, search string, limit int32) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT gate FROM (
		    SELECT "entryGate" AS gate FROM trip_expenses WHERE "entryGate" IS NOT NULL AND "voidedAt" IS NULL
		    UNION ALL
		    SELECT "exitGate" FROM trip_expenses WHERE "exitGate" IS NOT NULL AND "voidedAt" IS NULL
		) g
		WHERE $1 = '' OR gate ILIKE '%' || $1 || '%'
		GROUP BY gate
		ORDER BY COUNT(*) DESC, gate
		LIMIT $2`, search, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var g string
		if err := rows.Scan(&g); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (q *Queries) GetBookingTollBudget(ctx context.Context, bookingID int32) (sql.NullFloat64, error) {
	var v sql.NullFloat64
	err := q.db.QueryRowContext(ctx, `SELECT "tollBudget"::float8 FROM bookings WHERE id = $1`, bookingID).Scan(&v)
	return v, err
}

func (q *Queries) SetBookingTollBudget(ctx context.Context, bookingID int32, amount sql.NullFloat64) error {
	_, err := q.db.ExecContext(ctx, `UPDATE bookings SET "tollBudget" = $2 WHERE id = $1`, bookingID, amount)
	return err
}

// ListPendingTripReimbursements: transaksi uang pribadi supir yang belum
// diganti (tidak dibatalkan), urut supir lalu waktu.
func (q *Queries) ListPendingTripReimbursements(ctx context.Context) ([]TripExpense, error) {
	rows, err := q.db.QueryContext(ctx, tripExpenseSelect+`
		WHERE te."paymentMethod" = 'PERSONAL' AND te."reimbursementId" IS NULL AND te."voidedAt" IS NULL
		  AND te."driverId" IS NOT NULL
		ORDER BY du.name, te."occurredAt", te.id`)
	if err != nil {
		return nil, err
	}
	return scanTripExpenses(rows)
}

// MarkTripReimbursed mencatat satu pembayaran reimburse untuk transaksi
// [ids] milik supir [driverID] yang masih menunggu. Mengembalikan id
// pembayaran, jumlah transaksi, dan total; 0 transaksi = tidak ada yang cocok.
func (q *Queries) MarkTripReimbursed(ctx context.Context, driverID int32, ids []int32, paidAt time.Time, note sql.NullString, byID int32) (int32, int64, float64, error) {
	var count int64
	var total float64
	if err := q.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(amount), 0)::float8 FROM trip_expenses
		WHERE id = ANY($1::int[]) AND "driverId" = $2 AND "paymentMethod" = 'PERSONAL'
		  AND "reimbursementId" IS NULL AND "voidedAt" IS NULL`,
		intArrayLiteral(ids), driverID).Scan(&count, &total); err != nil {
		return 0, 0, 0, err
	}
	if count == 0 {
		return 0, 0, 0, nil
	}
	var rid int32
	if err := q.db.QueryRowContext(ctx, `
		INSERT INTO trip_reimbursements ("driverId", amount, "paidAt", note, "createdById")
		VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		driverID, total, paidAt, note, byID).Scan(&rid); err != nil {
		return 0, 0, 0, err
	}
	if _, err := q.db.ExecContext(ctx, `
		UPDATE trip_expenses SET "reimbursementId" = $3
		WHERE id = ANY($1::int[]) AND "driverId" = $2 AND "paymentMethod" = 'PERSONAL'
		  AND "reimbursementId" IS NULL AND "voidedAt" IS NULL`,
		intArrayLiteral(ids), driverID, rid); err != nil {
		return 0, 0, 0, err
	}
	return rid, count, total, nil
}

type TripReimbursement struct {
	ID            int32          `json:"id"`
	DriverID      int32          `json:"driverId"`
	DriverName    string         `json:"driverName"`
	Amount        float64        `json:"amount"`
	PaidAt        time.Time      `json:"paidAt"`
	Note          sql.NullString `json:"note"`
	CreatedByName string         `json:"createdByName"`
	ExpenseCount  int64          `json:"expenseCount"`
	CreatedAt     time.Time      `json:"createdAt"`
}

func (q *Queries) ListTripReimbursements(ctx context.Context, limit int32) ([]TripReimbursement, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT tr.id, tr."driverId", du.name, tr.amount::float8, tr."paidAt", tr.note, cu.name,
		       (SELECT COUNT(*) FROM trip_expenses te WHERE te."reimbursementId" = tr.id), tr."createdAt"
		FROM trip_reimbursements tr
		JOIN drivers d ON d.id = tr."driverId"
		JOIN users du ON du.id = d."userId"
		JOIN users cu ON cu.id = tr."createdById"
		ORDER BY tr."paidAt" DESC, tr.id DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TripReimbursement
	for rows.Next() {
		var r TripReimbursement
		if err := rows.Scan(&r.ID, &r.DriverID, &r.DriverName, &r.Amount, &r.PaidAt, &r.Note,
			&r.CreatedByName, &r.ExpenseCount, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
