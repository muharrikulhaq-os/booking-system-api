package service

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"booking-system-api/internal/repository"
	ws "booking-system-api/internal/websocket"
)

// Pelaku uji:
//
//	pemohon  = user 100
//	supir P  = driver 1 / user 11, supir tetap mobil A
//	supir R  = driver 2 / user 22, tanpa kendaraan tetap
//	supir Q  = driver 3 / user 33, supir tetap mobil B
//	supir S  = driver 4 / user 44, supir tetap mobil C
var (
	carA = vehicleBrief{id: 10, label: "Avanza (B 1 AA)", ownerID: sql.NullInt32{Int32: 1, Valid: true}, ownerUserID: 11, ownerName: "P"}
	carB = vehicleBrief{id: 20, label: "Innova (B 2 BB)", ownerID: sql.NullInt32{Int32: 3, Valid: true}, ownerUserID: 33, ownerName: "Q"}
	carC = vehicleBrief{id: 30, label: "Xenia (B 3 CC)", ownerID: sql.NullInt32{Int32: 4, Valid: true}, ownerUserID: 44, ownerName: "S"}
	carD = vehicleBrief{id: 40, label: "Hiace (B 4 DD)"} // tanpa supir tetap
)

func baseChange() assignChange {
	return assignChange{
		bookingID:     5,
		requesterID:   100,
		schedule:      "Sen, 6 Okt 08:00 WIB",
		prevTripLabel: "LAMA",
		newTripLabel:  "BARU",
	}
}

func withPrevDriver(c assignChange, id, userID int32, name string) assignChange {
	c.prevDriverID = sql.NullInt32{Int32: id, Valid: true}
	c.prevDriverUserID = userID
	c.prevDriverName = name
	return c
}

func withNewDriver(c assignChange, id, userID int32, name string) assignChange {
	c.newDriverID, c.newDriverUserID, c.newDriverName = id, userID, name
	return c
}

// byUser: notifikasi per penerima; gagal bila satu orang dapat >1 pesan.
func byUser(t *testing.T, ns []assignNotice) map[int32]assignNotice {
	t.Helper()
	out := map[int32]assignNotice{}
	for _, n := range ns {
		if _, dup := out[n.userID]; dup {
			t.Fatalf("user %d menerima lebih dari satu notifikasi: %+v", n.userID, ns)
		}
		out[n.userID] = n
	}
	return out
}

func expectNotice(t *testing.T, got map[int32]assignNotice, userID int32, notifType string, contains ...string) {
	t.Helper()
	n, ok := got[userID]
	if !ok {
		t.Fatalf("user %d tidak menerima notifikasi (%s)", userID, notifType)
	}
	if n.notifType != notifType {
		t.Errorf("user %d: tipe = %s, mau %s", userID, n.notifType, notifType)
	}
	for _, s := range contains {
		if !strings.Contains(n.title+" | "+n.body, s) {
			t.Errorf("user %d: %q tidak memuat %q", userID, n.title+" | "+n.body, s)
		}
	}
}

func expectRecipients(t *testing.T, got map[int32]assignNotice, users ...int32) {
	t.Helper()
	if len(got) != len(users) {
		t.Errorf("jumlah penerima = %d, mau %d (%v)", len(got), len(users), got)
	}
	for _, u := range users {
		if _, ok := got[u]; !ok {
			t.Errorf("user %d tidak menerima notifikasi", u)
		}
	}
}

func TestAssignNotices_FirstAssignmentOwnCar(t *testing.T) {
	// Booking mobil A, ditugaskan P (pemiliknya) dengan mobil A.
	c := withNewDriver(baseChange(), 1, 11, "P")
	c.prevVehicle, c.newVehicle = &carA, carA
	got := byUser(t, c.notices())
	expectRecipients(t, got, 11)
	expectNotice(t, got, 11, "NEW_BOOKING", "Penugasan baru", "BARU")
	if strings.Contains(got[11].body, "milik") {
		t.Errorf("supir memakai mobil sendiri tidak perlu catatan pemilik: %q", got[11].body)
	}
}

func TestAssignNotices_FirstAssignmentOtherCar(t *testing.T) {
	// Pemohon minta mobil A; admin menugaskan P dengan mobil B (milik Q).
	c := withNewDriver(baseChange(), 1, 11, "P")
	c.prevVehicle, c.newVehicle = &carA, carB
	c.newDriverOwnPlate = "B 1 AA"
	got := byUser(t, c.notices())
	expectRecipients(t, got, 100, 11, 33)
	expectNotice(t, got, 100, notifBookingReassigned, "Kendaraan dialihkan", "dari Avanza (B 1 AA) ke Innova (B 2 BB)", "Supir: P")
	expectNotice(t, got, 11, "NEW_BOOKING", "Kendaraan ini milik Q", "Kendaraan tetap Anda (B 1 AA) tidak dipakai")
	expectNotice(t, got, 33, notifVehicleLent, "Kendaraan Anda dipakai supir lain", "Innova (B 2 BB) akan dikemudikan P")
	if _, ok := got[33].data["bookingId"]; ok {
		t.Error("notifikasi pemilik mobil tidak boleh membawa bookingId (supir lain tidak bisa membuka booking)")
	}
}

func TestAssignNotices_DriverSwappedSameCar(t *testing.T) {
	// P memegang mobil A miliknya; admin mengganti supir menjadi R, mobil tetap A.
	c := withPrevDriver(withNewDriver(baseChange(), 2, 22, "R"), 1, 11, "P")
	c.prevVehicle, c.prevVehicleAssigned, c.newVehicle = &carA, true, carA
	got := byUser(t, c.notices())
	expectRecipients(t, got, 100, 22, 11)
	expectNotice(t, got, 100, notifBookingReassigned, "Supir diganti", "menjadi R", "kendaraan tetap Avanza (B 1 AA)")
	expectNotice(t, got, 22, "NEW_BOOKING", "Anda menggantikan P", "Kendaraan ini milik P")
	// P: satu pesan saja — tugas dilepas + mobilnya tetap dipakai R.
	expectNotice(t, got, 11, notifAssignmentReleased, "dialihkan ke R", "Kendaraan Anda (Avanza (B 1 AA)) tetap dipakai oleh R")
}

func TestAssignNotices_CarSwappedSameDriver(t *testing.T) {
	// P tetap supirnya; mobil dialihkan dari A (milik P) ke B (milik Q).
	c := withPrevDriver(withNewDriver(baseChange(), 1, 11, "P"), 1, 11, "P")
	c.prevVehicle, c.prevVehicleAssigned, c.newVehicle = &carA, true, carB
	c.newDriverOwnPlate = "B 1 AA"
	got := byUser(t, c.notices())
	expectRecipients(t, got, 100, 11, 33)
	expectNotice(t, got, 100, notifBookingReassigned, "Kendaraan dialihkan")
	expectNotice(t, got, 11, notifBookingReassigned, "Kendaraan tugas dialihkan", "dari Avanza (B 1 AA) ke Innova (B 2 BB)", "milik Q")
	expectNotice(t, got, 33, notifVehicleLent, "dikemudikan P")
}

func TestAssignNotices_BothSwapped(t *testing.T) {
	// P di mobil A → R di mobil B (milik Q).
	c := withPrevDriver(withNewDriver(baseChange(), 2, 22, "R"), 1, 11, "P")
	c.prevVehicle, c.prevVehicleAssigned, c.newVehicle = &carA, true, carB
	got := byUser(t, c.notices())
	expectRecipients(t, got, 100, 22, 11, 33)
	expectNotice(t, got, 100, notifBookingReassigned, "Kendaraan & supir dialihkan", "Innova (B 2 BB) dengan supir R")
	expectNotice(t, got, 22, "NEW_BOOKING", "menggantikan P", "milik Q")
	expectNotice(t, got, 11, notifAssignmentReleased, "dialihkan ke R")
	if strings.Contains(got[11].body, "tetap dipakai") {
		t.Errorf("mobil P tidak dipakai lagi, jangan bilang tetap dipakai: %q", got[11].body)
	}
	expectNotice(t, got, 33, notifVehicleLent, "dikemudikan R")
}

func TestAssignNotices_PreviousCarOwnerReleased(t *testing.T) {
	// R (tanpa mobil tetap) memakai mobil C milik S; dialihkan ke mobil D (tanpa pemilik).
	c := withPrevDriver(withNewDriver(baseChange(), 2, 22, "R"), 2, 22, "R")
	c.prevVehicle, c.prevVehicleAssigned, c.newVehicle = &carC, true, carD
	got := byUser(t, c.notices())
	expectRecipients(t, got, 100, 22, 44)
	expectNotice(t, got, 22, notifBookingReassigned, "dari Xenia (B 3 CC) ke Hiace (B 4 DD)")
	expectNotice(t, got, 44, notifVehicleLent, "tidak jadi dipakai", "Xenia (B 3 CC) batal dipakai")
}

func TestAssignNotices_RequestedCarOwnerNotToldWhenNeverAssigned(t *testing.T) {
	// Mobil C hanya DIMINTA pemohon (belum pernah ditugaskan) — pemiliknya
	// belum pernah diberi tahu, jadi tidak perlu kabar "batal dipakai".
	c := withNewDriver(baseChange(), 2, 22, "R")
	c.prevVehicle, c.newVehicle = &carC, carD
	got := byUser(t, c.notices())
	expectRecipients(t, got, 100, 22)
}

func TestAssignNotices_NoChange(t *testing.T) {
	// Penugasan ulang yang sama: hanya supir yang diingatkan.
	c := withPrevDriver(withNewDriver(baseChange(), 2, 22, "R"), 2, 22, "R")
	c.prevVehicle, c.prevVehicleAssigned, c.newVehicle = &carB, true, carB
	got := byUser(t, c.notices())
	expectRecipients(t, got, 22)
	expectNotice(t, got, 22, "NEW_BOOKING", "Penugasan kendaraan")
}

// assignQuerier: data kendaraan & supir untuk buildAssignChange.
type assignQuerier struct {
	*notifQuerier
	vehicles map[int32]repository.GetVehicleByIDRow
	byRes    map[int32]int32 // resourceId → vehicleId
	drvRows  map[int32]repository.GetDriverByIDRow
}

func (m *assignQuerier) GetVehicleByID(ctx context.Context, id int32) (repository.GetVehicleByIDRow, error) {
	v, ok := m.vehicles[id]
	if !ok {
		return v, sql.ErrNoRows
	}
	return v, nil
}

func (m *assignQuerier) GetVehicleIDByResourceID(ctx context.Context, resourceID int32) (int32, error) {
	id, ok := m.byRes[resourceID]
	if !ok {
		return 0, sql.ErrNoRows
	}
	return id, nil
}

func (m *assignQuerier) GetVehicleByFixedDriverID(ctx context.Context, driverID int32) (repository.Vehicle, error) {
	for _, v := range m.vehicles {
		if v.FixedDriverId.Valid && v.FixedDriverId.Int32 == driverID {
			return repository.Vehicle{ID: v.ID, PlateNumber: v.PlateNumber}, nil
		}
	}
	return repository.Vehicle{}, sql.ErrNoRows
}

func (m *assignQuerier) GetDriverByID(ctx context.Context, id int32) (repository.GetDriverByIDRow, error) {
	d, ok := m.drvRows[id]
	if !ok {
		return d, sql.ErrNoRows
	}
	return d, nil
}

func TestNotifyAssignment_LoadsNamesAndOwners(t *testing.T) {
	// Pemohon (user 100) minta mobil A (resource 501, milik P); admin
	// menugaskan R dengan mobil B (milik Q). Data diambil dari querier.
	fixed := func(id int32) sql.NullInt32 { return sql.NullInt32{Int32: id, Valid: true} }
	m := &assignQuerier{
		notifQuerier: &notifQuerier{MockQuerier: &MockQuerier{}},
		vehicles: map[int32]repository.GetVehicleByIDRow{
			10: {ID: 10, ResourceId: 501, ResourceName: "Avanza", PlateNumber: "B 1 AA", FixedDriverId: fixed(1)},
			20: {ID: 20, ResourceId: 502, ResourceName: "Innova", PlateNumber: "B 2 BB", FixedDriverId: fixed(3)},
		},
		byRes: map[int32]int32{501: 10, 502: 20},
		drvRows: map[int32]repository.GetDriverByIDRow{
			1: {ID: 1, UserId: 11, UserName: "P"},
			2: {ID: 2, UserId: 22, UserName: "R"},
			3: {ID: 3, UserId: 33, UserName: "Q"},
		},
	}
	svc := &BookingService{q: m, notif: &NotificationService{q: m, hub: ws.NewHub()}}
	start := time.Date(2026, 10, 6, 1, 0, 0, 0, time.UTC)
	before := repository.GetBookingByIDRow{ID: 5, UserId: 100, ResourceId: 501, ResourceName: "Avanza", StartDate: start}
	after := before
	after.ResourceId, after.ResourceName = 502, "Innova"
	after.PlateNumber = sql.NullString{String: "B 2 BB", Valid: true}

	svc.notifyAssignment(context.Background(), before, after, m.drvRows[2], 20)

	got := map[int32]repository.CreateNotificationParams{}
	for _, n := range m.sent {
		got[n.UserID] = n
	}
	if len(got) != 3 {
		t.Fatalf("penerima = %d, mau 3 (pemohon, R, Q): %+v", len(got), m.sent)
	}
	if b := got[100].Body; !strings.Contains(b, "dari Avanza (B 1 AA) ke Innova (B 2 BB)") || !strings.Contains(b, "Supir: R") {
		t.Errorf("pesan pemohon = %q", b)
	}
	if b := got[22].Body; !strings.Contains(b, "Kendaraan ini milik Q") {
		t.Errorf("pesan supir R = %q", b)
	}
	if n := got[33]; n.Type != notifVehicleLent || !strings.Contains(n.Body, "dikemudikan R") {
		t.Errorf("pesan pemilik Q = %+v", n)
	}
	if got[33].RelatedEntityID.Valid {
		t.Error("notifikasi pemilik mobil tidak boleh menautkan booking")
	}
}
