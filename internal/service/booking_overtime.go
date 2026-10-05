package service

import (
	"context"
	"time"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

// Pengaturan lembur supir (master_settings, migrasi 000019).
//
// Jam akhir kerja = jadwal selesai booking (endDate). Kendaraan yang kembali
// setelah itu dicatat lembur — tapi hanya bila kelebihannya MELEBIHI ambang
// (jam). Ambang hanya syarat: begitu terlewati, lembur dihitung penuh sejak
// jam akhir kerja. Ambang 0 = setiap kelebihan dihitung (perilaku lama).
const (
	settingOvertimeThresholdNonSPD = "overtime_threshold_hours_non_spd"
	settingOvertimeSPDEnabled      = "overtime_spd_enabled"         // 0/1
	settingOvertimeSPDSameAsNonSPD = "overtime_spd_same_as_non_spd" // 0/1
	settingOvertimeThresholdSPD    = "overtime_threshold_hours_spd" // dipakai bila SPD tidak ikut Non-SPD
	maxOvertimeThresholdHours      = 24.0
)

// settingFloat membaca master_settings numerik; def bila belum diatur/rusak.
func (s *BookingService) settingFloat(ctx context.Context, key string, def float64) float64 {
	m, err := s.q.GetMasterSettingByKey(ctx, key)
	if err != nil {
		return def
	}
	return util.ParseStringToFloat64(m.Value)
}

// overtimeRule: apakah booking ini kena hitungan lembur, dan ambangnya (menit).
func (s *BookingService) overtimeRule(ctx context.Context, b repository.GetBookingByIDRow) (bool, int) {
	if b.ResourceType != repository.ResourceTypeVEHICLE || !b.AssignedDriverId.Valid {
		return false, 0
	}
	key := settingOvertimeThresholdNonSPD
	if b.BookingType == repository.BookingTypeSPD {
		// Default: SPD (dinas resmi) tidak dihitung lembur.
		if s.settingFloat(ctx, settingOvertimeSPDEnabled, 0) < 1 {
			return false, 0
		}
		if s.settingFloat(ctx, settingOvertimeSPDSameAsNonSPD, 1) < 1 {
			key = settingOvertimeThresholdSPD
		}
	}
	hours := s.settingFloat(ctx, key, 0)
	if hours < 0 {
		hours = 0
	}
	if hours > maxOvertimeThresholdHours {
		hours = maxOvertimeThresholdHours
	}
	return true, int(hours * 60)
}

// overtimeMinutes: menit lembur antara jam akhir kerja dan waktu kembali —
// 0 bila tidak melebihi ambang.
func overtimeMinutes(scheduledEnd, actualEnd time.Time, thresholdMinutes int) int32 {
	mins := int32(actualEnd.Sub(scheduledEnd).Minutes())
	if mins <= 0 || int(mins) <= thresholdMinutes {
		return 0
	}
	return mins
}
