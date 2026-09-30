package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

// Complete() menyelaraskan pegangan supir; di fixture tidak ada booking aktif lain.
func (m *MockQuerier) GetDriverHoldVehicleID(ctx context.Context, driverID int32) (int32, error) {
	return 0, sql.ErrNoRows
}

func (m *MockQuerier) GetDriverCurrentAssignment(ctx context.Context, driverID int32) (repository.GetDriverCurrentAssignmentRow, error) {
	return repository.GetDriverCurrentAssignmentRow{}, sql.ErrNoRows
}

func (m *MockQuerier) CancelBooking(ctx context.Context, id int32) (repository.Booking, error) {
	return repository.Booking{ID: id, Status: repository.BookingStatusCANCELLED}, nil
}

// Hanya pemilik booking atau admin yang boleh membatalkan (skenario RL-07).
func TestCancel_OnlyOwnerOrAdmin(t *testing.T) {
	const ownerID = 10
	cases := []struct {
		name    string
		actorID int32
		role    string
		wantErr error
	}{
		{"pemilik (karyawan)", ownerID, "EMPLOYEE", nil},
		{"admin (bukan pemilik)", 99, "ADMIN", nil},
		{"karyawan lain", 11, "EMPLOYEE", util.ErrForbidden},
		{"supir", 12, "DRIVER", util.ErrForbidden},
		{"penjaga ruangan", 13, "ROOM_KEEPER", util.ErrForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := &MockQuerier{booking: repository.GetBookingByIDRow{
				ID: 1, UserId: ownerID, Status: repository.BookingStatusPENDING,
			}}
			svc := &BookingService{q: m}
			_, err := svc.Cancel(context.Background(), 1, AuditActor{UserID: tc.actorID}, tc.role)
			if !errors.Is(err, tc.wantErr) && err != tc.wantErr {
				t.Errorf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}
