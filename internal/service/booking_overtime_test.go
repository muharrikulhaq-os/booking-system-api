package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"booking-system-api/internal/repository"
)

func TestOvertimeMinutes_Threshold(t *testing.T) {
	end := time.Date(2026, 10, 5, 9, 30, 0, 0, time.UTC) // 16:30 WIB
	cases := []struct {
		name      string
		back      time.Duration
		threshold int
		want      int32
	}{
		{"kembali tepat waktu", 0, 0, 0},
		{"kembali lebih awal", -30 * time.Minute, 0, 0},
		{"ambang 0: tiap kelebihan dihitung", 90 * time.Minute, 0, 90},
		{"belum melebihi ambang 2 jam", 90 * time.Minute, 120, 0},
		{"tepat di ambang belum dihitung", 120 * time.Minute, 120, 0},
		{"melebihi ambang: dihitung penuh sejak jadwal selesai", 150 * time.Minute, 120, 150},
	}
	for _, tc := range cases {
		if got := overtimeMinutes(end, end.Add(tc.back), tc.threshold); got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestOvertimeRule_Settings(t *testing.T) {
	vehicle := func(bt repository.BookingType) repository.GetBookingByIDRow {
		return repository.GetBookingByIDRow{
			ResourceType:     repository.ResourceTypeVEHICLE,
			BookingType:      bt,
			AssignedDriverId: sql.NullInt32{Int32: 7, Valid: true},
		}
	}
	ctx := context.Background()
	svc := func(settings map[string]string) *BookingService {
		return &BookingService{q: &settingsQuerier{MockQuerier: &MockQuerier{}, settings: settings}}
	}
	cases := []struct {
		name      string
		settings  map[string]string
		bt        repository.BookingType
		wantApply bool
		wantMins  int
	}{
		{"default Non-SPD: ambang 0", nil, repository.BookingTypeNONSPD, true, 0},
		{"default SPD: tidak dihitung", nil, repository.BookingTypeSPD, false, 0},
		{"ambang Non-SPD 1.5 jam", map[string]string{settingOvertimeThresholdNonSPD: "1.5000"}, repository.BookingTypeNONSPD, true, 90},
		{"SPD aktif, ikut Non-SPD", map[string]string{
			settingOvertimeSPDEnabled: "1.0000", settingOvertimeThresholdNonSPD: "2", settingOvertimeThresholdSPD: "5",
		}, repository.BookingTypeSPD, true, 120},
		{"SPD aktif, aturan sendiri", map[string]string{
			settingOvertimeSPDEnabled: "1", settingOvertimeSPDSameAsNonSPD: "0",
			settingOvertimeThresholdNonSPD: "2", settingOvertimeThresholdSPD: "5",
		}, repository.BookingTypeSPD, true, 300},
		{"ambang Non-SPD tidak berlaku ke SPD nonaktif", map[string]string{settingOvertimeThresholdNonSPD: "2"}, repository.BookingTypeSPD, false, 0},
	}
	for _, tc := range cases {
		apply, mins := svc(tc.settings).overtimeRule(ctx, vehicle(tc.bt))
		if apply != tc.wantApply || mins != tc.wantMins {
			t.Errorf("%s: got (%v, %d), want (%v, %d)", tc.name, apply, mins, tc.wantApply, tc.wantMins)
		}
	}

	// Ruangan & kendaraan tanpa supir tidak pernah kena lembur.
	room := repository.GetBookingByIDRow{ResourceType: repository.ResourceTypeROOM, BookingType: repository.BookingTypeNONSPD}
	if apply, _ := svc(nil).overtimeRule(ctx, room); apply {
		t.Error("ruangan tidak boleh kena lembur")
	}
	noDriver := vehicle(repository.BookingTypeNONSPD)
	noDriver.AssignedDriverId = sql.NullInt32{}
	if apply, _ := svc(nil).overtimeRule(ctx, noDriver); apply {
		t.Error("kendaraan tanpa supir tidak boleh kena lembur")
	}
}

// Complete memakai pengaturan: SPD aktif → lembur tercatat; di bawah ambang → tidak.
func TestComplete_OvertimeFollowsSettings(t *testing.T) {
	run := func(bt repository.BookingType, late time.Duration, settings map[string]string) *repository.CreateDriverOvertimeParams {
		end := time.Now().Add(-5 * time.Hour)
		m := &MockQuerier{booking: repository.GetBookingByIDRow{
			ID: 9, Status: repository.BookingStatusRETURNED,
			ResourceType:     repository.ResourceTypeVEHICLE,
			BookingType:      bt,
			AssignedDriverId: sql.NullInt32{Int32: 7, Valid: true},
			EndDate:          end,
			ReturnedAt:       sql.NullTime{Time: end.Add(late), Valid: true},
		}}
		svc := &BookingService{q: &settingsQuerier{MockQuerier: m, settings: settings}}
		if _, err := svc.Complete(context.Background(), 9, AuditActor{UserID: 1}, "ADMIN"); err != nil {
			t.Fatalf("Complete: %v", err)
		}
		return m.createdOvertime
	}

	if ot := run(repository.BookingTypeSPD, 3*time.Hour, nil); ot != nil {
		t.Errorf("SPD default tidak boleh lembur, tercatat %d menit", ot.OvertimeMinutes)
	}
	if ot := run(repository.BookingTypeSPD, 3*time.Hour, map[string]string{settingOvertimeSPDEnabled: "1"}); ot == nil || ot.OvertimeMinutes < 179 || ot.OvertimeMinutes > 181 {
		t.Errorf("SPD aktif: lembur ~180 menit, got %+v", ot)
	}
	if ot := run(repository.BookingTypeNONSPD, 50*time.Minute, map[string]string{settingOvertimeThresholdNonSPD: "1"}); ot != nil {
		t.Errorf("di bawah ambang 1 jam tidak boleh lembur, tercatat %d menit", ot.OvertimeMinutes)
	}
	if ot := run(repository.BookingTypeNONSPD, 80*time.Minute, map[string]string{settingOvertimeThresholdNonSPD: "1"}); ot == nil || ot.OvertimeMinutes < 79 || ot.OvertimeMinutes > 81 {
		t.Errorf("melebihi ambang: lembur penuh ~80 menit, got %+v", ot)
	}
}
