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
