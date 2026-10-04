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

// notifQuerier menambahkan pencatatan notifikasi ke MockQuerier.
type notifQuerier struct {
	*MockQuerier
	drivers map[int32]int32 // driverID → userID akun supir
	admins  []int32
	sent    []repository.CreateNotificationParams
}

func (m *notifQuerier) CreateNotification(ctx context.Context, arg repository.CreateNotificationParams) (repository.Notification, error) {
	m.sent = append(m.sent, arg)
	return repository.Notification{ID: int32(len(m.sent))}, nil
}

func (m *notifQuerier) GetDriverByID(ctx context.Context, id int32) (repository.GetDriverByIDRow, error) {
	uid, ok := m.drivers[id]
	if !ok {
		return repository.GetDriverByIDRow{}, sql.ErrNoRows
	}
	return repository.GetDriverByIDRow{ID: id, UserId: uid}, nil
}

func (m *notifQuerier) ListActiveUserIDsByRole(ctx context.Context, role string) ([]int32, error) {
	if role == "ADMIN" {
		return m.admins, nil
	}
	return nil, nil
}

func (m *notifQuerier) sentTo(userID int32) []repository.CreateNotificationParams {
	var out []repository.CreateNotificationParams
	for _, n := range m.sent {
		if n.UserID == userID {
			out = append(out, n)
		}
	}
	return out
}

func newNotifFixture(b repository.GetBookingByIDRow) (*notifQuerier, *BookingService) {
	m := &notifQuerier{
		MockQuerier: &MockQuerier{booking: b},
		drivers:     map[int32]int32{7: 70, 8: 80},
		admins:      []int32{1},
	}
	notif := &NotificationService{q: m, hub: ws.NewHub()}
	return m, &BookingService{q: m, notif: notif}
}

func TestFormatWIB(t *testing.T) {
	// 01:00 UTC = 08:00 WIB, Selasa 6 Okt 2026.
	got := formatWIB(time.Date(2026, 10, 6, 1, 0, 0, 0, time.UTC))
	if got != "Sel, 6 Okt 08:00 WIB" {
		t.Errorf("formatWIB = %q", got)
	}
}

func TestBookingLabel_WithPlate(t *testing.T) {
	b := repository.GetBookingByIDRow{
		ResourceName: "Avanza",
		PlateNumber:  sql.NullString{String: "B 1234 CD", Valid: true},
		StartDate:    time.Date(2026, 10, 6, 1, 0, 0, 0, time.UTC),
	}
	if got := bookingLabel(b); got != "Avanza (B 1234 CD) · Sel, 6 Okt 08:00 WIB" {
		t.Errorf("bookingLabel = %q", got)
	}
}

func TestNotifyDriver_SendsToDriverUserAccount(t *testing.T) {
	m, svc := newNotifFixture(repository.GetBookingByIDRow{})
	svc.notifyDriver(context.Background(), sql.NullInt32{Int32: 7, Valid: true}, "NEW_BOOKING", "Booking masuk", "x", 5)
	got := m.sentTo(70)
	if len(got) != 1 || got[0].Type != "NEW_BOOKING" || !got[0].RelatedEntityID.Valid || got[0].RelatedEntityID.Int32 != 5 {
		t.Fatalf("notifikasi supir = %+v", m.sent)
	}
	// Tanpa supir → tidak ada notifikasi, tidak panik.
	svc.notifyDriver(context.Background(), sql.NullInt32{}, "NEW_BOOKING", "t", "x", 5)
	svc.notifyDriver(context.Background(), sql.NullInt32{Int32: 99, Valid: true}, "NEW_BOOKING", "t", "x", 5)
	if len(m.sent) != 1 {
		t.Errorf("notifikasi tambahan tak terduga: %d", len(m.sent))
	}
}

// Booking hangus otomatis memberi tahu pemohon & supir (dulu diam-diam).
func TestNotifySweptBookings_OwnerAndDriver(t *testing.T) {
	b := repository.GetBookingByIDRow{
		ID: 3, UserId: 10, ResourceName: "Avanza",
		AssignedDriverId: sql.NullInt32{Int32: 7, Valid: true},
		StartDate:        time.Date(2026, 10, 6, 1, 0, 0, 0, time.UTC),
	}
	m, svc := newNotifFixture(b)
	svc.notifySweptBookings(context.Background(), []repository.Booking{{ID: 3}}, nil)
	if len(m.sentTo(10)) != 1 || len(m.sentTo(70)) != 1 {
		t.Fatalf("pemohon=%d supir=%d, want 1/1", len(m.sentTo(10)), len(m.sentTo(70)))
	}
	if !strings.Contains(m.sentTo(70)[0].Body, "tidak lagi bertugas") {
		t.Errorf("pesan supir = %q", m.sentTo(70)[0].Body)
	}
}

// Pembatalan oleh admin: admin, pemohon, dan supir yang ditugaskan diberi tahu.
func TestCancel_NotifiesAssignedDriver(t *testing.T) {
	b := repository.GetBookingByIDRow{
		ID: 1, UserId: 10, Status: repository.BookingStatusAPPROVED,
		ResourceName:     "Avanza",
		AssignedDriverId: sql.NullInt32{Int32: 7, Valid: true},
		StartDate:        time.Date(2026, 10, 6, 1, 0, 0, 0, time.UTC),
	}
	m, svc := newNotifFixture(b)
	if _, err := svc.Cancel(context.Background(), 1, AuditActor{UserID: 1}, "ADMIN"); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if len(m.sentTo(70)) != 1 {
		t.Fatalf("supir tidak diberi tahu: %+v", m.sent)
	}
	if len(m.sentTo(10)) != 1 {
		t.Errorf("pemohon tidak diberi tahu")
	}
	if !strings.Contains(m.sentTo(70)[0].Body, "Avanza") {
		t.Errorf("pesan supir tanpa nama kendaraan: %q", m.sentTo(70)[0].Body)
	}
}
