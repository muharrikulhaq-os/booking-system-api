package service

import (
	"context"
	"database/sql"
	"errors"

	"booking-system-api/internal/repository"
)

// syncDriverHold menyelaraskan driver_assignments ("supir memegang kendaraan")
// dengan booking aktif supir: kendaraan dari booking yang sedang berjalan,
// atau booking APPROVED paling awal; tanpa booking aktif → dilepas.
//
// Dulu penugasan diubah per alur (approve menambah, complete melepas) dan
// alur lain terlewat — booking EXPIRED, pindah supir/kendaraan, gabung —
// sehingga supir tertahan "sibuk" selamanya (B4/B5). Panggil fungsi ini untuk
// SETIAP supir yang booking aktifnya mungkin berubah.
func syncDriverHold(ctx context.Context, q repository.ExtendedQuerier, driverID int32) {
	if driverID == 0 {
		return
	}
	want, wantErr := q.GetDriverHoldVehicleID(ctx, driverID)
	if wantErr != nil && !errors.Is(wantErr, sql.ErrNoRows) {
		return // gangguan DB: jangan ubah apa pun
	}
	cur, curErr := q.GetDriverCurrentAssignment(ctx, driverID)
	holding := curErr == nil

	if errors.Is(wantErr, sql.ErrNoRows) { // tidak ada booking aktif
		if holding {
			_ = q.ReleaseDriver(ctx, driverID)
		}
		return
	}
	if holding && cur.VehicleId == want {
		return
	}
	_ = q.ReleaseDriver(ctx, driverID)
	_, _ = q.AssignDriverToVehicle(ctx, repository.AssignDriverToVehicleParams{
		DriverId: driverID, VehicleId: want,
	})
}

// syncDriverHolds menjalankan syncDriverHold untuk setiap supir valid, sekali.
func syncDriverHolds(ctx context.Context, q repository.ExtendedQuerier, drivers ...sql.NullInt32) {
	seen := map[int32]bool{}
	for _, d := range drivers {
		if d.Valid && !seen[d.Int32] {
			seen[d.Int32] = true
			syncDriverHold(ctx, q, d.Int32)
		}
	}
}

// hasDriverConflict melaporkan apakah supir sudah ditugaskan di booking aktif
// lain yang jadwalnya bertumpuk dengan b (B7). Pasangan merge b dikecualikan —
// mereka satu perjalanan dengan supir yang sama.
func (s *BookingService) hasDriverConflict(ctx context.Context, driverID int32, b repository.GetBookingByIDRow) (bool, error) {
	ids, err := s.q.ListDriverConflictBookingIDs(ctx, driverID, b.StartDate, b.EndDate, b.ID)
	if err != nil || len(ids) == 0 {
		return false, err
	}
	partners := map[int32]bool{}
	if merges, _ := s.q.GetBookingMerges(ctx, b.ID); len(merges) > 0 {
		for _, m := range merges {
			partners[m.OtherBookingID] = true
		}
	}
	for _, id := range ids {
		if !partners[id] {
			return true, nil
		}
	}
	return false, nil
}
