package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

// TripExpenseService: biaya perjalanan (tol, parkir, lainnya) per booking
// kendaraan — docs/RANCANGAN_BIAYA_PERJALANAN.md. Dua tingkat fitur dipilih
// lewat master setting tersembunyi trip_expense_full_mode.
type TripExpenseService struct {
	q  repository.ExtendedQuerier
	db *sql.DB
}

func NewTripExpenseService(db *sql.DB) *TripExpenseService {
	return &TripExpenseService{q: repository.New(db), db: db}
}

const (
	settingTripExpenseFullMode = "trip_expense_full_mode"

	TripModeSimple = "SIMPLE"
	TripModeFull   = "FULL"

	tripCategoryToll    = "TOLL"
	tripCategoryParking = "PARKING"
	tripCategoryOther   = "OTHER"

	tripPayCompany  = "COMPANY_EMONEY"
	tripPayPersonal = "PERSONAL"

	tripMaxAmount  = 100_000_000
	tripMaxGateLen = 120
	tripMaxNoteLen = 500
)

// Mode: SIMPLE (default) atau FULL.
func (s *TripExpenseService) Mode(ctx context.Context) string {
	if m, err := s.q.GetMasterSettingByKey(ctx, settingTripExpenseFullMode); err == nil &&
		util.ParseStringToFloat64(m.Value) >= 1 {
		return TripModeFull
	}
	return TripModeSimple
}

func (s *TripExpenseService) Config(ctx context.Context) map[string]any {
	return map[string]any{"mode": s.Mode(ctx)}
}

// ── Validasi (murni) ────────────────────────────────────────────────────────

type CreateTripExpenseRequest struct {
	Category      string
	TollPoint     string
	EntryGate     string
	ExitGate      string
	Amount        float64
	PaymentMethod string
	Note          string
	OccurredAt    *time.Time
	ProofPhotoURL string
}

func tripBadRequest(msg string) error { return util.NewError(400, msg, util.ErrBadRequest) }

func trimMax(s string, n int) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

// normalizeTripExpense memvalidasi & merapikan isian sesuai mode. Isian milik
// mode lain dibuang (bukan ditolak) supaya klien versi lama/baru tetap jalan.
func normalizeTripExpense(mode string, req CreateTripExpenseRequest) (CreateTripExpenseRequest, error) {
	out := CreateTripExpenseRequest{
		Amount:        req.Amount,
		Note:          trimMax(req.Note, tripMaxNoteLen),
		OccurredAt:    req.OccurredAt,
		ProofPhotoURL: req.ProofPhotoURL,
	}
	if req.Amount <= 0 {
		return out, tripBadRequest("nominal wajib diisi")
	}
	if req.Amount > tripMaxAmount {
		return out, tripBadRequest("nominal terlalu besar")
	}
	if req.ProofPhotoURL == "" {
		return out, tripBadRequest("foto struk / bukti bayar wajib dilampirkan")
	}

	category := strings.ToUpper(strings.TrimSpace(req.Category))
	if category == "" {
		category = tripCategoryToll
	}

	if mode == TripModeSimple {
		if category != tripCategoryToll {
			return out, tripBadRequest("hanya biaya tol yang bisa dicatat")
		}
		out.Category = tripCategoryToll
		point := strings.ToUpper(strings.TrimSpace(req.TollPoint))
		if point != "ENTRY" && point != "EXIT" {
			return out, tripBadRequest("pilih Tol Masuk atau Tol Keluar")
		}
		out.TollPoint = point
		return out, nil
	}

	// FULL
	switch category {
	case tripCategoryToll, tripCategoryParking, tripCategoryOther:
	default:
		return out, tripBadRequest("jenis biaya tidak dikenal")
	}
	out.Category = category
	pay := strings.ToUpper(strings.TrimSpace(req.PaymentMethod))
	if pay != tripPayCompany && pay != tripPayPersonal {
		return out, tripBadRequest("pilih cara bayar: e-money kantor atau uang pribadi")
	}
	out.PaymentMethod = pay
	switch category {
	case tripCategoryToll:
		out.EntryGate = trimMax(req.EntryGate, tripMaxGateLen)
		out.ExitGate = trimMax(req.ExitGate, tripMaxGateLen)
		if out.EntryGate == "" && out.ExitGate == "" {
			return out, tripBadRequest("isi gerbang tol masuk atau keluar")
		}
	case tripCategoryOther:
		if out.Note == "" {
			return out, tripBadRequest("keterangan wajib diisi untuk biaya lainnya")
		}
	}
	return out, nil
}

// ── Hak akses ──────────────────────────────────────────────────────────────

// canRecordTrip: supir pemegang saat perjalanan berjalan; admin juga sesudahnya
// (input susulan). Hanya booking kendaraan.
func canRecordTrip(b repository.GetBookingByIDRow, role string, driverID int32) error {
	if b.ResourceType != repository.ResourceTypeVEHICLE {
		return tripBadRequest("biaya perjalanan hanya untuk booking kendaraan")
	}
	running := b.Status == repository.BookingStatusONGOING || b.Status == repository.BookingStatusOVERDUE
	switch role {
	case "ADMIN":
		if running || b.Status == repository.BookingStatusRETURNED || b.Status == repository.BookingStatusCOMPLETED {
			return nil
		}
		return util.NewError(409, "biaya perjalanan dicatat setelah perjalanan dimulai", util.ErrConflict)
	case "DRIVER":
		if !b.AssignedDriverId.Valid || b.AssignedDriverId.Int32 != driverID {
			return util.ErrForbidden
		}
		if !running {
			return util.NewError(409, "biaya perjalanan hanya bisa dicatat saat perjalanan berjalan", util.ErrConflict)
		}
		return nil
	default:
		return util.ErrForbidden
	}
}

// canViewBooking: sama dengan BookingService.GetByID.
func (s *TripExpenseService) canViewBooking(ctx context.Context, b repository.GetBookingByIDRow, userID int32, role string) error {
	switch role {
	case "EMPLOYEE":
		if b.UserId != userID {
			return util.ErrForbidden
		}
	case "DRIVER":
		d, err := s.q.GetDriverByUserID(ctx, userID)
		if err != nil || !b.AssignedDriverId.Valid || b.AssignedDriverId.Int32 != d.ID {
			return util.ErrForbidden
		}
	}
	return nil
}

// ── Serialisasi & ringkasan ────────────────────────────────────────────────

func serializeTripExpense(e repository.TripExpense) map[string]any {
	var driver any
	if e.DriverID.Valid {
		driver = map[string]any{"id": e.DriverID.Int32, "name": nullStr(e.DriverName)}
	}
	reimburse := "NONE"
	if e.PaymentMethod.Valid && e.PaymentMethod.String == tripPayPersonal {
		reimburse = "PENDING"
		if e.ReimbursementID.Valid {
			reimburse = "PAID"
		}
	}
	return map[string]any{
		"id":              e.ID,
		"bookingId":       e.BookingID,
		"vehicle":         map[string]any{"id": nullInt32(e.VehicleID), "plateNumber": nullStr(e.PlateNumber)},
		"driver":          driver,
		"category":        e.Category,
		"tollPoint":       nullStr(e.TollPoint),
		"entryGate":       nullStr(e.EntryGate),
		"exitGate":        nullStr(e.ExitGate),
		"amount":          e.Amount,
		"paymentMethod":   nullStr(e.PaymentMethod),
		"reimburseStatus": reimburse,
		"reimbursedAt":    nullTime(e.ReimbursedAt),
		"proofPhotoUrl":   e.ProofPhotoURL,
		"note":            nullStr(e.Note),
		"occurredAt":      e.OccurredAt,
		"recordedBy":      map[string]any{"id": e.RecordedByID, "name": e.RecordedByName},
		"isVoided":        e.VoidedAt.Valid,
		"voidedAt":        nullTime(e.VoidedAt),
		"voidedBy":        nullStr(e.VoidedByName),
		"voidReason":      nullStr(e.VoidReason),
		"createdAt":       e.CreatedAt,
	}
}

// tripSummary menghitung total (tanpa catatan dibatalkan), rincian per jenis,
// dan — untuk mode lengkap — pemakaian anggaran & reimburse.
func tripSummary(items []repository.TripExpense, budget sql.NullFloat64) map[string]any {
	total, company, personal, pending, paid := 0.0, 0.0, 0.0, 0.0, 0.0
	byCategory := map[string]float64{}
	count := 0
	for _, e := range items {
		if e.VoidedAt.Valid {
			continue
		}
		count++
		total += e.Amount
		byCategory[e.Category] += e.Amount
		switch e.PaymentMethod.String {
		case tripPayCompany:
			company += e.Amount
		case tripPayPersonal:
			personal += e.Amount
			if e.ReimbursementID.Valid {
				paid += e.Amount
			} else {
				pending += e.Amount
			}
		}
	}
	out := map[string]any{
		"total":            round2(total),
		"count":            count,
		"byCategory":       byCategory,
		"companyPaid":      round2(company),
		"personalPaid":     round2(personal),
		"reimbursePending": round2(pending),
		"reimbursePaid":    round2(paid),
		"tollBudget":       nil,
		"budgetRemaining":  nil,
	}
	if budget.Valid {
		out["tollBudget"] = round2(budget.Float64)
		out["budgetRemaining"] = round2(budget.Float64 - company)
	}
	return out
}

// ── Operasi ────────────────────────────────────────────────────────────────

// tripBookingIDs: booking ini + pasangan merge-nya (satu perjalanan bersama).
func (s *TripExpenseService) tripBookingIDs(ctx context.Context, bookingID int32) []int32 {
	ids := []int32{bookingID}
	if merges, err := s.q.GetBookingMerges(ctx, bookingID); err == nil {
		for _, m := range merges {
			ids = append(ids, m.OtherBookingID)
		}
	}
	return ids
}

func (s *TripExpenseService) ListForBooking(ctx context.Context, bookingID, userID int32, role string) (map[string]any, error) {
	b, err := s.q.GetBookingByID(ctx, bookingID)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if err := s.canViewBooking(ctx, b, userID, role); err != nil {
		return nil, err
	}
	var all []repository.TripExpense
	seen := map[int32]bool{}
	for _, id := range s.tripBookingIDs(ctx, bookingID) {
		list, err := s.q.ListTripExpensesByBooking(ctx, id)
		if err != nil {
			return nil, err
		}
		for _, e := range list {
			if !seen[e.ID] {
				seen[e.ID] = true
				all = append(all, e)
			}
		}
	}
	budget, _ := s.q.GetBookingTollBudget(ctx, bookingID)
	items := make([]map[string]any, len(all))
	for i, e := range all {
		items[i] = serializeTripExpense(e)
	}
	return map[string]any{
		"mode":    s.Mode(ctx),
		"items":   items,
		"summary": tripSummary(all, budget),
	}, nil
}

func (s *TripExpenseService) Create(ctx context.Context, bookingID int32, req CreateTripExpenseRequest, role string, actor AuditActor) (map[string]any, error) {
	b, err := s.q.GetBookingByID(ctx, bookingID)
	if err != nil {
		return nil, util.ErrNotFound
	}
	var driverID int32
	if role == "DRIVER" {
		d, err := s.q.GetDriverByUserID(ctx, actor.UserID)
		if err != nil {
			return nil, util.ErrForbidden
		}
		driverID = d.ID
	}
	if err := canRecordTrip(b, role, driverID); err != nil {
		return nil, err
	}
	mode := s.Mode(ctx)
	n, err := normalizeTripExpense(mode, req)
	if err != nil {
		return nil, err
	}
	occurred := time.Now().UTC()
	if n.OccurredAt != nil && !n.OccurredAt.IsZero() {
		if n.OccurredAt.After(occurred.Add(5 * time.Minute)) {
			return nil, tripBadRequest("waktu transaksi tidak boleh di masa depan")
		}
		occurred = n.OccurredAt.UTC()
	}
	ns := func(v string) sql.NullString { return sql.NullString{String: v, Valid: v != ""} }
	id, err := s.q.CreateTripExpense(ctx, repository.CreateTripExpenseParams{
		BookingID:     bookingID,
		VehicleID:     b.AssignedVehicleId,
		DriverID:      b.AssignedDriverId,
		Category:      n.Category,
		TollPoint:     ns(n.TollPoint),
		EntryGate:     ns(n.EntryGate),
		ExitGate:      ns(n.ExitGate),
		Amount:        round2(n.Amount),
		PaymentMethod: ns(n.PaymentMethod),
		ProofPhotoURL: n.ProofPhotoURL,
		Note:          ns(n.Note),
		OccurredAt:    occurred,
		RecordedByID:  actor.UserID,
	})
	if err != nil {
		return nil, err
	}
	logAudit(ctx, s.q, actor, "CREATE", "TripExpense", id,
		fmt.Sprintf("Catat biaya perjalanan %s Rp%.0f (booking #%d)", n.Category, n.Amount, bookingID))
	e, err := s.q.GetTripExpense(ctx, id)
	if err != nil {
		return nil, err
	}
	return serializeTripExpense(e), nil
}

func (s *TripExpenseService) Void(ctx context.Context, id int32, reason string, actor AuditActor) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return tripBadRequest("alasan pembatalan wajib diisi")
	}
	e, err := s.q.GetTripExpense(ctx, id)
	if err != nil {
		return util.ErrNotFound
	}
	if e.ReimbursementID.Valid {
		return util.NewError(409, "catatan yang sudah direimburse tidak bisa dibatalkan", util.ErrConflict)
	}
	ok, err := s.q.VoidTripExpense(ctx, id, actor.UserID, reason)
	if err != nil {
		return err
	}
	if !ok {
		return util.NewError(409, "catatan sudah dibatalkan", util.ErrConflict)
	}
	logAudit(ctx, s.q, actor, "VOID", "TripExpense", id, "Batalkan biaya perjalanan: "+reason)
	return nil
}

func (s *TripExpenseService) GateSuggestions(ctx context.Context, search string) ([]string, error) {
	list, err := s.q.ListTripGateSuggestions(ctx, trimMax(search, tripMaxGateLen), 10)
	if list == nil {
		list = []string{}
	}
	return list, err
}

// SetTollBudget: anggaran tol (mode lengkap) — nil menghapus anggaran.
func (s *TripExpenseService) SetTollBudget(ctx context.Context, bookingID int32, amount *float64, actor AuditActor) (map[string]any, error) {
	b, err := s.q.GetBookingByID(ctx, bookingID)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if b.ResourceType != repository.ResourceTypeVEHICLE {
		return nil, tripBadRequest("anggaran tol hanya untuk booking kendaraan")
	}
	v := sql.NullFloat64{}
	if amount != nil {
		if *amount < 0 || *amount > tripMaxAmount {
			return nil, tripBadRequest("nominal anggaran tidak valid")
		}
		v = sql.NullFloat64{Float64: round2(*amount), Valid: true}
	}
	if err := s.q.SetBookingTollBudget(ctx, bookingID, v); err != nil {
		return nil, err
	}
	desc := "Hapus anggaran tol"
	if v.Valid {
		desc = fmt.Sprintf("Anggaran tol Rp%.0f", v.Float64)
	}
	logAudit(ctx, s.q, actor, "UPDATE", "Booking", bookingID, desc)
	return map[string]any{"bookingId": bookingID, "tollBudget": nullFloat(v)}, nil
}

// PendingReimbursements: transaksi uang pribadi yang belum diganti,
// dikelompokkan per supir.
func (s *TripExpenseService) PendingReimbursements(ctx context.Context) ([]map[string]any, error) {
	list, err := s.q.ListPendingTripReimbursements(ctx)
	if err != nil {
		return nil, err
	}
	type group struct {
		driverID int32
		name     string
		total    float64
		items    []map[string]any
	}
	var order []int32
	groups := map[int32]*group{}
	for _, e := range list {
		id := e.DriverID.Int32
		g, ok := groups[id]
		if !ok {
			g = &group{driverID: id, name: e.DriverName.String}
			groups[id] = g
			order = append(order, id)
		}
		g.total += e.Amount
		g.items = append(g.items, serializeTripExpense(e))
	}
	out := make([]map[string]any, 0, len(order))
	for _, id := range order {
		g := groups[id]
		out = append(out, map[string]any{
			"driver": map[string]any{"id": g.driverID, "name": g.name},
			"total":  round2(g.total),
			"count":  len(g.items),
			"items":  g.items,
		})
	}
	return out, nil
}

type PayTripReimbursementRequest struct {
	DriverID   int32      `json:"driverId"   validate:"required"`
	ExpenseIDs []int32    `json:"expenseIds" validate:"required,min=1"`
	PaidAt     *time.Time `json:"paidAt"`
	Note       string     `json:"note"`
}

func (s *TripExpenseService) PayReimbursement(ctx context.Context, req PayTripReimbursementRequest, actor AuditActor) (map[string]any, error) {
	paidAt := time.Now().UTC()
	if req.PaidAt != nil && !req.PaidAt.IsZero() {
		paidAt = req.PaidAt.UTC()
	}
	note := sql.NullString{String: trimMax(req.Note, tripMaxNoteLen), Valid: strings.TrimSpace(req.Note) != ""}
	var rid int32
	var count int64
	var total float64
	run := func(q repository.ExtendedQuerier) error {
		var err error
		rid, count, total, err = q.MarkTripReimbursed(ctx, req.DriverID, req.ExpenseIDs, paidAt, note, actor.UserID)
		if err != nil {
			return err
		}
		if count == 0 {
			return util.NewError(409, "tidak ada transaksi yang menunggu reimburse untuk supir ini", util.ErrConflict)
		}
		return nil
	}
	var err error
	if s.db != nil {
		err = withTx(ctx, s.db, func(q *repository.Queries) error { return run(q) })
	} else {
		err = run(s.q)
	}
	if err != nil {
		return nil, err
	}
	logAudit(ctx, s.q, actor, "REIMBURSE", "TripReimbursement", rid,
		fmt.Sprintf("Reimburse biaya perjalanan %d transaksi Rp%.0f", count, total))
	return map[string]any{"id": rid, "driverId": req.DriverID, "count": count, "amount": round2(total), "paidAt": paidAt}, nil
}

func (s *TripExpenseService) ListReimbursements(ctx context.Context) ([]map[string]any, error) {
	list, err := s.q.ListTripReimbursements(ctx, 100)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, len(list))
	for i, r := range list {
		out[i] = map[string]any{
			"id":           r.ID,
			"driver":       map[string]any{"id": r.DriverID, "name": r.DriverName},
			"amount":       r.Amount,
			"paidAt":       r.PaidAt,
			"note":         nullStr(r.Note),
			"createdBy":    r.CreatedByName,
			"expenseCount": r.ExpenseCount,
			"createdAt":    r.CreatedAt,
		}
	}
	return out, nil
}
