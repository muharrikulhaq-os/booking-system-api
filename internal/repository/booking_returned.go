package repository

import (
	"context"
	"database/sql"
)

// Hand-written (sqlc CLI unavailable) — migrasi 000018.

// MarkBookingReturned: laporan pengembalian masuk → RETURNED ("Sudah
// Kembali") + catat jam kembali. Bersyarat status supaya laporan ganda /
// booking yang sudah selesai tidak berubah. false bila tidak ada yang diubah.
func (q *Queries) MarkBookingReturned(ctx context.Context, id int32) (bool, error) {
	res, err := q.db.ExecContext(ctx, `
		UPDATE bookings
		SET status = 'RETURNED', "returnedAt" = COALESCE("returnedAt", NOW()), "updatedAt" = NOW()
		WHERE id = $1 AND status IN ('ONGOING', 'OVERDUE')`, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// SetBookingLocations menyimpan lokasi penjemputan & tujuan (booking kendaraan).
func (q *Queries) SetBookingLocations(ctx context.Context, id int32, pickup, destination sql.NullString) error {
	_, err := q.db.ExecContext(ctx,
		`UPDATE bookings SET "pickupLocation" = $2, "destination" = $3 WHERE id = $1`,
		id, pickup, destination)
	return err
}
