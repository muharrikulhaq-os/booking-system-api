package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

type FuelExpenseService struct {
	db *sql.DB
	q  *repository.Queries
}

func NewFuelExpenseService(db *sql.DB) *FuelExpenseService {
	return &FuelExpenseService{db: db, q: repository.New(db)}
}

// CreateFuelExpenseRequest: pengisian LANGSUNG (tanpa voucher) - SPD,
// perjalanan jauh, darurat, SPBU non-mitra, atau charging listrik di kantor.
type CreateFuelExpenseRequest struct {
	VehicleID  int32  `json:"vehicleId"      validate:"required"`
	FuelTypeID int32  `json:"fuelTypeId"`
	BookingID  *int32 `json:"bookingId"`
	FuelGrade  string `json:"fuelGrade"`
	// Odometer saat mengisi. Klien lama mengirim odometerAfter (dipakai bila
	// odometer kosong); odometerBefore kini dihitung server dari titik saldo.
	Odometer       int32   `json:"odometer"`
	OdometerBefore int32   `json:"odometerBefore"`
	OdometerAfter  int32   `json:"odometerAfter"`
	Liter          float64 `json:"liter"`
	PricePerLiter  float64 `json:"pricePerLiter"`
	Kwh            float64 `json:"kwh"`
	PricePerKwh    float64 `json:"pricePerKwh"`
	// Listrik - kWh bebas: kwh langsung, ATAU angka meter awal/akhir, ATAU
	// estimasi % baterai × kapasitas baterai.
	MeterStartKwh float64 `json:"meterStartKwh"`
	MeterEndKwh   float64 `json:"meterEndKwh"`
	BatteryBefore float64 `json:"batteryBefore"`
	BatteryAfter  float64 `json:"batteryAfter"`
	StationID     int32   `json:"stationId"`
	StationName   string  `json:"stationName"`
	Reason        string  `json:"reason"`
	ProofPhotoUrl string  `json:"proofPhotoUrl"`
	Note          string  `json:"note"`
}

var fuelFillReasons = map[string]bool{"SPD": true, "LONG_TRIP": true, "EMERGENCY": true, "OFFICE": true, "OTHER": true}

// chargerEfficiency: rugi-rugi charger untuk estimasi kWh dari % baterai.
const chargerEfficiency = 0.9

func numericFromFloat(f float64) sql.NullString {
	return sql.NullString{String: fmt.Sprintf("%g", f), Valid: true}
}

func numericStr(f float64) string {
	return fmt.Sprintf("%g", f)
}

func serializeFuelExpenseRow(fe repository.ListFuelExpensesRow, ex repository.FuelExpenseExtra) map[string]any {
	var kwh, pricePerKwh float64
	if fe.FuelCategoryName == repository.FuelCategoryLISTRIK {
		if fe.Quantity.Valid {
			kwh = util.ParseStringToFloat64(fe.Quantity.String)
		}
		if fe.PricePerUnit.Valid {
			pricePerKwh = util.ParseStringToFloat64(fe.PricePerUnit.String)
		}
	}

	var liter, pricePerLiter float64
	if fe.FuelCategoryName != repository.FuelCategoryLISTRIK {
		if fe.Quantity.Valid {
			liter = util.ParseStringToFloat64(fe.Quantity.String)
		}
		if fe.PricePerUnit.Valid {
			pricePerLiter = util.ParseStringToFloat64(fe.PricePerUnit.String)
		}
	}

	totalCost := 0.0
	if fe.TotalCost.Valid {
		totalCost = util.ParseStringToFloat64(fe.TotalCost.String)
	}
	var batteryBefore, batteryAfter any
	if fe.BatteryBefore.Valid {
		batteryBefore = util.ParseStringToFloat64(fe.BatteryBefore.String)
	}
	if fe.BatteryAfter.Valid {
		batteryAfter = util.ParseStringToFloat64(fe.BatteryAfter.String)
	}

	source := ex.Source
	if source == "" {
		source = "DIRECT"
	}
	status := "ACTIVE"
	if ex.VoidedAt.Valid {
		status = "VOID"
	}

	return map[string]any{
		"id":        fe.ID,
		"vehicleId": fe.VehicleId,
		"vehicle": map[string]any{
			"id":          fe.VehicleId,
			"name":        fe.VehicleName,
			"plateNumber": fe.PlateNumber,
		},
		"fuelTypeId":       fe.FuelTypeId,
		"fuelType":         fe.FuelCategoryName,
		"fuelGrade":        fe.FuelGrade.String,
		"bookingId":        fe.BookingId.Int32,
		"driverId":         fe.DriverId.Int32,
		"driverName":       fe.DriverName.String,
		"recordedById":     fe.RecordedById,
		"odometerBefore":   fe.OdometerBefore.Int32,
		"odometerAfter":    fe.OdometerAfter.Int32,
		"distanceKm":       fe.DistanceKm.Int32,
		"liter":            liter,
		"pricePerLiter":    pricePerLiter,
		"kwh":              kwh,
		"pricePerKwh":      pricePerKwh,
		"totalCost":        totalCost,
		"batteryBefore":    batteryBefore,
		"batteryAfter":     batteryAfter,
		"meterStartKwh":    nullFloat(ex.MeterStartKwh),
		"meterEndKwh":      nullFloat(ex.MeterEndKwh),
		"quantitySource":   nullStr(ex.QuantitySource),
		"proofPhotoUrl":    fe.ProofPhotoUrl.String,
		"note":             fe.Note.String,
		"createdAt":        fe.CreatedAt,
		"source":           source,
		"voucherId":        nullInt(ex.VoucherID),
		"voucherCode":      nullStr(ex.VoucherCode),
		"stationId":        nullInt(ex.StationID),
		"stationName":      nullStr(ex.StationName),
		"isPartnerStation": ex.IsPartner,
		"reason":           nullStr(ex.Reason),
		"status":           status,
		"voidedAt":         nullTime(ex.VoidedAt),
		"voidedByName":     nullStr(ex.VoidedByName),
		"voidReason":       nullStr(ex.VoidReason),
		"ledger": map[string]any{
			"accrued":      nullFloat(ex.LedgerAccrued),
			"debit":        nullFloat(ex.LedgerDebit),
			"balanceAfter": nullFloat(ex.LedgerBalance),
			"kmPerUnit":    nullFloat(ex.KmPerUnit),
		},
	}
}

func (s *FuelExpenseService) List(ctx context.Context, page, limit int, driverID, vehicleID *int32, fuelType *string, bookingID *int32, actorID int32, role string, sortBy, sortOrder string) ([]map[string]any, int64, error) {
	if role == "DRIVER" {
		driver, err := s.q.GetDriverByUserID(ctx, actorID)
		if err == nil {
			driverID = &driver.ID
		}
	}

	params := repository.ListFuelExpensesParams{
		Limit:     int32(limit),
		Offset:    int32((page - 1) * limit),
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}
	if driverID != nil {
		params.DriverID = sql.NullInt32{Int32: *driverID, Valid: true}
	}
	if vehicleID != nil {
		params.VehicleID = sql.NullInt32{Int32: *vehicleID, Valid: true}
	}
	if fuelType != nil {
		params.FuelCategory = repository.NullFuelCategory{FuelCategory: repository.FuelCategory(*fuelType), Valid: true}
	}
	if bookingID != nil {
		params.BookingID = sql.NullInt32{Int32: *bookingID, Valid: true}
	}

	rows, err := s.q.ListFuelExpenses(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	total, _ := s.q.CountFuelExpenses(ctx, repository.CountFuelExpensesParams{
		DriverID: params.DriverID, VehicleID: params.VehicleID, FuelCategory: params.FuelCategory,
	})
	ids := make([]int32, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	extras, err := s.q.GetFuelExpenseExtras(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		out[i] = serializeFuelExpenseRow(r, extras[r.ID])
	}
	return out, total, nil
}

func (s *FuelExpenseService) GetByID(ctx context.Context, id int32) (map[string]any, error) {
	fe, err := s.q.GetFuelExpenseByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	extras, err := s.q.GetFuelExpenseExtras(ctx, []int32{id})
	if err != nil {
		return nil, err
	}
	return serializeFuelExpenseRow(repository.ListFuelExpensesRow(fe), extras[id]), nil
}

type fillQuantity struct {
	Quantity      float64
	Price         float64
	Source        string // INPUT | METER | ESTIMATE
	MeterStart    sql.NullFloat64
	MeterEnd      sql.NullFloat64
	BatteryBefore sql.NullString
	BatteryAfter  sql.NullString
}

// resolveQuantity: jumlah & harga per unit. Listrik menerima kWh langsung,
// selisih meter charger, atau estimasi dari % baterai (keputusan: "bebas").
func resolveQuantity(req CreateFuelExpenseRequest, energy repository.FuelCategory, p repository.VehicleFuelProfile) (fillQuantity, error) {
	bad := func(msg string) (fillQuantity, error) {
		return fillQuantity{}, util.NewError(400, msg, util.ErrBadRequest)
	}
	var out fillQuantity
	if req.BatteryBefore != 0 || req.BatteryAfter != 0 {
		if req.BatteryBefore < 0 || req.BatteryBefore > 100 || req.BatteryAfter < 0 || req.BatteryAfter > 100 {
			return bad("persentase baterai harus 0-100")
		}
		out.BatteryBefore = numericFromFloat(req.BatteryBefore)
		out.BatteryAfter = numericFromFloat(req.BatteryAfter)
	}
	if energy != repository.FuelCategoryLISTRIK {
		if req.Liter <= 0 {
			return bad("jumlah liter wajib diisi")
		}
		out.Quantity, out.Price, out.Source = req.Liter, req.PricePerLiter, "INPUT"
		return out, nil
	}
	out.Price = req.PricePerKwh
	if req.MeterStartKwh != 0 || req.MeterEndKwh != 0 {
		out.MeterStart = sql.NullFloat64{Float64: req.MeterStartKwh, Valid: true}
		out.MeterEnd = sql.NullFloat64{Float64: req.MeterEndKwh, Valid: true}
	}
	switch {
	case req.Kwh > 0:
		out.Quantity, out.Source = req.Kwh, "INPUT"
	case req.MeterEndKwh > req.MeterStartKwh && req.MeterStartKwh >= 0:
		out.Quantity, out.Source = round2(req.MeterEndKwh-req.MeterStartKwh), "METER"
	case req.BatteryAfter > req.BatteryBefore:
		if !p.BatteryCapacityKwh.Valid {
			return bad("estimasi dari % baterai butuh kapasitas baterai kendaraan - isi kWh atau angka meter charger")
		}
		out.Quantity = round2((req.BatteryAfter - req.BatteryBefore) / 100 * p.BatteryCapacityKwh.Float64 / chargerEfficiency)
		out.Source = "ESTIMATE"
	default:
		return bad("isi jumlah kWh, angka meter charger (awal & akhir), atau % baterai sebelum & sesudah")
	}
	if out.Quantity <= 0 {
		return bad("jumlah kWh harus lebih dari 0")
	}
	return out, nil
}

func (s *FuelExpenseService) Create(ctx context.Context, req CreateFuelExpenseRequest, recordedByID int32, driverID *int32, actor AuditActor) (map[string]any, error) {
	odometer := req.Odometer
	if odometer == 0 {
		odometer = req.OdometerAfter
	}
	if odometer <= 0 {
		return nil, util.NewError(400, "odometer saat pengisian wajib diisi", util.ErrBadRequest)
	}
	// Jenis bahan bakar harus sesuai sumber energi kendaraan (FL-04): BBM ↔ BBM,
	// LISTRIK ↔ LISTRIK, HYBRID boleh keduanya. UI sudah mengunci; API ikut.
	fuelType, ftErr := s.q.GetFuelTypeByID(ctx, req.FuelTypeID)
	if ftErr != nil {
		return nil, util.NewError(400, "jenis bahan bakar tidak ditemukan", util.ErrBadRequest)
	}
	energy := fuelType.Type
	reason := strings.ToUpper(strings.TrimSpace(req.Reason))
	if reason != "" && !fuelFillReasons[reason] {
		return nil, util.NewError(400, "alasan pengisian tidak dikenal", util.ErrBadRequest)
	}
	var station sql.NullInt32
	stationName := optStr(req.StationName)
	if req.StationID > 0 {
		st, err := s.q.GetFuelStation(ctx, req.StationID)
		if err != nil || !st.IsActive {
			return nil, util.NewError(400, "SPBU mitra tidak ditemukan / tidak aktif", util.ErrBadRequest)
		}
		station = sql.NullInt32{Int32: st.ID, Valid: true}
		stationName = sql.NullString{}
	}

	// driverId mereferensi drivers.id (BUKAN users.id). Resolve dari user yang login.
	// Jika user bukan driver (mis. admin), driverId dibiarkan NULL.
	var dID sql.NullInt32
	if driver, derr := s.q.GetDriverByUserID(ctx, recordedByID); derr == nil {
		dID = sql.NullInt32{Int32: driver.ID, Valid: true}
	}
	var bookingID sql.NullInt32
	if req.BookingID != nil {
		bookingID = sql.NullInt32{Int32: *req.BookingID, Valid: true}
	}

	var feID int32
	var qty fillQuantity
	var led ledgerResult
	var tracked bool
	var prof repository.VehicleFuelProfile
	err := withTx(ctx, s.db, func(q *repository.Queries) error {
		p, err := q.LockVehicleFuelProfile(ctx, req.VehicleID)
		if err != nil {
			return util.NewError(404, "Vehicle not found", util.ErrNotFound)
		}
		prof = p
		if p.EnergyType != repository.EnergyTypeHYBRID && string(p.EnergyType) != string(energy) {
			return util.NewError(400,
				fmt.Sprintf("kendaraan ini berenergi %s - jenis \"%s\" (%s) tidak sesuai", p.EnergyType, fuelType.Name, fuelType.Type),
				util.ErrBadRequest)
		}
		// Odometer harus data faktual kendaraan yang sama dipakai di mana-mana
		// (start trip, laporan pengembalian, maintenance) - tidak boleh mundur.
		if odometer < p.CurrentOdometer {
			return util.NewError(400,
				fmt.Sprintf("odometer tidak boleh kurang dari catatan kendaraan saat ini (%d km)", p.CurrentOdometer),
				util.ErrBadRequest)
		}
		qty, err = resolveQuantity(req, energy, p)
		if err != nil {
			return err
		}
		// Satu master harga: bila klien tidak mengirim harga, pakai fuel_types.default_price.
		if qty.Price <= 0 && fuelType.DefaultPrice.Valid {
			qty.Price = util.ParseStringToFloat64(fuelType.DefaultPrice.String)
		}
		if qty.Price <= 0 {
			return util.NewError(400, "harga per unit belum diatur", util.ErrBadRequest)
		}
		st, err := readLedgerState(ctx, q, p, energy)
		if err != nil {
			return err
		}
		fe, err := q.CreateFuelExpense(ctx, repository.CreateFuelExpenseParams{
			VehicleId:      req.VehicleID,
			FuelTypeId:     req.FuelTypeID,
			BookingId:      bookingID,
			DriverId:       dID,
			RecordedById:   recordedByID,
			FuelGrade:      sql.NullString{String: req.FuelGrade, Valid: req.FuelGrade != ""},
			ProofPhotoUrl:  sql.NullString{String: req.ProofPhotoUrl, Valid: req.ProofPhotoUrl != ""},
			OdometerBefore: sql.NullInt32{Int32: st.CheckpointOdometer, Valid: true},
			OdometerAfter:  sql.NullInt32{Int32: odometer, Valid: true},
			DistanceKm:     sql.NullInt32{Int32: max(odometer-st.CheckpointOdometer, 0), Valid: true},
			Quantity:       numericFromFloat(qty.Quantity),
			PricePerUnit:   numericFromFloat(qty.Price),
			TotalCost:      numericFromFloat(round2(qty.Quantity * qty.Price)),
			BatteryBefore:  qty.BatteryBefore,
			BatteryAfter:   qty.BatteryAfter,
			Location:       "Uploaded via API",
			Note:           sql.NullString{String: req.Note, Valid: req.Note != ""},
		})
		if err != nil {
			return err
		}
		feID = fe.ID
		if err := q.SetFuelExpenseExtra(ctx, repository.SetFuelExpenseExtraParams{
			ID: fe.ID, Source: "DIRECT", StationID: station, StationName: stationName,
			Reason:        sql.NullString{String: reason, Valid: reason != ""},
			MeterStartKwh: qty.MeterStart, MeterEndKwh: qty.MeterEnd,
			QuantitySource: sql.NullString{String: qty.Source, Valid: qty.Source != ""},
		}); err != nil {
			return err
		}
		// Tanpa km per unit, pengisian tidak dihitung ke saldo (hanya menggeser
		// titik hitung) - supaya saldo tidak minus menumpuk sebelum diatur.
		tracked = kmPerUnit(p, energy).Valid
		debit := 0.0
		note := ""
		if tracked {
			debit = qty.Quantity
		} else {
			note = "km per unit belum diatur - tidak dihitung ke saldo"
		}
		led, err = appendLedger(ctx, q, p, energy, ledgerEvent{
			EntryType:     ledgerDirectFill,
			Odometer:      sql.NullInt32{Int32: odometer, Valid: true},
			Debit:         debit,
			FuelExpenseID: sql.NullInt32{Int32: fe.ID, Valid: true},
			CreatedByID:   recordedByID,
			Note:          note,
		})
		if err != nil {
			return err
		}
		_, err = q.UpdateVehicleOdometer(ctx, repository.UpdateVehicleOdometerParams{
			ID: req.VehicleID, CurrentOdometer: odometer,
		})
		return err
	})
	if err != nil {
		return nil, err
	}

	logAudit(ctx, s.q, actor, "CREATE", "FuelExpense", feID,
		"Mencatat pengisian BBM/listrik kendaraan (langsung)")

	out, err := s.GetByID(ctx, feID)
	if err != nil {
		return nil, err
	}
	var warnings []string
	if tracked {
		entitled := round2(led.BalanceFrom + led.Accrued)
		if qty.Quantity > entitled+0.01 {
			warnings = append(warnings, fmt.Sprintf(
				"Pengisian %.2f %s melebihi hak saldo %.2f %s - saldo menjadi %.2f dan voucher berikutnya berkurang",
				qty.Quantity, unitFor(energy), entitled, unitFor(energy), led.BalanceAfter))
		}
	}
	if capacity := capacityFor(prof, energy); capacity.Valid && energy == repository.FuelCategoryBBM && qty.Quantity > capacity.Float64 {
		warnings = append(warnings, fmt.Sprintf("Pengisian %.2f L melebihi kapasitas tangki %.2f L", qty.Quantity, capacity.Float64))
	}
	out["warnings"] = warnings
	return out, nil
}

// Void membatalkan catatan pengisian langsung (pengganti hapus permanen, FL-06):
// data tetap ada (dicoret), liter kembali ke saldo. odometerTypo=true juga
// membatalkan hak dari odometer catatan ini dan mengembalikan odometer
// kendaraan ke bacaan sah sebelumnya - hanya untuk catatan TERAKHIR.
func (s *FuelExpenseService) Void(ctx context.Context, id int32, reason string, odometerTypo bool, actor AuditActor) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return util.NewError(400, "alasan pembatalan wajib diisi", util.ErrBadRequest)
	}
	fe, err := s.q.GetFuelExpenseByID(ctx, id)
	if err != nil {
		return util.ErrNotFound
	}
	extras, err := s.q.GetFuelExpenseExtras(ctx, []int32{id})
	if err != nil {
		return err
	}
	ex := extras[id]
	if ex.VoidedAt.Valid {
		return util.NewError(409, "catatan ini sudah dibatalkan", util.ErrConflict)
	}
	if ex.Source == "VOUCHER" {
		return util.NewError(409, "catatan ini berasal dari voucher - batalkan lewat menu Voucher", util.ErrConflict)
	}
	var restoredOdo int32 = -1
	err = withTx(ctx, s.db, func(q *repository.Queries) error {
		p, err := q.LockVehicleFuelProfile(ctx, fe.VehicleId)
		if err != nil {
			return err
		}
		entry, lerr := q.GetFuelLedgerEntryByExpense(ctx, id)
		hasLedger := lerr == nil
		if lerr != nil && !errors.Is(lerr, sql.ErrNoRows) {
			return lerr
		}
		if odometerTypo {
			if !fe.OdometerAfter.Valid || fe.OdometerAfter.Int32 != p.CurrentOdometer {
				return util.NewError(409,
					"odometer kendaraan sudah bergeser oleh catatan lain - koreksi odometer hanya untuk catatan terakhir",
					util.ErrConflict)
			}
			if hasLedger {
				cp, err := q.GetLastFuelCheckpoint(ctx, p.VehicleID, entry.Energy)
				if err != nil || cp.ID != entry.ID {
					return util.NewError(409,
						"sudah ada kejadian saldo sesudah catatan ini - gunakan Penyesuaian Saldo",
						util.ErrConflict)
				}
			} else if n, _ := q.CountFuelLedgerEntries(ctx, p.VehicleID); n > 0 {
				return util.NewError(409,
					"sudah ada kejadian saldo sesudah catatan ini - gunakan Penyesuaian Saldo",
					util.ErrConflict)
			}
		}
		if hasLedger {
			ev := ledgerEvent{
				EntryType:     ledgerVoid,
				Debit:         -entry.Debit,
				FuelExpenseID: sql.NullInt32{Int32: id, Valid: true},
				ReversesID:    sql.NullInt32{Int32: entry.ID, Valid: true},
				CreatedByID:   actor.UserID,
				Note:          "Pembatalan: " + reason,
			}
			if odometerTypo {
				// Titik hitung kembali ke kejadian sebelumnya, hak ikut dibatalkan.
				prev := baselineOdometer(p)
				if before, err := q.GetFuelCheckpointBefore(ctx, p.VehicleID, entry.Energy, entry.ID); err == nil && before.Odometer.Valid {
					prev = before.Odometer.Int32
				}
				neg := -entry.Accrued
				ev.Accrued = &neg
				ev.Odometer = sql.NullInt32{Int32: prev, Valid: true}
				ev.Note = "Pembatalan (odometer salah ketik): " + reason
			}
			if _, err := appendLedger(ctx, q, p, entry.Energy, ev); err != nil {
				return err
			}
		}
		if err := q.VoidFuelExpense(ctx, id, actor.UserID, reason); err != nil {
			return err
		}
		if odometerTypo {
			newOdo, err := q.MaxVehicleOdometerReading(ctx, p.VehicleID, id, hasLedger)
			if err != nil {
				return err
			}
			if vo, err := q.MaxFuelVoucherOdometer(ctx, p.VehicleID); err == nil && vo > newOdo {
				newOdo = vo
			}
			if err := q.SetVehicleOdometerExact(ctx, p.VehicleID, newOdo); err != nil {
				return err
			}
			if !hasLedger {
				// Belum ada kejadian saldo: titik awal ikut dikoreksi.
				if err := q.UpdateVehicleFuelProfile(ctx, repository.UpdateVehicleFuelProfileParams{
					VehicleID: p.VehicleID, KmPerLiter: p.KmPerLiter, TankCapacityLiter: p.TankCapacityLiter,
					KmPerKwh: p.KmPerKwh, BatteryCapacityKwh: p.BatteryCapacityKwh,
					FuelBaselineOdometer: sql.NullInt32{Int32: newOdo, Valid: true},
				}); err != nil {
					return err
				}
			}
			restoredOdo = newOdo
		}
		return nil
	})
	if err != nil {
		return err
	}
	desc := "Membatalkan catatan pengisian BBM/listrik kendaraan " + fe.PlateNumber + ": " + reason
	if restoredOdo >= 0 {
		desc += fmt.Sprintf(" (odometer dikoreksi ke %d km)", restoredOdo)
	}
	logAudit(ctx, s.q, actor, "VOID", "FuelExpense", id, desc)
	return nil
}

// Delete dipertahankan untuk klien lama - kini membatalkan (soft), bukan menghapus.
func (s *FuelExpenseService) Delete(ctx context.Context, id int32, actor AuditActor) error {
	return s.Void(ctx, id, "Dihapus admin", false, actor)
}
