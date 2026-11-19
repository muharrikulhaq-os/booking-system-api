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

// settingPassengerEditAfterApproval: 1 = penumpang boleh diubah setelah booking
// disetujui (sampai perjalanan dimulai); 0 = terkunci begitu disetujui.
const settingPassengerEditAfterApproval = "booking_passenger_edit_after_approval"

// passengersEditable: status booking ini masih mengizinkan daftar penumpang
// diubah - PENDING selalu; APPROVED hanya bila diizinkan pengaturan.
// (Siapa yang boleh - pemesan/admin - dicek terpisah.)
func (s *BookingService) passengersEditable(ctx context.Context, status repository.BookingStatus) bool {
	switch status {
	case repository.BookingStatusPENDING:
		return true
	case repository.BookingStatusAPPROVED:
		m, err := s.q.GetMasterSettingByKey(ctx, settingPassengerEditAfterApproval)
		return err == nil && util.ParseStringToFloat64(m.Value) >= 1
	}
	return false
}

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

// GuestInput: penumpang tamu tanpa akun (nama wajib; nomor HP & keterangan
// mis. instansi/jabatan opsional). Tidak mendapat notifikasi / akses.
type GuestInput struct {
	Name        string `json:"name"`
	PhoneNumber string `json:"phoneNumber"`
	Note        string `json:"note"`
}

const maxGuests = 20

// normalizeGuests merapikan & memvalidasi tamu.
func normalizeGuests(in []GuestInput) ([]repository.BookingGuest, error) {
	if len(in) > maxGuests {
		return nil, util.NewError(400, fmt.Sprintf("tamu maksimal %d orang", maxGuests), util.ErrBadRequest)
	}
	out := make([]repository.BookingGuest, 0, len(in))
	for _, g := range in {
		name := strings.TrimSpace(g.Name)
		if name == "" {
			return nil, util.NewError(400, "nama tamu wajib diisi", util.ErrBadRequest)
		}
		if len([]rune(name)) > 120 {
			return nil, util.NewError(400, "nama tamu terlalu panjang (maks 120 karakter)", util.ErrBadRequest)
		}
		phone, err := normalizePhone(g.PhoneNumber)
		if err != nil {
			return nil, err
		}
		note := strings.TrimSpace(g.Note)
		if r := []rune(note); len(r) > 120 {
			note = string(r[:120])
		}
		out = append(out, repository.BookingGuest{Name: name, PhoneNumber: phone, Note: note})
	}
	return out, nil
}

// checkSeats: karyawan + tamu tidak melebihi kapasitas dikurangi pemesan.
func checkSeats(users, guests int, capacity int32) error {
	if capacity > 0 && int32(users+guests) > capacity-1 {
		return util.NewError(400,
			fmt.Sprintf("penumpang yang ikut maksimal %d orang (kapasitas kendaraan %d termasuk pemesan)", capacity-1, capacity),
			util.ErrBadRequest)
	}
	return nil
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
	if guests, err := s.q.ListBookingGuests(ctx, ids); err == nil {
		for _, g := range guests {
			list = append(list, map[string]any{
				"userId": 0, "name": g.Name, "employeeId": "", "department": g.Note,
				"phoneNumber": g.PhoneNumber, "bookingId": g.BookingID, "role": "GUEST",
			})
		}
	}
	out["passengers"] = list
	// Status ini masih boleh diubah (PENDING / APPROVED + pengaturan) - klien
	// menambahkan cek pemesan/admin.
	out["passengersEditable"] = s.passengersEditable(ctx, b.Status)
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
	// nil = tamu tidak diubah (klien lama); [] = hapus semua tamu.
	Guests *[]GuestInput `json:"guests"`
}

// SetPassengers mengganti daftar karyawan yang ikut. Pemesan & admin selama
// PENDING; setelah disetujui hanya bila pengaturan
// booking_passenger_edit_after_approval = 1 (sampai perjalanan dimulai).
// Jumlah penumpang dinaikkan bila kurang dan tidak pernah melebihi kapasitas.
func (s *BookingService) SetPassengers(ctx context.Context, id int32, req SetPassengersRequest, actor AuditActor, role string) (map[string]any, error) {
	b, err := s.q.GetBookingByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if role != "ADMIN" && b.UserId != actor.UserID {
		return nil, util.ErrForbidden
	}
	if !s.passengersEditable(ctx, b.Status) {
		msg := "daftar penumpang hanya bisa diubah selama booking masih menunggu persetujuan"
		if b.Status != repository.BookingStatusAPPROVED {
			msg = "daftar penumpang hanya bisa diubah sebelum perjalanan dimulai"
		}
		return nil, util.NewError(409, msg, util.ErrConflict)
	}
	capacity, isVehicle := s.vehicleCapacity(ctx, b.ResourceId)
	if !isVehicle {
		return nil, util.NewError(400, "karyawan yang ikut hanya untuk booking kendaraan", util.ErrBadRequest)
	}
	ids, err := normalizePassengerIDs(ctx, s.q, b.UserId, req.PassengerIDs, capacity)
	if err != nil {
		return nil, err
	}
	var guests []repository.BookingGuest
	if req.Guests != nil {
		if guests, err = normalizeGuests(*req.Guests); err != nil {
			return nil, err
		}
	} else if guests, err = s.q.ListBookingGuests(ctx, []int32{id}); err != nil {
		return nil, err
	}
	if err := checkSeats(len(ids), len(guests), capacity); err != nil {
		return nil, err
	}
	added, err := s.q.ReplaceBookingPassengers(ctx, id, ids, actor.UserID)
	if err != nil {
		return nil, err
	}
	if req.Guests != nil {
		if err := s.q.ReplaceBookingGuests(ctx, id, guests, actor.UserID); err != nil {
			return nil, err
		}
	}
	// Jumlah penumpang: minimal pemesan + yang ikut, maksimal kapasitas
	// kendaraan (data lama yang terlanjur melebihi kapasitas ikut dirapikan).
	need := int32(len(ids)+len(guests)) + 1
	count := b.PassengerCount
	if count < need {
		count = need
	}
	if capacity > 0 && count > capacity {
		count = max(capacity, need)
	}
	if count != b.PassengerCount {
		_ = s.q.SetBookingPassengerCount(ctx, id, count)
	}
	logAudit(ctx, s.q, actor, "UPDATE", "Booking", id, fmt.Sprintf("Penumpang yang ikut: %d karyawan, %d tamu", len(ids), len(guests)))
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
