package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Hand-written (sqlc CLI unavailable) - voucher BBM untuk SPBU mitra.
// Lihat docs/RANCANGAN_VOUCHER_BBM.md §3.

type FuelVoucher struct {
	ID                int32           `json:"id"`
	Code              string          `json:"code"`
	VehicleID         int32           `json:"vehicleId"`
	VehicleName       string          `json:"vehicleName"`
	PlateNumber       string          `json:"plateNumber"`
	FuelTypeID        int32           `json:"fuelTypeId"`
	FuelTypeName      string          `json:"fuelTypeName"`
	StationID         int32           `json:"stationId"`
	StationName       string          `json:"stationName"`
	StationAddress    sql.NullString  `json:"stationAddress"`
	DriverID          sql.NullInt32   `json:"driverId"`
	DriverName        sql.NullString  `json:"driverName"`
	DriverUserID      sql.NullInt32   `json:"driverUserId"`
	BookingID         sql.NullInt32   `json:"bookingId"`
	IssuedByID        int32           `json:"issuedById"`
	IssuedByName      string          `json:"issuedByName"`
	Odometer          int32           `json:"odometer"`
	DistanceKm        int32           `json:"distanceKm"`
	KmPerLiter        float64         `json:"kmPerLiter"`
	AccruedLiter      float64         `json:"accruedLiter"`
	CarriedLiter      float64         `json:"carriedLiter"`
	TankCapacityLiter sql.NullFloat64 `json:"tankCapacityLiter"`
	Liter             float64         `json:"liter"`
	PricePerLiter     float64         `json:"pricePerLiter"`
	Amount            float64         `json:"amount"`
	ValidUntil        time.Time       `json:"validUntil"`
	Status            string          `json:"status"`
	UsedAt            sql.NullTime    `json:"usedAt"`
	UsedByName        sql.NullString  `json:"usedByName"`
	UsedOdometer      sql.NullInt32   `json:"usedOdometer"`
	ReceiptPhotoUrl   sql.NullString  `json:"receiptPhotoUrl"`
	FuelExpenseID     sql.NullInt32   `json:"fuelExpenseId"`
	CancelledAt       sql.NullTime    `json:"cancelledAt"`
	CancelledByName   sql.NullString  `json:"cancelledByName"`
	CancelReason      sql.NullString  `json:"cancelReason"`
	ReconciledAt      sql.NullTime    `json:"reconciledAt"`
	ReconciledByName  sql.NullString  `json:"reconciledByName"`
	InvoiceNumber     sql.NullString  `json:"invoiceNumber"`
	Note              sql.NullString  `json:"note"`
	CreatedAt         time.Time       `json:"createdAt"`
}

const fuelVoucherSelect = `
	SELECT fv.id, fv.code, fv."vehicleId", r.name, v."plateNumber", fv."fuelTypeId", ft.name,
	       fv."stationId", fs.name, fs.address, fv."driverId", du.name, d."userId", fv."bookingId",
	       fv."issuedById", iu.name, fv.odometer, fv."distanceKm", fv."kmPerLiter"::float8,
	       fv."accruedLiter"::float8, fv."carriedLiter"::float8, fv."tankCapacityLiter"::float8,
	       fv.liter::float8, fv."pricePerLiter"::float8, fv.amount::float8, fv."validUntil",
	       fv.status::text, fv."usedAt", uu.name, fv."usedOdometer", fv."receiptPhotoUrl",
	       fv."fuelExpenseId", fv."cancelledAt", cu.name, fv."cancelReason",
	       fv."reconciledAt", ru.name, fv."invoiceNumber", fv.note, fv."createdAt"
	FROM fuel_vouchers fv
	JOIN vehicles v ON v.id = fv."vehicleId"
	JOIN resources r ON r.id = v."resourceId"
	JOIN fuel_types ft ON ft.id = fv."fuelTypeId"
	JOIN fuel_stations fs ON fs.id = fv."stationId"
	JOIN users iu ON iu.id = fv."issuedById"
	LEFT JOIN drivers d ON d.id = fv."driverId"
	LEFT JOIN users du ON du.id = d."userId"
	LEFT JOIN users uu ON uu.id = fv."usedById"
	LEFT JOIN users cu ON cu.id = fv."cancelledById"
	LEFT JOIN users ru ON ru.id = fv."reconciledById"`

func scanFuelVoucher(s interface{ Scan(...any) error }) (FuelVoucher, error) {
	var f FuelVoucher
	err := s.Scan(&f.ID, &f.Code, &f.VehicleID, &f.VehicleName, &f.PlateNumber, &f.FuelTypeID, &f.FuelTypeName,
		&f.StationID, &f.StationName, &f.StationAddress, &f.DriverID, &f.DriverName, &f.DriverUserID, &f.BookingID,
		&f.IssuedByID, &f.IssuedByName, &f.Odometer, &f.DistanceKm, &f.KmPerLiter,
		&f.AccruedLiter, &f.CarriedLiter, &f.TankCapacityLiter,
		&f.Liter, &f.PricePerLiter, &f.Amount, &f.ValidUntil,
		&f.Status, &f.UsedAt, &f.UsedByName, &f.UsedOdometer, &f.ReceiptPhotoUrl,
		&f.FuelExpenseID, &f.CancelledAt, &f.CancelledByName, &f.CancelReason,
		&f.ReconciledAt, &f.ReconciledByName, &f.InvoiceNumber, &f.Note, &f.CreatedAt)
	return f, err
}

func (q *Queries) GetFuelVoucher(ctx context.Context, id int32) (FuelVoucher, error) {
	return scanFuelVoucher(q.db.QueryRowContext(ctx, fuelVoucherSelect+` WHERE fv.id = $1`, id))
}

// LockFuelVoucher mengunci baris voucher di dalam transaksi.
func (q *Queries) LockFuelVoucher(ctx context.Context, id int32) (string, int32, error) {
	var status string
	var vehicleID int32
	err := q.db.QueryRowContext(ctx,
		`SELECT status::text, "vehicleId" FROM fuel_vouchers WHERE id = $1 FOR UPDATE`, id).Scan(&status, &vehicleID)
	return status, vehicleID, err
}

func (q *Queries) GetActiveFuelVoucherID(ctx context.Context, vehicleID int32) (int32, error) {
	var id int32
	err := q.db.QueryRowContext(ctx,
		`SELECT id FROM fuel_vouchers WHERE "vehicleId" = $1 AND status = 'ISSUED' LIMIT 1`, vehicleID).Scan(&id)
	return id, err
}

// ActiveFuelVoucherByVehicle: id+kode voucher aktif per kendaraan (untuk daftar saldo).
func (q *Queries) ActiveFuelVoucherByVehicle(ctx context.Context) (map[int32]FuelVoucher, error) {
	rows, err := q.db.QueryContext(ctx, fuelVoucherSelect+` WHERE fv.status = 'ISSUED'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int32]FuelVoucher{}
	for rows.Next() {
		f, err := scanFuelVoucher(rows)
		if err != nil {
			return nil, err
		}
		out[f.VehicleID] = f
	}
	return out, rows.Err()
}

type ListFuelVouchersParams struct {
	Status     sql.NullString
	VehicleID  sql.NullInt32
	StationID  sql.NullInt32
	DriverID   sql.NullInt32
	From       sql.NullTime
	To         sql.NullTime
	Reconciled sql.NullBool
	Search     sql.NullString
	Limit      int32
	Offset     int32
}

const fuelVoucherFilter = `
	WHERE ($1::text IS NULL OR fv.status::text = $1)
	  AND ($2::int IS NULL OR fv."vehicleId" = $2)
	  AND ($3::int IS NULL OR fv."stationId" = $3)
	  AND ($4::int IS NULL OR fv."driverId" = $4)
	  AND ($5::timestamptz IS NULL OR fv."createdAt" >= $5)
	  AND ($6::timestamptz IS NULL OR fv."createdAt" < $6)
	  AND ($7::bool IS NULL OR (fv."reconciledAt" IS NOT NULL) = $7)
	  AND ($8::text IS NULL OR fv.code ILIKE '%' || $8 || '%' OR v."plateNumber" ILIKE '%' || $8 || '%')`

func (q *Queries) ListFuelVouchers(ctx context.Context, arg ListFuelVouchersParams) ([]FuelVoucher, int64, error) {
	args := []any{arg.Status, arg.VehicleID, arg.StationID, arg.DriverID, arg.From, arg.To, arg.Reconciled, arg.Search}
	var total int64
	if err := q.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM fuel_vouchers fv
		JOIN vehicles v ON v.id = fv."vehicleId"`+fuelVoucherFilter, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := q.db.QueryContext(ctx, fuelVoucherSelect+fuelVoucherFilter+
		fmt.Sprintf(` ORDER BY fv.id DESC LIMIT %d OFFSET %d`, arg.Limit, arg.Offset), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var items []FuelVoucher
	for rows.Next() {
		f, err := scanFuelVoucher(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, f)
	}
	return items, total, rows.Err()
}

// FuelVoucherSummary: rekap per status untuk filter yang sama (tanpa paging).
type FuelVoucherSummary struct {
	Status string  `json:"status"`
	Count  int64   `json:"count"`
	Liter  float64 `json:"liter"`
	Amount float64 `json:"amount"`
}

func (q *Queries) SummarizeFuelVouchers(ctx context.Context, arg ListFuelVouchersParams) ([]FuelVoucherSummary, error) {
	args := []any{sql.NullString{}, arg.VehicleID, arg.StationID, arg.DriverID, arg.From, arg.To, arg.Reconciled, arg.Search}
	rows, err := q.db.QueryContext(ctx, `
		SELECT fv.status::text, COUNT(*), COALESCE(SUM(fv.liter), 0)::float8, COALESCE(SUM(fv.amount), 0)::float8
		FROM fuel_vouchers fv JOIN vehicles v ON v.id = fv."vehicleId"`+fuelVoucherFilter+`
		GROUP BY fv.status`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []FuelVoucherSummary
	for rows.Next() {
		var s FuelVoucherSummary
		if err := rows.Scan(&s.Status, &s.Count, &s.Liter, &s.Amount); err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	return items, rows.Err()
}

type CreateFuelVoucherParams struct {
	Code              string
	VehicleID         int32
	FuelTypeID        int32
	StationID         int32
	DriverID          sql.NullInt32
	BookingID         sql.NullInt32
	IssuedByID        int32
	Odometer          int32
	DistanceKm        int32
	KmPerLiter        float64
	AccruedLiter      float64
	CarriedLiter      float64
	TankCapacityLiter sql.NullFloat64
	Liter             float64
	PricePerLiter     float64
	Amount            float64
	ValidUntil        time.Time
	Note              sql.NullString
}

func (q *Queries) CreateFuelVoucher(ctx context.Context, arg CreateFuelVoucherParams) (int32, error) {
	var id int32
	err := q.db.QueryRowContext(ctx, `
		INSERT INTO fuel_vouchers (code, "vehicleId", "fuelTypeId", "stationId", "driverId", "bookingId",
		    "issuedById", odometer, "distanceKm", "kmPerLiter", "accruedLiter", "carriedLiter",
		    "tankCapacityLiter", liter, "pricePerLiter", amount, "validUntil", note)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18) RETURNING id`,
		arg.Code, arg.VehicleID, arg.FuelTypeID, arg.StationID, arg.DriverID, arg.BookingID,
		arg.IssuedByID, arg.Odometer, arg.DistanceKm, numeric2(arg.KmPerLiter), numeric2(arg.AccruedLiter),
		numeric2(arg.CarriedLiter), arg.TankCapacityLiter, numeric2(arg.Liter), numeric2(arg.PricePerLiter),
		numeric2(arg.Amount), arg.ValidUntil, arg.Note,
	).Scan(&id)
	return id, err
}

func (q *Queries) MarkFuelVoucherUsed(ctx context.Context, id, usedByID, usedOdometer int32, receiptURL sql.NullString, expenseID int32) error {
	_, err := q.db.ExecContext(ctx, `
		UPDATE fuel_vouchers SET status = 'USED', "usedAt" = NOW(), "usedById" = $2, "usedOdometer" = $3,
		       "receiptPhotoUrl" = $4, "fuelExpenseId" = $5
		WHERE id = $1`, id, usedByID, usedOdometer, receiptURL, expenseID)
	return err
}

func (q *Queries) MarkFuelVoucherCancelled(ctx context.Context, id, byID int32, reason string) error {
	_, err := q.db.ExecContext(ctx, `
		UPDATE fuel_vouchers SET status = 'CANCELLED', "cancelledAt" = NOW(), "cancelledById" = $2, "cancelReason" = $3
		WHERE id = $1`, id, byID, reason)
	return err
}

func (q *Queries) MarkFuelVoucherExpired(ctx context.Context, id int32) error {
	_, err := q.db.ExecContext(ctx, `UPDATE fuel_vouchers SET status = 'EXPIRED' WHERE id = $1 AND status = 'ISSUED'`, id)
	return err
}

// ListExpiredIssuedVoucherIDs: voucher ISSUED yang sudah lewat masa berlaku.
func (q *Queries) ListExpiredIssuedVoucherIDs(ctx context.Context, now time.Time) ([]int32, error) {
	rows, err := q.db.QueryContext(ctx,
		`SELECT id FROM fuel_vouchers WHERE status = 'ISSUED' AND "validUntil" < $1 ORDER BY id`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int32
	for rows.Next() {
		var id int32
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ReconcileFuelVouchers menandai voucher USED yang belum direkonsiliasi.
func (q *Queries) ReconcileFuelVouchers(ctx context.Context, ids []int32, byID int32, invoiceNumber string) (int64, error) {
	res, err := q.db.ExecContext(ctx, `
		UPDATE fuel_vouchers SET "reconciledAt" = NOW(), "reconciledById" = $2, "invoiceNumber" = $3
		WHERE id = ANY($1::int[]) AND status = 'USED' AND "reconciledAt" IS NULL`,
		intArrayLiteral(ids), byID, invoiceNumber)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (q *Queries) UnreconcileFuelVoucher(ctx context.Context, id int32) error {
	_, err := q.db.ExecContext(ctx, `
		UPDATE fuel_vouchers SET "reconciledAt" = NULL, "reconciledById" = NULL, "invoiceNumber" = NULL WHERE id = $1`, id)
	return err
}

// MaxFuelVoucherOdometer: odometer tertinggi voucher yang tidak dibatalkan.
func (q *Queries) MaxFuelVoucherOdometer(ctx context.Context, vehicleID int32) (int32, error) {
	var odo int32
	err := q.db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(GREATEST(odometer, COALESCE("usedOdometer", 0))), 0)
		FROM fuel_vouchers WHERE "vehicleId" = $1 AND status <> 'CANCELLED'`, vehicleID).Scan(&odo)
	return odo, err
}
