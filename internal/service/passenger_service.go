package service

import (
	"context"
	"fmt"
	"strings"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

// Karyawan yang ikut booking — docs/RANCANGAN_PARKIR_PENUMPANG.md §3.
//
// Booking tetap SATU, atas nama pemesan: penumpang tidak membuat booking baru
// dan tidak dihitung di "Booking Saya" / statistik / laporan mereka. Mereka
// boleh membuka detail (baca saja) dan mendapat notifikasi perkembangan.

const maxPassengerLookup = 20

// employeeCanView: karyawan boleh membuka booking yang dia pesan atau ikuti.
func employeeCanView(ctx context.Context, q repository.ExtendedQuerier, b repository.GetBookingByIDRow, userID int32) bool {
	if b.UserId == userID {
		return true
	}
	ok, err := q.IsBookingPassenger(ctx, b.ID, userID)
	return err == nil && ok
}

// normalizePassengerIDs: buang duplikat & pemesan sendiri, pastikan semua
// karyawan aktif, dan tidak melebihi kapasitas kendaraan (dikurangi pemesan).
func normalizePassengerIDs(ctx context.Context, q repository.ExtendedQuerier, ownerID int32, ids []int32, capacity int32) ([]int32, error) {
	seen := map[int32]bool{ownerID: true}
	out := make([]int32, 0, len(ids))
	for _, id := range ids {
		if id > 0 && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	if len(out) == 0 {
		return out, nil
	}
	if capacity > 0 && int32(len(out)) > capacity-1 {
		return nil, util.NewError(400,
			fmt.Sprintf("karyawan yang ikut maksimal %d orang (kapasitas kendaraan %d termasuk pemesan)", capacity-1, capacity),
			util.ErrBadRequest)
	}
	found, err := q.LookupActiveUsers(ctx, "", out, int32(len(out)))
	if err != nil {
		return nil, err
	}
	if len(found) != len(out) {
		return nil, util.NewError(400, "ada karyawan yang dipilih tidak ditemukan atau tidak aktif", util.ErrBadRequest)
	}
	return out, nil
}

// vehicleCapacity: kapasitas kendaraan resource booking (0 = bukan kendaraan).
func (s *BookingService) vehicleCapacity(ctx context.Context, resourceID int32) (int32, bool) {
	vid, err := s.q.GetVehicleIDByResourceID(ctx, resourceID)
	if err != nil {
		return 0, false
	}
	v, err := s.q.GetVehicleByID(ctx, vid)
	if err != nil {
		return 0, true
	}
	return int32(v.Capacity), true
}

// attachPassengers: "passengers" untuk detail booking — karyawan yang ikut
// booking ini, plus (booking digabung) pemesan & penumpang booking pasangannya
// supaya supir melihat semua orang dalam satu perjalanan.
func (s *BookingService) attachPassengers(ctx context.Context, out map[string]any, b repository.GetBookingByIDRow) {
	ids := []int32{b.ID}
	type partner struct {
		bookingID int32
		userID    int32
		name      string
		empID     string
		dept      string
	}
	var partners []partner
	if merges, err := s.q.GetBookingMerges(ctx, b.ID); err == nil {
		for _, m := range merges {
			ids = append(ids, m.OtherBookingID)
			partners = append(partners, partner{m.OtherBookingID, m.OtherUserID, m.OtherUserName, m.OtherEmployeeID, m.OtherDepartment})
		}
	}
	list := []map[string]any{}
	for _, p := range partners {
		phone, _ := s.q.GetUserPhone(ctx, p.userID)
		list = append(list, map[string]any{
			"userId": p.userID, "name": p.name, "employeeId": p.empID, "department": p.dept,
			"phoneNumber": phone, "bookingId": p.bookingID, "role": "REQUESTER",
		})
	}
	if rows, err := s.q.ListBookingPassengers(ctx, ids); err == nil {
		for _, p := range rows {
			list = append(list, map[string]any{
				"userId": p.UserID, "name": p.Name, "employeeId": p.EmployeeID, "department": p.Department,
				"phoneNumber": p.PhoneNumber, "bookingId": p.BookingID, "role": "PASSENGER",
			})
		}
	}
	out["passengers"] = list
}

// notifyPassengers: kabar perkembangan booking ke karyawan yang ikut.
func (s *BookingService) notifyPassengers(ctx context.Context, bookingID int32, notifType, title, body string) {
	if s.notif == nil {
		return
	}
	ids, err := s.q.ListPassengerUserIDs(ctx, bookingID)
	if err != nil {
		return
	}
	for _, id := range ids {
		s.notif.Notify(id, notifType, title, body, map[string]any{"bookingId": bookingID})
	}
}

type SetPassengersRequest struct {
	PassengerIDs []int32 `json:"passengerIds"`
}

// SetPassengers mengganti daftar karyawan yang ikut. Pemesan (selama
// PENDING/APPROVED) & admin. Jumlah penumpang dinaikkan bila kurang.
func (s *BookingService) SetPassengers(ctx context.Context, id int32, req SetPassengersRequest, actor AuditActor, role string) (map[string]any, error) {
	b, err := s.q.GetBookingByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if role != "ADMIN" && b.UserId != actor.UserID {
		return nil, util.ErrForbidden
	}
	if b.Status != repository.BookingStatusPENDING && b.Status != repository.BookingStatusAPPROVED {
		return nil, util.NewError(409, "daftar karyawan yang ikut hanya bisa diubah sebelum perjalanan dimulai", util.ErrConflict)
	}
	capacity, isVehicle := s.vehicleCapacity(ctx, b.ResourceId)
	if !isVehicle {
		return nil, util.NewError(400, "karyawan yang ikut hanya untuk booking kendaraan", util.ErrBadRequest)
	}
	ids, err := normalizePassengerIDs(ctx, s.q, b.UserId, req.PassengerIDs, capacity)
	if err != nil {
		return nil, err
	}
	added, err := s.q.ReplaceBookingPassengers(ctx, id, ids, actor.UserID)
	if err != nil {
		return nil, err
	}
	if need := int32(len(ids)) + 1; b.PassengerCount < need {
		_ = s.q.SetBookingPassengerCount(ctx, id, need)
	}
	logAudit(ctx, s.q, actor, "UPDATE", "Booking", id, fmt.Sprintf("Karyawan yang ikut: %d orang", len(ids)))
	s.notifyPassengersAdded(ctx, b, added)
	full, _ := s.q.GetBookingByID(ctx, id)
	return s.GetByIDFor(ctx, full)
}

// notifyPassengersAdded: "Anda diikutkan" untuk karyawan yang baru ditambahkan.
func (s *BookingService) notifyPassengersAdded(ctx context.Context, b repository.GetBookingByIDRow, userIDs []int32) {
	if s.notif == nil {
		return
	}
	for _, uid := range userIDs {
		s.notif.Notify(uid, "BOOKING_PASSENGER_ADDED", "Anda diikutkan dalam perjalanan",
			b.UserName+" mengikutkan Anda: "+bookingLabel(b)+".", map[string]any{"bookingId": b.ID})
	}
}

// PassengerTrips: perjalanan yang diikuti user (Dashboard "Perjalanan yang Anda ikuti").
func (s *BookingService) PassengerTrips(ctx context.Context, userID int32) ([]map[string]any, error) {
	list, err := s.q.ListPassengerTrips(ctx, userID, 10)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, len(list))
	for i, t := range list {
		out[i] = map[string]any{
			"id":           t.ID,
			"status":       t.Status,
			"purpose":      t.Purpose,
			"startDate":    t.StartDate,
			"endDate":      t.EndDate,
			"resourceName": t.ResourceName,
			"plateNumber":  nullStr(t.PlateNumber),
			"ownerName":    t.OwnerName,
			"driverName":   nullStr(t.DriverName),
		}
	}
	return out, nil
}

// LookupPassengers: cari karyawan aktif untuk dipilih (tanpa nomor telepon).
func (s *UserService) LookupPassengers(ctx context.Context, search string) ([]repository.UserLookup, error) {
	search = strings.TrimSpace(search)
	if r := []rune(search); len(r) > 60 {
		search = string(r[:60])
	}
	list, err := s.q.LookupActiveUsers(ctx, search, nil, maxPassengerLookup)
	if list == nil {
		list = []repository.UserLookup{}
	}
	return list, err
}
