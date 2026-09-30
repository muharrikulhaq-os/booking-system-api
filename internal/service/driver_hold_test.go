package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"booking-system-api/internal/repository"
)

// holdFake meniru bagian ExtendedQuerier yang dipakai syncDriverHold.
type holdFake struct {
	repository.ExtendedQuerier
	want     *int32 // kendaraan dari booking aktif; nil = tidak ada booking aktif
	cur      *int32 // penugasan terbuka saat ini; nil = kosong
	released int
	assigned []int32
}

func (f *holdFake) GetDriverHoldVehicleID(ctx context.Context, driverID int32) (int32, error) {
	if f.want == nil {
		return 0, sql.ErrNoRows
	}
	return *f.want, nil
}

func (f *holdFake) GetDriverCurrentAssignment(ctx context.Context, driverID int32) (repository.GetDriverCurrentAssignmentRow, error) {
	if f.cur == nil {
		return repository.GetDriverCurrentAssignmentRow{}, sql.ErrNoRows
	}
	return repository.GetDriverCurrentAssignmentRow{DriverId: driverID, VehicleId: *f.cur}, nil
}

func (f *holdFake) ReleaseDriver(ctx context.Context, driverID int32) error {
	f.released++
	f.cur = nil
	return nil
}

func (f *holdFake) AssignDriverToVehicle(ctx context.Context, arg repository.AssignDriverToVehicleParams) (repository.DriverAssignment, error) {
	f.assigned = append(f.assigned, arg.VehicleId)
	v := arg.VehicleId
	f.cur = &v
	return repository.DriverAssignment{DriverId: arg.DriverId, VehicleId: arg.VehicleId, AssignedAt: time.Now()}, nil
}

func ptr(v int32) *int32 { return &v }

func TestSyncDriverHold(t *testing.T) {
	cases := []struct {
		name         string
		want, cur    *int32
		wantReleased int
		wantAssigned []int32
	}{
		{"booking hangus/selesai → supir dilepas (B4)", nil, ptr(5), 1, nil},
		{"tidak ada booking & tidak memegang → tidak ada aksi", nil, nil, 0, nil},
		{"baru disetujui → memegang kendaraan booking", ptr(5), nil, 1, []int32{5}},
		{"sudah memegang kendaraan yang benar → tidak ada aksi", ptr(5), ptr(5), 0, nil},
		{"pindah kendaraan → lepas lama, pegang baru (B5)", ptr(7), ptr(5), 1, []int32{7}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &holdFake{want: tc.want, cur: tc.cur}
			syncDriverHold(context.Background(), f, 9)
			if f.released != tc.wantReleased || len(f.assigned) != len(tc.wantAssigned) ||
				(len(tc.wantAssigned) > 0 && f.assigned[0] != tc.wantAssigned[0]) {
				t.Errorf("released=%d assigned=%v, want released=%d assigned=%v",
					f.released, f.assigned, tc.wantReleased, tc.wantAssigned)
			}
		})
	}
}
