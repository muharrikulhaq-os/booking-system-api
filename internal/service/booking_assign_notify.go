package service

import (
	"context"
	"database/sql"

	"booking-system-api/internal/repository"
)

// Notifikasi Tugaskan / Alihkan (AssignVehicle).
//
// Admin bebas memilih supir & kendaraan mana pun, jadi satu penugasan bisa
// menyentuh beberapa orang sekaligus. Masing-masing mendapat pesan yang
// sesuai posisinya:
//
//   - pemohon            → kendaraan/supir booking-nya berubah
//   - supir pemegang     → tugas baru, atau mobil tugasnya dialihkan
//   - supir yang diganti → tugasnya dilepas
//   - pemilik mobil      → mobilnya (supir tetap) dipakai supir lain,
//                          atau batal dipakai karena booking dialihkan
//
// Supir yang bukan pemegang booking tidak bisa membuka detailnya (403), jadi
// notifikasi untuk supir yang diganti & pemilik mobil sengaja tanpa
// bookingId — info saja, tidak membuka halaman apa pun.

const (
	notifBookingReassigned  = "BOOKING_REASSIGNED"
	notifAssignmentReleased = "ASSIGNMENT_RELEASED"
	notifVehicleLent        = "VEHICLE_LENT"
)

// vehicleBrief: kendaraan seperti yang disebut di teks notifikasi.
type vehicleBrief struct {
	id          int32
	label       string // "Avanza (B 1234 CD)"
	ownerID     sql.NullInt32
	ownerUserID int32 // akun supir tetap; 0 = tidak ada
	ownerName   string
}

// assignChange: keadaan sebelum & sesudah satu AssignVehicle.
type assignChange struct {
	bookingID     int32
	requesterID   int32
	schedule      string // "Sen, 6 Okt 08:00 WIB"
	prevTripLabel string // label booking sebelum dialihkan
	newTripLabel  string // label booking sesudah (+ lokasi jemput)

	// Kendaraan sebelumnya: yang sudah ditugaskan, atau yang diminta
	// pemohon bila belum ada penugasan. nil bila tidak diketahui.
	prevVehicle         *vehicleBrief
	prevVehicleAssigned bool // prevVehicle berasal dari penugasan (bukan sekadar permintaan)
	newVehicle          vehicleBrief

	prevDriverID     sql.NullInt32
	prevDriverUserID int32
	prevDriverName   string
	newDriverID      int32
	newDriverUserID  int32
	newDriverName    string
	// Plat kendaraan tetap supir baru bila BUKAN kendaraan tugas ini.
	newDriverOwnPlate string
}

type assignNotice struct {
	userID    int32
	notifType string
	title     string
	body      string
	data      map[string]any
}

func (c assignChange) vehicleChanged() bool {
	return c.prevVehicle != nil && c.prevVehicle.id != c.newVehicle.id
}

func (c assignChange) driverChanged() bool {
	return c.prevDriverID.Valid && c.prevDriverID.Int32 != c.newDriverID
}

// borrowed: kendaraan tugas punya supir tetap, tapi bukan supir yang ditugaskan.
func (c assignChange) borrowed() bool {
	return c.newVehicle.ownerID.Valid && c.newVehicle.ownerID.Int32 != c.newDriverID
}

// notices menyusun semua notifikasi untuk satu penugasan.
func (c assignChange) notices() []assignNotice {
	var out []assignNotice
	vehicleChanged, driverChanged := c.vehicleChanged(), c.driverChanged()
	newcomer := !c.prevDriverID.Valid || driverChanged
	booking := map[string]any{"bookingId": c.bookingID, "vehicleId": c.newVehicle.id}
	nv := c.newVehicle.label

	// ── Pemohon ──
	switch {
	case vehicleChanged && driverChanged:
		out = append(out, assignNotice{c.requesterID, notifBookingReassigned,
			"Kendaraan & supir dialihkan",
			"Booking Anda (" + c.schedule + ") dialihkan ke " + nv + " dengan supir " + c.newDriverName + ".",
			booking})
	case vehicleChanged:
		out = append(out, assignNotice{c.requesterID, notifBookingReassigned,
			"Kendaraan dialihkan",
			"Kendaraan booking Anda (" + c.schedule + ") dialihkan dari " + c.prevVehicle.label +
				" ke " + nv + ". Supir: " + c.newDriverName + ".",
			booking})
	case driverChanged:
		out = append(out, assignNotice{c.requesterID, notifBookingReassigned,
			"Supir diganti",
			"Supir booking Anda (" + c.schedule + ") diganti menjadi " + c.newDriverName +
				", kendaraan tetap " + nv + ".",
			booking})
	}

	// ── Supir pemegang booking ──
	ownerNote := ""
	if c.borrowed() {
		ownerNote = " Kendaraan ini milik " + c.newVehicle.ownerName + "."
	}
	ownNote := ""
	if c.newDriverOwnPlate != "" {
		ownNote = " Kendaraan tetap Anda (" + c.newDriverOwnPlate + ") tidak dipakai di tugas ini."
	}
	switch {
	case newcomer:
		body := c.newTripLabel + " — Anda ditugaskan."
		if driverChanged {
			body += " Anda menggantikan " + c.prevDriverName + "."
		}
		out = append(out, assignNotice{c.newDriverUserID, "NEW_BOOKING", "Penugasan baru",
			body + ownerNote + ownNote, booking})
	case vehicleChanged:
		out = append(out, assignNotice{c.newDriverUserID, notifBookingReassigned,
			"Kendaraan tugas dialihkan",
			"Mobil untuk tugas Anda (" + c.schedule + ") dialihkan dari " + c.prevVehicle.label +
				" ke " + nv + "." + ownerNote + ownNote,
			booking})
	default: // penugasan ulang tanpa perubahan
		out = append(out, assignNotice{c.newDriverUserID, "NEW_BOOKING", "Penugasan kendaraan",
			c.newTripLabel + " — Anda ditugaskan", booking})
	}

	// ── Supir yang digantikan ──
	if driverChanged && c.prevDriverUserID != 0 {
		body := c.prevTripLabel + " — dialihkan ke " + c.newDriverName + ", Anda tidak jadi bertugas."
		if c.borrowed() && c.newVehicle.ownerID.Int32 == c.prevDriverID.Int32 {
			body += " Kendaraan Anda (" + nv + ") tetap dipakai oleh " + c.newDriverName + "."
		}
		out = append(out, assignNotice{c.prevDriverUserID, notifAssignmentReleased,
			"Penugasan dipindahkan", body, map[string]any{"vehicleId": c.newVehicle.id}})
	}

	// ── Pemilik kendaraan tugas (dipakai supir lain) ──
	// Hanya bila ada yang berubah — penugasan ulang yang sama tidak diulang.
	// Pemilik yang juga supir lama sudah diberi tahu lewat pesan di atas.
	if c.borrowed() && c.newVehicle.ownerUserID != 0 && (vehicleChanged || newcomer) &&
		!(c.prevDriverID.Valid && c.prevDriverID.Int32 == c.newVehicle.ownerID.Int32) {
		out = append(out, assignNotice{c.newVehicle.ownerUserID, notifVehicleLent,
			"Kendaraan Anda dipakai supir lain",
			nv + " akan dikemudikan " + c.newDriverName + " pada " + c.schedule +
				" karena penugasan dari admin.",
			map[string]any{"vehicleId": c.newVehicle.id}})
	}

	// ── Pemilik kendaraan lama (batal dipakai) ──
	if vehicleChanged && c.prevVehicleAssigned {
		pv := c.prevVehicle
		involved := pv.ownerID.Valid && (pv.ownerID.Int32 == c.newDriverID ||
			(c.prevDriverID.Valid && pv.ownerID.Int32 == c.prevDriverID.Int32))
		if pv.ownerUserID != 0 && !involved {
			out = append(out, assignNotice{pv.ownerUserID, notifVehicleLent,
				"Kendaraan Anda tidak jadi dipakai",
				pv.label + " batal dipakai pada " + c.schedule + " — booking dialihkan ke kendaraan lain.",
				map[string]any{"vehicleId": pv.id}})
		}
	}
	return out
}

// ── Pengumpulan data ────────────────────────────────────────────────────────

func vehicleLabel(name, plate string) string {
	if plate == "" {
		return name
	}
	return name + " (" + plate + ")"
}

// loadVehicleBrief mengambil kendaraan + supir tetapnya (bila ada).
func (s *BookingService) loadVehicleBrief(ctx context.Context, vehicleID int32) (*vehicleBrief, bool) {
	v, err := s.q.GetVehicleByID(ctx, vehicleID)
	if err != nil {
		return nil, false
	}
	vb := &vehicleBrief{
		id:      v.ID,
		label:   vehicleLabel(v.ResourceName, v.PlateNumber),
		ownerID: v.FixedDriverId,
	}
	if v.FixedDriverId.Valid {
		if d, err := s.q.GetDriverByID(ctx, v.FixedDriverId.Int32); err == nil {
			vb.ownerUserID = d.UserId
			vb.ownerName = d.UserName
		}
		if vb.ownerName == "" && v.FixedDriverName.Valid {
			vb.ownerName = v.FixedDriverName.String
		}
	}
	return vb, true
}

// buildAssignChange merangkum perubahan AssignVehicle untuk notifikasi.
// before = booking sebelum diubah, after = sesudahnya.
func (s *BookingService) buildAssignChange(ctx context.Context, before, after repository.GetBookingByIDRow,
	newDriver repository.GetDriverByIDRow, newVehicleID int32) (assignChange, bool) {
	nv, ok := s.loadVehicleBrief(ctx, newVehicleID)
	if !ok {
		return assignChange{}, false
	}
	c := assignChange{
		bookingID:       after.ID,
		requesterID:     after.UserId,
		schedule:        formatWIB(after.StartDate),
		prevTripLabel:   bookingLabel(before),
		newTripLabel:    driverTripLabel(after),
		newVehicle:      *nv,
		prevDriverID:    before.AssignedDriverId,
		newDriverID:     newDriver.ID,
		newDriverUserID: newDriver.UserId,
		newDriverName:   newDriver.UserName,
	}

	// Kendaraan sebelumnya: yang ditugaskan, atau yang diminta pemohon.
	if before.AssignedVehicleId.Valid {
		c.prevVehicle, _ = s.loadVehicleBrief(ctx, before.AssignedVehicleId.Int32)
		c.prevVehicleAssigned = c.prevVehicle != nil
	} else if vid, err := s.q.GetVehicleIDByResourceID(ctx, before.ResourceId); err == nil {
		c.prevVehicle, _ = s.loadVehicleBrief(ctx, vid)
	}

	if before.AssignedDriverId.Valid {
		if d, err := s.q.GetDriverByID(ctx, before.AssignedDriverId.Int32); err == nil {
			c.prevDriverUserID = d.UserId
			c.prevDriverName = d.UserName
		}
	}
	if own, err := s.q.GetVehicleByFixedDriverID(ctx, newDriver.ID); err == nil && own.ID != newVehicleID {
		c.newDriverOwnPlate = own.PlateNumber
	}
	return c, true
}

// notifyAssignment mengirim semua notifikasi satu AssignVehicle.
func (s *BookingService) notifyAssignment(ctx context.Context, before, after repository.GetBookingByIDRow,
	newDriver repository.GetDriverByIDRow, newVehicleID int32) {
	if s.notif == nil {
		return
	}
	c, ok := s.buildAssignChange(ctx, before, after, newDriver, newVehicleID)
	if !ok {
		// Data kendaraan gagal dimuat — minimal supir tetap tahu tugasnya.
		s.notif.Notify(newDriver.UserId, "NEW_BOOKING", "Penugasan kendaraan",
			bookingLabel(after)+" — Anda ditugaskan", map[string]any{"bookingId": after.ID})
		return
	}
	for _, n := range c.notices() {
		s.notif.Notify(n.userID, n.notifType, n.title, n.body, n.data)
	}
}
