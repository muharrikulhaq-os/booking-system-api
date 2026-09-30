package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

// Buku saldo BBM/listrik per kendaraan (docs/RANCANGAN_VOUCHER_BBM.md §2):
//
//	hak   = (odometer kejadian − odometer titik hitung sebelumnya) ÷ km per unit
//	saldo = saldo sebelumnya + hak − keluar
//
// Setiap penulisan saldo berjalan dalam transaksi yang mengunci baris
// kendaraan (LockVehicleFuelProfile), jadi dua kejadian serentak pada
// kendaraan yang sama tidak membaca saldo terakhir yang sama.

const (
	ledgerOpening          = "OPENING"
	ledgerVoucher          = "VOUCHER"
	ledgerDirectFill       = "DIRECT_FILL"
	ledgerVoucherReturn    = "VOUCHER_RETURN"
	ledgerVoucherReinstate = "VOUCHER_REINSTATE"
	ledgerVoid             = "VOID"
	ledgerAdjustment       = "ADJUSTMENT"

	topicFuel = "fuel"
)

func round2(f float64) float64 { return math.Round(f*100) / 100 }

// kmPerUnit: km/L untuk BBM, km/kWh untuk LISTRIK.
func kmPerUnit(p repository.VehicleFuelProfile, energy repository.FuelCategory) sql.NullFloat64 {
	if energy == repository.FuelCategoryLISTRIK {
		return p.KmPerKwh
	}
	return p.KmPerLiter
}

// capacityFor: kapasitas tangki (liter) / baterai (kWh).
func capacityFor(p repository.VehicleFuelProfile, energy repository.FuelCategory) sql.NullFloat64 {
	if energy == repository.FuelCategoryLISTRIK {
		return p.BatteryCapacityKwh
	}
	return p.TankCapacityLiter
}

// energiesFor: buku saldo yang dimiliki kendaraan (HYBRID punya dua).
func energiesFor(e repository.EnergyType) []repository.FuelCategory {
	switch e {
	case repository.EnergyTypeLISTRIK:
		return []repository.FuelCategory{repository.FuelCategoryLISTRIK}
	case repository.EnergyTypeHYBRID:
		return []repository.FuelCategory{repository.FuelCategoryBBM, repository.FuelCategoryLISTRIK}
	}
	return []repository.FuelCategory{repository.FuelCategoryBBM}
}

func baselineOdometer(p repository.VehicleFuelProfile) int32 {
	if p.FuelBaselineOdometer.Valid {
		return p.FuelBaselineOdometer.Int32
	}
	return p.CurrentOdometer
}

// accrue: hak (liter/kWh) dari jarak tempuh; 0 bila km per unit belum diatur.
func accrue(distanceKm int32, kpu sql.NullFloat64) float64 {
	if !kpu.Valid || kpu.Float64 <= 0 || distanceKm <= 0 {
		return 0
	}
	return round2(float64(distanceKm) / kpu.Float64)
}

type ledgerState struct {
	CheckpointOdometer int32
	Balance            float64
}

func readLedgerState(ctx context.Context, q *repository.Queries, p repository.VehicleFuelProfile, energy repository.FuelCategory) (ledgerState, error) {
	st := ledgerState{CheckpointOdometer: baselineOdometer(p)}
	last, err := q.GetLastFuelLedgerEntry(ctx, p.VehicleID, energy)
	switch {
	case err == nil:
		st.Balance = last.BalanceAfter
	case !errors.Is(err, sql.ErrNoRows):
		return st, err
	}
	cp, err := q.GetLastFuelCheckpoint(ctx, p.VehicleID, energy)
	switch {
	case err == nil && cp.Odometer.Valid:
		st.CheckpointOdometer = cp.Odometer.Int32
	case err != nil && !errors.Is(err, sql.ErrNoRows):
		return st, err
	}
	return st, nil
}

type ledgerEvent struct {
	EntryType     string
	Odometer      sql.NullInt32 // titik hitung baru (NULL = tidak bergeser)
	Debit         float64       // keluar; negatif = kembali ke saldo
	Accrued       *float64      // override hak (pembalikan); nil = dihitung dari odometer
	FuelExpenseID sql.NullInt32
	VoucherID     sql.NullInt32
	ReversesID    sql.NullInt32
	CreatedByID   int32
	Note          string
}

type ledgerResult struct {
	ID           int32
	DistanceKm   int32
	Accrued      float64
	Debit        float64
	BalanceAfter float64
	KmPerUnit    sql.NullFloat64
	BalanceFrom  float64
}

func appendLedger(ctx context.Context, q *repository.Queries, p repository.VehicleFuelProfile, energy repository.FuelCategory, ev ledgerEvent) (ledgerResult, error) {
	st, err := readLedgerState(ctx, q, p, energy)
	if err != nil {
		return ledgerResult{}, err
	}
	kpu := kmPerUnit(p, energy)
	res := ledgerResult{KmPerUnit: kpu, Debit: round2(ev.Debit), BalanceFrom: st.Balance}
	var distance sql.NullInt32
	if ev.Odometer.Valid && ev.Accrued == nil {
		d := ev.Odometer.Int32 - st.CheckpointOdometer
		if d < 0 {
			return res, util.NewError(400,
				fmt.Sprintf("odometer %d km lebih kecil dari titik hitung saldo terakhir (%d km)", ev.Odometer.Int32, st.CheckpointOdometer),
				util.ErrBadRequest)
		}
		res.DistanceKm = d
		distance = sql.NullInt32{Int32: d, Valid: true}
		res.Accrued = accrue(d, kpu)
	}
	if ev.Accrued != nil {
		res.Accrued = round2(*ev.Accrued)
	}
	res.BalanceAfter = round2(st.Balance + res.Accrued - res.Debit)
	var kpuSnap sql.NullFloat64
	if ev.Odometer.Valid {
		kpuSnap = kpu
	}
	id, err := q.InsertFuelLedgerEntry(ctx, repository.InsertFuelLedgerEntryParams{
		VehicleID:     p.VehicleID,
		Energy:        energy,
		EntryType:     ev.EntryType,
		Odometer:      ev.Odometer,
		DistanceKm:    distance,
		KmPerUnit:     kpuSnap,
		Accrued:       res.Accrued,
		Debit:         res.Debit,
		BalanceAfter:  res.BalanceAfter,
		FuelExpenseID: ev.FuelExpenseID,
		VoucherID:     ev.VoucherID,
		ReversesID:    ev.ReversesID,
		CreatedByID:   sql.NullInt32{Int32: ev.CreatedByID, Valid: ev.CreatedByID != 0},
		Note:          sql.NullString{String: ev.Note, Valid: ev.Note != ""},
	})
	res.ID = id
	return res, err
}

// withTx menjalankan fn dalam satu transaksi.
func withTx(ctx context.Context, db *sql.DB, fn func(q *repository.Queries) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(repository.New(tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func nullFloat(f sql.NullFloat64) any {
	if !f.Valid {
		return nil
	}
	return f.Float64
}

func nullInt(i sql.NullInt32) any {
	if !i.Valid {
		return nil
	}
	return i.Int32
}

// ─── SERVICE ────────────────────────────────────────────────────────────────

type FuelLedgerService struct {
	db      *sql.DB
	q       *repository.Queries
	publish Publisher
}

func NewFuelLedgerService(db *sql.DB) *FuelLedgerService {
	return &FuelLedgerService{db: db, q: repository.New(db)}
}

func (s *FuelLedgerService) SetPublisher(p Publisher) { s.publish = p }

func serializeFuelProfile(p repository.VehicleFuelProfile) map[string]any {
	return map[string]any{
		"kmPerLiter":           nullFloat(p.KmPerLiter),
		"tankCapacityLiter":    nullFloat(p.TankCapacityLiter),
		"kmPerKwh":             nullFloat(p.KmPerKwh),
		"batteryCapacityKwh":   nullFloat(p.BatteryCapacityKwh),
		"fuelBaselineOdometer": baselineOdometer(p),
	}
}

// balanceView: saldo tercatat + hak dari jarak yang belum "dikunci" kejadian.
func balanceView(p repository.VehicleFuelProfile, energy repository.FuelCategory, st ledgerState, hasEntries bool) map[string]any {
	kpu := kmPerUnit(p, energy)
	pendingKm := p.CurrentOdometer - st.CheckpointOdometer
	if pendingKm < 0 {
		pendingKm = 0
	}
	pending := accrue(pendingKm, kpu)
	available := round2(st.Balance + pending)
	capacity := capacityFor(p, energy)

	var warnings []string
	if !kpu.Valid {
		if energy == repository.FuelCategoryLISTRIK {
			warnings = append(warnings, "Efisiensi km/kWh belum diatur - saldo kWh belum dihitung")
		} else {
			warnings = append(warnings, "Konsumsi km/liter belum diatur - saldo BBM belum dihitung")
		}
	}
	if energy == repository.FuelCategoryBBM && !capacity.Valid {
		warnings = append(warnings, "Kapasitas tangki belum diatur - voucher belum bisa diterbitkan")
	}
	if capacity.Valid && available > capacity.Float64*1.5 {
		warnings = append(warnings, "Saldo jauh melebihi kapasitas - kemungkinan ada pengisian yang belum dicatat")
	}

	out := map[string]any{
		"energy":             energy,
		"unit":               unitFor(energy),
		"kmPerUnit":          nullFloat(kpu),
		"capacity":           nullFloat(capacity),
		"checkpointOdometer": st.CheckpointOdometer,
		"currentOdometer":    p.CurrentOdometer,
		"pendingKm":          pendingKm,
		"pendingAccrued":     pending,
		"recordedBalance":    round2(st.Balance),
		"available":          available,
		"hasEntries":         hasEntries,
		"warnings":           warnings,
		"voucherable":        nil,
	}
	if energy == repository.FuelCategoryBBM && kpu.Valid && capacity.Valid {
		v := available
		if v > capacity.Float64 {
			v = capacity.Float64
		}
		if v < 0 {
			v = 0
		}
		out["voucherable"] = round2(v)
	}
	return out
}

func unitFor(energy repository.FuelCategory) string {
	if energy == repository.FuelCategoryLISTRIK {
		return "kWh"
	}
	return "L"
}

func serializeActiveVoucher(v repository.FuelVoucher) map[string]any {
	return map[string]any{
		"id": v.ID, "code": v.Code, "liter": v.Liter, "amount": v.Amount,
		"validUntil": v.ValidUntil, "stationName": v.StationName,
	}
}

func (s *FuelLedgerService) ListBalances(ctx context.Context) ([]map[string]any, error) {
	profiles, err := s.q.ListVehicleFuelProfiles(ctx)
	if err != nil {
		return nil, err
	}
	heads, err := s.q.ListFuelLedgerHeads(ctx)
	if err != nil {
		return nil, err
	}
	type key struct {
		v int32
		e repository.FuelCategory
	}
	headBy := map[key]repository.FuelLedgerHead{}
	for _, h := range heads {
		headBy[key{h.VehicleID, h.Energy}] = h
	}
	active, err := s.q.ActiveFuelVoucherByVehicle(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(profiles))
	for _, p := range profiles {
		balances := map[string]any{}
		for _, e := range energiesFor(p.EnergyType) {
			st := ledgerState{CheckpointOdometer: baselineOdometer(p)}
			h, ok := headBy[key{p.VehicleID, e}]
			if ok {
				st.Balance = h.Balance
				if h.CheckpointOdometer.Valid {
					st.CheckpointOdometer = h.CheckpointOdometer.Int32
				}
			}
			balances[string(e)] = balanceView(p, e, st, ok)
		}
		row := s.vehicleRow(p, balances)
		if v, ok := active[p.VehicleID]; ok {
			row["activeVoucher"] = serializeActiveVoucher(v)
		}
		out = append(out, row)
	}
	return out, nil
}

func (s *FuelLedgerService) vehicleRow(p repository.VehicleFuelProfile, balances map[string]any) map[string]any {
	return map[string]any{
		"vehicleId":       p.VehicleID,
		"vehicleName":     p.VehicleName,
		"plateNumber":     p.PlateNumber,
		"energyType":      p.EnergyType,
		"currentOdometer": p.CurrentOdometer,
		"fixedDriverId":   nullInt(p.FixedDriverID),
		"profile":         serializeFuelProfile(p),
		"balances":        balances,
		"activeVoucher":   nil,
	}
}

func (s *FuelLedgerService) GetBalance(ctx context.Context, vehicleID int32) (map[string]any, error) {
	p, err := s.q.GetVehicleFuelProfile(ctx, vehicleID)
	if err != nil {
		return nil, util.NewError(404, "Kendaraan tidak ditemukan", util.ErrNotFound)
	}
	return s.balanceFor(ctx, s.q, p)
}

func (s *FuelLedgerService) balanceFor(ctx context.Context, q *repository.Queries, p repository.VehicleFuelProfile) (map[string]any, error) {
	balances := map[string]any{}
	for _, e := range energiesFor(p.EnergyType) {
		st, err := readLedgerState(ctx, q, p, e)
		if err != nil {
			return nil, err
		}
		_, lerr := q.GetLastFuelLedgerEntry(ctx, p.VehicleID, e)
		balances[string(e)] = balanceView(p, e, st, lerr == nil)
	}
	row := s.vehicleRow(p, balances)
	n, _ := q.CountFuelLedgerEntries(ctx, p.VehicleID)
	row["baselineLocked"] = n > 0
	if id, err := q.GetActiveFuelVoucherID(ctx, p.VehicleID); err == nil {
		if v, err := q.GetFuelVoucher(ctx, id); err == nil {
			row["activeVoucher"] = serializeActiveVoucher(v)
		}
	}
	return row, nil
}

type FuelProfileRequest struct {
	KmPerLiter           *float64 `json:"kmPerLiter"`
	TankCapacityLiter    *float64 `json:"tankCapacityLiter"`
	KmPerKwh             *float64 `json:"kmPerKwh"`
	BatteryCapacityKwh   *float64 `json:"batteryCapacityKwh"`
	FuelBaselineOdometer *int32   `json:"fuelBaselineOdometer"`
}

func optPositive(v *float64, label string, max float64) (sql.NullFloat64, error) {
	if v == nil || *v == 0 {
		return sql.NullFloat64{}, nil
	}
	if *v < 0 || *v > max {
		return sql.NullFloat64{}, util.NewError(400, fmt.Sprintf("%s harus di antara 0 dan %g", label, max), util.ErrBadRequest)
	}
	return sql.NullFloat64{Float64: round2(*v), Valid: true}, nil
}

func (s *FuelLedgerService) UpdateProfile(ctx context.Context, vehicleID int32, req FuelProfileRequest, actor AuditActor) (map[string]any, error) {
	kmL, err := optPositive(req.KmPerLiter, "Konsumsi km/liter", 100)
	if err != nil {
		return nil, err
	}
	tank, err := optPositive(req.TankCapacityLiter, "Kapasitas tangki", 2000)
	if err != nil {
		return nil, err
	}
	kmK, err := optPositive(req.KmPerKwh, "Efisiensi km/kWh", 100)
	if err != nil {
		return nil, err
	}
	batt, err := optPositive(req.BatteryCapacityKwh, "Kapasitas baterai", 1000)
	if err != nil {
		return nil, err
	}
	err = withTx(ctx, s.db, func(q *repository.Queries) error {
		p, err := q.LockVehicleFuelProfile(ctx, vehicleID)
		if err != nil {
			return util.NewError(404, "Kendaraan tidak ditemukan", util.ErrNotFound)
		}
		var baseline sql.NullInt32
		if req.FuelBaselineOdometer != nil && *req.FuelBaselineOdometer != baselineOdometer(p) {
			n, err := q.CountFuelLedgerEntries(ctx, vehicleID)
			if err != nil {
				return err
			}
			if n > 0 {
				return util.NewError(409,
					"odometer awal BBM tidak bisa diubah karena sudah ada catatan saldo - gunakan Penyesuaian Saldo",
					util.ErrConflict)
			}
			if *req.FuelBaselineOdometer < 0 || *req.FuelBaselineOdometer > p.CurrentOdometer {
				return util.NewError(400,
					fmt.Sprintf("odometer awal BBM harus di antara 0 dan odometer kendaraan saat ini (%d km)", p.CurrentOdometer),
					util.ErrBadRequest)
			}
			baseline = sql.NullInt32{Int32: *req.FuelBaselineOdometer, Valid: true}
		}
		return q.UpdateVehicleFuelProfile(ctx, repository.UpdateVehicleFuelProfileParams{
			VehicleID: vehicleID, KmPerLiter: kmL, TankCapacityLiter: tank,
			KmPerKwh: kmK, BatteryCapacityKwh: batt, FuelBaselineOdometer: baseline,
		})
	})
	if err != nil {
		return nil, err
	}
	logAudit(ctx, s.q, actor, "UPDATE", "Vehicle", vehicleID, "Mengubah profil BBM kendaraan (konsumsi, kapasitas, odometer awal)")
	return s.GetBalance(ctx, vehicleID)
}

type FuelAdjustmentRequest struct {
	Energy string  `json:"energy"`
	Amount float64 `json:"amount"` // + menambah saldo, − mengurangi
	Note   string  `json:"note"`
}

func (s *FuelLedgerService) Adjust(ctx context.Context, vehicleID int32, req FuelAdjustmentRequest, actor AuditActor) (map[string]any, error) {
	req.Note = strings.TrimSpace(req.Note)
	if req.Note == "" {
		return nil, util.NewError(400, "alasan penyesuaian wajib diisi", util.ErrBadRequest)
	}
	if req.Amount == 0 || math.Abs(req.Amount) > 10000 {
		return nil, util.NewError(400, "jumlah penyesuaian tidak valid", util.ErrBadRequest)
	}
	energy := repository.FuelCategory(req.Energy)
	if energy == "" {
		energy = repository.FuelCategoryBBM
	}
	var res ledgerResult
	err := withTx(ctx, s.db, func(q *repository.Queries) error {
		p, err := q.LockVehicleFuelProfile(ctx, vehicleID)
		if err != nil {
			return util.NewError(404, "Kendaraan tidak ditemukan", util.ErrNotFound)
		}
		ok := false
		for _, e := range energiesFor(p.EnergyType) {
			ok = ok || e == energy
		}
		if !ok {
			return util.NewError(400, fmt.Sprintf("kendaraan %s tidak memiliki saldo %s", p.EnergyType, energy), util.ErrBadRequest)
		}
		res, err = appendLedger(ctx, q, p, energy, ledgerEvent{
			EntryType: ledgerAdjustment, Debit: -req.Amount, CreatedByID: actor.UserID, Note: req.Note,
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	logAudit(ctx, s.q, actor, "ADJUST", "FuelLedger", res.ID,
		fmt.Sprintf("Penyesuaian saldo %s %+.2f %s: %s", energy, req.Amount, unitFor(energy), req.Note))
	return s.GetBalance(ctx, vehicleID)
}

func serializeLedgerEntry(e repository.FuelLedgerEntry) map[string]any {
	return map[string]any{
		"id":            e.ID,
		"vehicleId":     e.VehicleID,
		"energy":        e.Energy,
		"entryType":     e.EntryType,
		"odometer":      nullInt(e.Odometer),
		"distanceKm":    nullInt(e.DistanceKm),
		"kmPerUnit":     nullFloat(e.KmPerUnit),
		"accrued":       e.Accrued,
		"debit":         e.Debit,
		"balanceAfter":  e.BalanceAfter,
		"fuelExpenseId": nullInt(e.FuelExpenseID),
		"reversesId":    nullInt(e.ReversesID),
		"createdByName": nullStr(e.CreatedByName),
		"note":          nullStr(e.Note),
		"createdAt":     e.CreatedAt,
	}
}

func (s *FuelLedgerService) ListLedger(ctx context.Context, vehicleID int32, energy *string, page, limit int) ([]map[string]any, int64, error) {
	arg := repository.ListFuelLedgerParams{VehicleID: vehicleID, Limit: int32(limit), Offset: int32((page - 1) * limit)}
	if energy != nil {
		arg.Energy = repository.NullFuelCategory{FuelCategory: repository.FuelCategory(*energy), Valid: true}
	}
	rows, total, err := s.q.ListFuelLedger(ctx, arg)
	if err != nil {
		return nil, 0, err
	}
	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		out[i] = serializeLedgerEntry(r)
	}
	return out, total, nil
}

// ─── SPBU MITRA ─────────────────────────────────────────────────────────────

type FuelStationRequest struct {
	Name          string `json:"name"`
	Address       string `json:"address"`
	Phone         string `json:"phone"`
	ContactPerson string `json:"contactPerson"`
	IsActive      *bool  `json:"isActive"`
}

func serializeFuelStation(f repository.FuelStation) map[string]any {
	return map[string]any{
		"id": f.ID, "name": f.Name, "address": nullStr(f.Address), "phone": nullStr(f.Phone),
		"contactPerson": nullStr(f.ContactPerson), "isActive": f.IsActive,
		"createdAt": f.CreatedAt, "updatedAt": f.UpdatedAt,
	}
}

func optStr(s string) sql.NullString {
	s = strings.TrimSpace(s)
	return sql.NullString{String: s, Valid: s != ""}
}

func (s *FuelLedgerService) ListStations(ctx context.Context, activeOnly bool) ([]map[string]any, error) {
	rows, err := s.q.ListFuelStations(ctx, activeOnly)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		out[i] = serializeFuelStation(r)
	}
	return out, nil
}

func (s *FuelLedgerService) stationParams(req FuelStationRequest) (repository.UpsertFuelStationParams, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return repository.UpsertFuelStationParams{}, util.NewError(400, "nama SPBU wajib diisi", util.ErrBadRequest)
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	return repository.UpsertFuelStationParams{
		Name: name, Address: optStr(req.Address), Phone: optStr(req.Phone),
		ContactPerson: optStr(req.ContactPerson), IsActive: active,
	}, nil
}

func duplicateStation(err error) error {
	if err != nil && strings.Contains(err.Error(), "fuel_stations_name_key") {
		return util.NewError(409, "nama SPBU mitra sudah ada", util.ErrConflict)
	}
	return err
}

func (s *FuelLedgerService) CreateStation(ctx context.Context, req FuelStationRequest, actor AuditActor) (map[string]any, error) {
	arg, err := s.stationParams(req)
	if err != nil {
		return nil, err
	}
	f, err := s.q.CreateFuelStation(ctx, arg)
	if err != nil {
		return nil, duplicateStation(err)
	}
	logAudit(ctx, s.q, actor, "CREATE", "FuelStation", f.ID, "Menambah SPBU mitra "+f.Name)
	return serializeFuelStation(f), nil
}

func (s *FuelLedgerService) UpdateStation(ctx context.Context, id int32, req FuelStationRequest, actor AuditActor) (map[string]any, error) {
	arg, err := s.stationParams(req)
	if err != nil {
		return nil, err
	}
	if _, err := s.q.GetFuelStation(ctx, id); err != nil {
		return nil, util.ErrNotFound
	}
	arg.ID = id
	f, err := s.q.UpdateFuelStation(ctx, arg)
	if err != nil {
		return nil, duplicateStation(err)
	}
	logAudit(ctx, s.q, actor, "UPDATE", "FuelStation", f.ID, "Mengubah SPBU mitra "+f.Name)
	return serializeFuelStation(f), nil
}

func (s *FuelLedgerService) DeleteStation(ctx context.Context, id int32, actor AuditActor) error {
	f, err := s.q.GetFuelStation(ctx, id)
	if err != nil {
		return util.ErrNotFound
	}
	if err := s.q.DeleteFuelStation(ctx, id); err != nil {
		return inUseError(err, "SPBU ini sudah dipakai di voucher/pengisian - nonaktifkan saja")
	}
	logAudit(ctx, s.q, actor, "DELETE", "FuelStation", id, "Menghapus SPBU mitra "+f.Name)
	return nil
}
