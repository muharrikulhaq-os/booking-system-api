package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Penjaga data ganda dari tombol yang ditekan beruntun / request yang
// terkirim ulang (lapis terakhir — klien web & mobile juga menjaga).

// LockBookingCreate mengunci (sampai transaksi selesai) pembuatan booking
// milik [userID], supaya cek duplikat + insert dari request serentak tidak
// saling mendahului. Harus dipanggil di dalam transaksi.
func (q *Queries) LockBookingCreate(ctx context.Context, userID int32) error {
	_, err := q.db.ExecContext(ctx,
		`SELECT pg_advisory_xact_lock(hashtext('booking:create'), $1)`, userID)
	return err
}

// FindActiveDuplicateBooking: booking AKTIF milik user yang sama untuk
// resource & jam yang persis sama. 0 bila tidak ada.
func (q *Queries) FindActiveDuplicateBooking(ctx context.Context, userID, resourceID int32, start, end time.Time) (int32, error) {
	var id int32
	err := q.db.QueryRowContext(ctx, `
		SELECT id FROM bookings
		WHERE "userId" = $1 AND "resourceId" = $2
		  AND "startDate" = $3 AND "endDate" = $4
		  AND status IN ('PENDING', 'APPROVED', 'ONGOING', 'OVERDUE')
		ORDER BY id DESC
		LIMIT 1`, userID, resourceID, start, end).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return id, err
}

// FindRecentDuplicateFuel: catatan pengisian (belum dibatalkan) kendaraan
// yang sama dengan odometer, jenis, dan jumlah yang sama dalam [within]
// terakhir. 0 bila tidak ada. Dipanggil di transaksi yang sudah mengunci
// baris kendaraan, jadi aman dari request serentak.
func (q *Queries) FindRecentDuplicateFuel(ctx context.Context, vehicleID, fuelTypeID, odometer int32, quantity float64, within time.Duration) (int32, error) {
	var id int32
	err := q.db.QueryRowContext(ctx, `
		SELECT id FROM fuel_expenses
		WHERE "vehicleId" = $1 AND "fuelTypeId" = $2 AND "odometerAfter" = $3
		  AND quantity = round($4::numeric, 2)
		  AND "voidedAt" IS NULL
		  AND "createdAt" > NOW() - make_interval(secs => $5)
		ORDER BY id DESC
		LIMIT 1`, vehicleID, fuelTypeID, odometer, quantity, within.Seconds()).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return id, err
}
