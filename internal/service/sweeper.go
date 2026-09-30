package service

import (
	"context"
	"time"
)

// Topik DATA_CHANGED - nilainya sama dengan middleware.Topic*.
const (
	topicBooking = "booking"
	topicVehicle = "vehicle"
	topicDriver  = "driver"
)

// Publisher menyiarkan DATA_CHANGED untuk perubahan yang dibuat SISTEM
// (bukan request tulis), mis. transisi otomatis berbasis waktu - supaya
// web & mobile ikut memuat ulang tanpa polling.
type Publisher func(topics ...string)

func (s *BookingService) SetPublisher(p Publisher) { s.publish = p }
func (s *VehicleService) SetPublisher(p Publisher) { s.publish = p }

// SetBeforeRead: dijalankan sebelum ringkasan dashboard dibaca (mis. sweep
// transisi otomatis) supaya angkanya tidak basi.
func (s *DashboardService) SetBeforeRead(f func(ctx context.Context)) { s.beforeRead = f }

// SweepNow menjalankan transisi otomatis (IGNORED/EXPIRED/OVERDUE + maintenance
// yang tiba waktunya) sekarang juga.
func (s *BookingService) SweepNow(ctx context.Context) { s.sweepStaleBookings(ctx) }

// RunSweeper menjalankan transisi otomatis berkala sampai ctx selesai, supaya
// status berubah tepat waktu walau tidak ada yang membuka aplikasi (B17).
// Siaran hanya dikirim bila memang ada yang berubah.
func (s *BookingService) RunSweeper(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.sweepStaleBookings(ctx)
		}
	}
}

// promoteDueMaintenance: lihat repository.PromoteDueMaintenance; menyiarkan
// perubahan status kendaraan bila ada.
func promoteDueMaintenance(ctx context.Context, q interface {
	PromoteDueMaintenance(ctx context.Context) (int64, error)
}, publish Publisher) int64 {
	n, err := q.PromoteDueMaintenance(ctx)
	if err != nil {
		return 0
	}
	if n > 0 && publish != nil {
		publish(topicVehicle)
	}
	return n
}
