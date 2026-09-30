package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

// Voucher BBM untuk SPBU mitra (docs/RANCANGAN_VOUCHER_BBM.md §3.1–3.3, §3.8).
// Liter voucher langsung memotong saldo saat terbit; kedaluwarsa/dibatalkan
// → liter kembali. Voucher dianggap terpakai penuh (nominal tetap).

const (
	voucherIssued    = "ISSUED"
	voucherUsed      = "USED"
	voucherExpired   = "EXPIRED"
	voucherCancelled = "CANCELLED"

	settingVoucherValidityDays = "fuel_voucher_validity_days"
)

type FuelVoucherRequest struct {
	VehicleID  int32  `json:"vehicleId"`
	FuelTypeID int32  `json:"fuelTypeId"`
	StationID  int32  `json:"stationId"`
	DriverID   *int32 `json:"driverId"`
	BookingID  *int32 `json:"bookingId"`
	Odometer   int32  `json:"odometer"`
	Note       string `json:"note"`
}

// voucherCalc: hasil hitung voucher (dipakai pratinjau & terbit).
type voucherCalc struct {
	Profile       repository.VehicleFuelProfile
	FuelType      repository.FuelType
	Station       repository.FuelStation
	Odometer      int32
	Checkpoint    int32
	DistanceKm    int32
	KmPerLiter    float64
	Accrued       float64
	Carried       float64
	Available     float64
	Tank          float64
	Liter         float64
	PricePerLiter float64
	Amount        float64
	ValidUntil    time.Time
	CappedByTank  bool
}

func (c voucherCalc) toMap() map[string]any {
	return map[string]any{
		"vehicleId":          c.Profile.VehicleID,
		"vehicleName":        c.Profile.VehicleName,
		"plateNumber":        c.Profile.PlateNumber,
		"fuelTypeId":         c.FuelType.ID,
		"fuelTypeName":       c.FuelType.Name,
		"stationId":          c.Station.ID,
		"stationName":        c.Station.Name,
		"odometer":           c.Odometer,
		"checkpointOdometer": c.Checkpoint,
		"distanceKm":         c.DistanceKm,
		"kmPerLiter":         c.KmPerLiter,
		"accruedLiter":       c.Accrued,
		"carriedLiter":       c.Carried,
		"availableLiter":     c.Available,
		"tankCapacityLiter":  c.Tank,
		"liter":              c.Liter,
		"remainingLiter":     round2(c.Available - c.Liter),
		"cappedByTank":       c.CappedByTank,
		"pricePerLiter":      c.PricePerLiter,
		"amount":             c.Amount,
		"validUntil":         c.ValidUntil,
	}
}

func (s *FuelLedgerService) validityDays(ctx context.Context, q *repository.Queries) int {
	m, err := q.GetMasterSettingByKey(ctx, settingVoucherValidityDays)
	if err != nil {
		return 1
	}
	d := int(util.ParseStringToFloat64(m.Value))
	if d < 1 {
		return 1
	}
	if d > 31 {
		return 31
	}
	return d
}

// voucherValidUntil: 23:59:59 WIB pada hari ke-N (1 = hari terbit).
func voucherValidUntil(now time.Time, days int) time.Time {
	return util.StartOfDayWIB(now).AddDate(0, 0, days).Add(-time.Second)
}

func (s *FuelLedgerService) calcVoucher(ctx context.Context, q *repository.Queries, p repository.VehicleFuelProfile, req FuelVoucherRequest) (voucherCalc, error) {
	c := voucherCalc{Profile: p}
	bad := func(msg string) (voucherCalc, error) { return c, util.NewError(400, msg, util.ErrBadRequest) }

	if p.EnergyType == repository.EnergyTypeLISTRIK {
		return bad("voucher hanya untuk kendaraan BBM/hybrid - kendaraan listrik diisi di kantor")
	}
	if !p.KmPerLiter.Valid {
		return bad("konsumsi km/liter kendaraan ini belum diatur")
	}
	if !p.TankCapacityLiter.Valid {
		return bad("kapasitas tangki kendaraan ini belum diatur")
	}
	ft, err := q.GetFuelTypeByID(ctx, req.FuelTypeID)
	if err != nil || !ft.IsActive {
		return bad("jenis BBM tidak ditemukan / tidak aktif")
	}
	if ft.Type != repository.FuelCategoryBBM {
		return bad("voucher hanya untuk jenis BBM (bukan listrik)")
	}
	c.FuelType = ft
	c.PricePerLiter = util.ParseStringToFloat64(ft.DefaultPrice.String)
	if !ft.DefaultPrice.Valid || c.PricePerLiter <= 0 {
		return bad(fmt.Sprintf("harga %s belum diatur di Pengaturan → Jenis Bahan Bakar", ft.Name))
	}
	st, err := q.GetFuelStation(ctx, req.StationID)
	if err != nil || !st.IsActive {
		return bad("SPBU mitra tidak ditemukan / tidak aktif")
	}
	c.Station = st

	c.Odometer = req.Odometer
	if c.Odometer == 0 {
		c.Odometer = p.CurrentOdometer
	}
	if c.Odometer < p.CurrentOdometer {
		return bad(fmt.Sprintf("odometer tidak boleh kurang dari catatan kendaraan saat ini (%d km)", p.CurrentOdometer))
	}
	state, err := readLedgerState(ctx, q, p, repository.FuelCategoryBBM)
	if err != nil {
		return c, err
	}
	c.Checkpoint = state.CheckpointOdometer
	c.DistanceKm = c.Odometer - state.CheckpointOdometer
	if c.DistanceKm < 0 {
		c.DistanceKm = 0
	}
	c.KmPerLiter = p.KmPerLiter.Float64
	c.Accrued = accrue(c.DistanceKm, p.KmPerLiter)
	c.Carried = round2(state.Balance)
	c.Available = round2(c.Carried + c.Accrued)
	c.Tank = p.TankCapacityLiter.Float64
	c.Liter = c.Available
	if c.Liter > c.Tank {
		c.Liter = c.Tank
		c.CappedByTank = true
	}
	c.Liter = round2(c.Liter)
	// Nominal tidak dibulatkan (keputusan pemilik 2026-10-01).
	c.Amount = round2(c.Liter * c.PricePerLiter)
	c.ValidUntil = voucherValidUntil(time.Now(), s.validityDays(ctx, q))
	return c, nil
}

func (s *FuelLedgerService) PreviewVoucher(ctx context.Context, req FuelVoucherRequest) (map[string]any, error) {
	p, err := s.q.GetVehicleFuelProfile(ctx, req.VehicleID)
	if err != nil {
		return nil, util.NewError(404, "Kendaraan tidak ditemukan", util.ErrNotFound)
	}
	c, err := s.calcVoucher(ctx, s.q, p, req)
	if err != nil {
		return nil, err
	}
	out := c.toMap()
	if id, err := s.q.GetActiveFuelVoucherID(ctx, p.VehicleID); err == nil {
		out["activeVoucherId"] = id
	}
	return out, nil
}

const voucherAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func newVoucherCode(now time.Time) string {
	var b strings.Builder
	for i := 0; i < 5; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(voucherAlphabet))))
		b.WriteByte(voucherAlphabet[n.Int64()])
	}
	return "VB" + now.In(util.WIB).Format("060102") + "-" + b.String()
}

func (s *FuelLedgerService) IssueVoucher(ctx context.Context, req FuelVoucherRequest, actor AuditActor) (map[string]any, error) {
	var voucherID int32
	var calc voucherCalc
	err := withTx(ctx, s.db, func(q *repository.Queries) error {
		p, err := q.LockVehicleFuelProfile(ctx, req.VehicleID)
		if err != nil {
			return util.NewError(404, "Kendaraan tidak ditemukan", util.ErrNotFound)
		}
		if id, err := q.GetActiveFuelVoucherID(ctx, p.VehicleID); err == nil {
			return util.NewError(409,
				fmt.Sprintf("kendaraan ini masih punya voucher aktif (#%d) - batalkan atau tunggu dipakai/kedaluwarsa dulu", id),
				util.ErrConflict)
		}
		calc, err = s.calcVoucher(ctx, q, p, req)
		if err != nil {
			return err
		}
		if calc.Liter <= 0 {
			return util.NewError(400,
				fmt.Sprintf("saldo BBM belum cukup untuk voucher (tersedia %.2f L)", calc.Available),
				util.ErrBadRequest)
		}
		driverID := p.FixedDriverID
		if req.DriverID != nil {
			driverID = sql.NullInt32{Int32: *req.DriverID, Valid: *req.DriverID > 0}
		}
		var bookingID sql.NullInt32
		if req.BookingID != nil && *req.BookingID > 0 {
			bookingID = sql.NullInt32{Int32: *req.BookingID, Valid: true}
		}
		// Kode 5 karakter acak per hari (32^5 kombinasi) - bentrok praktis mustahil;
		// bila terjadi, UNIQUE(code) menolak dan admin cukup menerbitkan ulang.
		voucherID, err = q.CreateFuelVoucher(ctx, repository.CreateFuelVoucherParams{
			Code: newVoucherCode(time.Now()), VehicleID: p.VehicleID, FuelTypeID: calc.FuelType.ID,
			StationID: calc.Station.ID, DriverID: driverID, BookingID: bookingID,
			IssuedByID: actor.UserID, Odometer: calc.Odometer, DistanceKm: calc.DistanceKm,
			KmPerLiter: calc.KmPerLiter, AccruedLiter: calc.Accrued, CarriedLiter: calc.Carried,
			TankCapacityLiter: p.TankCapacityLiter, Liter: calc.Liter,
			PricePerLiter: calc.PricePerLiter, Amount: calc.Amount, ValidUntil: calc.ValidUntil,
			Note: optStr(req.Note),
		})
		if err != nil {
			if strings.Contains(err.Error(), "uq_fuel_vouchers_one_active") {
				return util.NewError(409, "kendaraan ini masih punya voucher aktif", util.ErrConflict)
			}
			if strings.Contains(err.Error(), "violates foreign key") {
				return util.NewError(400, "driver/booking tidak ditemukan", util.ErrBadRequest)
			}
			return err
		}
		if _, err := appendLedger(ctx, q, p, repository.FuelCategoryBBM, ledgerEvent{
			EntryType:   ledgerVoucher,
			Odometer:    sql.NullInt32{Int32: calc.Odometer, Valid: true},
			Debit:       calc.Liter,
			VoucherID:   sql.NullInt32{Int32: voucherID, Valid: true},
			CreatedByID: actor.UserID,
		}); err != nil {
			return err
		}
		_, err = q.UpdateVehicleOdometer(ctx, repository.UpdateVehicleOdometerParams{ID: p.VehicleID, CurrentOdometer: calc.Odometer})
		return err
	})
	if err != nil {
		return nil, err
	}
	logAudit(ctx, s.q, actor, "CREATE", "FuelVoucher", voucherID,
		fmt.Sprintf("Menerbitkan voucher BBM %s %.2f L (Rp %.2f) untuk %s di %s",
			calc.FuelType.Name, calc.Liter, calc.Amount, calc.Profile.PlateNumber, calc.Station.Name))
	return s.GetVoucher(ctx, voucherID, actor.UserID, "ADMIN")
}

func serializeVoucher(v repository.FuelVoucher) map[string]any {
	return map[string]any{
		"id": v.ID, "code": v.Code,
		"vehicleId": v.VehicleID, "vehicleName": v.VehicleName, "plateNumber": v.PlateNumber,
		"fuelTypeId": v.FuelTypeID, "fuelTypeName": v.FuelTypeName,
		"stationId": v.StationID, "stationName": v.StationName, "stationAddress": nullStr(v.StationAddress),
		"driverId": nullInt(v.DriverID), "driverName": nullStr(v.DriverName), "driverUserId": nullInt(v.DriverUserID),
		"bookingId": nullInt(v.BookingID), "issuedById": v.IssuedByID, "issuedByName": v.IssuedByName,
		"odometer": v.Odometer, "distanceKm": v.DistanceKm, "kmPerLiter": v.KmPerLiter,
		"accruedLiter": v.AccruedLiter, "carriedLiter": v.CarriedLiter,
		"tankCapacityLiter": nullFloat(v.TankCapacityLiter),
		"liter":             v.Liter, "pricePerLiter": v.PricePerLiter, "amount": v.Amount,
		"validUntil": v.ValidUntil, "status": v.Status,
		"usedAt": nullTime(v.UsedAt), "usedByName": nullStr(v.UsedByName), "usedOdometer": nullInt(v.UsedOdometer),
		"receiptPhotoUrl": nullStr(v.ReceiptPhotoUrl), "fuelExpenseId": nullInt(v.FuelExpenseID),
		"cancelledAt": nullTime(v.CancelledAt), "cancelledByName": nullStr(v.CancelledByName),
		"cancelReason": nullStr(v.CancelReason),
		"reconciledAt": nullTime(v.ReconciledAt), "reconciledByName": nullStr(v.ReconciledByName),
		"invoiceNumber": nullStr(v.InvoiceNumber), "note": nullStr(v.Note), "createdAt": v.CreatedAt,
	}
}

// driverIDForUser: drivers.id milik user DRIVER (0 bila bukan driver).
func (s *FuelLedgerService) driverIDForUser(ctx context.Context, userID int32) int32 {
	if d, err := s.q.GetDriverByUserID(ctx, userID); err == nil {
		return d.ID
	}
	return 0
}

func (s *FuelLedgerService) canSeeVoucher(ctx context.Context, v repository.FuelVoucher, userID int32, role string) bool {
	if role == "ADMIN" {
		return true
	}
	return role == "DRIVER" && v.DriverUserID.Valid && v.DriverUserID.Int32 == userID
}

func (s *FuelLedgerService) GetVoucher(ctx context.Context, id, userID int32, role string) (map[string]any, error) {
	v, err := s.q.GetFuelVoucher(ctx, id)
	if err != nil {
		return nil, util.NewError(404, "Voucher tidak ditemukan", util.ErrNotFound)
	}
	if !s.canSeeVoucher(ctx, v, userID, role) {
		return nil, util.NewError(404, "Voucher tidak ditemukan", util.ErrNotFound)
	}
	return serializeVoucher(v), nil
}

type VoucherListFilter struct {
	Status     *string
	VehicleID  *int32
	StationID  *int32
	From       *time.Time
	To         *time.Time
	Reconciled *bool
	Search     *string
}

func (s *FuelLedgerService) ListVouchers(ctx context.Context, f VoucherListFilter, page, limit int, userID int32, role string) ([]map[string]any, int64, []repository.FuelVoucherSummary, error) {
	s.ExpireDue(ctx)
	arg := repository.ListFuelVouchersParams{Limit: int32(limit), Offset: int32((page - 1) * limit)}
	if f.Status != nil {
		arg.Status = sql.NullString{String: *f.Status, Valid: true}
	}
	if f.VehicleID != nil {
		arg.VehicleID = sql.NullInt32{Int32: *f.VehicleID, Valid: true}
	}
	if f.StationID != nil {
		arg.StationID = sql.NullInt32{Int32: *f.StationID, Valid: true}
	}
	if f.From != nil {
		arg.From = sql.NullTime{Time: *f.From, Valid: true}
	}
	if f.To != nil {
		arg.To = sql.NullTime{Time: *f.To, Valid: true}
	}
	if f.Reconciled != nil {
		arg.Reconciled = sql.NullBool{Bool: *f.Reconciled, Valid: true}
	}
	if f.Search != nil && strings.TrimSpace(*f.Search) != "" {
		arg.Search = sql.NullString{String: strings.TrimSpace(*f.Search), Valid: true}
	}
	if role != "ADMIN" {
		// Driver hanya melihat voucher yang ditujukan kepadanya.
		arg.DriverID = sql.NullInt32{Int32: s.driverIDForUser(ctx, userID), Valid: true}
	}
	rows, total, err := s.q.ListFuelVouchers(ctx, arg)
	if err != nil {
		return nil, 0, nil, err
	}
	summary, err := s.q.SummarizeFuelVouchers(ctx, arg)
	if err != nil {
		return nil, 0, nil, err
	}
	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		out[i] = serializeVoucher(r)
	}
	return out, total, summary, nil
}

type UseVoucherRequest struct {
	Odometer        int32
	ReceiptPhotoUrl string
	Note            string
}

// UseVoucher: driver penerima / admin menandai voucher sudah diisi. Voucher
// EXPIRED boleh ditandai terpakai oleh admin (tagihan mitra membuktikan).
func (s *FuelLedgerService) UseVoucher(ctx context.Context, id int32, req UseVoucherRequest, actor AuditActor, role string) (map[string]any, error) {
	v, err := s.q.GetFuelVoucher(ctx, id)
	if err != nil || !s.canSeeVoucher(ctx, v, actor.UserID, role) {
		return nil, util.NewError(404, "Voucher tidak ditemukan", util.ErrNotFound)
	}
	if role != "ADMIN" && req.ReceiptPhotoUrl == "" {
		return nil, util.NewError(400, "foto struk wajib", util.ErrBadRequest)
	}
	usedOdo := req.Odometer
	if usedOdo == 0 {
		usedOdo = v.Odometer
	}
	if usedOdo < v.Odometer {
		return nil, util.NewError(400,
			fmt.Sprintf("odometer saat isi tidak boleh kurang dari odometer voucher (%d km)", v.Odometer), util.ErrBadRequest)
	}
	reinstated := false
	err = withTx(ctx, s.db, func(q *repository.Queries) error {
		status, vehicleID, err := q.LockFuelVoucher(ctx, id)
		if err != nil {
			return err
		}
		p, err := q.LockVehicleFuelProfile(ctx, vehicleID)
		if err != nil {
			return err
		}
		switch status {
		case voucherIssued:
		case voucherExpired:
			if role != "ADMIN" {
				return util.NewError(409, "voucher sudah kedaluwarsa - hubungi admin", util.ErrConflict)
			}
			// Liter sudah kembali ke saldo saat kedaluwarsa → potong lagi.
			if _, err := appendLedger(ctx, q, p, repository.FuelCategoryBBM, ledgerEvent{
				EntryType: ledgerVoucherReinstate, Debit: v.Liter,
				VoucherID: sql.NullInt32{Int32: id, Valid: true}, CreatedByID: actor.UserID,
				Note: "Voucher kedaluwarsa ditandai terpakai (sesuai tagihan mitra)",
			}); err != nil {
				return err
			}
			reinstated = true
		default:
			return util.NewError(409, "voucher sudah "+strings.ToLower(status), util.ErrConflict)
		}
		note := "Voucher " + v.Code
		if strings.TrimSpace(req.Note) != "" {
			note += " - " + strings.TrimSpace(req.Note)
		}
		fe, err := q.CreateFuelExpense(ctx, repository.CreateFuelExpenseParams{
			VehicleId:      v.VehicleID,
			FuelTypeId:     v.FuelTypeID,
			BookingId:      v.BookingID,
			DriverId:       v.DriverID,
			RecordedById:   actor.UserID,
			ProofPhotoUrl:  sql.NullString{String: req.ReceiptPhotoUrl, Valid: req.ReceiptPhotoUrl != ""},
			OdometerBefore: sql.NullInt32{Int32: v.Odometer, Valid: true},
			OdometerAfter:  sql.NullInt32{Int32: usedOdo, Valid: true},
			DistanceKm:     sql.NullInt32{Int32: usedOdo - v.Odometer, Valid: true},
			Quantity:       numericFromFloat(v.Liter),
			PricePerUnit:   numericFromFloat(v.PricePerLiter),
			TotalCost:      numericFromFloat(v.Amount),
			Location:       v.StationName,
			Note:           sql.NullString{String: note, Valid: true},
		})
		if err != nil {
			return err
		}
		if err := q.SetFuelExpenseExtra(ctx, repository.SetFuelExpenseExtraParams{
			ID: fe.ID, Source: "VOUCHER",
			StationID: sql.NullInt32{Int32: v.StationID, Valid: true},
			VoucherID: sql.NullInt32{Int32: id, Valid: true},
		}); err != nil {
			return err
		}
		if err := q.MarkFuelVoucherUsed(ctx, id, actor.UserID, usedOdo,
			sql.NullString{String: req.ReceiptPhotoUrl, Valid: req.ReceiptPhotoUrl != ""}, fe.ID); err != nil {
			return err
		}
		_, err = q.UpdateVehicleOdometer(ctx, repository.UpdateVehicleOdometerParams{ID: p.VehicleID, CurrentOdometer: usedOdo})
		return err
	})
	if err != nil {
		return nil, err
	}
	desc := "Voucher BBM " + v.Code + " sudah diisi"
	if reinstated {
		desc = "Voucher BBM kedaluwarsa " + v.Code + " ditandai terpakai (saldo dipotong lagi)"
	}
	logAudit(ctx, s.q, actor, "USE", "FuelVoucher", id, desc)
	return s.GetVoucher(ctx, id, actor.UserID, role)
}

func (s *FuelLedgerService) CancelVoucher(ctx context.Context, id int32, reason string, actor AuditActor) (map[string]any, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, util.NewError(400, "alasan pembatalan wajib diisi", util.ErrBadRequest)
	}
	v, err := s.q.GetFuelVoucher(ctx, id)
	if err != nil {
		return nil, util.NewError(404, "Voucher tidak ditemukan", util.ErrNotFound)
	}
	err = withTx(ctx, s.db, func(q *repository.Queries) error {
		status, vehicleID, err := q.LockFuelVoucher(ctx, id)
		if err != nil {
			return err
		}
		p, err := q.LockVehicleFuelProfile(ctx, vehicleID)
		if err != nil {
			return err
		}
		switch status {
		case voucherIssued, voucherUsed:
			if _, err := appendLedger(ctx, q, p, repository.FuelCategoryBBM, ledgerEvent{
				EntryType: ledgerVoucherReturn, Debit: -v.Liter,
				VoucherID: sql.NullInt32{Int32: id, Valid: true}, CreatedByID: actor.UserID,
				Note: "Voucher dibatalkan: " + reason,
			}); err != nil {
				return err
			}
			if status == voucherUsed && v.FuelExpenseID.Valid {
				if err := q.VoidFuelExpense(ctx, v.FuelExpenseID.Int32, actor.UserID, "Voucher "+v.Code+" dibatalkan: "+reason); err != nil {
					return err
				}
			}
		case voucherExpired:
			// liter sudah kembali saat kedaluwarsa
		default:
			return util.NewError(409, "voucher sudah dibatalkan", util.ErrConflict)
		}
		if err := q.MarkFuelVoucherCancelled(ctx, id, actor.UserID, reason); err != nil {
			return err
		}
		return q.UnreconcileFuelVoucher(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	logAudit(ctx, s.q, actor, "CANCEL", "FuelVoucher", id, "Membatalkan voucher BBM "+v.Code+": "+reason)
	return s.GetVoucher(ctx, id, actor.UserID, "ADMIN")
}

type ReconcileRequest struct {
	IDs           []int32 `json:"ids"`
	InvoiceNumber string  `json:"invoiceNumber"`
}

func (s *FuelLedgerService) Reconcile(ctx context.Context, req ReconcileRequest, actor AuditActor) (int64, error) {
	req.InvoiceNumber = strings.TrimSpace(req.InvoiceNumber)
	if len(req.IDs) == 0 {
		return 0, util.NewError(400, "pilih minimal satu voucher", util.ErrBadRequest)
	}
	if req.InvoiceNumber == "" {
		return 0, util.NewError(400, "nomor tagihan mitra wajib diisi", util.ErrBadRequest)
	}
	n, err := s.q.ReconcileFuelVouchers(ctx, req.IDs, actor.UserID, req.InvoiceNumber)
	if err != nil {
		return 0, err
	}
	logAudit(ctx, s.q, actor, "RECONCILE", "FuelVoucher", req.IDs[0],
		fmt.Sprintf("Rekonsiliasi %d voucher BBM dengan tagihan %s", n, req.InvoiceNumber))
	return n, nil
}

// ExpireDue: voucher ISSUED yang lewat masa berlaku → EXPIRED, liter kembali
// ke saldo. Dipanggil sweeper berkala dan sebelum daftar voucher dibaca.
func (s *FuelLedgerService) ExpireDue(ctx context.Context) int {
	ids, err := s.q.ListExpiredIssuedVoucherIDs(ctx, time.Now())
	if err != nil || len(ids) == 0 {
		return 0
	}
	n := 0
	for _, id := range ids {
		var code string
		err := withTx(ctx, s.db, func(q *repository.Queries) error {
			status, vehicleID, err := q.LockFuelVoucher(ctx, id)
			if err != nil {
				return err
			}
			if status != voucherIssued {
				return errSkip
			}
			v, err := q.GetFuelVoucher(ctx, id)
			if err != nil {
				return err
			}
			code = v.Code
			p, err := q.LockVehicleFuelProfile(ctx, vehicleID)
			if err != nil {
				return err
			}
			if _, err := appendLedger(ctx, q, p, repository.FuelCategoryBBM, ledgerEvent{
				EntryType: ledgerVoucherReturn, Debit: -v.Liter,
				VoucherID: sql.NullInt32{Int32: id, Valid: true},
				Note:      "Voucher kedaluwarsa tanpa konfirmasi pengisian",
			}); err != nil {
				return err
			}
			return q.MarkFuelVoucherExpired(ctx, id)
		})
		if err == nil {
			n++
			logAudit(ctx, s.q, AuditActor{}, "EXPIRED", "FuelVoucher", id,
				"Voucher BBM "+code+" kedaluwarsa tanpa konfirmasi pengisian - liter kembali ke saldo")
		}
	}
	if n > 0 && s.publish != nil {
		s.publish(topicFuel, topicVehicle)
	}
	return n
}

var errSkip = errors.New("skip")

// RunVoucherSweeper menjalankan ExpireDue berkala sampai ctx selesai.
func (s *FuelLedgerService) RunVoucherSweeper(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.ExpireDue(ctx)
		}
	}
}
