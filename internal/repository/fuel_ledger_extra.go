package repository

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"
)

// Hand-written (sqlc CLI unavailable) - buku saldo BBM/listrik per kendaraan,
// profil BBM kendaraan, master SPBU mitra, dan kolom tambahan fuel_expenses
// (sumber, SPBU, alasan, pembatalan). Lihat docs/RANCANGAN_VOUCHER_BBM.md.

// ─── PROFIL BBM KENDARAAN ───────────────────────────────────────────────────

type VehicleFuelProfile struct {
	VehicleID            int32           `json:"vehicleId"`
	VehicleName          string          `json:"vehicleName"`
	PlateNumber          string          `json:"plateNumber"`
	EnergyType           EnergyType      `json:"energyType"`
	CurrentOdometer      int32           `json:"currentOdometer"`
	KmPerLiter           sql.NullFloat64 `json:"kmPerLiter"`
	TankCapacityLiter    sql.NullFloat64 `json:"tankCapacityLiter"`
	KmPerKwh             sql.NullFloat64 `json:"kmPerKwh"`
	BatteryCapacityKwh   sql.NullFloat64 `json:"batteryCapacityKwh"`
	FuelBaselineOdometer sql.NullInt32   `json:"fuelBaselineOdometer"`
	FixedDriverID        sql.NullInt32   `json:"fixedDriverId"`
}

const vehicleFuelProfileSelect = `
	SELECT v.id, r.name, v."plateNumber", v.energy_type, v."currentOdometer",
	       v."kmPerLiter"::float8, v."tankCapacityLiter"::float8, v."kmPerKwh"::float8,
	       v."batteryCapacityKwh"::float8, v."fuelBaselineOdometer", v."fixedDriverId"
	FROM vehicles v JOIN resources r ON r.id = v."resourceId"`

func scanVehicleFuelProfile(s interface{ Scan(...any) error }) (VehicleFuelProfile, error) {
	var p VehicleFuelProfile
	err := s.Scan(&p.VehicleID, &p.VehicleName, &p.PlateNumber, &p.EnergyType, &p.CurrentOdometer,
		&p.KmPerLiter, &p.TankCapacityLiter, &p.KmPerKwh, &p.BatteryCapacityKwh,
		&p.FuelBaselineOdometer, &p.FixedDriverID)
	return p, err
}

func (q *Queries) GetVehicleFuelProfile(ctx context.Context, vehicleID int32) (VehicleFuelProfile, error) {
	return scanVehicleFuelProfile(q.db.QueryRowContext(ctx, vehicleFuelProfileSelect+` WHERE v.id = $1`, vehicleID))
}

// LockVehicleFuelProfile mengunci baris kendaraan (SELECT … FOR UPDATE) -
// dipakai di dalam transaksi supaya dua penulisan saldo serentak pada
// kendaraan yang sama tidak membaca saldo terakhir yang sama.
func (q *Queries) LockVehicleFuelProfile(ctx context.Context, vehicleID int32) (VehicleFuelProfile, error) {
	return scanVehicleFuelProfile(q.db.QueryRowContext(ctx, vehicleFuelProfileSelect+` WHERE v.id = $1 FOR UPDATE OF v`, vehicleID))
}

func (q *Queries) ListVehicleFuelProfiles(ctx context.Context) ([]VehicleFuelProfile, error) {
	rows, err := q.db.QueryContext(ctx, vehicleFuelProfileSelect+` ORDER BY r.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []VehicleFuelProfile
	for rows.Next() {
		p, err := scanVehicleFuelProfile(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}

type UpdateVehicleFuelProfileParams struct {
	VehicleID            int32
	KmPerLiter           sql.NullFloat64
	TankCapacityLiter    sql.NullFloat64
	KmPerKwh             sql.NullFloat64
	BatteryCapacityKwh   sql.NullFloat64
	FuelBaselineOdometer sql.NullInt32
}

func (q *Queries) UpdateVehicleFuelProfile(ctx context.Context, arg UpdateVehicleFuelProfileParams) error {
	_, err := q.db.ExecContext(ctx, `
		UPDATE vehicles SET "kmPerLiter" = $2, "tankCapacityLiter" = $3, "kmPerKwh" = $4,
		       "batteryCapacityKwh" = $5, "fuelBaselineOdometer" = COALESCE($6, "fuelBaselineOdometer")
		WHERE id = $1`,
		arg.VehicleID, arg.KmPerLiter, arg.TankCapacityLiter, arg.KmPerKwh, arg.BatteryCapacityKwh,
		arg.FuelBaselineOdometer)
	return err
}

// SetVehicleOdometerExact menulis odometer kendaraan apa adanya (boleh turun) -
// HANYA untuk koreksi odometer salah ketik saat membatalkan catatan BBM.
func (q *Queries) SetVehicleOdometerExact(ctx context.Context, vehicleID, odometer int32) error {
	_, err := q.db.ExecContext(ctx, `UPDATE vehicles SET "currentOdometer" = $2 WHERE id = $1`, vehicleID, odometer)
	return err
}

// MaxVehicleOdometerReading: bacaan odometer sah tertinggi kendaraan dari
// semua sumber faktual (catatan BBM yang tidak dibatalkan, titik saldo,
// awal/akhir trip, maintenance), tanpa catatan BBM excludeExpenseID.
func (q *Queries) MaxVehicleOdometerReading(ctx context.Context, vehicleID, excludeExpenseID int32, includeBaseline bool) (int32, error) {
	var odo int32
	err := q.db.QueryRowContext(ctx, `
		SELECT GREATEST(
		  COALESCE((SELECT MAX("odometerAfter") FROM fuel_expenses
		            WHERE "vehicleId" = $1 AND "voidedAt" IS NULL AND id <> $2), 0),
		  CASE WHEN $3 THEN COALESCE((SELECT "fuelBaselineOdometer" FROM vehicles WHERE id = $1), 0) ELSE 0 END,
		  COALESCE((SELECT MAX(b."odometerStart") FROM bookings b
		            WHERE b."assignedVehicleId" = $1), 0),
		  COALESCE((SELECT MAX(rr.odometer) FROM booking_return_reports rr
		            JOIN bookings b ON b.id = rr."bookingId" WHERE b."assignedVehicleId" = $1), 0),
		  COALESCE((SELECT MAX(GREATEST(COALESCE(mr.odometer, 0), COALESCE(mr."handoverOdometer", 0), COALESCE(mr."returnOdometer", 0)))
		            FROM maintenance_records mr WHERE mr."vehicleId" = $1), 0)
		)`, vehicleID, excludeExpenseID, includeBaseline).Scan(&odo)
	return odo, err
}

// ─── BUKU SALDO ─────────────────────────────────────────────────────────────

type FuelLedgerEntry struct {
	ID            int32           `json:"id"`
	VehicleID     int32           `json:"vehicleId"`
	Energy        FuelCategory    `json:"energy"`
	EntryType     string          `json:"entryType"`
	Odometer      sql.NullInt32   `json:"odometer"`
	DistanceKm    sql.NullInt32   `json:"distanceKm"`
	KmPerUnit     sql.NullFloat64 `json:"kmPerUnit"`
	Accrued       float64         `json:"accrued"`
	Debit         float64         `json:"debit"`
	BalanceAfter  float64         `json:"balanceAfter"`
	FuelExpenseID sql.NullInt32   `json:"fuelExpenseId"`
	ReversesID    sql.NullInt32   `json:"reversesId"`
	CreatedByID   sql.NullInt32   `json:"createdById"`
	CreatedByName sql.NullString  `json:"createdByName"`
	Note          sql.NullString  `json:"note"`
	CreatedAt     time.Time       `json:"createdAt"`
}

const fuelLedgerSelect = `
	SELECT l.id, l."vehicleId", l.energy, l."entryType"::text, l.odometer, l."distanceKm",
	       l."kmPerUnit"::float8, l.accrued::float8, l.debit::float8, l."balanceAfter"::float8,
	       l."fuelExpenseId", l."reversesId", l."createdById", u.name, l.note, l."createdAt"
	FROM fuel_ledger l LEFT JOIN users u ON u.id = l."createdById"`

func scanFuelLedgerEntry(s interface{ Scan(...any) error }) (FuelLedgerEntry, error) {
	var e FuelLedgerEntry
	err := s.Scan(&e.ID, &e.VehicleID, &e.Energy, &e.EntryType, &e.Odometer, &e.DistanceKm,
		&e.KmPerUnit, &e.Accrued, &e.Debit, &e.BalanceAfter,
		&e.FuelExpenseID, &e.ReversesID, &e.CreatedByID, &e.CreatedByName, &e.Note, &e.CreatedAt)
	return e, err
}

// GetLastFuelLedgerEntry: entri terakhir (saldo terkini). sql.ErrNoRows bila belum ada.
func (q *Queries) GetLastFuelLedgerEntry(ctx context.Context, vehicleID int32, energy FuelCategory) (FuelLedgerEntry, error) {
	return scanFuelLedgerEntry(q.db.QueryRowContext(ctx, fuelLedgerSelect+`
		WHERE l."vehicleId" = $1 AND l.energy = $2 ORDER BY l.id DESC LIMIT 1`, vehicleID, energy))
}

// GetLastFuelCheckpoint: entri terakhir yang membawa odometer (titik hitung
// hak berikutnya). sql.ErrNoRows bila belum ada.
func (q *Queries) GetLastFuelCheckpoint(ctx context.Context, vehicleID int32, energy FuelCategory) (FuelLedgerEntry, error) {
	return scanFuelLedgerEntry(q.db.QueryRowContext(ctx, fuelLedgerSelect+`
		WHERE l."vehicleId" = $1 AND l.energy = $2 AND l.odometer IS NOT NULL
		ORDER BY l.id DESC LIMIT 1`, vehicleID, energy))
}

// GetFuelCheckpointBefore: titik hitung terakhir sebelum entri beforeID.
func (q *Queries) GetFuelCheckpointBefore(ctx context.Context, vehicleID int32, energy FuelCategory, beforeID int32) (FuelLedgerEntry, error) {
	return scanFuelLedgerEntry(q.db.QueryRowContext(ctx, fuelLedgerSelect+`
		WHERE l."vehicleId" = $1 AND l.energy = $2 AND l.odometer IS NOT NULL AND l.id < $3
		ORDER BY l.id DESC LIMIT 1`, vehicleID, energy, beforeID))
}

func (q *Queries) GetFuelLedgerEntryByExpense(ctx context.Context, expenseID int32) (FuelLedgerEntry, error) {
	return scanFuelLedgerEntry(q.db.QueryRowContext(ctx, fuelLedgerSelect+`
		WHERE l."fuelExpenseId" = $1 AND l."entryType" = 'DIRECT_FILL' ORDER BY l.id LIMIT 1`, expenseID))
}

func (q *Queries) CountFuelLedgerEntries(ctx context.Context, vehicleID int32) (int64, error) {
	var n int64
	err := q.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM fuel_ledger WHERE "vehicleId" = $1`, vehicleID).Scan(&n)
	return n, err
}

type InsertFuelLedgerEntryParams struct {
	VehicleID     int32
	Energy        FuelCategory
	EntryType     string
	Odometer      sql.NullInt32
	DistanceKm    sql.NullInt32
	KmPerUnit     sql.NullFloat64
	Accrued       float64
	Debit         float64
	BalanceAfter  float64
	FuelExpenseID sql.NullInt32
	ReversesID    sql.NullInt32
	VoucherID     sql.NullInt32
	CreatedByID   sql.NullInt32
	Note          sql.NullString
}

func (q *Queries) InsertFuelLedgerEntry(ctx context.Context, arg InsertFuelLedgerEntryParams) (int32, error) {
	var id int32
	err := q.db.QueryRowContext(ctx, `
		INSERT INTO fuel_ledger ("vehicleId", energy, "entryType", odometer, "distanceKm", "kmPerUnit",
		                         accrued, debit, "balanceAfter", "fuelExpenseId", "reversesId", "createdById", note, "voucherId")
		VALUES ($1, $2, $3::fuel_ledger_type, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id`,
		arg.VehicleID, arg.Energy, arg.EntryType, arg.Odometer, arg.DistanceKm, arg.KmPerUnit,
		numeric2(arg.Accrued), numeric2(arg.Debit), numeric2(arg.BalanceAfter),
		arg.FuelExpenseID, arg.ReversesID, arg.CreatedByID, arg.Note, arg.VoucherID,
	).Scan(&id)
	return id, err
}

func numeric2(f float64) string { return strconv.FormatFloat(f, 'f', 2, 64) }

// intArrayLiteral: literal array Postgres ("{1,2,3}") untuk parameter $n::int[].
func intArrayLiteral(ids []int32) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(int(id))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

type ListFuelLedgerParams struct {
	VehicleID int32
	Energy    NullFuelCategory
	Limit     int32
	Offset    int32
}

func (q *Queries) ListFuelLedger(ctx context.Context, arg ListFuelLedgerParams) ([]FuelLedgerEntry, int64, error) {
	var total int64
	if err := q.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM fuel_ledger
		WHERE "vehicleId" = $1 AND ($2::fuel_category IS NULL OR energy = $2::fuel_category)`,
		arg.VehicleID, arg.Energy).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := q.db.QueryContext(ctx, fuelLedgerSelect+`
		WHERE l."vehicleId" = $1 AND ($2::fuel_category IS NULL OR l.energy = $2::fuel_category)
		ORDER BY l.id DESC LIMIT $3 OFFSET $4`, arg.VehicleID, arg.Energy, arg.Limit, arg.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var items []FuelLedgerEntry
	for rows.Next() {
		e, err := scanFuelLedgerEntry(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, e)
	}
	return items, total, rows.Err()
}

// FuelLedgerHead: saldo terakhir + titik hitung terakhir per kendaraan & energi.
type FuelLedgerHead struct {
	VehicleID          int32
	Energy             FuelCategory
	Balance            float64
	CheckpointOdometer sql.NullInt32
	LastEntryAt        time.Time
}

func (q *Queries) ListFuelLedgerHeads(ctx context.Context) ([]FuelLedgerHead, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT last."vehicleId", last.energy, last."balanceAfter"::float8, cp.odometer, last."createdAt"
		FROM (SELECT DISTINCT ON ("vehicleId", energy) "vehicleId", energy, "balanceAfter", "createdAt"
		      FROM fuel_ledger ORDER BY "vehicleId", energy, id DESC) last
		LEFT JOIN (SELECT DISTINCT ON ("vehicleId", energy) "vehicleId", energy, odometer
		           FROM fuel_ledger WHERE odometer IS NOT NULL
		           ORDER BY "vehicleId", energy, id DESC) cp
		  ON cp."vehicleId" = last."vehicleId" AND cp.energy = last.energy`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []FuelLedgerHead
	for rows.Next() {
		var h FuelLedgerHead
		if err := rows.Scan(&h.VehicleID, &h.Energy, &h.Balance, &h.CheckpointOdometer, &h.LastEntryAt); err != nil {
			return nil, err
		}
		items = append(items, h)
	}
	return items, rows.Err()
}

// ─── SPBU MITRA ─────────────────────────────────────────────────────────────

type FuelStation struct {
	ID            int32          `json:"id"`
	Name          string         `json:"name"`
	Address       sql.NullString `json:"address"`
	Phone         sql.NullString `json:"phone"`
	ContactPerson sql.NullString `json:"contactPerson"`
	IsActive      bool           `json:"isActive"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}

const fuelStationColumns = `id, name, address, phone, "contactPerson", "isActive", "createdAt", "updatedAt"`

func scanFuelStation(s interface{ Scan(...any) error }) (FuelStation, error) {
	var f FuelStation
	err := s.Scan(&f.ID, &f.Name, &f.Address, &f.Phone, &f.ContactPerson, &f.IsActive, &f.CreatedAt, &f.UpdatedAt)
	return f, err
}

func (q *Queries) ListFuelStations(ctx context.Context, activeOnly bool) ([]FuelStation, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT `+fuelStationColumns+` FROM fuel_stations
		WHERE (NOT $1 OR "isActive") ORDER BY name`, activeOnly)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []FuelStation
	for rows.Next() {
		f, err := scanFuelStation(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, f)
	}
	return items, rows.Err()
}

func (q *Queries) GetFuelStation(ctx context.Context, id int32) (FuelStation, error) {
	return scanFuelStation(q.db.QueryRowContext(ctx, `SELECT `+fuelStationColumns+` FROM fuel_stations WHERE id = $1`, id))
}

type UpsertFuelStationParams struct {
	ID            int32
	Name          string
	Address       sql.NullString
	Phone         sql.NullString
	ContactPerson sql.NullString
	IsActive      bool
}

func (q *Queries) CreateFuelStation(ctx context.Context, arg UpsertFuelStationParams) (FuelStation, error) {
	return scanFuelStation(q.db.QueryRowContext(ctx, `
		INSERT INTO fuel_stations (name, address, phone, "contactPerson", "isActive")
		VALUES ($1, $2, $3, $4, $5) RETURNING `+fuelStationColumns,
		arg.Name, arg.Address, arg.Phone, arg.ContactPerson, arg.IsActive))
}

func (q *Queries) UpdateFuelStation(ctx context.Context, arg UpsertFuelStationParams) (FuelStation, error) {
	return scanFuelStation(q.db.QueryRowContext(ctx, `
		UPDATE fuel_stations SET name = $2, address = $3, phone = $4, "contactPerson" = $5, "isActive" = $6
		WHERE id = $1 RETURNING `+fuelStationColumns,
		arg.ID, arg.Name, arg.Address, arg.Phone, arg.ContactPerson, arg.IsActive))
}

func (q *Queries) DeleteFuelStation(ctx context.Context, id int32) error {
	_, err := q.db.ExecContext(ctx, `DELETE FROM fuel_stations WHERE id = $1`, id)
	return err
}

// ─── KOLOM TAMBAHAN FUEL_EXPENSES ───────────────────────────────────────────

type FuelExpenseExtra struct {
	ID           int32
	Source       string
	StationID    sql.NullInt32
	StationName  sql.NullString // nama SPBU mitra, atau nama bebas SPBU lain
	IsPartner    bool
	Reason       sql.NullString
	VoidedAt     sql.NullTime
	VoidedByName sql.NullString
	VoidReason   sql.NullString
	// Dari entri ledger DIRECT_FILL (NULL untuk catatan sebelum fitur saldo).
	LedgerAccrued  sql.NullFloat64
	LedgerDebit    sql.NullFloat64
	LedgerBalance  sql.NullFloat64
	KmPerUnit      sql.NullFloat64
	VoucherID      sql.NullInt32
	VoucherCode    sql.NullString
	MeterStartKwh  sql.NullFloat64
	MeterEndKwh    sql.NullFloat64
	QuantitySource sql.NullString
}

func (q *Queries) GetFuelExpenseExtras(ctx context.Context, ids []int32) (map[int32]FuelExpenseExtra, error) {
	out := make(map[int32]FuelExpenseExtra, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := q.db.QueryContext(ctx, `
		SELECT fe.id, fe.source, fe."stationId", COALESCE(fs.name, fe."stationName"), fs.id IS NOT NULL,
		       fe.reason, fe."voidedAt", vu.name, fe."voidReason",
		       l.accrued::float8, l.debit::float8, l."balanceAfter"::float8, l."kmPerUnit"::float8,
		       fe."voucherId", fv.code, fe."meterStartKwh"::float8, fe."meterEndKwh"::float8, fe."quantitySource"
		FROM fuel_expenses fe
		LEFT JOIN fuel_vouchers fv ON fv.id = fe."voucherId"
		LEFT JOIN fuel_stations fs ON fs.id = fe."stationId"
		LEFT JOIN users vu ON vu.id = fe."voidedById"
		LEFT JOIN LATERAL (SELECT * FROM fuel_ledger x
		                   WHERE x."fuelExpenseId" = fe.id AND x."entryType" = 'DIRECT_FILL'
		                   ORDER BY x.id LIMIT 1) l ON TRUE
		WHERE fe.id = ANY($1::int[])`, intArrayLiteral(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var e FuelExpenseExtra
		if err := rows.Scan(&e.ID, &e.Source, &e.StationID, &e.StationName, &e.IsPartner,
			&e.Reason, &e.VoidedAt, &e.VoidedByName, &e.VoidReason,
			&e.LedgerAccrued, &e.LedgerDebit, &e.LedgerBalance, &e.KmPerUnit,
			&e.VoucherID, &e.VoucherCode, &e.MeterStartKwh, &e.MeterEndKwh, &e.QuantitySource); err != nil {
			return nil, err
		}
		out[e.ID] = e
	}
	return out, rows.Err()
}

type SetFuelExpenseExtraParams struct {
	ID             int32
	Source         string
	StationID      sql.NullInt32
	StationName    sql.NullString
	Reason         sql.NullString
	VoucherID      sql.NullInt32
	MeterStartKwh  sql.NullFloat64
	MeterEndKwh    sql.NullFloat64
	QuantitySource sql.NullString
}

func (q *Queries) SetFuelExpenseExtra(ctx context.Context, arg SetFuelExpenseExtraParams) error {
	_, err := q.db.ExecContext(ctx, `
		UPDATE fuel_expenses SET source = $2, "stationId" = $3, "stationName" = $4, reason = $5,
		       "voucherId" = $6, "meterStartKwh" = $7, "meterEndKwh" = $8, "quantitySource" = $9
		WHERE id = $1`,
		arg.ID, arg.Source, arg.StationID, arg.StationName, arg.Reason,
		arg.VoucherID, arg.MeterStartKwh, arg.MeterEndKwh, arg.QuantitySource)
	return err
}

func (q *Queries) VoidFuelExpense(ctx context.Context, id, voidedByID int32, reason string) error {
	_, err := q.db.ExecContext(ctx, `
		UPDATE fuel_expenses SET "voidedAt" = NOW(), "voidedById" = $2, "voidReason" = $3
		WHERE id = $1 AND "voidedAt" IS NULL`, id, voidedByID, reason)
	return err
}
