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

// Notifikasi untuk SUPIR yang ditugaskan pada booking.
//
// Prinsipnya: setiap perubahan yang membuat supir mendapat, kehilangan, atau
// perlu tahu perubahan tugasnya diberitahukan — termasuk saat booking baru
// masuk (supir dipilih/otomatis saat dibuat, status masih PENDING).

var (
	hariID  = [...]string{"Min", "Sen", "Sel", "Rab", "Kam", "Jum", "Sab"}
	bulanID = [...]string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}
)

// formatWIB: "Sen, 6 Okt 08:00 WIB".
func formatWIB(t time.Time) string {
	w := t.In(util.WIB)
	return fmt.Sprintf("%s, %d %s %02d:%02d WIB",
		hariID[w.Weekday()], w.Day(), bulanID[w.Month()-1], w.Hour(), w.Minute())
}

// bookingLabel: "Avanza (B 1234 CD) · Sen, 6 Okt 08:00 WIB" — cukup untuk
// supir mengenali booking mana tanpa membuka aplikasi.
func bookingLabel(b repository.GetBookingByIDRow) string {
	name := b.ResourceName
	if b.PlateNumber.Valid && b.PlateNumber.String != "" {
		name += " (" + b.PlateNumber.String + ")"
	}
	return name + " · " + formatWIB(b.StartDate)
}

// notifyDriver mengirim notifikasi ke akun pengguna milik supir [driverID]
// (bila ada). Aman dipanggil dengan driverID kosong.
func (s *BookingService) notifyDriver(ctx context.Context, driverID sql.NullInt32, notifType, title, message string, bookingID int32) {
	if s.notif == nil || !driverID.Valid {
		return
	}
	drv, err := s.q.GetDriverByID(ctx, driverID.Int32)
	if err != nil {
		return
	}
	s.notif.Notify(drv.UserId, notifType, title, message, map[string]any{"bookingId": bookingID})
}

// notifySweptBookings memberi tahu pemohon & supir booking yang hangus
// otomatis (EXPIRED: disetujui tapi tidak dimulai; IGNORED: tidak direspons
// admin). Sebelumnya transisi ini terjadi diam-diam.
func (s *BookingService) notifySweptBookings(ctx context.Context, expired, ignored []repository.Booking) {
	if s.notif == nil {
		return
	}
	send := func(rows []repository.Booking, ownerMsg, driverMsg string) {
		for _, row := range rows {
			b, err := s.q.GetBookingByID(ctx, row.ID)
			if err != nil {
				continue
			}
			label := bookingLabel(b)
			s.notif.Notify(b.UserId, "BOOKING_CANCELLED", "Booking hangus",
				label+" — "+ownerMsg, map[string]any{"bookingId": b.ID})
			s.notifyDriver(ctx, b.AssignedDriverId, "BOOKING_CANCELLED", "Booking hangus",
				label+" — "+driverMsg, b.ID)
		}
	}
	send(expired,
		"sudah disetujui tetapi tidak dimulai sampai waktu selesai",
		"tidak dimulai sampai waktu selesai; Anda tidak lagi bertugas")
	send(ignored,
		"tidak mendapat respons admin sampai waktu selesai",
		"tidak diproses admin; Anda tidak jadi bertugas")
}

// ── Penjaga booking ganda ───────────────────────────────────────────────────

// errDuplicateBooking: request yang sama terkirim lagi (tombol ditekan
// beruntun / dikirim ulang) — booking yang sudah ada tidak dibuat ulang.
func errDuplicateBooking(id int32) error {
	return util.NewError(409,
		fmt.Sprintf("Booking yang sama sudah Anda ajukan (#%d) - tidak dibuat ulang", id),
		util.ErrConflict)
}

// insertBookingOnce menolak booking AKTIF yang identik (user, resource, jam
// mulai & selesai sama), lalu menyimpan. Cek + insert berada dalam satu
// transaksi dengan advisory lock per user, sehingga dua request serentak
// tidak sama-sama lolos.
// bookingLocs: lokasi penjemputan & tujuan (hanya booking kendaraan).
type bookingLocs struct{ pickup, destination sql.NullString }

func bookingLocations(isVehicle bool, pickup, destination string) bookingLocs {
	if !isVehicle {
		return bookingLocs{}
	}
	ns := func(s string) sql.NullString {
		s = strings.TrimSpace(s)
		return sql.NullString{String: s, Valid: s != ""}
	}
	return bookingLocs{pickup: ns(pickup), destination: ns(destination)}
}

func (s *BookingService) insertBookingOnce(ctx context.Context, p repository.CreateBookingParams, locs bookingLocs) (repository.Booking, error) {
	check := func(q repository.ExtendedQuerier) error {
		id, err := q.FindActiveDuplicateBooking(ctx, p.UserId, p.ResourceId, p.StartDate, p.EndDate)
		if err != nil {
			return err
		}
		if id > 0 {
			return errDuplicateBooking(id)
		}
		return nil
	}
	if s.db == nil { // unit test (mock querier)
		if err := check(s.q); err != nil {
			return repository.Booking{}, err
		}
		b, err := s.q.CreateBooking(ctx, p)
		if err == nil && (locs.pickup.Valid || locs.destination.Valid) {
			err = s.q.SetBookingLocations(ctx, b.ID, locs.pickup, locs.destination)
		}
		return b, err
	}
	var b repository.Booking
	err := withTx(ctx, s.db, func(q *repository.Queries) error {
		if err := q.LockBookingCreate(ctx, p.UserId); err != nil {
			return err
		}
		if err := check(q); err != nil {
			return err
		}
		var err error
		b, err = q.CreateBooking(ctx, p)
		if err != nil {
			return err
		}
		if locs.pickup.Valid || locs.destination.Valid {
			return q.SetBookingLocations(ctx, b.ID, locs.pickup, locs.destination)
		}
		return nil
	})
	return b, err
}

// driverTripLabel: label booking + lokasi penjemputan (bila ada) untuk
// notifikasi tugas supir.
func driverTripLabel(b repository.GetBookingByIDRow) string {
	l := bookingLabel(b)
	if b.PickupLocation.Valid && b.PickupLocation.String != "" {
		l += " · Jemput: " + b.PickupLocation.String
	}
	return l
}

// ── Mulai lebih awal (Pengaturan) ───────────────────────────────────────────

// Kunci master_settings (migrasi 000018) + nilai bawaan bila belum diatur.
const (
	settingStartEarlySPD    = "booking_start_early_minutes_spd"
	settingStartEarlyNonSPD = "booking_start_early_minutes_non_spd"
	settingStartEarlyRoom   = "booking_start_early_minutes_room"
)

var startEarlyDefaults = map[string]int{
	settingStartEarlySPD:    180,
	settingStartEarlyNonSPD: 15,
	settingStartEarlyRoom:   30,
}

// startEarlyMinutes: berapa menit sebelum jadwal booking ini boleh dimulai.
// Ruangan punya setting sendiri; kendaraan dibedakan SPD / Non-SPD.
func (s *BookingService) startEarlyMinutes(ctx context.Context, b repository.GetBookingByIDRow) int {
	key := settingStartEarlyNonSPD
	switch {
	case b.ResourceType == repository.ResourceTypeROOM:
		key = settingStartEarlyRoom
	case b.BookingType == repository.BookingTypeSPD:
		key = settingStartEarlySPD
	}
	mins := startEarlyDefaults[key]
	if m, err := s.q.GetMasterSettingByKey(ctx, key); err == nil {
		if v := int(util.ParseStringToFloat64(m.Value)); v >= 0 {
			mins = v
		}
	}
	if mins > 24*60 { // batas wajar: maksimal 1 hari lebih awal
		mins = 24 * 60
	}
	return mins
}

// startableFrom: waktu paling awal booking ini boleh dimulai.
func (s *BookingService) startableFrom(ctx context.Context, b repository.GetBookingByIDRow) time.Time {
	return b.StartDate.Add(-time.Duration(s.startEarlyMinutes(ctx, b)) * time.Minute)
}
