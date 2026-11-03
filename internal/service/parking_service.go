package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

// Uang parkir per booking — docs/RANCANGAN_PARKIR_PENUMPANG.md §2.
//
// Admin membekali uang parkir (nominal) per booking, supir mencatat setiap
// parkir + karcis. Setelah perjalanan kembali/selesai selisihnya diselesaikan
// admin: sisa dikembalikan supir (TO_OFFICE) atau kekurangan diganti ke supir
// (TO_DRIVER, ikut pembayaran reimburse). Booking yang digabung = satu
// perjalanan, dihitung di booking utamanya.

const (
	parkingToDriver = "TO_DRIVER"
	parkingToOffice = "TO_OFFICE"
	parkingNone     = "NONE"
)

// parkingBalance: arah & besar selisih. allowance 0 = tidak dibekali.
func parkingBalance(allowance, used float64) (string, float64) {
	diff := round2(allowance - used)
	switch {
	case diff > 0:
		return parkingToOffice, diff
	case diff < 0:
		return parkingToDriver, -diff
	}
	return parkingNone, 0
}

// parkingLeadID: booking utama perjalanan (sisi "digabung" → booking utamanya).
func (s *TripExpenseService) parkingLeadID(ctx context.Context, bookingID int32) int32 {
	if merges, err := s.q.GetBookingMerges(ctx, bookingID); err == nil {
		for _, m := range merges {
			if !m.IsPrimary {
				return m.PrimaryBookingID
			}
		}
	}
	return bookingID
}

func serializeParkingSettlement(p repository.ParkingSettlement) map[string]any {
	return map[string]any{
		"id":              p.ID,
		"bookingId":       p.BookingID,
		"driver":          map[string]any{"id": p.DriverID, "name": p.DriverName},
		"direction":       p.Direction,
		"amount":          round2(p.Amount),
		"allowance":       round2(p.Allowance),
		"used":            round2(p.Used),
		"reimbursementId": nullInt32(p.ReimbursementID),
		"settledAt":       p.SettledAt,
		"note":            nullStr(p.Note),
		"createdBy":       p.CreatedByName,
		"bookingPurpose":  p.BookingPurpose,
		"bookingStart":    p.BookingStart,
	}
}

// parkingInfo: ringkasan uang parkir perjalanan untuk detail booking.
func (s *TripExpenseService) parkingInfo(ctx context.Context, lead int32, items []repository.TripExpense) map[string]any {
	used, count := 0.0, 0
	for _, e := range items {
		if e.Category == tripCategoryParking && !e.VoidedAt.Valid {
			used += e.Amount
			count++
		}
	}
	allowance, _ := s.q.GetBookingParkingAllowance(ctx, lead)
	direction, amount := parkingBalance(allowance.Float64, used)
	out := map[string]any{
		"leadBookingId": lead,
		"allowance":     round2(allowance.Float64),
		"hasAllowance":  allowance.Valid && allowance.Float64 > 0,
		"used":          round2(used),
		"count":         count,
		"direction":     direction,
		"amount":        amount,
		"settlement":    nil,
		"canSettle":     false,
	}
	if p, err := s.q.GetParkingSettlement(ctx, lead); err == nil {
		out["settlement"] = serializeParkingSettlement(p)
	} else if b, err := s.q.GetBookingByID(ctx, lead); err == nil {
		out["canSettle"] = direction != parkingNone &&
			(b.Status == repository.BookingStatusRETURNED || b.Status == repository.BookingStatusCOMPLETED)
	}
	return out
}

// ensureParkingOpen menolak perubahan parkir/bekal setelah selisih diselesaikan.
func (s *TripExpenseService) ensureParkingOpen(ctx context.Context, bookingID int32) error {
	lead := s.parkingLeadID(ctx, bookingID)
	if _, err := s.q.GetParkingSettlement(ctx, lead); err == nil {
		return util.NewError(409, "uang parkir perjalanan ini sudah diselesaikan - catatan parkir terkunci", util.ErrConflict)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	return nil
}

// SetParkingAllowance: bekal uang parkir (admin). 0 = tidak dibekali.
// Disimpan di booking utama perjalanan.
func (s *TripExpenseService) SetParkingAllowance(ctx context.Context, bookingID int32, amount float64, actor AuditActor) (map[string]any, error) {
	b, err := s.q.GetBookingByID(ctx, bookingID)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if b.ResourceType != repository.ResourceTypeVEHICLE {
		return nil, tripBadRequest("uang parkir hanya untuk booking kendaraan")
	}
	switch b.Status {
	case repository.BookingStatusREJECTED, repository.BookingStatusCANCELLED,
		repository.BookingStatusEXPIRED, repository.BookingStatusIGNORED:
		return nil, util.NewError(409, "booking ini tidak berjalan - uang parkir tidak bisa diatur", util.ErrConflict)
	}
	if amount < 0 || amount > tripMaxAmount {
		return nil, tripBadRequest("nominal uang parkir tidak valid")
	}
	if err := s.ensureParkingOpen(ctx, bookingID); err != nil {
		return nil, err
	}
	lead := s.parkingLeadID(ctx, bookingID)
	v := sql.NullFloat64{Float64: round2(amount), Valid: amount > 0}
	if err := s.q.SetBookingParkingAllowance(ctx, lead, v); err != nil {
		return nil, err
	}
	desc := "Tanpa bekal uang parkir"
	if v.Valid {
		desc = fmt.Sprintf("Bekal uang parkir Rp%.0f", v.Float64)
	}
	logAudit(ctx, s.q, actor, "UPDATE", "Booking", lead, desc)
	return map[string]any{"bookingId": lead, "parkingAllowance": round2(v.Float64)}, nil
}

func serializePendingParking(p repository.PendingParking) map[string]any {
	direction, amount := parkingBalance(p.Allowance, p.Used)
	return map[string]any{
		"bookingId":    p.BookingID,
		"purpose":      p.Purpose,
		"startDate":    p.StartDate,
		"resourceName": p.ResourceName,
		"plateNumber":  p.PlateNumber,
		"status":       p.Status,
		"allowance":    round2(p.Allowance),
		"used":         round2(p.Used),
		"count":        p.Count,
		"direction":    direction,
		"amount":       amount,
	}
}

// PendingParking: selisih parkir yang belum diselesaikan, per supir.
func (s *TripExpenseService) PendingParking(ctx context.Context) ([]map[string]any, error) {
	list, err := s.q.ListPendingParking(ctx, nil)
	if err != nil {
		return nil, err
	}
	type group struct {
		id                 int32
		name               string
		toDriver, toOffice float64
		items              []map[string]any
	}
	var order []int32
	groups := map[int32]*group{}
	for _, p := range list {
		g, ok := groups[p.DriverID]
		if !ok {
			g = &group{id: p.DriverID, name: p.DriverName}
			groups[p.DriverID] = g
			order = append(order, p.DriverID)
		}
		item := serializePendingParking(p)
		if item["direction"] == parkingToDriver {
			g.toDriver += item["amount"].(float64)
		} else {
			g.toOffice += item["amount"].(float64)
		}
		g.items = append(g.items, item)
	}
	out := make([]map[string]any, 0, len(order))
	for _, id := range order {
		g := groups[id]
		out = append(out, map[string]any{
			"driver":   map[string]any{"id": g.id, "name": g.name},
			"toDriver": round2(g.toDriver),
			"toOffice": round2(g.toOffice),
			"count":    len(g.items),
			"items":    g.items,
		})
	}
	return out, nil
}

type SettleParkingRequest struct {
	DriverID   int32      `json:"driverId"   validate:"required"`
	BookingIDs []int32    `json:"bookingIds" validate:"required,min=1"`
	SettledAt  *time.Time `json:"settledAt"`
	Note       string     `json:"note"`
}

// SettleParking menyelesaikan selisih beberapa perjalanan satu supir sekaligus.
// Kekurangan (TO_DRIVER) digabung jadi satu pembayaran reimburse; sisa
// (TO_OFFICE) dicatat sudah dikembalikan.
func (s *TripExpenseService) SettleParking(ctx context.Context, req SettleParkingRequest, actor AuditActor) (map[string]any, error) {
	settledAt := time.Now().UTC()
	if req.SettledAt != nil && !req.SettledAt.IsZero() {
		settledAt = req.SettledAt.UTC()
	}
	note := sql.NullString{String: trimMax(req.Note, tripMaxNoteLen), Valid: strings.TrimSpace(req.Note) != ""}
	var toDriver, toOffice float64
	var nDriver, nOffice int
	var rid sql.NullInt32
	run := func(q repository.ExtendedQuerier) error {
		rows, err := q.ListPendingParking(ctx, req.BookingIDs)
		if err != nil {
			return err
		}
		var mine []repository.PendingParking
		for _, p := range rows {
			if p.DriverID == req.DriverID {
				mine = append(mine, p)
			}
		}
		if len(mine) == 0 {
			return util.NewError(409, "tidak ada selisih parkir yang menunggu untuk supir ini", util.ErrConflict)
		}
		for _, p := range mine {
			if d, amt := parkingBalance(p.Allowance, p.Used); d == parkingToDriver {
				toDriver += amt
			}
		}
		toDriver = round2(toDriver)
		if toDriver > 0 {
			id, err := q.CreateTripReimbursement(ctx, req.DriverID, toDriver, settledAt, note, actor.UserID)
			if err != nil {
				return err
			}
			rid = sql.NullInt32{Int32: id, Valid: true}
		}
		for _, p := range mine {
			d, amt := parkingBalance(p.Allowance, p.Used)
			params := repository.CreateParkingSettlementParams{
				BookingID: p.BookingID, DriverID: p.DriverID, Direction: d, Amount: amt,
				Allowance: p.Allowance, Used: p.Used, SettledAt: settledAt, Note: note, CreatedByID: actor.UserID,
			}
			if d == parkingToDriver {
				params.ReimbursementID = rid
				nDriver++
			} else {
				toOffice += amt
				nOffice++
			}
			ok, err := q.CreateParkingSettlement(ctx, params)
			if err != nil {
				return err
			}
			if !ok {
				return util.NewError(409, fmt.Sprintf("booking #%d sudah diselesaikan", p.BookingID), util.ErrConflict)
			}
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
	toOffice = round2(toOffice)
	logAudit(ctx, s.q, actor, "SETTLE", "Parking", req.DriverID,
		fmt.Sprintf("Selesaikan uang parkir: diganti ke supir Rp%.0f (%d), sisa dikembalikan Rp%.0f (%d)",
			toDriver, nDriver, toOffice, nOffice))
	return map[string]any{
		"driverId":        req.DriverID,
		"toDriver":        toDriver,
		"toDriverCount":   nDriver,
		"toOffice":        toOffice,
		"toOfficeCount":   nOffice,
		"reimbursementId": nullInt32(rid),
		"settledAt":       settledAt,
	}, nil
}

func (s *TripExpenseService) ListParkingSettlements(ctx context.Context) ([]map[string]any, error) {
	list, err := s.q.ListParkingSettlements(ctx, 100)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, len(list))
	for i, p := range list {
		out[i] = serializeParkingSettlement(p)
	}
	return out, nil
}
