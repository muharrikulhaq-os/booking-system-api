package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"booking-system-api/internal/repository"
)

// settingsQuerier: MockQuerier + master_settings sederhana.
type settingsQuerier struct {
	*MockQuerier
	settings map[string]string
}

func (m *settingsQuerier) GetMasterSettingByKey(ctx context.Context, key string) (repository.MasterSetting, error) {
	v, ok := m.settings[key]
	if !ok {
		return repository.MasterSetting{}, sql.ErrNoRows
	}
	return repository.MasterSetting{Key: key, Value: v}, nil
}

func TestStartEarlyMinutes_PerType(t *testing.T) {
	vehicle := func(bt repository.BookingType) repository.GetBookingByIDRow {
		return repository.GetBookingByIDRow{ResourceType: repository.ResourceTypeVEHICLE, BookingType: bt}
	}
	room := repository.GetBookingByIDRow{ResourceType: repository.ResourceTypeROOM, BookingType: repository.BookingTypeNONSPD}

	// Belum diatur → nilai bawaan migrasi 000018.
	svc := &BookingService{q: &settingsQuerier{MockQuerier: &MockQuerier{}}}
	ctx := context.Background()
	if got := svc.startEarlyMinutes(ctx, vehicle(repository.BookingTypeSPD)); got != 180 {
		t.Errorf("SPD default = %d, want 180", got)
	}
	if got := svc.startEarlyMinutes(ctx, vehicle(repository.BookingTypeNONSPD)); got != 15 {
		t.Errorf("Non-SPD default = %d, want 15", got)
	}
	if got := svc.startEarlyMinutes(ctx, room); got != 30 {
		t.Errorf("ruangan default = %d, want 30", got)
	}

	// Diatur admin; ruangan memakai setting sendiri walau bookingType NON_SPD.
	svc = &BookingService{q: &settingsQuerier{MockQuerier: &MockQuerier{}, settings: map[string]string{
		settingStartEarlySPD: "120", settingStartEarlyNonSPD: "0", settingStartEarlyRoom: "45",
	}}}
	if got := svc.startEarlyMinutes(ctx, vehicle(repository.BookingTypeSPD)); got != 120 {
		t.Errorf("SPD = %d, want 120", got)
	}
	if got := svc.startEarlyMinutes(ctx, vehicle(repository.BookingTypeNONSPD)); got != 0 {
		t.Errorf("Non-SPD = %d, want 0 (tidak boleh lebih awal)", got)
	}
	if got := svc.startEarlyMinutes(ctx, room); got != 45 {
		t.Errorf("ruangan = %d, want 45", got)
	}
}

// RETURNED bisa diselesaikan admin; lembur dihitung sampai jam laporan
// pengembalian, bukan jam admin menekan Selesaikan.
func TestComplete_Returned_OvertimeUntilReturnTime(t *testing.T) {
	end := time.Now().Add(-3 * time.Hour)
	returned := end.Add(40 * time.Minute) // kembali 40 menit lewat jadwal
	m := &MockQuerier{booking: repository.GetBookingByIDRow{
		ID: 9, Status: repository.BookingStatusRETURNED,
		ResourceType:     repository.ResourceTypeVEHICLE,
		BookingType:      repository.BookingTypeNONSPD,
		AssignedDriverId: sql.NullInt32{Int32: 7, Valid: true},
		EndDate:          end,
		ReturnedAt:       sql.NullTime{Time: returned, Valid: true},
	}}
	svc := &BookingService{q: m}
	if _, err := svc.Complete(context.Background(), 9, AuditActor{UserID: 1}, "ADMIN"); err != nil {
		t.Fatalf("Complete(RETURNED): %v", err)
	}
	if m.createdOvertime == nil {
		t.Fatal("lembur tidak tercatat")
	}
	if got := m.createdOvertime.OvertimeMinutes; got < 39 || got > 41 {
		t.Errorf("lembur = %d menit, want ~40 (sampai jam laporan, bukan sekarang ~180)", got)
	}
}
