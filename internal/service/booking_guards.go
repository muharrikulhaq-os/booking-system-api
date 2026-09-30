package service

import (
	"context"
	"time"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

// bookingStartGrace: toleransi waktu mulai di masa lalu saat membuat booking
// (jeda antara memilih jam di form dan menekan kirim).
const bookingStartGrace = 30 * time.Minute

// requireRoomKeeperOf: penjaga ruangan hanya boleh memulai/menyelesaikan
// booking ruangan yang dijaganya (ST-11), bukan ruangan penjaga lain.
func (s *BookingService) requireRoomKeeperOf(ctx context.Context, userID, resourceID int32) error {
	rk, err := s.q.GetRoomKeeperByUserID(ctx, userID)
	if err != nil || !rk.IsActive {
		return util.ErrForbidden
	}
	keeperID, err := s.q.GetRoomKeeperIDByResourceID(ctx, resourceID)
	if err != nil || !keeperID.Valid || keeperID.Int32 != rk.ID {
		return util.NewError(403, "Anda bukan penjaga ruangan ini", util.ErrForbidden)
	}
	return nil
}

// checkResourceBookable menolak resource yang dinonaktifkan admin (B16).
func (s *BookingService) checkResourceBookable(ctx context.Context, resourceID int32) error {
	r, err := s.q.GetResourceByID(ctx, resourceID)
	if err != nil {
		return util.ErrNotFound
	}
	if r.Status == repository.ResourceStatusINACTIVE {
		return util.NewError(409, "resource ini sedang nonaktif dan tidak bisa dibooking", util.ErrConflict)
	}
	return nil
}
